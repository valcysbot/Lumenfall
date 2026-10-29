package world

import (
	"math"
	"testing"
)

func clairiere(t *testing.T) *Map {
	t.Helper()
	m, err := LoadMap("clairiere")
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestCarteValide(t *testing.T) {
	m := clairiere(t)
	if m.Width != 40 || m.Height != 40 {
		t.Fatalf("taille %dx%d", m.Width, m.Height)
	}
	if !m.Free(m.Spawn[0], m.Spawn[1]) {
		t.Fatal("apparition bloquée")
	}
	if !m.BlockedCell(-1, 5) || !m.BlockedCell(0, 0) {
		t.Fatal("les bords doivent bloquer")
	}
}

func TestOnNeTraversePasLesMurs(t *testing.T) {
	m := clairiere(t)
	b := Body{X: m.Spawn[0], Y: m.Spawn[1]}
	b.SetInput(0, -1) // tout droit vers le nord, jusqu'à la lisière
	for i := 0; i < 400; i++ {
		b.Step(m, 0.1)
		if !m.Free(b.X, b.Y) {
			t.Fatalf("héros dans un obstacle en (%.2f, %.2f)", b.X, b.Y)
		}
	}
	if b.Y < 1 {
		t.Fatalf("le héros est sorti de la carte (y=%.2f)", b.Y)
	}
}

func TestVitesseBornee(t *testing.T) {
	m := clairiere(t)
	b := Body{X: m.Spawn[0], Y: m.Spawn[1]}
	b.SetInput(1000, 0) // un client tricheur demande une direction énorme
	b.Step(m, 0.1)
	if d := math.Hypot(b.X-m.Spawn[0], b.Y-m.Spawn[1]); d > WalkSpeed*0.1+1e-9 {
		t.Fatalf("déplacement de %.3f cases en 0,1 s", d)
	}
	b.SetInput(math.NaN(), 1)
	if b.Moving() {
		t.Fatal("une direction NaN doit arrêter le héros")
	}
}

func TestTrajetAutourDeLEtang(t *testing.T) {
	m := clairiere(t)
	// De l'ouest de l'étang à l'est de l'étang.
	p := m.FindPath(21.5, 12.5, 33.5, 12.5, 5000)
	if p == nil {
		t.Fatal("aucun trajet trouvé")
	}
	b := Body{X: 21.5, Y: 12.5}
	b.SetPath(p)
	for i := 0; i < 300 && b.Moving(); i++ {
		b.Step(m, 0.1)
		if !m.Free(b.X, b.Y) {
			t.Fatalf("héros dans un obstacle en (%.2f, %.2f)", b.X, b.Y)
		}
	}
	if math.Hypot(b.X-33.5, b.Y-12.5) > 0.5 {
		t.Fatalf("arrivée en (%.2f, %.2f) au lieu de (33.5, 12.5)", b.X, b.Y)
	}
	if m.FindPath(20.5, 20.5, 27.5, 12.5, 5000) != nil {
		t.Fatal("on ne doit pas pouvoir viser le milieu de l'étang")
	}
}

func TestNoms(t *testing.T) {
	ok := []string{"Elowan", "Aëlys", "Morgan-Le", "D'Arvel", "élise"}
	for _, n := range ok {
		if err := ValidateName(NormalizeName(n)); err != nil {
			t.Errorf("%q refusé : %v", n, err)
		}
	}
	ko := []string{"Al", "Un nom", "Bob42", "-Ari", "Ari--el", "Braise", "XxAdminxX", "abcdefghijklmnopq"}
	for _, n := range ko {
		if ValidateName(NormalizeName(n)) == nil {
			t.Errorf("%q accepté à tort", n)
		}
	}
	if NameKey("Élowan") != NameKey("elowan") || NameKey("D'Arvel") != NameKey("darvel") {
		t.Fatal("les clés d'unicité doivent ignorer accents, casse et signes")
	}
	if NormalizeName("  élise ") != "Élise" {
		t.Fatalf("normalisation : %q", NormalizeName("  élise "))
	}
}

func TestDirections(t *testing.T) {
	cases := map[[2]float64]int{{1, 0}: 0, {1, 1}: 1, {0, 1}: 2, {-1, 0}: 4, {0, -1}: 6, {1, -1}: 7}
	for v, want := range cases {
		if got := DirOf(v[0], v[1]); got != want {
			t.Errorf("DirOf(%v) = %d, attendu %d", v, got, want)
		}
	}
}

func TestLesHerosNeSeTraversentPas(t *testing.T) {
	m := clairiere(t)
	a := &Body{X: 18.5, Y: 20.5}
	b := &Body{X: 22.5, Y: 20.5}
	a.SetInput(1, 0)
	b.SetInput(-1, 0)
	for i := 0; i < 30; i++ {
		crowd := NewCrowd([]*Body{a, b})
		a.StepAmong(m, crowd, 0.1)
		b.StepAmong(m, crowd, 0.1)
		Separate(m, []*Body{a, b})
	}
	if d := math.Hypot(a.X-b.X, a.Y-b.Y); d < 2*HeroRadius-0.25 {
		t.Fatalf("les héros se traversent (distance %.2f)", d)
	}
	if a.X > b.X {
		t.Fatal("les deux héros ont échangé leur place : ils se sont traversés")
	}
}

func TestSeparationHerosSuperposes(t *testing.T) {
	m := clairiere(t)
	bodies := []*Body{{X: 20.5, Y: 20.5}, {X: 20.5, Y: 20.5}, {X: 20.5, Y: 20.5}}
	for i := 0; i < 10; i++ {
		Separate(m, bodies)
	}
	for i := range bodies {
		if !m.Free(bodies[i].X, bodies[i].Y) {
			t.Fatal("un héros a été poussé dans un obstacle")
		}
		for j := i + 1; j < len(bodies); j++ {
			if d := math.Hypot(bodies[i].X-bodies[j].X, bodies[i].Y-bodies[j].Y); d < 2*HeroRadius-0.05 {
				t.Fatalf("héros %d et %d encore superposés (%.2f)", i, j, d)
			}
		}
	}
}

func TestCourse(t *testing.T) {
	m := clairiere(t)
	for _, run := range []bool{false, true} {
		b := Body{X: m.Spawn[0], Y: m.Spawn[1], Run: run}
		b.SetInput(0, 1000)
		b.Step(m, 0.1)
		want := WalkSpeed * 0.1
		if run {
			want = RunSpeed * 0.1
		}
		if d := math.Hypot(b.X-m.Spawn[0], b.Y-m.Spawn[1]); math.Abs(d-want) > 1e-6 {
			t.Fatalf("course=%v : %.3f cases en 0,1 s, attendu %.3f", run, d, want)
		}
	}
}
