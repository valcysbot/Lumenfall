package world

import "math"

// Le Grand Sanglier de Mabon, premier boss d'Erydor. Ses grosses attaques sont
// annoncées au sol (zones rouges) : on a le temps de rouler hors de la zone.
//
// Phase 1 : il charge en ligne droite et piétine autour de lui.
// Phase 2 (sous la moitié de sa vie) : la Flétrissure le dévore. Il devient
// un Effacé (élément ombre : la lumière du Paladin lui fait plus mal), plus
// rapide et plus enragé, et des éclats gris tombent sur chaque héros.

// Telegraph : une attaque annoncée au sol.
type Telegraph struct {
	ID     int
	Src    int
	Line   bool    // une ligne (charge) ; sinon un cercle
	X, Y   float64 // centre du cercle, ou départ de la ligne
	X2, Y2 float64 // arrivée de la ligne
	R      float64 // rayon du cercle, ou demi-largeur de la ligne
	Left   float64 // secondes avant l'impact
	Total  float64
	Dmg    float64
	Elem   Element
	Fx     Effect
	FxDur  float64
	FxPow  int
	Charge bool // le boss fonce le long de la ligne à l'impact
}

// Réglages du Grand Sanglier.
const (
	ChargeLen   = 9.0
	ChargeWidth = 1.1
	StompRadius = 3.6
	ShardRadius = 2.0
)

func (a *Arena) tele(t *Telegraph) {
	a.teleID++
	t.ID = a.teleID
	t.Total = t.Left
	a.Teles = append(a.Teles, t)
	a.emit(Event{Kind: EvTele, Src: t.Src, Value: t.ID})
}

// Tele renvoie une attaque annoncée par son numéro.
func (a *Arena) Tele(id int) *Telegraph {
	for _, t := range a.Teles {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func (a *Arena) cancelTeles(src int) {
	keep := a.Teles[:0]
	for _, t := range a.Teles {
		if t.Src != src {
			keep = append(keep, t)
		}
	}
	a.Teles = keep
}

// Inside dit si un point est dans la zone (avec la taille d'un héros).
func (t *Telegraph) Inside(x, y float64) bool {
	if !t.Line {
		return math.Hypot(x-t.X, y-t.Y) <= t.R+HeroRadius
	}
	dx, dy := t.X2-t.X, t.Y2-t.Y
	l2 := dx*dx + dy*dy
	k := 0.0
	if l2 > 0 {
		k = math.Max(0, math.Min(1, ((x-t.X)*dx+(y-t.Y)*dy)/l2))
	}
	return math.Hypot(x-(t.X+dx*k), y-(t.Y+dy*k)) <= t.R+HeroRadius
}

func (a *Arena) tickTeles(dt float64) {
	keep := a.Teles[:0]
	var due []*Telegraph
	for _, t := range a.Teles {
		t.Left -= dt
		if t.Left <= 1e-6 {
			due = append(due, t)
		} else {
			keep = append(keep, t)
		}
	}
	a.Teles = keep
	for _, t := range due {
		src := a.Get(t.Src)
		if src == nil || src.Dead {
			continue
		}
		if t.Charge {
			src.Body.Dash(t.X2-src.Body.X, t.Y2-src.Body.Y, math.Hypot(t.X2-src.Body.X, t.Y2-src.Body.Y), 0.45)
			a.emit(Event{Kind: EvAnim, Src: src.SID, Text: "attaque"})
		}
		for _, h := range a.Heroes {
			if h.Dead || !t.Inside(h.Body.X, h.Body.Y) {
				continue
			}
			if a.hit(src, h, t.Dmg, t.Elem, SkillAuto, 0, 1) > 0 && t.FxDur > 0 {
				h.Apply(t.Fx, t.FxDur, t.FxPow, src.SID)
			}
		}
		if src.Mon != nil {
			src.Mon.windup = 0
		}
	}
}

// bossThink : les attaques spéciales. Renvoie vrai quand le boss est occupé
// (il prépare un coup et ne bouge pas).
func (a *Arena) bossThink(f, t *Fighter, dt float64) bool {
	mo := f.Mon
	if mo.Phase == 1 && f.HP*2 <= f.MaxHP {
		mo.Phase = 2
		f.Elem = Ombre
		f.dirty = true
		mo.shardCD = 1.5
		a.emit(Event{Kind: EvPhase, Src: f.SID, Value: 2})
	}
	enrage := 1.0
	if mo.Phase == 2 {
		enrage = 0.65
		f.Body.Mul *= 1.25
	}
	mo.chargeCD -= dt
	mo.stompCD -= dt
	mo.shardCD -= dt
	if mo.windup > 0 || f.Body.Dashing() {
		mo.windup -= dt
		f.Body.Stop()
		return true
	}
	d := dist(f, t)
	switch {
	case mo.stompCD <= 0 && d <= StompRadius:
		mo.stompCD = 10 * enrage
		mo.windup = 1.4
		a.tele(&Telegraph{Src: f.SID, X: f.Body.X, Y: f.Body.Y, R: StompRadius, Left: mo.windup, Dmg: 34,
			Fx: Ralenti, FxDur: 3})
	case mo.chargeCD <= 0 && d > 2.5:
		mo.chargeCD = 7 * enrage
		mo.windup = 1.2 * math.Sqrt(enrage)
		dx, dy := (t.Body.X-f.Body.X)/d, (t.Body.Y-f.Body.Y)/d
		// La charge s'arrête au premier obstacle.
		l := 0.0
		for l < ChargeLen && a.M.Free(f.Body.X+dx*(l+0.5), f.Body.Y+dy*(l+0.5)) {
			l += 0.5
		}
		if l < 2 {
			mo.chargeCD = 1 // un arbre dans le chemin : il se rapproche d'abord
			return false
		}
		f.Body.Face(t.Body.X, t.Body.Y)
		a.tele(&Telegraph{Src: f.SID, Line: true, X: f.Body.X, Y: f.Body.Y, X2: f.Body.X + dx*l, Y2: f.Body.Y + dy*l,
			R: ChargeWidth, Left: mo.windup, Dmg: 55, Fx: Etourdi, FxDur: 1.5, Charge: true})
	case mo.Phase == 2 && mo.shardCD <= 0:
		mo.shardCD = 8
		mo.windup = 0.6
		n := 0
		for sid := range mo.threat {
			h := a.Heroes[sid]
			if h == nil || h.Dead || n >= 5 {
				continue
			}
			n++
			a.tele(&Telegraph{Src: f.SID, X: h.Body.X, Y: h.Body.Y, R: ShardRadius, Left: 1.6, Dmg: 40, Elem: Ombre,
				Fx: Poison, FxDur: 5, FxPow: 4})
		}
	default:
		return false
	}
	f.Body.Stop()
	return true
}

// resetBoss : le boss abandonne le combat et redevient lui-même.
func (a *Arena) resetBoss(f *Fighter) {
	mo := f.Mon
	a.cancelTeles(f.SID)
	if mo.Phase != 1 {
		a.emit(Event{Kind: EvPhase, Src: f.SID, Value: 1})
	}
	mo.Phase, mo.windup, mo.chargeCD, mo.stompCD, mo.shardCD = 1, 0, 4, 8, 6
	f.Elem = mo.Kind.Elem
	f.dirty = true
}
