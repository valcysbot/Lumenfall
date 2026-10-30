package world

import (
	"testing"
	"time"
)

func TestJourEtNuit(t *testing.T) {
	debut := time.UnixMilli(0).Add(DayLength * 1000) // un lever de soleil
	cas := []struct {
		apres time.Duration
		nuit  bool
	}{
		{0, false},
		{79 * time.Minute, false},
		{80 * time.Minute, true},
		{119 * time.Minute, true},
		{120 * time.Minute, false}, // le lendemain matin
	}
	for _, c := range cas {
		if got := IsNight(debut.Add(c.apres)); got != c.nuit {
			t.Errorf("après %v : nuit=%v, attendu %v", c.apres, got, c.nuit)
		}
	}
	if p := DayPhase(debut.Add(30 * time.Minute)); p < 0.249 || p > 0.251 {
		t.Errorf("30 min après l'aube, la journée doit être avancée d'un quart (%.3f)", p)
	}
}

func TestMeteo(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	w := NewWeather(now, 42)
	if w.Kind != Clear {
		t.Fatal("la météo commence par un temps clair")
	}
	if w.Update(now.Add(time.Minute)) {
		t.Fatal("la météo ne doit pas changer avant son heure")
	}
	vus := map[string]int{}
	for i := 0; i < 2000; i++ {
		avant := w.Kind
		now = w.Until
		w.Update(now)
		if avant == Storm && w.Kind != Rain {
			t.Fatalf("un orage doit finir en pluie, pas en %q", w.Kind)
		}
		d := w.Until.Sub(now)
		if d < 4*time.Minute || d > 35*time.Minute {
			t.Fatalf("durée de météo étrange : %v", d)
		}
		vus[w.Kind]++
	}
	for _, k := range []string{Clear, Rain, Fog, Storm} {
		if vus[k] == 0 {
			t.Errorf("le temps %q n'arrive jamais", k)
		}
	}
	if vus[Clear] < vus[Storm]*2 {
		t.Errorf("le beau temps doit dominer : %v", vus)
	}
}
