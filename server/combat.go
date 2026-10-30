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
			_ = d.BroadcastMessage(opCast, mustJSON(map[string]int{"s": e.Src, "k": e.Skill, "ms": e.Value}), nil, nil, true)
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
					"r": cent(t.R), "ms": int(t.Left * 1000),
				}), nil, nil, true)
			}
		case world.EvPhase:
			if m := a.Get(e.Src); m != nil && e.Value == 2 {
				_ = d.BroadcastMessage(opNotice, mustJSON(map[string]string{
					"m": "Le " + m.Name + " hurle : la Flétrissure le dévore !"}), nil, nil, true)
			}
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
	}
}

// sendMe envoie au joueur l'état de son héros que lui seul voit.
func sendMe(d runtime.MatchDispatcher, pl *player) {
	f := pl.fighter
	h := f.Hero
	ms := func(v float64) int { return int(math.Ceil(v * 1000)) }
	cd := make([]int, 5)
	for k := 1; k <= 5; k++ {
		cd[k-1] = ms(h.CD[k])
	}
	_ = d.BroadcastMessage(opMe, mustJSON(map[string]interface{}{
		"f": int(h.Ferveur), "en": int(h.Endurance * 10), "cd": cd, "gcd": ms(h.GCD),
		"po": h.Potions, "pocd": ms(h.PotionCD), "lv": f.Level, "xp": h.XP, "xpn": world.XPForLevel(f.Level),
		"or": h.Gold, "tr": h.Trophies, "hp": f.HP, "hpm": f.MaxHP,
	}), []runtime.Presence{pl.presence}, nil, true)
}
