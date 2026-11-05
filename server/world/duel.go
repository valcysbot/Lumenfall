package world

import "math"

// Les duels sur invitation : un héros en défie un autre, qui accepte ou non.
// Un drapeau est planté entre eux, un cercle se dessine au sol, et après un
// compte à rebours ils peuvent se frapper. Personne ne meurt : celui qui tombe
// à 1 point de vie est hors du duel. Sortir du cercle, c'est abandonner.
//
// Acte VI : le chef d'un groupe qui défie le chef d'un autre groupe lance un
// duel de groupe (jusqu'à 5 contre 5) : chaque camp réunit les membres présents
// près du drapeau, et le dernier camp debout gagne. Les autres héros peuvent
// regarder autour du cercle.

// Réglages des duels.
const (
	DuelInviteTime = 30.0  // secondes pour répondre à un défi
	DuelReach      = 10.0  // distance maximale pour lancer un défi
	DuelRadius     = 12.0  // rayon du cercle autour du drapeau
	DuelCountdown  = 3.0   // 3, 2, 1… combattez !
	DuelTime       = 120.0 // au-delà, match nul
	DuelGroupTime  = 240.0 // un duel de groupe dure plus longtemps
)

// Fins de duel (Event.Text de EvDuel « fin »).
const (
	DuelWon     = "victoire"
	DuelFled    = "abandon"    // le perdant est sorti du cercle
	DuelDraw    = "nul"        // le temps est écoulé
	DuelStopped = "interrompu" // un monstre s'en est mêlé, ou quelqu'un est parti
)

// Duel : deux héros (ou deux groupes) qui se battent pour l'honneur.
type Duel struct {
	ID    int
	A, B  int      // le héros qui a lancé le défi, et celui qui l'a accepté (les chefs)
	Sides [2][]int // les deux camps (A est dans le premier)
	Out   map[int]bool
	X, Y  float64 // le drapeau, au centre du cercle
	Count float64 // secondes avant le début
	Left  float64 // secondes avant le match nul
}

type invite struct {
	from int
	left float64
}

// Started dit si les coups comptent déjà.
func (d *Duel) Started() bool { return d.Count <= 0 }

// Side : le camp de sid (0 ou 1), -1 s'il n'est pas du duel.
func (d *Duel) Side(sid int) int {
	for s, side := range d.Sides {
		for _, x := range side {
			if x == sid {
				return s
			}
		}
	}
	return -1
}

// Opposed : x et y sont-ils adversaires, tous deux encore dans le duel ?
func (d *Duel) Opposed(x, y int) bool {
	sx, sy := d.Side(x), d.Side(y)
	return sx >= 0 && sy >= 0 && sx != sy && !d.Out[x] && !d.Out[y]
}

// Group dit si c'est un duel de groupe.
func (d *Duel) Group() bool { return len(d.Sides[0]) > 1 || len(d.Sides[1]) > 1 }

// Other : le chef du camp d'en face.
func (d *Duel) Other(sid int) int {
	if d.Side(sid) == 0 {
		return d.B
	}
	return d.A
}

// standing : combien de héros de ce camp sont encore dans le duel.
func (d *Duel) standing(s int) int {
	n := 0
	for _, x := range d.Sides[s] {
		if !d.Out[x] {
			n++
		}
	}
	return n
}

// DuelOf renvoie le duel d'un héros (ou nil).
func (a *Arena) DuelOf(sid int) *Duel {
	for _, d := range a.Duels {
		if d.Side(sid) >= 0 {
			return d
		}
	}
	return nil
}

// Duel renvoie un duel par son numéro.
func (a *Arena) Duel(id int) *Duel {
	for _, d := range a.Duels {
		if d.ID == id {
			return d
		}
	}
	return nil
}

// busy : le héros est déjà pris (duel, Cercle).
func (a *Arena) busy(sid int) bool { return a.DuelOf(sid) != nil || a.Cercle.BoutOf(sid) != nil }

// Challenge : sid défie target en duel.
func (a *Arena) Challenge(sid, target int) {
	h, t := a.Heroes[sid], a.Heroes[target]
	gh := a.groupOf[sid]
	gt := a.groupOf[target]
	switch {
	case h == nil || h.Dead:
		return
	case t == nil || t.Dead || t == h:
		a.refuse(h, "Choisis d'abord un héros à défier !")
	case a.SameGroup(sid, target):
		a.refuse(h, t.Name+" est de ton groupe !")
	case a.busy(sid):
		a.refuse(h, "Tu es déjà en duel !")
	case a.busy(target):
		a.refuse(h, t.Name+" est déjà en duel.")
	case gh != nil && gt != nil && gh.Leader != sid:
		a.refuse(h, "Seul le chef de ton groupe peut défier un autre groupe.")
	case gh != nil && gt != nil && gt.Leader != target:
		a.refuse(h, "Pour un duel de groupe, défie le chef de son groupe.")
	case dist(h, t) > DuelReach:
		a.refuse(h, "Approche-toi pour lancer ton défi !")
	case a.Ignores != nil && a.Ignores(target, sid):
		a.refuse(h, t.Name+" ne relève pas ton défi.")
	case a.invites[target].from == sid && a.invites[target].left > 0:
		a.refuse(h, t.Name+" n'a pas encore répondu.")
	default:
		a.invites[target] = invite{from: sid, left: DuelInviteTime}
		flags := 0
		if gh != nil && gt != nil {
			flags = 1 // un duel de groupe
		}
		a.emit(Event{Kind: EvDuel, Src: sid, Tgt: target, Text: "defi", Flags: flags})
		a.refuse(h, "Tu as défié "+t.Name+". Attends sa réponse…")
	}
}

// AnswerDuel : sid accepte (ou refuse) le dernier défi reçu.
func (a *Arena) AnswerDuel(sid int, accept bool) {
	in, ok := a.invites[sid]
	delete(a.invites, sid)
	h, c := a.Heroes[sid], a.Heroes[in.from]
	if !ok || in.left <= 0 || h == nil || c == nil {
		if h != nil && accept {
			a.refuse(h, "Ce défi n'est plus valable.")
		}
		return
	}
	if !accept {
		a.refuse(c, h.Name+" refuse ton défi.")
		return
	}
	switch {
	case h.Dead || c.Dead:
		a.refuse(h, "Ce défi n'est plus valable.")
	case a.busy(sid) || a.busy(c.SID):
		a.refuse(h, "L'un de vous est déjà en duel.")
	case dist(h, c) > DuelRadius:
		a.refuse(h, "Vous êtes trop loin l'un de l'autre.")
		a.refuse(c, h.Name+" a accepté, mais vous êtes trop loin l'un de l'autre.")
	default:
		a.duelID++
		d := &Duel{ID: a.duelID, A: c.SID, B: sid, X: (h.Body.X + c.Body.X) / 2, Y: (h.Body.Y + c.Body.Y) / 2,
			Count: DuelCountdown, Left: DuelTime, Out: map[int]bool{}}
		d.Sides[0] = a.duelSide(c, d)
		d.Sides[1] = a.duelSide(h, d)
		if d.Group() {
			d.Left = DuelGroupTime
		}
		a.Duels = append(a.Duels, d)
		for s, side := range d.Sides {
			lead := a.Heroes[[2]int{d.B, d.A}[s]]
			for _, x := range side {
				f := a.Heroes[x]
				f.Hero.Target = lead.SID
				f.Body.Face(lead.Body.X, lead.Body.Y)
			}
		}
		a.emit(Event{Kind: EvDuel, Src: d.A, Tgt: d.B, Value: d.ID, Text: "debut"})
	}
}

// duelSide : le camp d'un chef (lui, et les membres de son groupe près du drapeau).
func (a *Arena) duelSide(lead *Fighter, d *Duel) []int {
	out := []int{lead.SID}
	g := a.groupOf[lead.SID]
	if g == nil || a.groupOf[d.A] == nil || a.groupOf[d.B] == nil {
		return out
	}
	for _, x := range g.Members {
		f := a.Heroes[x]
		if x == lead.SID || f == nil || f.Dead || a.busy(x) {
			continue
		}
		if math.Hypot(f.Body.X-d.X, f.Body.Y-d.Y) <= DuelRadius {
			out = append(out, x)
		}
	}
	return out
}

// duelBlow : un coup de duel qui aurait été fatal laisse le héros à 1 point de
// vie, hors du duel. Renvoie vrai si le coup a été arrêté.
func (a *Arena) duelBlow(src, tgt *Fighter) bool {
	if tgt.HP > 0 || src.Hero == nil || tgt.Hero == nil {
		return false
	}
	if a.cercleBlow(src, tgt) {
		return true
	}
	d := a.DuelOf(tgt.SID)
	if d == nil || !d.Opposed(src.SID, tgt.SID) {
		return false
	}
	tgt.HP = 1
	a.duelOut(d, tgt.SID)
	return true
}

// duelOut : un héros quitte le duel ; si son camp est vide, l'autre gagne.
func (a *Arena) duelOut(d *Duel, sid int) {
	if d.Out[sid] {
		return
	}
	d.Out[sid] = true
	if f := a.Heroes[sid]; f != nil {
		a.calmDuel(d, f)
	}
	s := d.Side(sid)
	if d.standing(s) == 0 {
		a.endDuel(d, [2]int{d.B, d.A}[s], DuelWon)
		return
	}
	a.emit(Event{Kind: EvDuel, Src: sid, Value: d.ID, Text: "sort"})
}

// calmDuel : le héros n'est plus une cible, ses brûlures s'éteignent, ses adversaires le laissent.
func (a *Arena) calmDuel(d *Duel, f *Fighter) {
	s := d.Side(f.SID)
	for _, x := range d.Sides[1-s] {
		if o := a.Heroes[x]; o != nil && o.Hero.Target == f.SID {
			o.Hero.Target = 0
		}
	}
	if f.Hero.Target != 0 && d.Side(f.Hero.Target) == 1-s {
		f.Hero.Target = 0
	}
	for e := range f.fx {
		if src := f.fx[e].src; src != f.SID && d.Side(src) == 1-s {
			f.fx[e] = effect{}
			f.dirty = true
		}
	}
	a.interrupt(f)
}

// endDuel termine un duel. winner = 0 : personne ne gagne (sinon : le chef du camp gagnant).
func (a *Arena) endDuel(d *Duel, winner int, why string) {
	for i, o := range a.Duels {
		if o == d {
			a.Duels = append(a.Duels[:i], a.Duels[i+1:]...)
			break
		}
	}
	for _, side := range d.Sides {
		for _, sid := range side {
			if f := a.Heroes[sid]; f != nil {
				a.calmDuel(d, f)
			}
		}
	}
	loser := 0
	if winner != 0 {
		loser = d.Other(winner)
		ws := d.Side(winner)
		for s, side := range d.Sides {
			for _, sid := range side {
				if f := a.Heroes[sid]; f != nil {
					if s == ws {
						a.Count(f, "duel_v", 1)
					} else {
						a.Count(f, "duel_d", 1)
					}
				}
			}
		}
	}
	a.emit(Event{Kind: EvDuel, Src: winner, Tgt: loser, Value: d.ID, Text: "fin", Flags: duelEnds[why]})
}

// Les fins de duel envoyées aux joueurs (Event.Flags).
var duelEnds = map[string]int{DuelWon: 0, DuelFled: 1, DuelDraw: 2, DuelStopped: 3}

// tickDuels : défis qui expirent, compte à rebours, cercle, temps limite.
func (a *Arena) tickDuels(dt float64) {
	for sid, in := range a.invites {
		in.left -= dt
		if in.left <= 0 || a.Heroes[sid] == nil || a.Heroes[in.from] == nil {
			delete(a.invites, sid)
			continue
		}
		a.invites[sid] = in
	}
	for _, d := range append([]*Duel(nil), a.Duels...) {
		gone := false
		for s := range d.Sides {
			for _, x := range d.Sides[s] {
				if f := a.Heroes[x]; f == nil || f.Dead {
					if d.Group() && f != nil {
						d.Out[x] = true
						continue
					}
					gone = true
				}
			}
		}
		if gone || d.standing(0) == 0 || d.standing(1) == 0 {
			a.endDuel(d, 0, DuelStopped)
			continue
		}
		if !d.Started() {
			before := math.Ceil(d.Count)
			d.Count -= dt
			if math.Ceil(d.Count) != before {
				a.emit(Event{Kind: EvDuel, Src: d.A, Tgt: d.B, Value: d.ID, Text: "compte", Flags: int(math.Max(0, math.Ceil(d.Count)))})
			}
			continue
		}
		d.Left -= dt
		for s := range d.Sides {
			for _, x := range d.Sides[s] {
				f := a.Heroes[x]
				if !d.Out[x] && math.Hypot(f.Body.X-d.X, f.Body.Y-d.Y) > DuelRadius {
					if !d.Group() {
						a.endDuel(d, [2]int{d.B, d.A}[s], DuelFled)
						break
					}
					a.duelOut(d, x)
				}
			}
			if a.Duel(d.ID) == nil {
				break
			}
		}
		if a.Duel(d.ID) != nil && d.Left <= 0 {
			a.endDuel(d, 0, DuelDraw)
		}
	}
}
