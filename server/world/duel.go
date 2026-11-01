package world

import "math"

// Les duels sur invitation : un héros en défie un autre, qui accepte ou non.
// Un drapeau est planté entre eux, un cercle se dessine au sol, et après un
// compte à rebours ils peuvent se frapper. Personne ne meurt : le premier qui
// tombe à 1 point de vie a perdu. Sortir du cercle, c'est abandonner.

// Réglages des duels.
const (
	DuelInviteTime = 30.0  // secondes pour répondre à un défi
	DuelReach      = 10.0  // distance maximale pour lancer un défi
	DuelRadius     = 12.0  // rayon du cercle autour du drapeau
	DuelCountdown  = 3.0   // 3, 2, 1… combattez !
	DuelTime       = 120.0 // au-delà, match nul
)

// Fins de duel (Event.Text de EvDuel « fin »).
const (
	DuelWon     = "victoire"
	DuelFled    = "abandon"    // le perdant est sorti du cercle
	DuelDraw    = "nul"        // le temps est écoulé
	DuelStopped = "interrompu" // un monstre s'en est mêlé, ou quelqu'un est parti
)

// Duel : deux héros qui se battent pour l'honneur.
type Duel struct {
	ID    int
	A, B  int     // le héros qui a lancé le défi, et celui qui l'a accepté
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

// Other : l'adversaire de sid dans ce duel.
func (d *Duel) Other(sid int) int {
	if d.A == sid {
		return d.B
	}
	return d.A
}

// DuelOf renvoie le duel d'un héros (ou nil).
func (a *Arena) DuelOf(sid int) *Duel {
	for _, d := range a.Duels {
		if d.A == sid || d.B == sid {
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

// foe dit si h peut frapper t : un monstre, ou son adversaire d'un duel commencé.
func (a *Arena) foe(h, t *Fighter) bool {
	if t == nil || t.Dead {
		return false
	}
	if t.Mon != nil {
		return !t.Mon.gone
	}
	d := a.DuelOf(h.SID)
	return d != nil && d.Started() && d.Other(h.SID) == t.SID
}

// noFoe explique pourquoi on ne peut pas frapper cette cible.
func (a *Arena) noFoe(h, t *Fighter) {
	switch d := a.DuelOf(h.SID); {
	case t == nil || t.Dead || t.Hero == nil:
		a.refuse(h, "Choisis d'abord un monstre !")
	case d != nil && d.Other(h.SID) == t.SID:
		a.refuse(h, "Attends la fin du compte à rebours !")
	default:
		a.refuse(h, "Pour te battre contre "+t.Name+", défie-le en duel !")
	}
}

// Challenge : sid défie target en duel.
func (a *Arena) Challenge(sid, target int) {
	h, t := a.Heroes[sid], a.Heroes[target]
	switch {
	case h == nil || h.Dead:
		return
	case t == nil || t.Dead || t == h:
		a.refuse(h, "Choisis d'abord un héros à défier !")
	case a.DuelOf(sid) != nil:
		a.refuse(h, "Tu es déjà en duel !")
	case a.DuelOf(target) != nil:
		a.refuse(h, t.Name+" est déjà en duel.")
	case dist(h, t) > DuelReach:
		a.refuse(h, "Approche-toi pour lancer ton défi !")
	case a.invites[target].from == sid && a.invites[target].left > 0:
		a.refuse(h, t.Name+" n'a pas encore répondu.")
	default:
		a.invites[target] = invite{from: sid, left: DuelInviteTime}
		a.emit(Event{Kind: EvDuel, Src: sid, Tgt: target, Text: "defi"})
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
	case a.DuelOf(sid) != nil || a.DuelOf(c.SID) != nil:
		a.refuse(h, "L'un de vous est déjà en duel.")
	case dist(h, c) > DuelRadius:
		a.refuse(h, "Vous êtes trop loin l'un de l'autre.")
		a.refuse(c, h.Name+" a accepté, mais vous êtes trop loin l'un de l'autre.")
	default:
		a.duelID++
		d := &Duel{ID: a.duelID, A: c.SID, B: sid, X: (h.Body.X + c.Body.X) / 2, Y: (h.Body.Y + c.Body.Y) / 2,
			Count: DuelCountdown, Left: DuelTime}
		a.Duels = append(a.Duels, d)
		for _, f := range []*Fighter{h, c} {
			f.Hero.Target = d.Other(f.SID)
			f.Body.Face(a.Heroes[d.Other(f.SID)].Body.X, a.Heroes[d.Other(f.SID)].Body.Y)
		}
		a.emit(Event{Kind: EvDuel, Src: d.A, Tgt: d.B, Value: d.ID, Text: "debut"})
	}
}

// duelBlow : un coup de duel qui aurait été fatal laisse le perdant à 1 point de vie.
// Renvoie vrai si le duel vient de se finir.
func (a *Arena) duelBlow(src, tgt *Fighter) bool {
	if tgt.HP > 0 || src.Hero == nil || tgt.Hero == nil {
		return false
	}
	d := a.DuelOf(tgt.SID)
	if d == nil || d.Other(tgt.SID) != src.SID {
		return false
	}
	tgt.HP = 1
	a.endDuel(d, src.SID, DuelWon)
	return true
}

// endDuel termine un duel. winner = 0 : personne ne gagne.
func (a *Arena) endDuel(d *Duel, winner int, why string) {
	for i, o := range a.Duels {
		if o == d {
			a.Duels = append(a.Duels[:i], a.Duels[i+1:]...)
			break
		}
	}
	for _, sid := range []int{d.A, d.B} {
		f := a.Heroes[sid]
		if f == nil {
			continue
		}
		// L'adversaire n'est plus une cible, et ses brûlures s'éteignent.
		if f.Hero.Target == d.Other(sid) {
			f.Hero.Target = 0
		}
		for e := range f.fx {
			if f.fx[e].src == d.Other(sid) {
				f.fx[e] = effect{}
				f.dirty = true
			}
		}
		a.interrupt(f)
	}
	loser := 0
	if winner != 0 {
		loser = d.Other(winner)
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
		ha, hb := a.Heroes[d.A], a.Heroes[d.B]
		if ha == nil || hb == nil || ha.Dead || hb.Dead {
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
		out := func(f *Fighter) bool { return math.Hypot(f.Body.X-d.X, f.Body.Y-d.Y) > DuelRadius }
		switch {
		case out(ha):
			a.endDuel(d, d.B, DuelFled)
		case out(hb):
			a.endDuel(d, d.A, DuelFled)
		case d.Left <= 0:
			a.endDuel(d, 0, DuelDraw)
		}
	}
}
