package world

import (
	"errors"
	"math"
	"strconv"
)

// Le moteur des compétences : il fait tourner les SkillDef de classes.go
// sans rien savoir de chaque classe.

// SpecChangeGold : le prix, au temple, pour changer de voie.
func SpecChangeGold(level int) int { return 20 * level }

// UseSkill : compétence k (1 à 7, et 8 pour l'ultime de la voie) sur la cible actuelle.
func (a *Arena) UseSkill(sid, k int) {
	h := a.Heroes[sid]
	if !a.canAct(h) || k < 1 || k > NSkills {
		return
	}
	hs := h.Hero
	sk := hs.Skill(k)
	if sk == nil {
		a.refuse(h, "Choisis d'abord ta voie : l'ultime arrive avec elle, au niveau 10.")
		return
	}
	if h.Level < sk.Level {
		a.refuse(h, "Tu apprendras "+sk.Name+" au niveau "+strconv.Itoa(sk.Level)+".")
		return
	}
	if hs.CD[k] > 0 || hs.GCD > 0 || hs.Cast != 0 {
		return // le client grise déjà le bouton : inutile d'insister
	}
	if sk.Magic && h.Has(Silence) {
		a.refuse(h, "Tu es réduit au silence !")
		return
	}
	if !a.canPay(h, sk, true) {
		return
	}
	t, ok := a.skillTarget(h, sk, true)
	if !ok {
		return
	}
	if sk.Cast > 0 {
		hs.Cast, hs.CastLeft, hs.castOn = k, sk.Cast, t.SID
		hs.Blocking = false
		hs.GCD = GCD
		hs.me = true
		h.dirty = true
		if t != h {
			h.Body.Face(t.Body.X, t.Body.Y)
		}
		a.emit(Event{Kind: EvCast, Src: h.SID, Skill: k, Value: int(sk.Cast * 1000)})
		return
	}
	a.execute(h, k, sk, t)
}

// skillTarget : sur qui part la compétence, et est-ce à portée ?
func (a *Arena) skillTarget(h *Fighter, sk *SkillDef, say bool) (*Fighter, bool) {
	t := a.Get(h.Hero.Target)
	switch sk.Target {
	case TSelf:
		return h, true
	case TAlly:
		on := h
		if t != nil && t.Hero != nil && !t.Dead && !a.foe(h, t) {
			on = t
		}
		if dist(h, on) > sk.Range {
			if say {
				a.refuse(h, "Trop loin pour l'atteindre !")
			}
			return nil, false
		}
		return on, true
	}
	if !a.foe(h, t) {
		if say {
			a.noFoe(h, t)
		}
		return nil, false
	}
	reach := sk.Range
	if reach <= 0 {
		reach = reachOf(t)
	}
	if dist(h, t) > reach {
		if say {
			a.refuse(h, "Trop loin pour "+sk.Name+" !")
		}
		return nil, false
	}
	return t, true
}

// castTargetOK : à la fin d'une incantation, la cible est-elle toujours là ?
func (a *Arena) castTargetOK(h *Fighter, sk *SkillDef, t *Fighter) bool {
	if t == nil || t.Dead {
		return false
	}
	if sk.Magic && h.Has(Silence) {
		return false
	}
	switch sk.Target {
	case TSelf:
		return t == h
	case TAlly:
		return !a.foe(h, t) && t.Hero != nil && dist(h, t) <= sk.Range+1
	}
	reach := sk.Range
	if reach <= 0 {
		reach = reachOf(t)
	}
	return a.foe(h, t) && dist(h, t) <= reach+1
}

// canPay : le héros a-t-il de quoi lancer la compétence ?
func (a *Arena) canPay(h *Fighter, sk *SkillDef, say bool) bool {
	hs := h.Hero
	r := hs.Class.Res
	ok := true
	text := "Pas assez de " + r.Name + " !"
	switch {
	case r.Heat:
		ok = hs.overheat <= 0
		text = "Surchauffe ! Attends que la Vapeur retombe."
	case sk.Finisher:
		ok = hs.Res >= 1
	default:
		ok = hs.Res+1e-6 >= sk.Cost
	}
	if !ok && say {
		a.refuse(h, text)
	}
	return ok
}

// execute : la compétence part pour de bon.
func (a *Arena) execute(h *Fighter, k int, sk *SkillDef, t *Fighter) {
	hs := h.Hero
	r := hs.Class.Res

	// Le prix, et ce que la compétence rapporte.
	finisher := 0.0
	switch {
	case r.Heat:
		hs.Res = math.Max(0, hs.Res+sk.Cost-sk.Gain)
		if hs.Res >= r.Max {
			hs.Res = r.Max
			hs.overheat = OverheatTime
			a.refuse(h, "Surchauffe ! Ton armure crache sa vapeur.")
		}
	case sk.Finisher:
		finisher = math.Floor(hs.Res)
		hs.Res = 0
	default:
		hs.Res -= sk.Cost
		a.gainRes(h, sk.Gain)
	}
	if r.ID == "harmonie" && k != hs.lastSkill {
		a.gainRes(h, 10) // le barde enchaîne des airs différents
	}
	hs.lastSkill = k
	hs.CD[k] = sk.CD
	if sk.Cast == 0 {
		hs.GCD = GCD
	}
	hs.Blocking = false
	hs.me = true
	h.dirty = true

	if t != h {
		h.Body.Face(t.Body.X, t.Body.Y)
	}

	// Le saut : vers la cible, ou en arrière.
	if sk.Dash > 0 && t != h {
		d := math.Max(0, dist(h, t)-reachOf(t)*0.8)
		if d > 0.2 {
			h.Body.Dash(t.Body.X-h.Body.X, t.Body.Y-h.Body.Y, math.Min(d, sk.Dash), RollTime)
		}
	} else if sk.Dash < 0 {
		ang := float64(h.Body.Dir)*math.Pi/4 + math.Pi
		dx, dy := math.Cos(ang), math.Sin(ang)
		if f := a.Get(hs.Target); a.foe(h, f) {
			dx, dy = h.Body.X-f.Body.X, h.Body.Y-f.Body.Y
		}
		h.Body.Dash(dx, dy, -sk.Dash, RollTime)
	}

	// Les dégâts : la cible, ou toute une zone.
	var zone []*Fighter
	if sk.Radius > 0 && sk.Target != TAlly {
		zone = a.foesAround(h, t, sk.Radius)
	} else if sk.Target == TFoe {
		zone = []*Fighter{t}
	}
	if sk.Power > 0 {
		base := h.Power(sk.Magic) * sk.Power
		if sk.Finisher {
			base *= 1 + 0.6*finisher
		}
		for _, f := range zone {
			dmg := base * a.championMul(h, f)
			if sk.Execute && f.Pct() < 0.3 {
				dmg *= 2
			}
			n := a.hit(h, f, dmg, sk.Elem, k, hs.Stats.Crit, 1+sk.Threat)
			if sk.Drain > 0 && n > 0 {
				a.heal(h, h, int(math.Round(float64(n)*sk.Drain)), k)
			}
		}
	} else if sk.Threat > 0 {
		for _, f := range zone {
			if f.Mon != nil {
				a.addThreat(f, h, sk.Threat*h.Weapon()*hs.Stats.Threat)
			}
		}
	}

	// Les soins.
	if sk.Heal > 0 {
		a.heal(h, t, int(math.Round(sk.Heal*float64(t.MaxHP)*hs.Stats.Heal)), k)
	}
	if sk.GroupHeal > 0 {
		for _, o := range a.group(h) {
			a.heal(h, o, int(math.Round(sk.GroupHeal*float64(o.MaxHP)*hs.Stats.Heal)), k)
		}
	}

	// Les effets.
	for _, ap := range sk.Apply {
		var who []*Fighter
		switch ap.Who {
		case WCible:
			who = []*Fighter{t}
		case WSoi:
			who = []*Fighter{h}
		case WGroupe:
			who = a.group(h)
		case WZone:
			who = zone
			if sk.Radius <= 0 && len(zone) == 0 && t != h {
				who = []*Fighter{t}
			}
		}
		pow := a.applyPower(h, sk, ap)
		for _, f := range who {
			a.applyTo(h, f, ap, pow)
		}
	}

	if sk.Anim != "" {
		a.emit(Event{Kind: EvAnim, Src: h.SID, Text: sk.Anim})
	}
}

// foesAround : les ennemis du héros autour de la cible (ou de lui-même).
func (a *Arena) foesAround(h, c *Fighter, radius float64) []*Fighter {
	var out []*Fighter
	for _, m := range a.Monsters {
		if a.foe(h, m) && dist(c, m) <= radius+m.Mon.Kind.Radius {
			out = append(out, m)
		}
	}
	for _, o := range a.Heroes {
		if o != h && a.foe(h, o) && dist(c, o) <= radius {
			out = append(out, o)
		}
	}
	return out
}

// group : le héros et ses alliés proches.
func (a *Arena) group(h *Fighter) []*Fighter {
	out := []*Fighter{h}
	for _, o := range a.Heroes {
		if o != h && !o.Dead && dist(h, o) <= GroupRadius && !a.foe(h, o) {
			out = append(out, o)
		}
	}
	return out
}

// applyPower : la force d'un effet posé par le héros.
func (a *Arena) applyPower(h *Fighter, sk *SkillDef, ap Apply) int {
	if ap.Pow <= 0 {
		return 0
	}
	s := h.Hero.Stats
	p := ap.Pow * h.Power(sk.Magic)
	switch ap.E {
	case Bouclier:
		p *= s.Shields
	case Regen:
		p *= s.Heal
	case Compagnon:
		p *= s.Pet
	}
	return int(math.Max(1, math.Round(p)))
}

// applyTo pose un effet, avec ce que chaque effet a de particulier.
func (a *Arena) applyTo(h, f *Fighter, ap Apply, pow int) {
	if f == nil || f.Dead {
		return
	}
	switch ap.E {
	case Peur, Charme:
		if f.Mon == nil || f.Mon.Kind.Boss {
			return // les boss ne se laissent ni charmer ni effrayer
		}
	case Provoque:
		if f.Mon == nil {
			return
		}
		a.addThreat(f, h, 1)
	case Furtif:
		// Les monstres oublient le héros.
		for _, m := range a.Monsters {
			delete(m.Mon.threat, f.SID)
		}
	}
	f.Apply(ap.E, ap.Dur, pow, h.SID)
	if ap.E == Compagnon && f.Hero != nil {
		f.Hero.petCD = 0.5
	}
}

// ChooseSpec : le héros choisit sa voie (après l'épreuve), ou en change au temple.
func (a *Arena) ChooseSpec(sid, i int) error {
	h := a.Heroes[sid]
	if h == nil || i < 0 || i > 2 {
		return errors.New("Cette voie n'existe pas.")
	}
	hs := h.Hero
	if h.Level < SpecLevel {
		return errors.New("Les voies s'ouvrent au niveau 10.")
	}
	if hs.Spec == i {
		return errors.New("Tu suis déjà cette voie.")
	}
	if hs.Spec < 0 {
		if hs.Quest < VoieQuest {
			return errors.New("Termine d'abord l'épreuve : " + strconv.Itoa(VoieQuest-hs.Quest) + " Effacés à vaincre.")
		}
	} else {
		if math.Hypot(h.Body.X-a.Temple[0], h.Body.Y-a.Temple[1]) > VillageRadius {
			return errors.New("On ne change de voie qu'au temple.")
		}
		price := SpecChangeGold(h.Level)
		if hs.Gold < price {
			return errors.New("Changer de voie coûte " + strconv.Itoa(price) + " or.")
		}
		hs.Gold -= price
	}
	hs.Spec = i
	hs.Talents = hs.Talents.KeepCommon() // les points de l'ancienne voie reviennent
	hs.CD[NSkills] = 0
	h.Restat()
	a.emit(Event{Kind: EvVoie, Tgt: h.SID, Text: "choisie", Value: i})
	return nil
}

// SetTalents : le héros place ses points de talent. En ajouter est gratuit ;
// en reprendre se fait au temple, contre de l'or.
func (a *Arena) SetTalents(sid int, t Talents) error {
	h := a.Heroes[sid]
	if h == nil {
		return errors.New("Héros introuvable.")
	}
	hs := h.Hero
	if err := CheckTalents(hs.Class, hs.Spec, h.Level, t); err != nil {
		return err
	}
	if t == hs.Talents {
		return nil
	}
	if !t.Within(hs.Talents) {
		if math.Hypot(h.Body.X-a.Temple[0], h.Body.Y-a.Temple[1]) > VillageRadius {
			return errors.New("On ne reprend ses points de talent qu'au temple.")
		}
		price := TalentResetGold(h.Level)
		if hs.Gold < price {
			return errors.New("Reprendre ses talents coûte " + strconv.Itoa(price) + " or.")
		}
		hs.Gold -= price
	}
	hs.Talents = t
	h.Restat()
	return nil
}

// SetAttrs : le héros place ses points d'attributs.
func (a *Arena) SetAttrs(sid int, at Attrs) error {
	h := a.Heroes[sid]
	if h == nil {
		return errors.New("Héros introuvable.")
	}
	if err := ValidateAttrs(h.Hero.Attrs, at, h.Level); err != nil {
		return err
	}
	h.Hero.Attrs = at
	h.Restat()
	return nil
}
