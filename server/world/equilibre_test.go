package world

import (
	"math/rand"
	"testing"
)

// duel simule un Paladin de niveau lv qui attaque un camp et joue
// raisonnablement : compétences dès qu'elles sont prêtes, soin et potions
// quand il faiblit. Renvoie s'il a gagné, sa vie restante et la durée.
func duel(t *testing.T, sorte string, camp, lv int, seed int64) (bool, int, float64) {
	t.Helper()
	m, _ := LoadMap("glennmor")
	camps, _ := LoadCamps(m)
	a := NewArena(m, camps, rand.New(rand.NewSource(seed)))
	a.Temple = [2]float64{111.5, 98.5}
	a.Village = [2]float64{95.5, 100.5}
	n := 0
	var cible []*Fighter
	for ci, c := range camps {
		if c.Kind == sorte {
			if n == camp {
				for _, f := range a.Monsters {
					if f.Mon.Camp == ci {
						cible = append(cible, f)
					}
				}
			}
			n++
		}
	}
	h := a.AddHero(1, "Essai", &Body{}, lv, 0, 0, MaxPotions, nil)
	face(h, cible[0])
	a.hit(h, cible[0], 1, Physique, 0, 0, 1)
	aura := false
	for i := 0; i < 1200; i++ {
		alive := 0
		var t0 *Fighter
		for _, f := range cible {
			if !f.Dead {
				alive++
				if t0 == nil || dist(h, f) < dist(h, t0) {
					t0 = f
				}
			}
		}
		if alive == 0 {
			return true, h.HP, float64(i) / 10
		}
		if h.Dead {
			return false, 0, float64(i) / 10
		}
		if a.Get(h.Hero.Target) == nil || a.Get(h.Hero.Target).Dead {
			a.SetTarget(h.SID, t0.SID)
		}
		switch {
		case h.Pct() < 0.4 && h.Hero.Potions > 0 && h.Hero.PotionCD <= 0:
			a.Potion(h.SID)
		case h.Pct() < 0.55 && h.Hero.Ferveur >= 30:
			a.SetTarget(h.SID, h.SID)
			a.UseSkill(h.SID, 3)
			a.SetTarget(h.SID, t0.SID)
		case !aura:
			a.UseSkill(h.SID, 4)
			aura = true
		default:
			for _, k := range []int{5, 1, 2} {
				a.UseSkill(h.SID, k)
			}
		}
		// Le héros se tourne vers sa cible et la suit au besoin.
		if t := a.Get(h.Hero.Target); t != nil && t.Mon != nil && h.Hero.Cast == 0 && dist(h, t) > reachOf(t) {
			h.Body.Teleport(t.Body.X+0.9, t.Body.Y)
		}
		h.Body.Step(a.M, 0.1)
		a.Tick(0.1)
		a.Events = a.Events[:0]
	}
	return false, h.HP, 120
}

func TestEquilibre(t *testing.T) {
	cas := []struct {
		sorte     string
		camp, lv  int
		victoires int // sur 10 essais, au moins
	}{
		{"sanglier", 0, 1, 9},
		{"loup", 0, 4, 6},
		{"loup", 0, 6, 9},
		{"loup_efface", 0, 7, 5},
		{"corbeau_efface", 0, 8, 6},
		{"grand_sanglier", 0, 10, 0},
	}
	for _, c := range cas {
		gagne, vie, temps := 0, 0, 0.0
		for s := int64(0); s < 10; s++ {
			ok, hp, d := duel(t, c.sorte, c.camp, c.lv, s)
			if ok {
				gagne++
				vie += hp
				temps += d
			}
		}
		t.Logf("%-15s niv. %d : %d/10 victoires, vie moyenne restante %d, %.0f s", c.sorte, c.lv, gagne, vie/max(1, gagne), temps/float64(max(1, gagne)))
		if gagne < c.victoires {
			t.Errorf("%s au niveau %d : seulement %d victoires sur 10", c.sorte, c.lv, gagne)
		}
	}
}
