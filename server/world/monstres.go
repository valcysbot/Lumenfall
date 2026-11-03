package world

import "math"

// Les petits monstres ne se contentent plus d'attendre au contact (retour de la
// soirée de jeu de la V0.5) : chacun a un coup annoncé au sol, comme le Grand
// Sanglier mais en plus petit, et les plus vifs tournent autour de leur proie
// entre deux morsures.
//
//   - Sanglier : la charge, en ligne droite, qui étourdit un instant.
//   - Loups : le bond sur leur proie, qui la fait saigner.
//   - Corbeau effacé : le piqué, un cercle sous la proie, qui aveugle.
//
// Chaque coup s'annonce assez tôt pour être évité d'une roulade ou d'un pas.

const (
	SpecialEvery = 9.0  // secondes entre deux coups annoncés (± 20 %)
	DanceTime    = 0.6  // durée d'un pas de côté
	DanceChance  = 0.55 // chance d'un pas de côté après une morsure
)

// specialNames : ce que le jeu affiche au-dessus du monstre quand il prépare son coup.
var specialNames = map[string]string{"charge": "Charge !", "bond": "Bond !", "pique": "Piqué !"}

// specialThink : le coup annoncé. Renvoie vrai quand le monstre est occupé
// (il le prépare, ou il fonce) et ne fait rien d'autre.
func (a *Arena) specialThink(f, t *Fighter, dt float64) bool {
	mo := f.Mon
	k := mo.Kind
	mo.specialCD -= dt
	if mo.windup > 0 || f.Body.Dashing() {
		mo.windup -= dt
		f.Body.Stop()
		return true
	}
	if k.Special == "" || mo.specialCD > 0 || f.Has(Gel) || f.Has(Ralenti) {
		return false
	}
	d := dist(f, t)
	if d < 1e-6 {
		return false
	}
	dx, dy := (t.Body.X-f.Body.X)/d, (t.Body.Y-f.Body.Y)/d
	dmg := monsterDmg(k, f.Level)
	tl := &Telegraph{Src: f.SID, Name: specialNames[k.Special], Elem: k.Elem, Charge: true}
	switch k.Special {
	case "charge":
		if d < 2.5 || d > 7 {
			return false
		}
		l := a.clearLine(f, dx, dy, math.Min(d+1.5, 7))
		if l < 2.5 {
			return false
		}
		tl.Line, tl.X, tl.Y, tl.X2, tl.Y2, tl.R = true, f.Body.X, f.Body.Y, f.Body.X+dx*l, f.Body.Y+dy*l, 0.8
		tl.Left, tl.Dmg, tl.Fx, tl.FxDur = 1.0, 1.8*dmg, Etourdi, 0.8
	case "bond":
		if d < 2 || d > 6 {
			return false
		}
		l := a.clearLine(f, dx, dy, d)
		if l < 2 {
			return false
		}
		tl.Line, tl.X, tl.Y, tl.X2, tl.Y2, tl.R = true, f.Body.X, f.Body.Y, f.Body.X+dx*l, f.Body.Y+dy*l, 0.7
		tl.Left, tl.Dmg, tl.Fx, tl.FxDur = 0.75, 1.5*dmg, Saignement, 4
		tl.FxPow = int(math.Max(1, math.Round(0.5*float64(f.Level))))
	case "pique":
		if d > 6 {
			return false
		}
		tl.X, tl.Y, tl.X2, tl.Y2, tl.R = t.Body.X, t.Body.Y, t.Body.X, t.Body.Y, 1.3
		tl.Left, tl.Dmg, tl.Fx, tl.FxDur = 0.9, 1.6*dmg, Aveugle, 2.5
	default:
		return false
	}
	mo.specialCD = SpecialEvery * (0.8 + 0.4*a.rng.Float64())
	mo.windup = tl.Left
	mo.attackCD = math.Max(mo.attackCD, k.Every)
	mo.dance = 0
	f.Body.Stop()
	f.Body.Face(t.Body.X, t.Body.Y)
	a.tele(tl)
	return true
}

// clearLine : jusqu'où le monstre peut foncer avant le premier obstacle.
func (a *Arena) clearLine(f *Fighter, dx, dy, max float64) float64 {
	l := 0.0
	for l < max && a.M.Free(f.Body.X+dx*(l+0.5), f.Body.Y+dy*(l+0.5)) {
		l += 0.5
	}
	return math.Min(l, max)
}

// danceStep : un pas de côté autour de la proie, entre deux morsures.
// Renvoie vrai tant qu'il danse.
func (a *Arena) danceStep(f, t *Fighter, dt float64) bool {
	mo := f.Mon
	if mo.dance <= 0 || f.Has(Gel) {
		return false
	}
	mo.dance -= dt
	dx, dy := t.Body.X-f.Body.X, t.Body.Y-f.Body.Y
	d := math.Hypot(dx, dy)
	if d < 1e-6 {
		return false
	}
	// De côté, en gardant à peu près la même distance.
	px, py := -dy/d*mo.danceSide, dx/d*mo.danceSide
	keep := (d - mo.Kind.Reach*0.8) / d
	f.Body.SetInput(px+dx*keep, py+dy*keep)
	return true
}

// afterBite : après une morsure, les monstres vifs font un pas de côté.
func (a *Arena) afterBite(f *Fighter) {
	mo := f.Mon
	if !mo.Kind.Dance || a.rng.Float64() >= DanceChance {
		return
	}
	mo.dance = DanceTime
	mo.danceSide = 1
	if a.rng.Float64() < 0.5 {
		mo.danceSide = -1
	}
}
