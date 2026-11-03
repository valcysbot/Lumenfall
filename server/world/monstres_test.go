package world

import "testing"

// Chaque petit monstre finit par annoncer son coup quand on se bat à distance.
func TestLesMonstresAnnoncentLeurCoup(t *testing.T) {
	for sorte, nom := range map[string]string{"sanglier": "Charge !", "loup": "Bond !", "corbeau_efface": "Piqué !"} {
		a, h := arene(t)
		m := premier(a, sorte)
		h.Level, h.MaxHP, h.HP = 10, HeroMaxHP(10), HeroMaxHP(10)
		// Le héros se tient à 4 cases, là où il y a de la place.
		placed := false
		for _, d := range [][2]float64{{4, 0}, {-4, 0}, {0, 4}, {0, -4}, {3, 3}, {-3, -3}} {
			if a.M.Free(m.Body.X+d[0], m.Body.Y+d[1]) && a.clearLine(m, d[0]/4, d[1]/4, 4) >= 3.5 {
				h.Body.Teleport(m.Body.X+d[0], m.Body.Y+d[1])
				placed = true
				break
			}
		}
		if !placed {
			t.Fatalf("%s : pas de place autour du camp", sorte)
		}
		a.hit(h, m, 1, Physique, 0, 0, 1)
		m.Mon.specialCD = 0
		var vu *Telegraph
		for i := 0; i < 5 && vu == nil; i++ {
			a.Tick(0.1)
			for _, tl := range a.Teles {
				if tl.Src == m.SID {
					vu = tl
				}
			}
		}
		if vu == nil {
			t.Fatalf("%s n'a pas annoncé son coup", sorte)
		}
		if vu.Name != nom || vu.Left <= 0.5 {
			t.Fatalf("%s : coup %q annoncé %.2f s avant", sorte, vu.Name, vu.Left)
		}
		// Le coup part, puis le monstre reprend la chasse.
		combat(a, 1.5)
		if m.Mon.windup > 0 {
			t.Fatalf("%s reste figé après son coup", sorte)
		}
	}
}

// Après une morsure, un loup fait parfois un pas de côté : il ne reste pas planté.
func TestLesLoupsTournentAutour(t *testing.T) {
	a, h := arene(t)
	m := premier(a, "loup")
	h.Level, h.MaxHP, h.HP = 10, HeroMaxHP(10), HeroMaxHP(10)
	face(h, m)
	a.hit(h, m, 1, Physique, 0, 0, 1)
	danse := false
	for i := 0; i < 200 && !danse; i++ {
		a.Tick(0.1)
		danse = m.Mon.dance > 0
	}
	if !danse {
		t.Fatal("le loup ne bouge jamais entre deux morsures")
	}
}
