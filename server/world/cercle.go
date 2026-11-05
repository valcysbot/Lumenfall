package world

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
)

// Le Cercle des Braves (Acte VI) : l'arène du village, derrière la forge de
// Doran. On s'inscrit seul ou avec son groupe pour un combat à 1 contre 1,
// 2 contre 2 ou 3 contre 3 ; dès que deux camps sont prêts et qu'une salle est
// libre, ils y sont emmenés. Personne ne meurt : à 1 point de vie, on est hors
// du combat. Le dernier camp debout gagne ; au bout du temps, le camp qui a le
// plus de vie. Chaque héros a une cote par format (comme aux échecs), remise à
// zéro à chaque saison de 4 semaines, et gagne des Insignes de bravoure.

const (
	CercleCountdown = 5.0   // secondes avant le premier coup
	CercleTime      = 180.0 // au-delà, le camp le plus vaillant l'emporte
	CercleReach     = 12.0  // cases autour du cercle pour s'inscrire
	CoteStart       = 1500
	CoteK           = 32
	InsWin          = 30 // Insignes de bravoure : victoire, défaite, égalité
	InsLose         = 10
	InsDraw         = 15
)

// CercleFormats : 1 contre 1, 2 contre 2, 3 contre 3.
var CercleFormats = []int{1, 2, 3}

// Salle : une salle du Cercle, loin au sud avec les intérieurs.
type Salle struct {
	Nom   string        `json:"nom"`
	X0    int           `json:"x0"`
	Y0    int           `json:"y0"`
	W     int           `json:"w"`
	H     int           `json:"h"`
	Spots [2][2]float64 `json:"-"` // où chaque camp commence
}

// CercleData : le Cercle d'une carte (server/world/maps/<carte>.cercle.json).
type CercleData struct {
	Anneau  [2]float64 `json:"anneau"`  // le cercle de pierres, au village
	Honneur [2]float64 `json:"honneur"` // le tableau d'honneur, sur la place
	Salles  []Salle    `json:"salles"`
}

// LoadCercle lit et vérifie le Cercle d'une carte (absent : pas de Cercle).
func LoadCercle(m *Map) (*CercleData, error) {
	raw, err := mapFiles.ReadFile("maps/" + m.ID + ".cercle.json")
	if err != nil {
		return nil, nil
	}
	d := &CercleData{}
	if err := json.Unmarshal(raw, d); err != nil {
		return nil, err
	}
	if !m.Free(d.Anneau[0], d.Anneau[1]) || !m.Free(d.Honneur[0], d.Honneur[1]) {
		return nil, fmt.Errorf("cercle de %q : l'anneau ou le tableau est sur une case bloquée", m.ID)
	}
	for i := range d.Salles {
		s := &d.Salles[i]
		cy := float64(s.Y0) + float64(s.H)/2
		s.Spots[0] = [2]float64{float64(s.X0) + 2.5, cy}
		s.Spots[1] = [2]float64{float64(s.X0+s.W) - 2.5, cy}
		for _, p := range s.Spots {
			if !m.Free(p[0], p[1]) {
				return nil, fmt.Errorf("salle %q : départ bloqué en %v", s.Nom, p)
			}
		}
		if m.FindPath(s.Spots[0][0], s.Spots[0][1], s.Spots[1][0], s.Spots[1][1], 5000) == nil {
			return nil, fmt.Errorf("salle %q : les deux camps ne peuvent pas se rejoindre", s.Nom)
		}
	}
	return d, nil
}

// Bout : un combat du Cercle.
type Bout struct {
	ID     int
	Format int
	Salle  int
	Teams  [2][]int
	Out    map[int]bool
	Count  float64
	Left   float64
	back   map[int][2]float64 // où chacun était avant
}

// Side : le camp de sid (-1 : pas de ce combat).
func (b *Bout) Side(sid int) int {
	for s, t := range b.Teams {
		for _, x := range t {
			if x == sid {
				return s
			}
		}
	}
	return -1
}

// Opposed : x et y s'affrontent, le signal est donné et ils sont encore debout.
func (b *Bout) Opposed(x, y int) bool {
	sx, sy := b.Side(x), b.Side(y)
	return b.Count <= 0 && sx >= 0 && sy >= 0 && sx != sy && !b.Out[x] && !b.Out[y]
}

func (b *Bout) standing(s int) int {
	n := 0
	for _, x := range b.Teams[s] {
		if !b.Out[x] {
			n++
		}
	}
	return n
}

type cercleEntry struct {
	sids []int
	wait float64
}

// Cercle : les files d'attente et les combats en cours.
type Cercle struct {
	a      *Arena
	Data   *CercleData
	Queues map[int][]*cercleEntry // par format
	Bouts  []*Bout
	boutID int
	// Season : la saison en cours (réglée par le match ; 1 par défaut).
	Season func() int
}

func newCercle(a *Arena) *Cercle {
	return &Cercle{a: a, Queues: map[int][]*cercleEntry{}, Season: func() int { return 1 }}
}

// BoutOf : le combat d'un héros (ou nil).
func (c *Cercle) BoutOf(sid int) *Bout {
	if c == nil {
		return nil
	}
	for _, b := range c.Bouts {
		if b.Side(sid) >= 0 {
			return b
		}
	}
	return nil
}

// Bout renvoie un combat par son numéro.
func (c *Cercle) Bout(id int) *Bout {
	for _, b := range c.Bouts {
		if b.ID == id {
			return b
		}
	}
	return nil
}

// QueuedFor : le format pour lequel sid attend (0 : aucun).
func (c *Cercle) QueuedFor(sid int) int {
	for f, q := range c.Queues {
		for _, e := range q {
			for _, x := range e.sids {
				if x == sid {
					return f
				}
			}
		}
	}
	return 0
}

// QueueSize : combien de héros attendent pour ce format.
func (c *Cercle) QueueSize(format int) int {
	n := 0
	for _, e := range c.Queues[format] {
		n += len(e.sids)
	}
	return n
}

// Join : sid s'inscrit (avec son groupe, s'il en est le chef) pour un format.
func (c *Cercle) Join(sid, format int) {
	a := c.a
	h := a.Heroes[sid]
	if h == nil {
		return
	}
	ok := false
	for _, f := range CercleFormats {
		ok = ok || f == format
	}
	sids := []int{sid}
	if g := a.groupOf[sid]; g != nil {
		if g.Leader != sid {
			a.refuse(h, "C'est le chef du groupe qui inscrit tout le monde au Cercle.")
			return
		}
		sids = append([]int(nil), g.Members...)
	}
	switch {
	case !ok:
		return
	case c.Data == nil || len(c.Data.Salles) == 0:
		a.refuse(h, "Le Cercle des Braves est fermé.")
		return
	case math.Hypot(h.Body.X-c.Data.Anneau[0], h.Body.Y-c.Data.Anneau[1]) > CercleReach:
		a.refuse(h, "On s'inscrit au Cercle des Braves, derrière la forge de Doran.")
		return
	case len(sids) > format:
		a.refuse(h, "Ton groupe est trop nombreux pour ce combat.")
		return
	}
	for _, x := range sids {
		f := a.Heroes[x]
		switch {
		case f == nil:
			return
		case f.Dead:
			a.refuse(h, f.Name+" doit d'abord se relever.")
			return
		case a.busy(x):
			a.refuse(h, f.Name+" est déjà en plein combat.")
			return
		}
	}
	for _, x := range sids {
		c.leave(x)
	}
	c.Queues[format] = append(c.Queues[format], &cercleEntry{sids: sids})
	for _, x := range sids {
		a.emit(Event{Kind: EvCercle, Tgt: x, Text: "file", Value: format})
	}
}

// Leave : sid quitte la file (son groupe avec lui).
func (c *Cercle) Leave(sid int) {
	if f := c.QueuedFor(sid); f != 0 {
		for _, e := range c.Queues[f] {
			for _, x := range e.sids {
				if x == sid {
					for _, y := range e.sids {
						c.leave(y)
					}
					return
				}
			}
		}
	}
}

// leave : sid n'attend plus, et abandonne son combat s'il en avait un.
func (c *Cercle) leave(sid int) {
	if c == nil {
		return
	}
	for f, q := range c.Queues {
		for i := 0; i < len(q); i++ {
			e := q[i]
			for k, x := range e.sids {
				if x == sid {
					e.sids = append(e.sids[:k], e.sids[k+1:]...)
					c.a.emit(Event{Kind: EvCercle, Tgt: sid, Text: "file", Value: 0})
					break
				}
			}
			if len(e.sids) == 0 {
				q = append(q[:i], q[i+1:]...)
				i--
			}
		}
		c.Queues[f] = q
	}
	if b := c.BoutOf(sid); b != nil && !b.Out[sid] {
		c.out(b, sid)
	}
}

// pickTeam : des inscriptions qui font exactement n héros (les premiers arrivés d'abord).
func pickTeam(q []*cercleEntry, n int, skip map[*cercleEntry]bool) []*cercleEntry {
	var best []*cercleEntry
	var try func(i, left int, cur []*cercleEntry) bool
	try = func(i, left int, cur []*cercleEntry) bool {
		if left == 0 {
			best = append([]*cercleEntry(nil), cur...)
			return true
		}
		for ; i < len(q); i++ {
			if skip[q[i]] || len(q[i].sids) > left {
				continue
			}
			if try(i+1, left-len(q[i].sids), append(cur, q[i])) {
				return true
			}
		}
		return false
	}
	try(0, n, nil)
	return best
}

func (c *Cercle) freeSalle() int {
	if c.Data == nil {
		return -1
	}
	used := map[int]bool{}
	for _, b := range c.Bouts {
		used[b.Salle] = true
	}
	for i := range c.Data.Salles {
		if !used[i] {
			return i
		}
	}
	return -1
}

func (c *Cercle) tick(dt float64) {
	if c == nil {
		return
	}
	a := c.a
	for _, f := range CercleFormats {
		for _, e := range c.Queues[f] {
			e.wait += dt
		}
		for {
			s := c.freeSalle()
			if s < 0 {
				break
			}
			q := c.Queues[f]
			t1 := pickTeam(q, f, nil)
			if t1 == nil {
				break
			}
			skip := map[*cercleEntry]bool{}
			for _, e := range t1 {
				skip[e] = true
			}
			t2 := pickTeam(q, f, skip)
			if t2 == nil {
				break
			}
			for _, e := range t2 {
				skip[e] = true
			}
			var rest []*cercleEntry
			for _, e := range q {
				if !skip[e] {
					rest = append(rest, e)
				}
			}
			c.Queues[f] = rest
			c.start(f, s, flatten(t1), flatten(t2))
		}
	}
	for _, b := range append([]*Bout(nil), c.Bouts...) {
		for s := range b.Teams {
			for _, x := range b.Teams[s] {
				if h := a.Heroes[x]; (h == nil || h.Dead) && !b.Out[x] {
					c.out(b, x)
				}
			}
		}
		if c.Bout(b.ID) == nil {
			continue
		}
		if b.Count > 0 {
			before := math.Ceil(b.Count)
			b.Count -= dt
			if math.Ceil(b.Count) != before {
				a.emit(Event{Kind: EvCercle, Text: "compte", Value: b.ID, Flags: int(math.Max(0, math.Ceil(b.Count)))})
			}
			continue
		}
		if b.Left -= dt; b.Left <= 0 {
			c.timeUp(b)
		}
	}
}

func flatten(es []*cercleEntry) []int {
	var out []int
	for _, e := range es {
		out = append(out, e.sids...)
	}
	return out
}

func (c *Cercle) start(format, salle int, t1, t2 []int) {
	a := c.a
	c.boutID++
	b := &Bout{ID: c.boutID, Format: format, Salle: salle, Teams: [2][]int{t1, t2}, Out: map[int]bool{},
		Count: CercleCountdown, Left: CercleTime, back: map[int][2]float64{}}
	c.Bouts = append(c.Bouts, b)
	sa := c.Data.Salles[salle]
	for s, team := range b.Teams {
		for i, x := range team {
			h := a.Heroes[x]
			b.back[x] = [2]float64{h.Body.X, h.Body.Y}
			sp := sa.Spots[s]
			c.refresh(h)
			h.Body.Teleport(sp[0], sp[1]+float64(i)-float64(len(team)-1)/2)
			h.Hero.lastX, h.Hero.lastY = h.Body.X, h.Body.Y
			h.Hero.Target = 0
		}
	}
	a.emit(Event{Kind: EvCercle, Text: "debut", Value: b.ID})
}

// refresh : vie pleine, plus d'effets ni de recharges, ressource de départ.
func (c *Cercle) refresh(h *Fighter) {
	a := c.a
	hs := h.Hero
	a.interrupt(h)
	a.stopWork(h, "")
	h.Clear()
	h.HP = h.MaxHP
	hs.Res = hs.Class.Res.Start
	hs.Endurance = MaxEndurance
	for k := range hs.CD {
		hs.CD[k] = 0
	}
	hs.PotionCD = 0
	hs.dr = nil
	h.Body.Stop()
	h.dirty = true
	hs.me = true
}

// cercleBlow : un coup fatal dans le Cercle laisse le héros à 1 point de vie, hors du combat.
func (a *Arena) cercleBlow(src, tgt *Fighter) bool {
	b := a.Cercle.BoutOf(tgt.SID)
	if b == nil || !b.Opposed(src.SID, tgt.SID) {
		return false
	}
	tgt.HP = 1
	a.Cercle.out(b, tgt.SID)
	return true
}

func (c *Cercle) out(b *Bout, sid int) {
	a := c.a
	b.Out[sid] = true
	s := b.Side(sid)
	if h := a.Heroes[sid]; h != nil {
		a.interrupt(h)
		h.Clear()
		h.dirty = true
		for _, x := range b.Teams[1-s] {
			if o := a.Heroes[x]; o != nil && o.Hero.Target == sid {
				o.Hero.Target = 0
			}
		}
	}
	a.emit(Event{Kind: EvCercle, Text: "sort", Value: b.ID, Tgt: sid})
	if b.standing(s) == 0 {
		c.end(b, 1-s)
	}
}

// timeUp : le temps est écoulé ; le camp qui a gardé le plus de vie gagne.
func (c *Cercle) timeUp(b *Bout) {
	var pct [2]float64
	for s, team := range b.Teams {
		for _, x := range team {
			if h := c.a.Heroes[x]; h != nil && !b.Out[x] {
				pct[s] += h.Pct()
			}
		}
		pct[s] /= float64(len(team))
	}
	switch {
	case math.Abs(pct[0]-pct[1]) < 0.02:
		c.end(b, -1)
	case pct[0] > pct[1]:
		c.end(b, 0)
	default:
		c.end(b, 1)
	}
}

// end : le combat est fini (winner : 0 ou 1, -1 : égalité). Les cotes changent,
// les insignes tombent, et chacun revient où il était.
func (c *Cercle) end(b *Bout, winner int) {
	a := c.a
	for i, o := range c.Bouts {
		if o == b {
			c.Bouts = append(c.Bouts[:i], c.Bouts[i+1:]...)
			break
		}
	}
	key := strconv.Itoa(b.Format)
	season := c.Season()
	var avg [2]float64
	for s, team := range b.Teams {
		n := 0
		for _, x := range team {
			if h := a.Heroes[x]; h != nil && h.Hero.Gloire != nil {
				avg[s] += float64(h.Hero.Gloire.CoteOf(key, season))
				n++
			} else {
				avg[s] += CoteStart
				n++
			}
		}
		avg[s] /= float64(max(1, n))
	}
	for s, team := range b.Teams {
		score := 0.5
		ins := InsDraw
		if winner == s {
			score, ins = 1, InsWin
		} else if winner == 1-s {
			score, ins = 0, InsLose
		}
		expect := 1 / (1 + math.Pow(10, (avg[1-s]-avg[s])/400))
		delta := int(math.Round(CoteK * (score - expect)))
		for _, x := range team {
			h := a.Heroes[x]
			if h == nil {
				continue
			}
			if gl := h.Hero.Gloire; gl != nil {
				cote := gl.CoteOf(key, season) + delta
				gl.Cote[key] = max(0, cote)
				gl.Meilleur[key] = max(gl.Meilleur[key], gl.Cote[key])
				gl.Insignes += ins
				a.CountMax(h, "cote", gl.Cote[key])
			}
			switch {
			case winner == s:
				a.Count(h, "cercle_v", 1)
			case winner == 1-s:
				a.Count(h, "cercle_d", 1)
			}
			c.refresh(h)
			if p, ok := b.back[x]; ok {
				h.Body.Teleport(p[0], p[1])
				h.Hero.lastX, h.Hero.lastY = p[0], p[1]
			}
			a.emit(Event{Kind: EvCercle, Text: "fin", Value: b.Format, Src: s, Tgt: x, Flags: winner + 1, XP: delta, Skill: ins})
		}
	}
}

// Teams triés, pour les tests.
func (b *Bout) sorted() [2][]int {
	var out [2][]int
	for s := range b.Teams {
		out[s] = append([]int(nil), b.Teams[s]...)
		sort.Ints(out[s])
	}
	return out
}
