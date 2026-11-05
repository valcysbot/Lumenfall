package main

import (
	"math"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/valcysbot/lumenfall/server/world"
)

// sendCombat diffuse ce que l'arène a produit pendant ce tick : coups, morts,
// animations, vie des combattants, et à chaque joueur ce qui ne regarde que lui.
func (w *worldMatch) sendCombat(d runtime.MatchDispatcher, s *worldState, tick int64) {
	a := s.arena
	bySID := make(map[int]*player, len(s.players))
	for _, pl := range s.players {
		bySID[pl.sid] = pl
	}
	only := func(sid int) []runtime.Presence {
		if pl := bySID[sid]; pl != nil {
			return []runtime.Presence{pl.presence}
		}
		return nil
	}
	var hits [][6]int
	var anims [][2]interface{}
	for _, e := range a.Events {
		switch e.Kind {
		case world.EvHit:
			hits = append(hits, [6]int{e.Src, e.Tgt, e.Value, e.Flags, e.Skill, int(e.Elem)})
		case world.EvAnim:
			anims = append(anims, [2]interface{}{e.Src, e.Text})
		case world.EvRespawn:
			anims = append(anims, [2]interface{}{e.Src, "releve"})
		case world.EvCast:
			_ = d.BroadcastMessage(opCast, mustJSON(map[string]interface{}{"s": e.Src, "k": e.Skill, "ms": e.Value, "t": e.Text}), nil, nil, true)
		case world.EvDeath:
			_ = d.BroadcastMessage(opDeath, mustJSON(map[string]int{"s": e.Tgt, "k": e.Src, "f": e.Flags}), nil, nil, true)
		case world.EvLevel:
			_ = d.BroadcastMessage(opLevel, mustJSON(map[string]int{"s": e.Src, "lv": e.Value}), nil, nil, true)
		case world.EvSpawn:
			if m := a.Get(e.Src); m != nil {
				_ = d.BroadcastMessage(opJoin, mustJSON(describeMonster(m)), nil, nil, true)
			}
		case world.EvDespawn:
			_ = d.BroadcastMessage(opLeave, mustJSON(map[string]int{"s": e.Src}), nil, nil, true)
		case world.EvLoot:
			if to := only(e.Tgt); to != nil {
				_ = d.BroadcastMessage(opLoot, mustJSON(map[string]interface{}{"m": e.Src, "or": e.Value, "xp": e.XP, "tr": e.Text}), to, nil, true)
			}
		case world.EvTele:
			if t := a.Tele(e.Value); t != nil {
				_ = d.BroadcastMessage(opTele, mustJSON(map[string]interface{}{
					"s": t.Src, "l": t.Line, "a": [2]int{cent(t.X), cent(t.Y)}, "b": [2]int{cent(t.X2), cent(t.Y2)},
					"r": cent(t.R), "ms": int(t.Left * 1000), "n": t.Name,
				}), nil, nil, true)
			}
		case world.EvPhase:
			if m := a.Get(e.Src); m != nil && e.Value == 2 {
				_ = d.BroadcastMessage(opNotice, mustJSON(map[string]string{
					"m": "Le " + m.Name + " hurle : la Flétrissure le dévore !"}), nil, nil, true)
			}
		case world.EvVoie:
			if to := only(e.Tgt); to != nil {
				_ = d.BroadcastMessage(opVoieOut, mustJSON(map[string]interface{}{"k": e.Text, "n": e.Value, "max": world.VoieQuest}), to, nil, true)
			}
		case world.EvDuel:
			sendDuel(d, a, e, only)
		case world.EvGround, world.EvItem, world.EvNode, world.EvMetier, world.EvLook, world.EvTrade:
			sendActeV(d, s, e, only)
		case world.EvGroup, world.EvGuildXP, world.EvFlag, world.EvGloire, world.EvCercle:
			s.sendActeVI(s.ctx, s.logger, s.db, d, e)
		case world.EvError:
			if to := only(e.Tgt); to != nil {
				_ = d.BroadcastMessage(opRefuse, mustJSON(map[string]string{"m": e.Text}), to, nil, true)
			}
		}
	}
	a.Events = a.Events[:0]
	if len(hits) > 0 {
		_ = d.BroadcastMessage(opHits, mustJSON(map[string]interface{}{"h": hits}), nil, nil, true)
	}
	if len(anims) > 0 {
		_ = d.BroadcastMessage(opAnim, mustJSON(map[string]interface{}{"a": anims}), nil, nil, true)
	}

	var vitals [][5]int
	for _, pl := range s.players {
		if f := pl.fighter; f.Dirty() {
			vitals = append(vitals, [5]int{f.SID, f.HP, f.MaxHP, f.Bits(), f.Level})
		}
	}
	for _, f := range a.Monsters {
		if f.Dirty() && f.Visible() {
			vitals = append(vitals, [5]int{f.SID, f.HP, f.MaxHP, f.Bits(), f.Level})
		}
	}
	if len(vitals) > 0 {
		_ = d.BroadcastMessage(opVitals, mustJSON(map[string]interface{}{"v": vitals}), nil, nil, true)
	}
	for _, pl := range s.players {
		if pl.fighter.Hero.TakeMe() {
			sendMe(d, pl)
		}
		if inv := pl.fighter.Hero.Inv; inv != nil && inv.TakeDirty() {
			sendSac(d, pl)
			sendMe(d, pl) // les potions du sac comptent sur le bouton R
			if pl.panel[0] != "" {
				s.openPanel(d, pl, pl.panel[0], pl.panel[1])
			}
		}
	}
}

// sendMe envoie au joueur l'état de son héros que lui seul voit.
func sendMe(d runtime.MatchDispatcher, pl *player) {
	f := pl.fighter
	h := f.Hero
	ms := func(v float64) int { return int(math.Ceil(v * 1000)) }
	po := h.Potions
	if h.Inv != nil {
		for _, id := range []string{"petite_potion", "potion", "grande_potion"} {
			po += h.Inv.Count(id)
		}
	}
	cd := make([]int, world.NSkills)
	cdt := make([]int, world.NSkills) // la recharge complète, talents compris
	for k := 1; k <= world.NSkills; k++ {
		cd[k-1] = ms(h.CD[k])
		if sk := h.Skill(k); sk != nil {
			cdt[k-1] = ms(sk.CD)
		}
	}
	_ = d.BroadcastMessage(opMe, mustJSON(map[string]interface{}{
		"r": int(h.Res), "rm": int(h.Class.Res.Max), "oh": ms(h.Overheat()),
		"en": int(h.Endurance * 10), "cd": cd, "cdt": cdt, "gcd": ms(h.GCD),
		"po": po, "pof": h.Potions, "pocd": ms(h.PotionCD), "lv": f.Level, "xp": h.XP, "xpn": world.XPForLevel(f.Level),
		"or": h.Gold, "tr": h.Trophies, "hp": f.HP, "hpm": f.MaxHP,
		"cl": h.Class.ID, "v": h.Spec, "q": h.Quest, "qm": world.VoieQuest, "at": h.Attrs, "pt": world.PointsAt(f.Level),
		"mir": pl.hero.miroir, "ta": h.Talents, "tp": world.TalentPoints(f.Level),
	}), []runtime.Presence{pl.presence}, nil, true)
}

// sendDuel raconte un duel : le défi au seul héros défié, le reste à tous.
func sendDuel(d runtime.MatchDispatcher, a *world.Arena, e world.Event, only func(int) []runtime.Presence) {
	msg := map[string]interface{}{"k": e.Text, "id": e.Value, "a": e.Src, "b": e.Tgt}
	switch e.Text {
	case "defi":
		to := only(e.Tgt)
		if to == nil {
			return
		}
		name := ""
		if f := a.Get(e.Src); f != nil {
			name = f.Name
		}
		msg["n"] = name
		msg["sec"] = int(world.DuelInviteTime)
		msg["g"] = e.Flags // 1 : un duel de groupe
		_ = d.BroadcastMessage(opDuel, mustJSON(msg), to, nil, true)
		return
	case "debut":
		du := a.Duel(e.Value)
		if du == nil {
			return
		}
		msg["c"] = [2]int{cent(du.X), cent(du.Y)}
		msg["r"] = cent(world.DuelRadius)
		msg["sec"] = int(world.DuelCountdown)
		msg["camps"] = du.Sides
	case "sort":
		msg["s"] = e.Src
	case "compte":
		msg["n"] = e.Flags
	case "fin":
		msg["f"] = e.Flags
	}
	_ = d.BroadcastMessage(opDuel, mustJSON(msg), nil, nil, true)
}
