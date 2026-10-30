package world

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestHabitantsDeGlennmor(t *testing.T) {
	m, err := LoadMap("glennmor")
	if err != nil {
		t.Fatal(err)
	}
	npcs, err := LoadNPCs(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(npcs) < 10 {
		t.Fatalf("%d habitants seulement", len(npcs))
	}
	for _, n := range npcs {
		if n.ID == "mabon" && n.Work[1] < 230 {
			t.Errorf("Mabon doit rester dans son auberge")
		}
	}
}

// Une journée complète, seconde par seconde : chacun va au travail le matin,
// flâne autour, puis rentre chez lui et disparaît le soir (l'inverse pour
// Corbin, l'allumeur de lanternes).
func TestUneJourneeAGlennmor(t *testing.T) {
	m, _ := LoadMap("glennmor")
	npcs, err := LoadNPCs(m)
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(1))
	matin := time.UnixMilli(0).Add(DayLength * 1000)
	soir := matin.Add(DaylightLength)
	for _, n := range npcs {
		n.Start(matin.Add(-time.Minute)) // la nuit : chacun chez soi (sauf Corbin)
	}
	appear, vanish := map[string]int{}, map[string]int{}
	loin := map[string]float64{}
	for s := 0; s < int(DayLength.Seconds()); s++ {
		now := matin.Add(time.Duration(s) * time.Second)
		for _, n := range npcs {
			switch n.Think(m, now, rng) {
			case NPCAppear:
				appear[n.ID]++
			case NPCVanish:
				vanish[n.ID]++
			}
			for k := 0; k < 10; k++ {
				n.Body.Step(m, 0.1)
			}
			if !m.Free(n.Body.X, n.Body.Y) {
				t.Fatalf("%s est dans un mur en (%.1f, %.1f)", n.ID, n.Body.X, n.Body.Y)
			}
			if n.Visible && n.Awake(now) && now.Sub(matin) > 10*time.Minute && now.Before(soir) {
				loin[n.ID] = math.Max(loin[n.ID], math.Hypot(n.Body.X-n.Work[0], n.Body.Y-n.Work[1])-n.Wander)
			}
		}
	}
	for _, n := range npcs {
		if appear[n.ID] != 1 || vanish[n.ID] != 1 {
			t.Errorf("%s : sorti %d fois, rentré %d fois", n.ID, appear[n.ID], vanish[n.ID])
		}
		if !n.Nocturnal && loin[n.ID] > 1.5 {
			t.Errorf("%s s'éloigne de son travail (%.1f cases de trop)", n.ID, loin[n.ID])
		}
	}
}
