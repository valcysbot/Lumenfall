package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/valcysbot/lumenfall/server/world"
)

const (
	worldModule  = "world"
	defaultMap   = "clairiere"
	tickRate     = 10            // le monde « respire » 10 fois par seconde
	saveEvery    = 30 * tickRate // sauvegarde des positions toutes les 30 s
	statsEvery   = 60 * tickRate // statistiques de charge toutes les 60 s
	maxMsgTick   = 6             // messages acceptés par joueur et par tick
	chatMaxLen   = 120           // caractères par message
	chatCooldown = 1 * tickRate  // un message par seconde
	pathCooldown = 2             // un calcul de trajet tous les 2 ticks
	pathMaxNodes = 2500
)

// Codes des messages échangés pendant la partie.
const (
	opMoveDir  = 1  // client → serveur : {"d":[dx,dy],"r":course}
	opMoveTo   = 2  // client → serveur : {"t":[x,y],"r":course}
	opChat     = 3  // client → serveur : {"m":"Bonjour !"}
	opEmote    = 4  // client → serveur : {"e":"salut"}
	opSnapshot = 10 // serveur → client : état complet à l'arrivée
	opDelta    = 11 // serveur → client : héros qui ont bougé
	opJoin     = 12 // serveur → client : un héros arrive
	opLeave    = 13 // serveur → client : un héros part
	opChatOut  = 14 // serveur → client : message de chat
	opNotice   = 15 // serveur → client : message système
	opEmoteOut = 16 // serveur → client : {"s":sid,"e":"salut"}
)

// Les émotes connues ; les autres sont ignorées.
var emotes = map[string]bool{"salut": true}

var worldMaps = []string{defaultMap}
var worldMapsByID = map[string]*world.Map{}

func init() {
	for _, id := range worldMaps {
		m, err := world.LoadMap(id)
		if err != nil {
			panic(err)
		}
		worldMapsByID[id] = m
	}
}

// worlds associe chaque carte à sa partie Nakama.
var worlds = &worldRegistry{ids: map[string]string{}}

type worldRegistry struct {
	sync.RWMutex
	ids map[string]string
}

func (r *worldRegistry) set(mapID, matchID string) { r.Lock(); r.ids[mapID] = matchID; r.Unlock() }
func (r *worldRegistry) get(mapID string) (string, bool) {
	r.RLock()
	defer r.RUnlock()
	id, ok := r.ids[mapID]
	return id, ok
}

type player struct {
	sid       int
	presence  runtime.Presence
	hero      Hero
	body      world.Body
	lastChat  int64
	lastPath  int64
	lastEmote int64
}

type worldState struct {
	m       *world.Map
	players map[string]*player // par identifiant de session
	pending map[string]Hero    // héros validés en attente d'arrivée
	nextSID int
	// Mesure de charge.
	loopTime time.Duration
	loops    int
	peak     int
}

type worldMatch struct{}

func newWorldMatch(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule) (runtime.Match, error) {
	return &worldMatch{}, nil
}

func (w *worldMatch) MatchInit(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, params map[string]interface{}) (interface{}, int, string) {
	mapID, _ := params["map"].(string)
	m := worldMapsByID[mapID]
	label, _ := json.Marshal(map[string]string{"kind": "world", "map": mapID})
	return &worldState{m: m, players: map[string]*player{}, pending: map[string]Hero{}, nextSID: 1}, tickRate, string(label)
}

func (w *worldMatch) MatchJoinAttempt(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, d runtime.MatchDispatcher, tick int64, state interface{}, p runtime.Presence, meta map[string]string) (interface{}, bool, string) {
	s := state.(*worldState)
	h, err := loadHero(ctx, db, p.GetUserId(), meta["hero_id"])
	if err != nil {
		return s, false, "Ce héros n'existe pas sur ton compte."
	}
	if h.Map != s.m.ID {
		return s, false, "Ce héros se trouve dans une autre région."
	}
	s.pending[p.GetSessionId()] = h
	return s, true, ""
}

func (w *worldMatch) MatchJoin(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, d runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	s := state.(*worldState)
	for _, p := range presences {
		h, ok := s.pending[p.GetSessionId()]
		delete(s.pending, p.GetSessionId())
		if !ok {
			continue
		}
		// Un seul héros en ligne par compte : l'ancienne session est renvoyée.
		for _, other := range s.players {
			if other.presence.GetUserId() == p.GetUserId() {
				_ = d.BroadcastMessage(opNotice, mustJSON(map[string]string{"m": "Ton compte vient de se connecter ailleurs."}), []runtime.Presence{other.presence}, nil, true)
				_ = d.MatchKick([]runtime.Presence{other.presence})
				w.removePlayer(ctx, logger, db, d, s, other.presence.GetSessionId())
			}
		}
		pl := &player{sid: s.nextSID, presence: p, hero: h, body: world.Body{X: h.X, Y: h.Y, Dir: 2}}
		s.nextSID++
		if !s.m.Free(pl.body.X, pl.body.Y) {
			pl.body.X, pl.body.Y = s.m.Spawn[0], s.m.Spawn[1]
		}
		s.players[p.GetSessionId()] = pl

		everyone := make([]map[string]interface{}, 0, len(s.players))
		for _, o := range s.players {
			everyone = append(everyone, describe(o))
		}
		_ = d.BroadcastMessage(opSnapshot, mustJSON(map[string]interface{}{
			"you": pl.sid, "map": s.m.ID, "tick_rate": tickRate, "players": everyone,
		}), []runtime.Presence{p}, nil, true)
		// Attention : une liste vide voudrait dire « tout le monde » pour Nakama.
		if others := s.others(p.GetSessionId()); len(others) > 0 {
			_ = d.BroadcastMessage(opJoin, mustJSON(describe(pl)), others, nil, true)
		}
		if n := len(s.players); n > s.peak {
			s.peak = n
		}
	}
	return s
}

func (w *worldMatch) MatchLeave(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, d runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	s := state.(*worldState)
	for _, p := range presences {
		w.removePlayer(ctx, logger, db, d, s, p.GetSessionId())
	}
	return s
}

func (w *worldMatch) removePlayer(ctx context.Context, logger runtime.Logger, db *sql.DB, d runtime.MatchDispatcher, s *worldState, sessionID string) {
	pl, ok := s.players[sessionID]
	if !ok {
		return
	}
	delete(s.players, sessionID)
	if err := saveHeroes(ctx, db, s.m.ID, []heroPos{{pl.hero.ID, pl.body.X, pl.body.Y}}); err != nil {
		logger.Error("Sauvegarde de %q au départ: %v", pl.hero.Name, err)
	}
	_ = d.BroadcastMessage(opLeave, mustJSON(map[string]int{"s": pl.sid}), nil, nil, true)
}

func (w *worldMatch) MatchLoop(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, d runtime.MatchDispatcher, tick int64, state interface{}, messages []runtime.MatchData) interface{} {
	s := state.(*worldState)
	start := time.Now()

	counts := map[string]int{}
	for _, msg := range messages {
		pl, ok := s.players[msg.GetSessionId()]
		if !ok {
			continue
		}
		counts[msg.GetSessionId()]++
		if counts[msg.GetSessionId()] > maxMsgTick {
			continue // trop de messages : on ignore le surplus
		}
		w.handle(d, s, pl, tick, msg.GetOpCode(), msg.GetData())
	}

	// Simulation des déplacements.
	dt := 1.0 / tickRate
	changed := make(map[*player]bool)
	all := make([]*player, 0, len(s.players))
	bodies := make([]*world.Body, 0, len(s.players))
	for _, pl := range s.players {
		all = append(all, pl)
		bodies = append(bodies, &pl.body)
	}
	crowd := world.NewCrowd(bodies)
	for _, pl := range all {
		if pl.body.Moving() && pl.body.StepAmong(s.m, crowd, dt) {
			changed[pl] = true
		}
	}
	// Les héros se bousculent mais ne se traversent pas.
	for _, i := range world.Separate(s.m, bodies) {
		changed[all[i]] = true
	}
	moved := make([][4]int, 0, len(changed))
	for pl := range changed {
		moved = append(moved, [4]int{pl.sid, cent(pl.body.X), cent(pl.body.Y), pl.body.Dir})
	}
	if len(moved) > 0 {
		_ = d.BroadcastMessage(opDelta, mustJSON(map[string]interface{}{"t": tick, "p": moved}), nil, nil, false)
	}

	if tick%saveEvery == 0 && len(s.players) > 0 {
		ps := make([]heroPos, 0, len(s.players))
		for _, pl := range s.players {
			ps = append(ps, heroPos{pl.hero.ID, pl.body.X, pl.body.Y})
		}
		if err := saveHeroes(ctx, db, s.m.ID, ps); err != nil {
			logger.Error("Sauvegarde périodique: %v", err)
		}
	}

	s.loopTime += time.Since(start)
	s.loops++
	if tick%statsEvery == 0 {
		if len(s.players) > 0 || s.peak > 0 {
			logger.Info("Monde %s : %d héros (pic %d), tick moyen %.2f ms sur %d ms disponibles",
				s.m.ID, len(s.players), s.peak, float64(s.loopTime.Microseconds())/float64(s.loops)/1000, 1000/tickRate)
		}
		s.loopTime, s.loops, s.peak = 0, 0, len(s.players)
	}
	return s
}

func (w *worldMatch) handle(d runtime.MatchDispatcher, s *worldState, pl *player, tick, op int64, data []byte) {
	switch op {
	case opMoveDir:
		var in struct {
			D [2]float64 `json:"d"`
			R bool       `json:"r"`
		}
		if json.Unmarshal(data, &in) == nil {
			pl.body.SetInput(in.D[0], in.D[1])
			pl.body.Run = in.R
		}
	case opMoveTo:
		var in struct {
			T [2]float64 `json:"t"`
			R bool       `json:"r"`
		}
		if json.Unmarshal(data, &in) != nil || tick-pl.lastPath < pathCooldown {
			return
		}
		pl.lastPath = tick
		tx, ty := in.T[0], in.T[1]
		if math.IsNaN(tx) || math.IsNaN(ty) || tx < 0 || ty < 0 || tx >= float64(s.m.Width) || ty >= float64(s.m.Height) {
			return
		}
		if p := s.m.FindPath(pl.body.X, pl.body.Y, tx, ty, pathMaxNodes); p != nil {
			pl.body.SetPath(p)
			pl.body.Run = in.R
		}
	case opChat:
		var in struct {
			M string `json:"m"`
		}
		if json.Unmarshal(data, &in) != nil || tick-pl.lastChat < chatCooldown {
			return
		}
		text := cleanChat(in.M)
		if text == "" {
			return
		}
		pl.lastChat = tick
		_ = d.BroadcastMessage(opChatOut, mustJSON(map[string]interface{}{"s": pl.sid, "n": pl.hero.Name, "m": text}), nil, nil, true)
	case opEmote:
		var in struct {
			E string `json:"e"`
		}
		if json.Unmarshal(data, &in) != nil || !emotes[in.E] || tick-pl.lastEmote < chatCooldown {
			return
		}
		pl.lastEmote = tick
		_ = d.BroadcastMessage(opEmoteOut, mustJSON(map[string]interface{}{"s": pl.sid, "e": in.E}), nil, nil, true)
	}
}

// cleanChat retire les caractères invisibles et coupe les messages trop longs.
func cleanChat(s string) string {
	if !utf8.ValidString(s) {
		return ""
	}
	var b strings.Builder
	n := 0
	for _, c := range s {
		if unicode.IsControl(c) || unicode.Is(unicode.Cf, c) {
			c = ' '
		}
		if n >= chatMaxLen {
			break
		}
		b.WriteRune(c)
		n++
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func (w *worldMatch) MatchTerminate(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, d runtime.MatchDispatcher, tick int64, state interface{}, grace int) interface{} {
	s := state.(*worldState)
	ps := make([]heroPos, 0, len(s.players))
	for _, pl := range s.players {
		ps = append(ps, heroPos{pl.hero.ID, pl.body.X, pl.body.Y})
	}
	if err := saveHeroes(ctx, db, s.m.ID, ps); err != nil {
		logger.Error("Sauvegarde à l'arrêt: %v", err)
	}
	return s
}

func (w *worldMatch) MatchSignal(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, d runtime.MatchDispatcher, tick int64, state interface{}, data string) (interface{}, string) {
	return state, ""
}

func (s *worldState) others(sessionID string) []runtime.Presence {
	out := make([]runtime.Presence, 0, len(s.players))
	for id, pl := range s.players {
		if id != sessionID {
			out = append(out, pl.presence)
		}
	}
	return out
}

func describe(pl *player) map[string]interface{} {
	return map[string]interface{}{
		"s": pl.sid, "n": pl.hero.Name, "c": pl.hero.Color,
		"x": cent(pl.body.X), "y": cent(pl.body.Y), "d": pl.body.Dir,
	}
}

// cent transmet les positions en centièmes de case, pour des messages courts.
func cent(v float64) int { return int(math.Round(v * 100)) }

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
