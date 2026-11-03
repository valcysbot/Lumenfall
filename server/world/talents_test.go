package world

import (
	"fmt"
	"go/format"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"
)

// rendement : dégâts et soins d'un héros en 60 s contre le Grand Sanglier,
// sans jamais mourir, avec un allié blessé à côté (pour les soins).
func rendement(c *Class, spec, lv int, tal Talents, seed int64) float64 {
	m, _ := LoadMap("glennmor")
	camps, _ := LoadCamps(m)
	a := NewArena(m, camps, rand.New(rand.NewSource(seed)))
	a.Temple = [2]float64{111.5, 98.5}
	var boss *Fighter
	for _, f := range a.Monsters {
		if f.Mon.Kind.Boss {
			boss = f
		}
	}
	boss.MaxHP = 1 << 30 // un mannequin qui ne tombe jamais
	in := heros(c, lv, spec)
	in.Talents = tal
	h := a.AddHero(1, "Essai", &Body{}, in)
	ami := a.AddHero(2, "Ami", &Body{}, heros(c, lv, spec))
	face(h, boss)
	ami.Body.Teleport(h.Body.X+1, h.Body.Y)
	total := 0.0
	for i := 0; i < 600; i++ {
		boss.HP = boss.MaxHP
		h.HP = h.MaxHP * 6 / 10
		ami.HP = ami.MaxHP / 2
		a.SetTarget(h.SID, boss.SID)
		for k := NSkills; k >= 1; k-- {
			sk := h.Hero.Skill(k)
			if sk == nil || sk.Level > h.Level || h.Hero.CD[k] > 0 || sk.Dash < 0 {
				continue
			}
			if sk.Target == TAlly {
				a.SetTarget(h.SID, ami.SID)
			}
			a.UseSkill(h.SID, k)
			a.SetTarget(h.SID, boss.SID)
		}
		if dist(h, boss) > a.autoReach(h, boss) {
			h.Body.Teleport(boss.Body.X+0.9, boss.Body.Y)
			ami.Body.Teleport(boss.Body.X+1.5, boss.Body.Y)
		}
		h.Body.Step(a.M, 0.1)
		a.Tick(0.1)
		for _, e := range a.Events {
			if e.Kind == EvHit && e.Src == h.SID {
				total += float64(e.Value)
			}
		}
		a.Events = a.Events[:0]
	}
	return total
}

// poids : combien les talents conseillés rendent la voie plus forte au niveau 50.
func poids(c *Class, v int) float64 {
	sans, avec := 0.0, 0.0
	for s := int64(0); s < 3; s++ {
		sans += rendement(c, v, 50, Talents{}, s)
		avec += rendement(c, v, 50, RecommendTalents(c, v, 50), s)
	}
	return avec / sans
}

// Le but du Conseil des Talents : environ 15 % de force en plus.
const poidsVise = 1.15

// TestEchelleDesTalents règle la force des talents de chaque voie (avec LF_ECRIRE=1).
func TestEchelleDesTalents(t *testing.T) {
	if os.Getenv("LF_ECRIRE") == "" {
		t.Skip("LF_ECRIRE=1 pour régler l'échelle des talents")
	}
	var lines []string
	for _, c := range Classes {
		for v := range c.Specs {
			// On essaie toutes les échelles de 0 à 1,5 et on garde la plus proche du but
			// (les arrondis rendent la courbe un peu irrégulière).
			f, best := 0.0, 9.0
			for x := 0.0; x <= 1.5001; x += 0.05 {
				buildSpec(c, v, x)
				if d := math.Abs(poids(c, v) - poidsVise); d < best {
					f, best = x, d
				}
			}
			buildSpec(c, v, f)
			t.Logf("%s/%s : échelle %.2f, ×%.2f", c.ID, c.Specs[v].ID, f, poids(c, v))
			lines = append(lines, fmt.Sprintf("\t%q: %.3f,", c.ID+"/"+c.Specs[v].ID, f))
		}
	}
	sort.Strings(lines)
	src, _ := os.ReadFile("talents_echelle.go")
	head := string(src[:strings.Index(string(src), "var talentScale")])
	out := head + "var talentScale = map[string]float64{\n" + strings.Join(lines, "\n") + "\n}\n"
	b, err := format.Source([]byte(out))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("talents_echelle.go", b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestPoidsDesTalents : les talents conseillés rendent un héros de niveau 50
// environ 15 % plus fort (Conseil des Talents), dans chaque voie.
func TestPoidsDesTalents(t *testing.T) {
	for _, c := range Classes {
		for v := 0; v < 3; v++ {
			r := poids(c, v)
			t.Logf("%-17s %-22s ×%.2f", c.ID, c.Specs[v].ID, r)
			if r < 1.08 || r > 1.25 {
				t.Errorf("%s/%s : les talents changent la force de ×%.2f", c.ID, c.Specs[v].ID, r)
			}
		}
	}
}

// TestArbres : 10 paliers par voie, 3 talents par palier, tous différents.
func TestArbres(t *testing.T) {
	n := 0
	for _, c := range Classes {
		for v := 0; v < 3; v++ {
			tree := c.Tree(v)
			if len(tree) != NTiers {
				t.Fatalf("%s/%d : %d paliers", c.ID, v, len(tree))
			}
			seen := map[string]bool{}
			for i, tr := range tree {
				if tr.Level != TierLevel(i) {
					t.Errorf("%s : palier %d au niveau %d", c.ID, i, tr.Level)
				}
				for _, tl := range tr.T {
					if seen[tl.Desc] || seen[tl.Name] || tl.Name == "" || tl.Max < 1 {
						t.Errorf("%s/%d : talent %q en double ou vide", c.ID, v, tl.Name)
					}
					seen[tl.Desc], seen[tl.Name] = true, true
					if i > 0 || v == 0 {
						n++
					}
				}
			}
		}
	}
	if n != 17*3+51*27 {
		t.Errorf("%d talents", n)
	}
}

// TestPlacerSesTalents : points, paliers, temple et changement de voie.
func TestPlacerSesTalents(t *testing.T) {
	a, _ := arene(t)
	c := ClassByID("mage")
	in := heros(c, 12, -1)
	in.Quest = VoieQuest
	h := a.AddHero(5, "Essai", &Body{X: a.Temple[0], Y: a.Temple[1]}, in)
	h.Hero.Gold = 1000
	var tal Talents
	tal[1][0] = 1
	if a.SetTalents(5, tal) == nil {
		t.Fatal("un palier de voie sans voie")
	}
	tal = Talents{}
	tal[0][0] = 3
	if err := a.SetTalents(5, tal); err != nil {
		t.Fatal(err)
	}
	if a.ChooseSpec(5, 0) != nil {
		t.Fatal("voie refusée")
	}
	if h.Hero.Talents[0][0] != 3 {
		t.Fatal("le palier commun est perdu en prenant la voie")
	}
	before := h.Hero.Skill(1).Power
	tal[1][0] = 3
	tal[0][2] = 3 // 3+3+3 = 9 > 8 points au niveau 12
	if a.SetTalents(5, tal) == nil {
		t.Fatal("trop de points accepté")
	}
	tal[0][2] = 2
	if err := a.SetTalents(5, tal); err != nil {
		t.Fatal(err)
	}
	_ = before
	// Reprendre des points : seulement au temple, contre de l'or.
	h.Body.Teleport(a.Temple[0]+40, a.Temple[1])
	tal[0][0] = 0
	if a.SetTalents(5, tal) == nil {
		t.Fatal("points repris loin du temple")
	}
	h.Body.Teleport(a.Temple[0], a.Temple[1])
	if err := a.SetTalents(5, tal); err != nil || h.Hero.Gold != 1000-TalentResetGold(12) {
		t.Fatalf("reprise au temple : %v, or %d", err, h.Hero.Gold)
	}
}
