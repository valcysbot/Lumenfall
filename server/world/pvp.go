package world

import (
	"math"
	"strconv"
)

// Le PvP (Acte VI) : qui peut frapper qui, le drapeau de guerre, et les règles
// communes des combats entre héros (choix d'Elowan) : les contrôles répétés
// durent moins, les soins font −30 % sur un héros qui se bat contre des héros,
// et tomber sous les coups d'un héros ne coûte rien (ni usure, ni Faiblesse,
// ni la vie d'un héros hardcore).

const (
	FlagLowerWait = 30.0 // secondes sans combat contre un héros avant de baisser son drapeau
	PvPHealMul    = 0.7  // les soins sur un héros en plein PvP
	DRReset       = 15.0 // secondes avant qu'un contrôle retrouve toute sa durée
)

// Les durées successives d'un même contrôle sur un héros : 100 %, 50 %, 25 %, puis plus rien.
var drSteps = [...]float64{1, 0.5, 0.25, 0}

// Les contrôles qui s'affaiblissent quand on les répète.
var pvpControls = map[Effect]bool{Etourdi: true, Gel: true, Ralenti: true, Silence: true, Aveugle: true}

type diminish struct {
	n     int
	reset float64
}

// Faction : la faction du peuple du héros (concorde, sylvestre, cendres).
func (f *Fighter) Faction() string {
	if f.Hero == nil {
		return ""
	}
	return Races[f.Hero.Race].Faction
}

// InVillage : le héros est-il à Bourg-Lanterne (où l'on ne se bat pas) ?
func (a *Arena) InVillage(f *Fighter) bool {
	return math.Hypot(f.Body.X-a.Village[0], f.Body.Y-a.Village[1]) < VillageRadius || f.Body.Y >= float64(a.M.IndoorsFrom)
}

// foe dit si h peut frapper t : un monstre, un adversaire de duel ou du Cercle,
// ou un héros d'une faction ennemie quand vos deux drapeaux sont levés.
func (a *Arena) foe(h, t *Fighter) bool {
	if t == nil || t.Dead || t == h {
		return false
	}
	if t.Mon != nil {
		return !t.Mon.gone
	}
	if h.Hero == nil || t.Hero == nil {
		return false
	}
	if d := a.DuelOf(h.SID); d != nil {
		return d.Started() && d.Opposed(h.SID, t.SID)
	}
	if b := a.Cercle.BoutOf(h.SID); b != nil {
		return b.Opposed(h.SID, t.SID)
	}
	return a.AtWar(h, t)
}

// AtWar : deux héros de factions ennemies, drapeau levé, hors du village.
func (a *Arena) AtWar(h, t *Fighter) bool {
	hs, ts := h.Hero, t.Hero
	return hs.Flag && ts.Flag && h.Faction() != t.Faction() && !a.SameGroup(h.SID, t.SID) &&
		!a.InVillage(h) && !a.InVillage(t) && a.DuelOf(t.SID) == nil && a.Cercle.BoutOf(t.SID) == nil
}

// noFoe explique pourquoi on ne peut pas frapper cette cible.
func (a *Arena) noFoe(h, t *Fighter) {
	switch d := a.DuelOf(h.SID); {
	case t == nil || t.Dead || t.Hero == nil:
		a.refuse(h, "Choisis d'abord un monstre !")
	case d != nil && d.Opposed(h.SID, t.SID):
		a.refuse(h, "Attends la fin du compte à rebours !")
	case a.Cercle.BoutOf(h.SID) != nil:
		a.refuse(h, "Attends le signal du Cercle !")
	case h.Hero.Flag && t.Hero.Flag && h.Faction() != t.Faction() && (a.InVillage(h) || a.InVillage(t)):
		a.refuse(h, "On ne se bat pas au village !")
	case h.Hero.Flag && t.Hero.Flag && h.Faction() == t.Faction():
		a.refuse(h, t.Name+" est de ta faction.")
	case h.Hero.Flag && !t.Hero.Flag && h.Faction() != t.Faction():
		a.refuse(h, t.Name+" n'a pas levé son drapeau de guerre.")
	default:
		a.refuse(h, "Pour te battre contre "+t.Name+", défie-le en duel !")
	}
}

// ToggleFlag lève ou baisse le drapeau de guerre.
func (a *Arena) ToggleFlag(sid int) {
	h := a.Heroes[sid]
	if h == nil {
		return
	}
	hs := h.Hero
	if hs.Flag && hs.flagCD > 0 {
		a.refuse(h, "Tu t'es battu contre un héros il y a peu : attends encore "+strconv.Itoa(int(math.Ceil(hs.flagCD)))+" s.")
		return
	}
	hs.Flag = !hs.Flag
	v := 0
	if hs.Flag {
		v = 1
		a.Count(h, "drapeau", 1)
		a.refuse(h, "🚩 Drapeau de guerre levé ! Hors du village, les héros des autres factions qui l'ont levé peuvent t'attaquer.")
	} else {
		a.refuse(h, "Drapeau baissé : tu es en paix.")
	}
	a.emit(Event{Kind: EvFlag, Src: sid, Value: v})
}

// InPvP : le héros échange des coups avec des héros (soins réduits).
func (hs *HeroState) InPvP() bool { return hs.pvp < CombatTime }

// PvPDeath : le héros est tombé sous les coups d'un héros (il revient sans perte).
func (hs *HeroState) PvPDeath() bool { return hs.pvpDeath }

// pvpHit : un coup entre deux héros.
func (a *Arena) pvpHit(src, tgt *Fighter) {
	for _, f := range []*Fighter{src, tgt} {
		f.Hero.pvp = 0
		if f.Hero.Flag && a.DuelOf(f.SID) == nil && a.Cercle.BoutOf(f.SID) == nil {
			f.Hero.flagCD = FlagLowerWait
		}
	}
}

// applyCC pose un effet ; sur un héros, un contrôle venu d'un héros s'affaiblit à chaque répétition.
func (a *Arena) applyCC(src, f *Fighter, e Effect, dur float64, pow int) {
	if pvpControls[e] && f.Hero != nil && src.Hero != nil && src != f {
		hs := f.Hero
		if hs.dr == nil {
			hs.dr = map[Effect]*diminish{}
		}
		d := hs.dr[e]
		if d == nil {
			d = &diminish{}
			hs.dr[e] = d
		}
		if d.reset <= 0 {
			d.n = 0
		}
		mul := drSteps[min(d.n, len(drSteps)-1)]
		d.n++
		d.reset = DRReset
		if mul <= 0 {
			a.emit(Event{Kind: EvHit, Src: src.SID, Tgt: f.SID, Flags: FlagDodge})
			return
		}
		dur *= mul
	}
	f.Apply(e, dur, pow, src.SID)
}

func (a *Arena) tickPvP(h *Fighter, dt float64) {
	hs := h.Hero
	hs.pvp += dt
	hs.flagCD = math.Max(0, hs.flagCD-dt)
	for _, d := range hs.dr {
		d.reset -= dt
	}
}
