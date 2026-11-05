package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/valcysbot/lumenfall/server/world"
)

// L'Acte VI dans le monde : le groupe, les canaux de discussion, les amis, les
// comptes ignorés et les signalements, les guildes, le drapeau de guerre, le
// Cercle des Braves, les hauts faits et le tableau d'honneur. Le client envoie
// opSocial avec une action ; le serveur répond par opSocialOut avec un genre.

const (
	opSocial    = 46 // client → serveur : {"a":"inviter","t":sid}, {"a":"ami","n":"Nom"}…
	opSocialOut = 47 // serveur → toi : {"k":"groupe"|"amis"|"guilde"|"cercle"|…, …}

	saidKeep      = 20 // messages gardés par compte (pour les signalements)
	boardEvery    = 60 * time.Second
	guildInvTime  = 60 * tickRate // ticks pour répondre à une invitation de guilde
	groupResEvery = tickRate      // la ressource des alliés, chaque seconde
	honneurReach  = 6.0           // cases autour du tableau d'honneur
	fiacreID      = "fiacre"      // le registre des guildes
)

// Les canaux de discussion.
var chatChannels = map[string]bool{"region": true, "groupe": true, "guilde": true, "commerce": true, "faction": true, "chuchot": true}

// Les émotes connues (danser : la danse du peuple du héros).
func init() {
	for _, e := range []string{"rire", "pleurer", "applaudir", "assis", "dormir", "montrer", "incliner", "danser",
		"acclamer", "hausser", "colere", "coeur"} {
		emotes[e] = true
	}
}

// La saison 1 du Cercle commence le lundi 2 novembre 2026 ; une saison dure 4 semaines.
var seasonStart = time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)

const seasonLength = 28 * 24 * time.Hour

func seasonAt(t time.Time) int {
	if t.Before(seasonStart) {
		return 1
	}
	return int(t.Sub(seasonStart)/seasonLength) + 1
}

func seasonEnd(n int) time.Time { return seasonStart.Add(time.Duration(n) * seasonLength) }

type chatLine struct {
	T int64  `json:"t"`
	C string `json:"c"`
	M string `json:"m"`
	N string `json:"n"`
}

type guildInvite struct {
	gid  int64
	from string
	left int64
}

// social : ce que le match garde pour l'Acte VI.
type social struct {
	guilds   map[int64]*world.Guilde
	guildInv map[int]guildInvite // invitation reçue, par héros
	said     map[string][]chatLine
	board    map[string]interface{}
	boardAt  time.Time
	season   int // la dernière saison vérifiée
}

func newSocial() *social {
	return &social{guilds: map[int64]*world.Guilde{}, guildInv: map[int]guildInvite{}, said: map[string][]chatLine{}}
}

func (pl *player) account() string { return pl.presence.GetUserId() }

func (s *worldState) bySID(sid int) *player {
	for _, pl := range s.players {
		if pl.sid == sid {
			return pl
		}
	}
	return nil
}

func (s *worldState) byName(name string) *player {
	k := world.NameKey(world.NormalizeName(name))
	for _, pl := range s.players {
		if world.NameKey(pl.hero.Name) == k {
			return pl
		}
	}
	return nil
}

func (s *worldState) byAccount(uid string) *player {
	for _, pl := range s.players {
		if pl.account() == uid {
			return pl
		}
	}
	return nil
}

func (s *worldState) byHeroID(id string) *player {
	for _, pl := range s.players {
		if pl.hero.ID == id {
			return pl
		}
	}
	return nil
}

func sendSocial(d runtime.MatchDispatcher, pl *player, k string, v map[string]interface{}) {
	if v == nil {
		v = map[string]interface{}{}
	}
	v["k"] = k
	_ = d.BroadcastMessage(opSocialOut, mustJSON(v), []runtime.Presence{pl.presence}, nil, true)
}

// ignores : x ignore-t-il le compte de y ?
func (s *worldState) ignores(x, y *player) bool {
	return x != nil && y != nil && x.ignored[y.account()]
}

// --- L'arrivée et le départ ---------------------------------------------------------

// socialJoin : un héros arrive. On lit ses comptes ignorés, sa guilde, ses amis,
// et ses amis en ligne apprennent qu'il est là.
func (s *worldState) socialJoin(ctx context.Context, logger runtime.Logger, db *sql.DB, d runtime.MatchDispatcher, pl *player) {
	pl.ignored = map[string]bool{}
	if rows, err := db.QueryContext(ctx, `SELECT ignored FROM lf_ignores WHERE user_id = $1`, pl.account()); err == nil {
		for rows.Next() {
			var u string
			if rows.Scan(&u) == nil {
				pl.ignored[u] = true
			}
		}
		rows.Close()
	}
	if pl.hero.guilde != 0 {
		if g := s.loadGuild(ctx, logger, db, pl.hero.guilde); g != nil {
			s.setGuild(pl, g, pl.hero.rang)
		} else {
			pl.hero.guilde = 0
		}
	}
	s.arena.CountMaxSID(pl.sid, "niveau", pl.fighter.Level)
	friends := s.sendFriends(ctx, db, d, pl)
	for _, f := range friends {
		if o := s.byAccount(f); o != nil && o != pl {
			sendSocial(d, o, "connexion", map[string]interface{}{"n": pl.hero.Name, "u": pl.account()})
			s.sendFriends(ctx, db, d, o)
		}
	}
	if pl.guild != nil {
		g := pl.guild
		g.PruneAgenda(time.Now().UnixMilli())
		if g.Motd != "" {
			sendSocial(d, pl, "motd", map[string]interface{}{"g": g.Nom, "m": g.Motd})
		}
		for _, o := range s.players {
			if o != pl && o.guild == g {
				sendSocial(d, o, "connexion", map[string]interface{}{"n": pl.hero.Name, "g": true})
			}
		}
	}
	sendSocial(d, pl, "gloire", world.GloireView(pl.fighter.Hero.Gloire))
	sendSocial(d, pl, "saison", map[string]interface{}{"n": seasonAt(time.Now()), "fin": seasonEnd(seasonAt(time.Now())).UnixMilli()})
}

// socialLeave : un héros part. Ses amis l'apprennent, sa guilde est sauvegardée.
func (s *worldState) socialLeave(ctx context.Context, logger runtime.Logger, db *sql.DB, d runtime.MatchDispatcher, pl *player) {
	delete(s.social.guildInv, pl.sid)
	for _, o := range s.players {
		if o.friends[pl.account()] {
			s.sendFriends(ctx, db, d, o)
		}
	}
	if g := pl.guild; g != nil {
		s.saveGuild(ctx, logger, db, g)
		still := false
		for _, o := range s.players {
			if o.guild == g {
				still = true
			}
		}
		if !still {
			delete(s.social.guilds, g.ID)
		}
	}
}

// --- Les messages -------------------------------------------------------------------

// chat : un message sur un canal.
func (s *worldState) chat(d runtime.MatchDispatcher, pl *player, tick int64, data []byte) {
	var in struct {
		M string `json:"m"`
		C string `json:"c"`
		A string `json:"a"`
	}
	if json.Unmarshal(data, &in) != nil || tick-pl.lastChat < chatCooldown {
		return
	}
	text := cleanChat(in.M)
	if text == "" {
		return
	}
	c := in.C
	if !chatChannels[c] {
		c = "region"
	}
	var to []*player
	switch c {
	case "region", "commerce":
		for _, o := range s.players {
			to = append(to, o)
		}
	case "faction":
		for _, o := range s.players {
			if o.fighter.Faction() == pl.fighter.Faction() {
				to = append(to, o)
			}
		}
	case "groupe":
		g := s.arena.GroupOf(pl.sid)
		if g == nil {
			refuse(d, pl, "Tu n'es dans aucun groupe.")
			return
		}
		for _, sid := range g.Members {
			if o := s.bySID(sid); o != nil {
				to = append(to, o)
			}
		}
	case "guilde":
		if pl.guild == nil {
			refuse(d, pl, "Tu n'es dans aucune guilde.")
			return
		}
		for _, o := range s.players {
			if o.guild == pl.guild {
				to = append(to, o)
			}
		}
	case "chuchot":
		o := s.byName(in.A)
		if o == nil {
			refuse(d, pl, "Personne de ce nom n'est en ligne.")
			return
		}
		if o == pl {
			refuse(d, pl, "Tu te parles à toi-même ? Braise trouve ça très sain.")
			return
		}
		to = []*player{o, pl}
	}
	pl.lastChat = tick
	msg := map[string]interface{}{"s": pl.sid, "n": pl.hero.Name, "m": text, "c": c, "f": pl.fighter.Faction()}
	if c == "chuchot" {
		msg["a"] = to[0].hero.Name
	}
	if pl.guild != nil {
		msg["g"] = pl.guild.Nom
	}
	var pres []runtime.Presence
	for _, o := range to {
		if o != pl && s.ignores(o, pl) {
			continue
		}
		pres = append(pres, o.presence)
	}
	if len(pres) > 0 {
		_ = d.BroadcastMessage(opChatOut, mustJSON(msg), pres, nil, true)
	}
	said := append(s.social.said[pl.account()], chatLine{T: time.Now().Unix(), C: c, M: text, N: pl.hero.Name})
	if len(said) > saidKeep {
		said = said[len(said)-saidKeep:]
	}
	s.social.said[pl.account()] = said
	s.arena.CountSID(pl.sid, "chat", 1)
}

// emote : une émote, que ceux qui t'ignorent ne voient pas.
func (s *worldState) emote(d runtime.MatchDispatcher, pl *player, e string) {
	var pres []runtime.Presence
	for _, o := range s.players {
		if o == pl || !s.ignores(o, pl) {
			pres = append(pres, o.presence)
		}
	}
	_ = d.BroadcastMessage(opEmoteOut, mustJSON(map[string]interface{}{"s": pl.sid, "e": e}), pres, nil, true)
	if e == "danser" {
		s.arena.CountSID(pl.sid, "danse", 1)
	}
	s.arena.CountSID(pl.sid, "emote", 1)
}

// --- Les actions --------------------------------------------------------------------

func (w *worldMatch) social(d runtime.MatchDispatcher, s *worldState, pl *player, tick int64, data []byte) {
	var in struct {
		A  string          `json:"a"`
		T  int             `json:"t"`
		N  string          `json:"n"`
		U  string          `json:"u"`
		H  string          `json:"h"`
		R  int             `json:"r"`
		I  int             `json:"i"`
		O  int             `json:"o"`
		K  int             `json:"k"`
		F  int             `json:"f"`
		Q  int             `json:"q"`
		M  string          `json:"m"`
		OK bool            `json:"ok"`
		Bl world.Blason    `json:"bl"`
		Rs []world.Rang    `json:"rangs"`
		Ev world.Evenement `json:"ev"`
	}
	if json.Unmarshal(data, &in) != nil {
		return
	}
	ctx, db, logger := s.ctx, s.db, s.logger
	a := s.arena
	sid := pl.sid
	fail := func(err error) {
		if err != nil && err.Error() != "" {
			refuse(d, pl, err.Error())
		}
	}
	target := func() *player {
		if in.T != 0 {
			return s.bySID(in.T)
		}
		if in.N != "" {
			return s.byName(in.N)
		}
		return nil
	}
	switch in.A {
	// Le groupe.
	case "inviter":
		o := target()
		if o == nil {
			refuse(d, pl, "Personne de ce nom n'est en ligne.")
			return
		}
		a.GroupInvite(sid, o.sid)
	case "groupe_rep":
		a.GroupAnswer(sid, in.OK)
	case "quitter":
		a.GroupLeave(sid)
	case "exclure":
		a.GroupKick(sid, in.T)
	case "chef":
		a.GroupLead(sid, in.T)

	// Les amis.
	case "amis":
		s.sendFriends(ctx, db, d, pl)
	case "ami":
		fail(s.askFriend(ctx, db, d, pl, in.N))
	case "ami_rep":
		fail(s.answerFriend(ctx, db, d, pl, in.U, in.OK))
	case "ami_retrait":
		_, _ = db.ExecContext(ctx, `DELETE FROM lf_amis WHERE (a = $1 AND b = $2) OR (a = $2 AND b = $1)`, pl.account(), in.U)
		s.sendFriends(ctx, db, d, pl)
		if o := s.byAccount(in.U); o != nil {
			s.sendFriends(ctx, db, d, o)
		}
	case "ignorer":
		fail(s.ignore(ctx, db, d, pl, in.N))
	case "ignorer_fin":
		_, _ = db.ExecContext(ctx, `DELETE FROM lf_ignores WHERE user_id = $1 AND ignored = $2`, pl.account(), in.U)
		delete(pl.ignored, in.U)
		s.sendFriends(ctx, db, d, pl)
	case "signaler":
		fail(s.report(ctx, logger, db, pl, in.N, in.M))

	// Les guildes.
	case "guilde":
		s.sendGuild(ctx, db, d, pl)
	case "guilde_fonder":
		fail(s.foundGuild(ctx, logger, db, d, pl, in.N, in.Bl))
	case "guilde_inviter":
		o := target()
		fail(s.guildInvite(d, pl, o))
	case "guilde_rep":
		fail(s.guildAnswer(ctx, logger, db, d, pl, in.OK))
	case "guilde_quitter":
		fail(s.guildLeave(ctx, logger, db, d, pl))
	case "guilde_exclure":
		fail(s.guildKick(ctx, logger, db, d, pl, in.H))
	case "guilde_rang":
		fail(s.guildSetRank(ctx, logger, db, d, pl, in.H, in.R))
	case "guilde_rangs", "guilde_motd", "guilde_blason", "guilde_agenda", "guilde_agenda_suppr":
		fail(s.guildEdit(ctx, logger, db, d, pl, in.A, in.Rs, in.M, in.Bl, in.Ev, in.I))
	case "gcoffre", "gcoffre_depot", "gcoffre_retrait", "gcoffre_or":
		if pl.guild == nil {
			refuse(d, pl, "Tu n'es dans aucune guilde.")
			return
		}
		if !s.nearRegistre(pl) {
			refuse(d, pl, "Le coffre des guildes est au registre de Fiacre, sur la place.")
			return
		}
		g := pl.guild
		switch in.A {
		case "gcoffre_depot":
			fail(a.GuildeDepot(sid, in.I, in.O, g))
		case "gcoffre_retrait":
			fail(a.GuildeRetrait(sid, in.O, in.K, pl.rang, g))
		case "gcoffre_or":
			fail(a.GuildeArgent(sid, in.Q, pl.rang, g))
		}
		if in.A != "gcoffre" {
			s.saveGuild(ctx, logger, db, g)
		}
		pl.panel = [2]string{"gcoffre", ""}
		s.sendGuildCoffre(d, pl)

	// Le PvP.
	case "drapeau":
		a.ToggleFlag(sid)
	case "cercle":
		a.Cercle.Join(sid, in.F)
	case "cercle_quitter":
		a.Cercle.Leave(sid)

	// L'intendance de bravoure, au Cercle.
	case "bravoure", "bravoure_achat":
		if c := a.Cercle.Data; c == nil || math.Hypot(pl.body.X-c.Anneau[0], pl.body.Y-c.Anneau[1]) > world.CercleReach {
			refuse(d, pl, "L'intendance de bravoure est au Cercle des Braves, derrière la forge de Doran.")
			return
		}
		if in.A == "bravoure_achat" {
			if err := a.BuyBravoure(sid, in.N); err != nil {
				refuse(d, pl, err.Error())
			}
		}
		sendSocial(d, pl, "bravoure", map[string]interface{}{"liste": world.Bravoure})

	// La gloire.
	case "titre":
		a.SetTitre(sid, in.N)
		sendSocial(d, pl, "gloire", world.GloireView(pl.fighter.Hero.Gloire))
	case "gloire":
		sendSocial(d, pl, "gloire", world.GloireView(pl.fighter.Hero.Gloire))
	case "hauts_faits":
		sendSocial(d, pl, "hauts_faits", map[string]interface{}{"liste": world.HautsFaits})
	case "honneur":
		if c := a.Cercle.Data; c == nil || math.Hypot(pl.body.X-c.Honneur[0], pl.body.Y-c.Honneur[1]) > honneurReach {
			refuse(d, pl, "Le tableau d'honneur est sur la place du village.")
			return
		}
		sendSocial(d, pl, "honneur", s.honorBoard(ctx, logger, db))
	case "rencontre":
		if s.nearNPC(pl, in.N) {
			a.Meet(sid, in.N)
		}
	}
}

// --- Les amis -----------------------------------------------------------------------

// heroAccount : le compte d'un héros, par son nom.
func heroAccount(ctx context.Context, db *sql.DB, name string) (string, string, error) {
	var uid, n string
	err := db.QueryRowContext(ctx, `SELECT user_id, name FROM lf_heroes WHERE name_key = $1`, world.NameKey(world.NormalizeName(name))).Scan(&uid, &n)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", errors.New("Aucun héros d'Erydor ne porte ce nom.")
	}
	return uid, n, err
}

func (s *worldState) askFriend(ctx context.Context, db *sql.DB, d runtime.MatchDispatcher, pl *player, name string) error {
	uid, n, err := heroAccount(ctx, db, name)
	if err != nil {
		return err
	}
	me := pl.account()
	if uid == me {
		return errors.New("C'est un de tes propres héros !")
	}
	if pl.ignored[uid] {
		return errors.New("Tu ignores ce joueur : retire-le d'abord de ta liste des ignorés.")
	}
	var theyIgnore bool
	_ = db.QueryRowContext(ctx, `SELECT true FROM lf_ignores WHERE user_id = $1 AND ignored = $2`, uid, me).Scan(&theyIgnore)
	// Une demande dans l'autre sens existe déjà : vous voilà amis.
	res, err := db.ExecContext(ctx, `UPDATE lf_amis SET etat = 1 WHERE a = $1 AND b = $2 AND etat = 0`, uid, me)
	if err != nil {
		return err
	}
	if k, _ := res.RowsAffected(); k > 0 {
		s.friendsChanged(ctx, db, d, me, uid)
		refuse(d, pl, "🤝 Toi et "+n+" êtes désormais amis !")
		return nil
	}
	if !theyIgnore {
		if _, err := db.ExecContext(ctx, `INSERT INTO lf_amis (a, b) VALUES ($1, $2) ON CONFLICT DO NOTHING`, me, uid); err != nil {
			return err
		}
		if o := s.byAccount(uid); o != nil {
			sendSocial(d, o, "demande_ami", map[string]interface{}{"n": pl.hero.Name, "u": me})
			s.sendFriends(ctx, db, d, o)
		}
	}
	refuse(d, pl, "Demande d'ami envoyée à "+n+".")
	s.sendFriends(ctx, db, d, pl)
	return nil
}

func (s *worldState) answerFriend(ctx context.Context, db *sql.DB, d runtime.MatchDispatcher, pl *player, uid string, ok bool) error {
	me := pl.account()
	if !ok {
		_, err := db.ExecContext(ctx, `DELETE FROM lf_amis WHERE a = $1 AND b = $2 AND etat = 0`, uid, me)
		s.sendFriends(ctx, db, d, pl)
		return err
	}
	res, err := db.ExecContext(ctx, `UPDATE lf_amis SET etat = 1 WHERE a = $1 AND b = $2 AND etat = 0`, uid, me)
	if err != nil {
		return err
	}
	if k, _ := res.RowsAffected(); k == 0 {
		return errors.New("Cette demande n'est plus valable.")
	}
	s.friendsChanged(ctx, db, d, me, uid)
	return nil
}

func (s *worldState) friendsChanged(ctx context.Context, db *sql.DB, d runtime.MatchDispatcher, uids ...string) {
	for _, u := range uids {
		if o := s.byAccount(u); o != nil {
			s.sendFriends(ctx, db, d, o)
		}
	}
}

// sendFriends envoie la liste d'amis (et d'ignorés) ; renvoie les comptes amis.
func (s *worldState) sendFriends(ctx context.Context, db *sql.DB, d runtime.MatchDispatcher, pl *player) []string {
	me := pl.account()
	type entry struct {
		u              string
		etat           int
		incoming       bool
		name, cl, mp   string
		lv             int
		online, ignore bool
	}
	var list []*entry
	rows, err := db.QueryContext(ctx, `
SELECT CASE WHEN a = $1 THEN b ELSE a END, etat, b = $1 FROM lf_amis WHERE a = $1 OR b = $1`, me)
	if err != nil {
		return nil
	}
	for rows.Next() {
		e := &entry{}
		if rows.Scan(&e.u, &e.etat, &e.incoming) == nil {
			list = append(list, e)
		}
	}
	rows.Close()
	for u := range pl.ignored {
		list = append(list, &entry{u: u, ignore: true})
	}
	ids := make([]string, 0, len(list))
	for _, e := range list {
		ids = append(ids, e.u)
	}
	names := map[string]*entry{}
	if len(ids) > 0 {
		r2, err := db.QueryContext(ctx, `
SELECT DISTINCT ON (user_id) user_id, name, level, classe, map FROM lf_heroes
WHERE user_id::text = ANY(string_to_array($1, ',')) ORDER BY user_id, updated_at DESC`, strings.Join(ids, ","))
		if err == nil {
			for r2.Next() {
				e := &entry{}
				if r2.Scan(&e.u, &e.name, &e.lv, &e.cl, &e.mp) == nil {
					names[e.u] = e
				}
			}
			r2.Close()
		}
	}
	pl.friends = map[string]bool{}
	var out []map[string]interface{}
	var friends []string
	nFriends := 0
	for _, e := range list {
		v := map[string]interface{}{"u": e.u}
		if n := names[e.u]; n != nil {
			v["n"], v["lv"], v["cl"], v["r"] = n.name, n.lv, n.cl, regionName(n.mp)
		}
		if o := s.byAccount(e.u); o != nil {
			v["n"], v["lv"], v["cl"], v["r"] = o.hero.Name, o.fighter.Level, o.fighter.Hero.Class.ID, regionName(s.m.ID)
			v["on"] = true
		}
		switch {
		case e.ignore:
			v["ig"] = true
		case e.etat == 1:
			v["ami"] = true
			pl.friends[e.u] = true
			friends = append(friends, e.u)
			nFriends++
		case e.incoming:
			v["demande"] = true
		default:
			v["attente"] = true
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		oi, oj := out[i]["on"] != nil, out[j]["on"] != nil
		if oi != oj {
			return oi
		}
		ni, _ := out[i]["n"].(string)
		nj, _ := out[j]["n"].(string)
		return ni < nj
	})
	sendSocial(d, pl, "amis", map[string]interface{}{"l": out})
	s.arena.CountMaxSID(pl.sid, "amis", nFriends)
	return friends
}

func regionName(mapID string) string {
	if m := worldMapsByID[mapID]; m != nil {
		return m.Name
	}
	return "Erydor"
}

func (s *worldState) ignore(ctx context.Context, db *sql.DB, d runtime.MatchDispatcher, pl *player, name string) error {
	uid, n, err := heroAccount(ctx, db, name)
	if err != nil {
		return err
	}
	if uid == pl.account() {
		return errors.New("Tu ne peux pas t'ignorer toi-même.")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO lf_ignores (user_id, ignored) VALUES ($1, $2) ON CONFLICT DO NOTHING`, pl.account(), uid); err != nil {
		return err
	}
	_, _ = db.ExecContext(ctx, `DELETE FROM lf_amis WHERE (a = $1 AND b = $2) OR (a = $2 AND b = $1)`, pl.account(), uid)
	pl.ignored[uid] = true
	if o := s.byAccount(uid); o != nil {
		if g := s.arena.GroupOf(pl.sid); g != nil && g.Has(o.sid) && g.Leader == pl.sid {
			s.arena.GroupKick(pl.sid, o.sid)
		}
		s.sendFriends(ctx, db, d, o)
	}
	refuse(d, pl, "Tu ignores désormais "+n+" : plus de messages, d'invitations ni de défis.")
	s.sendFriends(ctx, db, d, pl)
	return nil
}

// report garde les derniers messages d'un joueur pour qu'Elowan les lise.
func (s *worldState) report(ctx context.Context, logger runtime.Logger, db *sql.DB, pl *player, name, why string) error {
	uid, n, err := heroAccount(ctx, db, name)
	if err != nil {
		return err
	}
	why = world.CleanText(why, 200)
	if why == "" {
		why = "(sans raison)"
	}
	msgs := s.social.said[uid]
	if msgs == nil {
		msgs = []chatLine{}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO lf_signalements (de, contre, heros, raison, messages) VALUES ($1, $2, $3, $4, $5)`,
		pl.account(), uid, n, why, string(mustJSON(msgs))); err != nil {
		return err
	}
	logger.Warn("Signalement : %q signale %q (%s), %d messages gardés", pl.hero.Name, n, why, len(msgs))
	return errors.New("Merci : ton signalement est enregistré, avec ses derniers messages.")
}

// --- Les guildes --------------------------------------------------------------------

func (s *worldState) loadGuild(ctx context.Context, logger runtime.Logger, db *sql.DB, id int64) *world.Guilde {
	if g := s.social.guilds[id]; g != nil {
		return g
	}
	var raw []byte
	var nom string
	if err := db.QueryRowContext(ctx, `SELECT nom, data FROM lf_guildes WHERE id = $1`, id).Scan(&nom, &raw); err != nil {
		return nil
	}
	g := &world.Guilde{}
	if err := json.Unmarshal(raw, g); err != nil {
		logger.Error("Guilde %d illisible: %v", id, err)
		return nil
	}
	g.ID, g.Nom = id, nom
	g.Fix()
	s.social.guilds[id] = g
	return g
}

func (s *worldState) saveGuild(ctx context.Context, logger runtime.Logger, db *sql.DB, g *world.Guilde) {
	if g == nil || !g.TakeDirty() {
		return
	}
	if _, err := db.ExecContext(ctx, `UPDATE lf_guildes SET data = $2, updated_at = now() WHERE id = $1`, g.ID, string(mustJSON(g))); err != nil {
		logger.Error("Sauvegarde de la guilde %q: %v", g.Nom, err)
		g.Touch()
	}
}

// setGuild : le héros est (ou n'est plus) membre de la guilde g.
func (s *worldState) setGuild(pl *player, g *world.Guilde, rang int) {
	pl.guild, pl.rang = g, rang
	h := pl.fighter.Hero
	if g == nil {
		pl.hero.guilde, h.Guild, h.GuildLv = 0, 0, 0
		return
	}
	pl.hero.guilde, pl.hero.rang = g.ID, rang
	h.Guild, h.GuildLv = int(g.ID), g.Niveau
}

func setHeroGuild(ctx context.Context, db *sql.DB, heroID string, gid int64, rang int) error {
	var v interface{}
	if gid != 0 {
		v = gid
	}
	_, err := db.ExecContext(ctx, `UPDATE lf_heroes SET guilde = $2, rang = $3 WHERE id = $1`, heroID, v, rang)
	return err
}

func (s *worldState) foundGuild(ctx context.Context, logger runtime.Logger, db *sql.DB, d runtime.MatchDispatcher, pl *player, nom string, bl world.Blason) error {
	f := pl.fighter
	switch {
	case pl.guild != nil:
		return errors.New("Tu es déjà dans une guilde.")
	case !s.nearRegistre(pl):
		return errors.New("On fonde une guilde au registre de Fiacre, sur la place.")
	case f.Hero.Gold < world.GuildePrix:
		return errors.New("Fiacre demande " + world.Money(world.GuildePrix) + " pour ouvrir une page de son registre.")
	}
	nom, err := world.ValidateGuildeNom(nom)
	if err != nil {
		return err
	}
	if err := bl.Check(1); err != nil {
		return err
	}
	g := world.NewGuilde(nom, bl)
	var id int64
	err = db.QueryRowContext(ctx, `INSERT INTO lf_guildes (nom, nom_key, data) VALUES ($1, $2, $3) RETURNING id`,
		nom, world.NameKey(nom), string(mustJSON(g))).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "nom_key") {
			return errors.New("Une guilde porte déjà ce nom.")
		}
		logger.Error("Fondation de guilde: %v", err)
		return errors.New("Fiacre a renversé son encrier : réessaie dans un instant.")
	}
	g.ID = id
	if err := setHeroGuild(ctx, db, pl.hero.ID, id, 0); err != nil {
		logger.Error("Fondation de guilde (héros): %v", err)
	}
	f.Hero.Gold -= world.GuildePrix
	s.arena.Ledger.Out["guilde"] += world.GuildePrix
	f.Hero.TouchMe()
	s.social.guilds[id] = g
	s.setGuild(pl, g, 0)
	s.arena.CountSID(pl.sid, "fonde", 1)
	s.arena.CountSID(pl.sid, "guilde", 1)
	logger.Info("Nouvelle guilde %q fondée par %q", nom, pl.hero.Name)
	_ = d.BroadcastMessage(opNotice, mustJSON(map[string]string{"m": "🔔 La cloche de la place sonne : " + pl.hero.Name + " fonde la guilde « " + nom + " » !"}), nil, nil, true)
	_ = d.BroadcastMessage(opJoin, mustJSON(describe(pl)), nil, nil, true)
	s.sendGuild(ctx, db, d, pl)
	return nil
}

func (s *worldState) guildInvite(d runtime.MatchDispatcher, pl, o *player) error {
	g := pl.guild
	switch {
	case g == nil:
		return errors.New("Tu n'es dans aucune guilde.")
	case !g.Can(pl.rang, world.DroitInviter):
		return world.ErrGuildeDroit
	case o == nil:
		return errors.New("Personne de ce nom n'est en ligne.")
	case o == pl:
		return errors.New("Tu es déjà dans ta guilde !")
	case o.guild != nil:
		return errors.New(o.hero.Name + " est déjà dans une guilde.")
	case s.ignores(o, pl):
		return errors.New(o.hero.Name + " n'accepte pas ton invitation.")
	}
	s.social.guildInv[o.sid] = guildInvite{gid: g.ID, from: pl.hero.Name, left: guildInvTime}
	sendSocial(d, o, "invite_guilde", map[string]interface{}{"n": pl.hero.Name, "g": g.Nom, "bl": g.Blason, "sec": guildInvTime / tickRate})
	return errors.New("Tu as invité " + o.hero.Name + " dans « " + g.Nom + " ».")
}

func (s *worldState) guildAnswer(ctx context.Context, logger runtime.Logger, db *sql.DB, d runtime.MatchDispatcher, pl *player, ok bool) error {
	in, has := s.social.guildInv[pl.sid]
	delete(s.social.guildInv, pl.sid)
	if !has || in.left <= 0 {
		if ok {
			return errors.New("Cette invitation n'est plus valable.")
		}
		return nil
	}
	if !ok {
		for _, o := range s.players {
			if o.hero.Name == in.from {
				refuse(d, o, pl.hero.Name+" décline ton invitation.")
			}
		}
		return nil
	}
	if pl.guild != nil {
		return errors.New("Tu es déjà dans une guilde.")
	}
	g := s.loadGuild(ctx, logger, db, in.gid)
	if g == nil {
		return errors.New("Cette guilde n'existe plus.")
	}
	var n int
	_ = db.QueryRowContext(ctx, `SELECT count(*) FROM lf_heroes WHERE guilde = $1`, g.ID).Scan(&n)
	if n >= world.GuildeMax {
		return world.ErrGuildePleine
	}
	if err := setHeroGuild(ctx, db, pl.hero.ID, g.ID, world.GuildeRangs-1); err != nil {
		return err
	}
	s.setGuild(pl, g, world.GuildeRangs-1)
	s.arena.CountSID(pl.sid, "guilde", 1)
	s.guildNotice(d, g, "🏰 "+pl.hero.Name+" rejoint la guilde !")
	_ = d.BroadcastMessage(opJoin, mustJSON(describe(pl)), nil, nil, true)
	s.guildRefresh(ctx, db, d, g)
	return nil
}

func (s *worldState) guildNotice(d runtime.MatchDispatcher, g *world.Guilde, text string) {
	var pres []runtime.Presence
	for _, o := range s.players {
		if o.guild == g {
			pres = append(pres, o.presence)
		}
	}
	if len(pres) > 0 {
		_ = d.BroadcastMessage(opChatOut, mustJSON(map[string]interface{}{"s": 0, "n": "", "m": text, "c": "guilde", "g": g.Nom}), pres, nil, true)
	}
}

// guildRefresh renvoie la fiche de la guilde aux membres en ligne qui l'ont ouverte.
func (s *worldState) guildRefresh(ctx context.Context, db *sql.DB, d runtime.MatchDispatcher, g *world.Guilde) {
	for _, o := range s.players {
		if o.guild == g {
			s.sendGuild(ctx, db, d, o)
		}
	}
}

type guildMember struct {
	id, name, cl, race string
	lv, rang           int
	seen               time.Time
}

func guildMembers(ctx context.Context, db *sql.DB, gid int64) []guildMember {
	rows, err := db.QueryContext(ctx, `SELECT id, name, classe, race, level, rang, updated_at FROM lf_heroes WHERE guilde = $1`, gid)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []guildMember
	for rows.Next() {
		var m guildMember
		if rows.Scan(&m.id, &m.name, &m.cl, &m.race, &m.lv, &m.rang, &m.seen) == nil {
			out = append(out, m)
		}
	}
	return out
}

func (s *worldState) sendGuild(ctx context.Context, db *sql.DB, d runtime.MatchDispatcher, pl *player) {
	g := pl.guild
	if g == nil {
		sendSocial(d, pl, "guilde", map[string]interface{}{"aucune": true, "px": world.GuildePrix, "motifs": world.BlasonMotifs[:world.MotifsAuNiveau(1)]})
		return
	}
	g.PruneAgenda(time.Now().UnixMilli())
	var ms []map[string]interface{}
	for _, m := range guildMembers(ctx, db, g.ID) {
		v := map[string]interface{}{"h": m.id, "n": m.name, "cl": m.cl, "lv": m.lv, "rg": m.rang, "vu": m.seen.UnixMilli()}
		if o := s.byHeroID(m.id); o != nil {
			v["on"], v["lv"], v["r"] = true, o.fighter.Level, regionName(s.m.ID)
		}
		ms = append(ms, v)
	}
	sort.Slice(ms, func(i, j int) bool {
		oi, oj := ms[i]["on"] != nil, ms[j]["on"] != nil
		if oi != oj {
			return oi
		}
		if ms[i]["rg"].(int) != ms[j]["rg"].(int) {
			return ms[i]["rg"].(int) < ms[j]["rg"].(int)
		}
		return ms[i]["n"].(string) < ms[j]["n"].(string)
	})
	sendSocial(d, pl, "guilde", map[string]interface{}{
		"id": g.ID, "nom": g.Nom, "bl": g.Blason, "lv": g.Niveau, "xp": g.XP, "xpn": world.GuildeXP(g.Niveau),
		"motd": g.Motd, "rangs": g.Rangs, "agenda": g.Agenda, "membres": ms, "moi": pl.hero.ID, "rg": pl.rang,
		"max": world.GuildeMax, "motifs": world.BlasonMotifs[:world.MotifsAuNiveau(g.Niveau)], "onglets": world.OngletsAuNiveau(g.Niveau),
		"bonus": g.Niveau >= 2,
	})
}

func (s *worldState) sendGuildCoffre(d runtime.MatchDispatcher, pl *player) {
	g := pl.guild
	if g == nil {
		return
	}
	cl := pl.fighter.Hero.Class.ID
	n := world.OngletsAuNiveau(g.Niveau)
	tabs := make([][]*world.ItemView, n)
	for i := 0; i < n; i++ {
		tabs[i] = world.ItemsView(g.Onglets[i], cl)
	}
	panneau(d, pl, "gcoffre", map[string]interface{}{"onglets": tabs, "or": g.Argent, "journal": g.Journal,
		"retrait": g.Can(pl.rang, world.DroitCoffre), "g": g.Nom})
}

func (s *worldState) guildLeave(ctx context.Context, logger runtime.Logger, db *sql.DB, d runtime.MatchDispatcher, pl *player) error {
	g := pl.guild
	if g == nil {
		return nil
	}
	members := guildMembers(ctx, db, g.ID)
	if pl.rang == 0 && len(members) > 1 {
		return errors.New("Passe d'abord la main à un autre membre (le nommer chef), puis tu pourras partir.")
	}
	if err := setHeroGuild(ctx, db, pl.hero.ID, 0, world.GuildeRangs-1); err != nil {
		return err
	}
	s.setGuild(pl, nil, 0)
	if len(members) <= 1 {
		_, _ = db.ExecContext(ctx, `DELETE FROM lf_guildes WHERE id = $1`, g.ID)
		delete(s.social.guilds, g.ID)
		logger.Info("Guilde %q dissoute", g.Nom)
	} else {
		s.guildNotice(d, g, "🏰 "+pl.hero.Name+" a quitté la guilde.")
		s.guildRefresh(ctx, db, d, g)
	}
	_ = d.BroadcastMessage(opJoin, mustJSON(describe(pl)), nil, nil, true)
	s.sendGuild(ctx, db, d, pl)
	return errors.New("Tu as quitté « " + g.Nom + " ».")
}

// member : un membre de la guilde par son identifiant de héros.
func member(ms []guildMember, id string) *guildMember {
	for i := range ms {
		if ms[i].id == id {
			return &ms[i]
		}
	}
	return nil
}

func (s *worldState) guildKick(ctx context.Context, logger runtime.Logger, db *sql.DB, d runtime.MatchDispatcher, pl *player, heroID string) error {
	g := pl.guild
	if g == nil {
		return nil
	}
	m := member(guildMembers(ctx, db, g.ID), heroID)
	switch {
	case m == nil || heroID == pl.hero.ID:
		return nil
	case !g.Can(pl.rang, world.DroitExclure) || m.rang <= pl.rang:
		return world.ErrGuildeDroit
	}
	if err := setHeroGuild(ctx, db, heroID, 0, world.GuildeRangs-1); err != nil {
		return err
	}
	if o := s.byHeroID(heroID); o != nil {
		s.setGuild(o, nil, 0)
		refuse(d, o, "Tu as été exclu de la guilde « "+g.Nom+" ».")
		_ = d.BroadcastMessage(opJoin, mustJSON(describe(o)), nil, nil, true)
		s.sendGuild(ctx, db, d, o)
	}
	s.guildNotice(d, g, "🏰 "+m.name+" a été exclu de la guilde.")
	s.guildRefresh(ctx, db, d, g)
	return nil
}

func (s *worldState) guildSetRank(ctx context.Context, logger runtime.Logger, db *sql.DB, d runtime.MatchDispatcher, pl *player, heroID string, rang int) error {
	g := pl.guild
	if g == nil || rang < 0 || rang >= world.GuildeRangs {
		return nil
	}
	m := member(guildMembers(ctx, db, g.ID), heroID)
	if m == nil || heroID == pl.hero.ID {
		return nil
	}
	if rang == 0 {
		// Passer la main : seul le chef le peut ; il devient officier.
		if pl.rang != 0 {
			return world.ErrGuildeDroit
		}
		if err := setHeroGuild(ctx, db, heroID, g.ID, 0); err != nil {
			return err
		}
		if err := setHeroGuild(ctx, db, pl.hero.ID, g.ID, 1); err != nil {
			return err
		}
		s.setGuild(pl, g, 1)
		if o := s.byHeroID(heroID); o != nil {
			s.setGuild(o, g, 0)
		}
		s.guildNotice(d, g, "👑 "+m.name+" est le nouveau chef de la guilde.")
		s.guildRefresh(ctx, db, d, g)
		return nil
	}
	if !g.Can(pl.rang, world.DroitRangs) || m.rang <= pl.rang || rang <= pl.rang {
		return world.ErrGuildeDroit
	}
	if err := setHeroGuild(ctx, db, heroID, g.ID, rang); err != nil {
		return err
	}
	if o := s.byHeroID(heroID); o != nil {
		s.setGuild(o, g, rang)
	}
	s.guildNotice(d, g, "🏰 "+m.name+" devient "+g.Rangs[rang].Nom+".")
	s.guildRefresh(ctx, db, d, g)
	return nil
}

func (s *worldState) guildEdit(ctx context.Context, logger runtime.Logger, db *sql.DB, d runtime.MatchDispatcher, pl *player,
	what string, rangs []world.Rang, text string, bl world.Blason, ev world.Evenement, i int) error {
	g := pl.guild
	if g == nil {
		return nil
	}
	now := time.Now().UnixMilli()
	switch what {
	case "guilde_rangs":
		if pl.rang != 0 {
			return world.ErrGuildeDroit
		}
		if err := g.SetRangs(rangs); err != nil {
			return err
		}
	case "guilde_motd":
		if !g.Can(pl.rang, world.DroitMotd) {
			return world.ErrGuildeDroit
		}
		g.Motd = world.CleanText(text, world.GuildeMotdMax)
		g.Touch()
		if g.Motd != "" {
			for _, o := range s.players {
				if o.guild == g {
					sendSocial(d, o, "motd", map[string]interface{}{"g": g.Nom, "m": g.Motd})
				}
			}
		}
	case "guilde_blason":
		if pl.rang != 0 {
			return world.ErrGuildeDroit
		}
		if err := bl.Check(g.Niveau); err != nil {
			return err
		}
		g.Blason = bl
		g.Touch()
		for _, o := range s.players {
			if o.guild == g {
				_ = d.BroadcastMessage(opJoin, mustJSON(describe(o)), nil, nil, true)
			}
		}
	case "guilde_agenda":
		if !g.Can(pl.rang, world.DroitAgenda) {
			return world.ErrGuildeDroit
		}
		ev.Auteur = pl.hero.Name
		if err := g.AddEvent(ev, now); err != nil {
			return err
		}
		s.guildNotice(d, g, "📅 "+pl.hero.Name+" prévoit une soirée : « "+world.CleanText(ev.Titre, 60)+" ».")
	case "guilde_agenda_suppr":
		if !g.Can(pl.rang, world.DroitAgenda) {
			return world.ErrGuildeDroit
		}
		if i < 0 || i >= len(g.Agenda) {
			return nil
		}
		g.Agenda = append(g.Agenda[:i], g.Agenda[i+1:]...)
		g.Touch()
	}
	s.saveGuild(ctx, logger, db, g)
	s.guildRefresh(ctx, db, d, g)
	return nil
}

// leaveGuildOnDelete : un héros effacé quitte sa guilde. S'il en était le chef,
// le membre le plus gradé prend sa place ; une guilde vide disparaît.
func leaveGuildOnDelete(ctx context.Context, db *sql.DB, heroID string) error {
	var gid sql.NullInt64
	var rang int
	if err := db.QueryRowContext(ctx, `SELECT guilde, rang FROM lf_heroes WHERE id = $1`, heroID).Scan(&gid, &rang); err != nil || !gid.Valid {
		return nil
	}
	if _, err := db.ExecContext(ctx, `UPDATE lf_heroes SET guilde = NULL WHERE id = $1`, heroID); err != nil {
		return err
	}
	var next string
	err := db.QueryRowContext(ctx, `SELECT id FROM lf_heroes WHERE guilde = $1 ORDER BY rang, level DESC LIMIT 1`, gid.Int64).Scan(&next)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = db.ExecContext(ctx, `DELETE FROM lf_guildes WHERE id = $1`, gid.Int64)
		return err
	}
	if err == nil && rang == 0 {
		_, err = db.ExecContext(ctx, `UPDATE lf_heroes SET rang = 0 WHERE id = $1`, next)
	}
	return err
}

// --- Chaque tick --------------------------------------------------------------------

// socialTick : invitations qui expirent, ressource des alliés, guildes à
// sauvegarder, fin de saison du Cercle.
func (s *worldState) socialTick(ctx context.Context, logger runtime.Logger, db *sql.DB, d runtime.MatchDispatcher, tick int64) {
	for sid, in := range s.social.guildInv {
		if in.left--; in.left <= 0 {
			delete(s.social.guildInv, sid)
		} else {
			s.social.guildInv[sid] = in
		}
	}
	if tick%groupResEvery == 0 {
		for _, g := range s.arena.Groups {
			res := make([][3]int, 0, len(g.Members))
			for _, sid := range g.Members {
				if f := s.arena.Heroes[sid]; f != nil {
					res = append(res, [3]int{sid, int(f.Hero.Res), int(f.Hero.Class.Res.Max)})
				}
			}
			for _, sid := range g.Members {
				if o := s.bySID(sid); o != nil {
					sendSocial(d, o, "groupe_res", map[string]interface{}{"m": res})
				}
			}
		}
	}
	if tick%saveEvery == 0 {
		for _, g := range s.social.guilds {
			s.saveGuild(ctx, logger, db, g)
		}
	}
	if tick%(60*tickRate) == 0 {
		s.closeSeasons(ctx, logger, db, d)
	}
}

// closeSeasons : à la fin d'une saison du Cercle, les deux premiers de chaque
// format gagnent leur titre.
func (s *worldState) closeSeasons(ctx context.Context, logger runtime.Logger, db *sql.DB, d runtime.MatchDispatcher) {
	cur := seasonAt(time.Now())
	if s.social.season == cur {
		return
	}
	s.social.season = cur
	for n := cur - 1; n >= 1 && n >= cur-3; n-- {
		res, err := db.ExecContext(ctx, `INSERT INTO lf_saisons (numero) VALUES ($1) ON CONFLICT DO NOTHING`, n)
		if err != nil {
			return
		}
		if k, _ := res.RowsAffected(); k == 0 {
			continue // déjà close
		}
		// Les cotes des héros en ligne sont plus récentes que la base.
		var ps []heroPos
		for _, pl := range s.players {
			if pl.fallen == 0 {
				ps = append(ps, pl.save())
			}
		}
		_ = saveHeroes(ctx, db, s.m.ID, ps)
		for _, f := range world.CercleFormats {
			k := strconv.Itoa(f)
			rows, err := db.QueryContext(ctx, `
SELECT id, name FROM lf_heroes
WHERE (gloire->>'saison')::int = $1 AND gloire->'cote' ? $2
ORDER BY (gloire->'cote'->>$2)::int DESC LIMIT 3`, n, k)
			if err != nil {
				continue
			}
			var winners [][2]string
			for rows.Next() {
				var id, name string
				if rows.Scan(&id, &name) == nil {
					winners = append(winners, [2]string{id, name})
				}
			}
			rows.Close()
			for i, w := range winners {
				titre := "saison_" + k + "_2"
				if i == 0 {
					titre = "saison_" + k + "_1"
				}
				if o := s.byHeroID(w[0]); o != nil && o.fighter.Hero.Gloire != nil {
					o.fighter.Hero.Gloire.AddTitre(titre)
					sendSocial(d, o, "gloire", world.GloireView(o.fighter.Hero.Gloire))
				} else {
					_, _ = db.ExecContext(ctx, `
UPDATE lf_heroes SET gloire = jsonb_set(gloire, '{ti}', COALESCE(gloire->'ti', '[]'::jsonb) || to_jsonb($2::text))
WHERE id = $1 AND NOT COALESCE(gloire->'ti', '[]'::jsonb) ? $2`, w[0], titre)
				}
				logger.Info("Saison %d du Cercle (%s c. %s) : %q gagne le titre %q", n, k, k, w[1], world.TitreText(titre))
			}
		}
		_ = d.BroadcastMessage(opNotice, mustJSON(map[string]string{"m": "🏟️ La saison " + strconv.Itoa(n) + " du Cercle des Braves est close ! Les cotes repartent de 1 500."}), nil, nil, true)
	}
}

// honorBoard : le tableau d'honneur (relu toutes les minutes).
func (s *worldState) honorBoard(ctx context.Context, logger runtime.Logger, db *sql.DB) map[string]interface{} {
	if s.social.board != nil && time.Since(s.social.boardAt) < boardEvery {
		return s.social.board
	}
	season := seasonAt(time.Now())
	var lv []map[string]interface{}
	if rows, err := db.QueryContext(ctx, `SELECT name, level, classe, race FROM lf_heroes ORDER BY level DESC, xp DESC LIMIT 10`); err == nil {
		for rows.Next() {
			var n, cl, r string
			var l int
			if rows.Scan(&n, &l, &cl, &r) == nil {
				lv = append(lv, map[string]interface{}{"n": n, "lv": l, "cl": cl, "r": r})
			}
		}
		rows.Close()
	}
	cotes := map[string][]map[string]interface{}{}
	for _, f := range world.CercleFormats {
		k := strconv.Itoa(f)
		rows, err := db.QueryContext(ctx, `
SELECT name, classe, (gloire->'cote'->>$2)::int FROM lf_heroes
WHERE (gloire->>'saison')::int = $1 AND gloire->'cote' ? $2
ORDER BY 3 DESC LIMIT 10`, season, k)
		if err != nil {
			continue
		}
		for rows.Next() {
			var n, cl string
			var c int
			if rows.Scan(&n, &cl, &c) == nil {
				cotes[k] = append(cotes[k], map[string]interface{}{"n": n, "cl": cl, "c": c})
			}
		}
		rows.Close()
	}
	var gs []map[string]interface{}
	if rows, err := db.QueryContext(ctx, `
SELECT g.nom, g.data->'bl', COALESCE((g.data->>'lv')::int, 1), COALESCE((g.data->>'xp')::int, 0), count(h.id)
FROM lf_guildes g LEFT JOIN lf_heroes h ON h.guilde = g.id
GROUP BY g.id ORDER BY 3 DESC, 4 DESC, 5 DESC LIMIT 5`); err == nil {
		for rows.Next() {
			var n string
			var bl []byte
			var l, xp, m int
			if rows.Scan(&n, &bl, &l, &xp, &m) == nil {
				var b world.Blason
				_ = json.Unmarshal(bl, &b)
				gs = append(gs, map[string]interface{}{"n": n, "bl": b, "lv": l, "m": m})
			}
		}
		rows.Close()
	}
	s.social.board = map[string]interface{}{"niveaux": lv, "cotes": cotes, "guildes": gs, "saison": season,
		"fin": seasonEnd(season).UnixMilli()}
	s.social.boardAt = time.Now()
	return s.social.board
}

// --- Ce que l'arène raconte ------------------------------------------------------------

func (s *worldState) groupView(g *world.Group) map[string]interface{} {
	if g == nil {
		return map[string]interface{}{"m": []interface{}{}}
	}
	var ms []map[string]interface{}
	for _, sid := range g.Members {
		if o := s.bySID(sid); o != nil {
			f := o.fighter
			ms = append(ms, map[string]interface{}{"s": sid, "n": o.hero.Name, "cl": f.Hero.Class.ID, "lv": f.Level,
				"hp": f.HP, "hpm": f.MaxHP, "rm": int(f.Hero.Class.Res.Max)})
		}
	}
	return map[string]interface{}{"id": g.ID, "chef": g.Leader, "m": ms, "max": world.GroupMax}
}

// sendActeVI diffuse les événements de l'Acte VI.
func (s *worldState) sendActeVI(ctx context.Context, logger runtime.Logger, db *sql.DB, d runtime.MatchDispatcher, e world.Event) {
	a := s.arena
	switch e.Kind {
	case world.EvGroup:
		switch e.Text {
		case "invite":
			if o := s.bySID(e.Tgt); o != nil {
				name := ""
				if f := a.Heroes[e.Src]; f != nil {
					name = f.Name
				}
				sendSocial(d, o, "invite_groupe", map[string]interface{}{"n": name, "s": e.Src, "sec": int(world.GroupInviteTime)})
			}
		case "quitte":
			if o := s.bySID(e.Tgt); o != nil && a.GroupOf(e.Tgt) == nil {
				sendSocial(d, o, "groupe", s.groupView(nil))
			}
			fallthrough
		default:
			if g := a.Groups[e.Src]; g != nil {
				v := s.groupView(g)
				for _, sid := range g.Members {
					if o := s.bySID(sid); o != nil {
						sendSocial(d, o, "groupe", v)
					}
				}
			}
		}
	case world.EvGuildXP:
		g := s.social.guilds[int64(e.Src)]
		if g == nil {
			return
		}
		if up := g.AddXP(e.Value); up > 0 {
			for _, o := range s.players {
				if o.guild == g {
					o.fighter.Hero.GuildLv = up
				}
			}
			s.guildNotice(d, g, "🎉 La guilde passe au niveau "+strconv.Itoa(up)+" !")
			s.saveGuild(ctx, logger, db, g)
			s.guildRefresh(ctx, db, d, g)
		}
	case world.EvFlag:
		if o := s.bySID(e.Src); o != nil {
			_ = d.BroadcastMessage(opJoin, mustJSON(describe(o)), nil, nil, true)
		}
	case world.EvGloire:
		o := s.bySID(e.Tgt)
		if o != nil && e.Text == "" { // les Insignes ou les titres ont changé
			sendSocial(d, o, "gloire", world.GloireView(o.fighter.Hero.Gloire))
			_ = d.BroadcastMessage(opJoin, mustJSON(describe(o)), nil, nil, true)
			return
		}
		h := world.HautsFaitsByID[e.Text]
		if o == nil || h == nil {
			return
		}
		v := map[string]interface{}{"id": h.ID, "nom": h.Nom, "ds": h.Desc, "ic": h.Ic, "pts": h.Pts}
		if h.Titre != "" {
			v["titre"] = world.TitreText(h.Titre)
		}
		sendSocial(d, o, "haut_fait", v)
		sendSocial(d, o, "gloire", world.GloireView(o.fighter.Hero.Gloire))
		if o.guild != nil && h.Pts >= 15 {
			s.guildNotice(d, o.guild, "🏅 "+o.hero.Name+" réussit le haut fait « "+h.Nom+" » !")
		}
	case world.EvCercle:
		s.sendCercle(d, e)
	}
}

func (s *worldState) sendCercle(d runtime.MatchDispatcher, e world.Event) {
	c := s.arena.Cercle
	switch e.Text {
	case "file":
		if o := s.bySID(e.Tgt); o != nil {
			sendSocial(d, o, "file", map[string]interface{}{"f": e.Value, "n": c.QueueSize(e.Value)})
		}
	case "debut", "compte", "sort":
		b := c.Bout(e.Value)
		if b == nil {
			return
		}
		names := map[int]string{}
		for _, t := range b.Teams {
			for _, x := range t {
				if f := s.arena.Heroes[x]; f != nil {
					names[x] = f.Name
				}
			}
		}
		v := map[string]interface{}{"e": e.Text, "id": b.ID, "f": b.Format, "camps": b.Teams, "n": e.Flags, "s": e.Tgt,
			"sec": int(math.Ceil(b.Count)), "duree": int(b.Left), "salle": c.Data.Salles[b.Salle].Nom}
		for _, t := range b.Teams {
			for _, x := range t {
				if o := s.bySID(x); o != nil {
					w := map[string]interface{}{}
					for k, val := range v {
						w[k] = val
					}
					w["noms"] = names
					w["moi"] = b.Side(x)
					sendSocial(d, o, "cercle", w)
				}
			}
		}
	case "fin":
		if o := s.bySID(e.Tgt); o != nil {
			res := "nul"
			switch {
			case e.Flags == 0:
			case e.Flags-1 == e.Src:
				res = "victoire"
			default:
				res = "defaite"
			}
			gl := o.fighter.Hero.Gloire
			cote := 0
			if gl != nil {
				cote = gl.Cote[strconv.Itoa(e.Value)]
			}
			sendSocial(d, o, "cercle", map[string]interface{}{"e": "fin", "res": res, "delta": e.XP, "ins": e.Skill, "cote": cote})
			sendSocial(d, o, "gloire", world.GloireView(gl))
		}
	}
}

// nearRegistre : le registre des guildes, à l'étal de Fiacre sur la place. Il
// sert de jour comme de nuit, que Fiacre soit là ou non.
func (s *worldState) nearRegistre(pl *player) bool {
	for _, n := range s.npcs {
		if n.ID == fiacreID && math.Hypot(n.Work[0]-pl.body.X, n.Work[1]-pl.body.Y) <= npcReach {
			return true
		}
	}
	return s.nearNPC(pl, fiacreID)
}
