package world

import (
	"math"
	"math/rand"
	"testing"
)

// heros : un héros de la classe c, de niveau lv, points placés comme conseillé.
func heros(c *Class, lv, spec int) HeroInit {
	return HeroInit{Level: lv, Potions: MaxPotions, Class: c.ID, Spec: spec, Race: "humain", Attrs: Recommend(c, PointsAt(lv))}
}

// soigneur : la compétence k soigne-t-elle le héros ?
func soigneur(sk *SkillDef) bool {
	if sk.Heal > 0 || sk.GroupHeal > 0 {
		return true
	}
	for _, ap := range sk.Apply {
		if (ap.E == Regen || ap.E == Bouclier) && ap.Who != WZone {
			return true
		}
	}
	return false
}

// duel simule un héros de niveau lv qui attaque un camp et joue
// raisonnablement : compétences dès qu'elles sont prêtes, soins et potions
// quand il faiblit. Renvoie s'il a gagné, sa vie restante et la durée.
func duel(t *testing.T, c *Class, spec int, sorte string, camp, lv int, seed int64) (bool, int, float64) {
	t.Helper()
	m, _ := LoadMap("glennmor")
	camps, _ := LoadCamps(m)
	a := NewArena(m, camps, rand.New(rand.NewSource(seed)))
	a.Temple = [2]float64{111.5, 98.5}
	a.Village = [2]float64{95.5, 100.5}
	n := 0
	var cible []*Fighter
	for ci, cp := range camps {
		if cp.Kind == sorte {
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
	h := a.AddHero(1, "Essai", &Body{}, heros(c, lv, spec))
	face(h, cible[0])
	a.hit(h, cible[0], 1, Physique, 0, 0, 1)
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
		if h.Pct() < 0.4 && h.Hero.Potions > 0 && h.Hero.PotionCD <= 0 {
			a.Potion(h.SID)
		}
		for k := NSkills; k >= 1; k-- {
			sk := h.Hero.Skill(k)
			if sk == nil || sk.Level > h.Level || h.Hero.CD[k] > 0 {
				continue
			}
			if soigneur(sk) {
				if h.Pct() > 0.6 {
					continue
				}
				a.SetTarget(h.SID, h.SID)
				a.UseSkill(h.SID, k)
				a.SetTarget(h.SID, t0.SID)
				continue
			}
			if sk.Dash < 0 {
				continue // le bot ne recule pas
			}
			a.UseSkill(h.SID, k)
		}
		// Le héros suit sa cible au besoin (les tireurs restent à distance).
		if t := a.Get(h.Hero.Target); t != nil && t.Mon != nil && h.Hero.Cast == 0 && dist(h, t) > a.autoReach(h, t) {
			r := math.Max(0.9, a.autoReach(h, t)-1)
			h.Body.Teleport(t.Body.X+r, t.Body.Y)
		}
		h.Body.Step(a.M, 0.1)
		a.Tick(0.1)
		a.Events = a.Events[:0]
	}
	return false, h.HP, 120
}

// TestEquilibre : chaque classe doit pouvoir faire sa route dans le val.
func TestEquilibre(t *testing.T) {
	cas := []struct {
		sorte     string
		camp, lv  int
		spec      int
		victoires int // sur 10 essais, au moins
	}{
		{"sanglier", 0, 1, -1, 8},
		{"loup", 0, 4, -1, 6},
		{"loup", 0, 6, -1, 8},
		{"loup_efface", 0, 7, -1, 5},
		{"corbeau_efface", 0, 8, -1, 6},
		{"grand_sanglier", 0, 10, 0, 0},
	}
	for _, c := range Classes {
		for _, cs := range cas {
			specs := []int{cs.spec}
			if cs.spec >= 0 {
				specs = []int{0, 1, 2}
			}
			for _, sp := range specs {
				gagne, vie, temps := 0, 0, 0.0
				for s := int64(0); s < 10; s++ {
					ok, hp, d := duel(t, c, sp, cs.sorte, cs.camp, cs.lv, s)
					if ok {
						gagne++
						vie += hp
						temps += d
					}
				}
				t.Logf("%-17s %d %-15s niv. %2d : %2d/10, vie %4d, %3.0f s", c.ID, sp, cs.sorte, cs.lv, gagne, vie/max(1, gagne), temps/float64(max(1, gagne)))
				if gagne < cs.victoires {
					t.Errorf("%s contre %s au niveau %d : seulement %d victoires sur 10", c.ID, cs.sorte, cs.lv, gagne)
				}
			}
		}
	}
}
