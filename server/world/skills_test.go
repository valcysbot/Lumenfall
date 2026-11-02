package world

import "testing"

func TestToutesLesClassesSeDefinissent(t *testing.T) {
	if len(Classes) != 17 {
		t.Fatalf("%d classes", len(Classes))
	}
	ids := map[string]bool{}
	for _, c := range Classes {
		n := 0
		for _, sk := range c.Skills {
			if sk.Level > 9 || sk.Name == "" || ids[c.ID+sk.ID] {
				t.Fatalf("%s : compétence %q mal réglée", c.ID, sk.ID)
			}
			ids[c.ID+sk.ID] = true
			if sk.Level == 1 {
				n++
			}
		}
		if n < 2 {
			t.Fatalf("%s n'a que %d compétence au niveau 1", c.ID, n)
		}
		for _, sp := range c.Specs {
			if sp.Ultime.Level != SpecLevel || sp.Name == "" {
				t.Fatalf("%s : la voie %q n'a pas d'ultime", c.ID, sp.ID)
			}
		}
	}
}

func TestLaVoieSeGagne(t *testing.T) {
	a, h := arene(t)
	h.Level = SpecLevel
	h.Restat()
	if a.ChooseSpec(h.SID, 0) == nil {
		t.Fatal("il faut d'abord vaincre les Effacés")
	}
	e := premier(a, "loup_efface")
	for i := 0; i < VoieQuest; i++ {
		e.Dead = false
		e.HP = 1
		a.addThreat(e, h, 1)
		a.kill(e, h)
	}
	if h.Hero.Quest != VoieQuest {
		t.Fatalf("épreuve : %d", h.Hero.Quest)
	}
	if err := a.ChooseSpec(h.SID, 2); err != nil || h.Hero.Spec != 2 {
		t.Fatalf("voie refusée : %v", err)
	}
	// Changer de voie se paie, au temple.
	h.Body.Teleport(a.Temple[0], a.Temple[1])
	h.Hero.Gold = 0
	if a.ChooseSpec(h.SID, 0) == nil {
		t.Fatal("changer de voie coûte de l'or")
	}
	h.Hero.Gold = SpecChangeGold(h.Level)
	if err := a.ChooseSpec(h.SID, 0); err != nil || h.Hero.Gold != 0 {
		t.Fatalf("changement refusé : %v", err)
	}
}

func TestLesAttributs(t *testing.T) {
	a, h := arene(t)
	h.Level = 3
	h.Restat()
	var at Attrs
	at[Vitalite] = 10
	hp := h.MaxHP
	if err := a.SetAttrs(h.SID, at); err != nil || h.MaxHP != hp+80 {
		t.Fatalf("vitalité : %v, %d", err, h.MaxHP)
	}
	at[Force] = 1
	if a.SetAttrs(h.SID, at) == nil {
		t.Fatal("11 points au niveau 3 : trop")
	}
	h.Level = 12
	at = Attrs{}
	if a.SetAttrs(h.SID, at) == nil {
		t.Fatal("après le niveau 10, on ne reprend plus ses points")
	}
	if r := Recommend(ClassByID("mage"), 20); r.Points() != 20 || r[Intelligence] == 0 {
		t.Fatalf("répartition conseillée : %v", r)
	}
}

func TestLeHardcoreNeRevientPas(t *testing.T) {
	a, h := arene(t)
	h.Hero.Hardcore = true
	s := premier(a, "sanglier")
	a.kill(h, s)
	combat(a, RespawnHero+1)
	if !h.Dead {
		t.Fatal("un héros hardcore ne revient pas")
	}
	ok := false
	for _, e := range a.Events {
		ok = ok || (e.Kind == EvDeath && e.Flags&FlagHardcore != 0)
	}
	if !ok {
		t.Fatal("la mort hardcore doit être annoncée")
	}
}

func TestLaVapeurSurchauffe(t *testing.T) {
	a, _ := arene(t)
	c := ClassByID("ingenieur")
	h := a.AddHero(3, "Bricole", &Body{X: 111.5, Y: 98.5}, heros(c, 1, -1))
	h.Hero.Res = c.Res.Max - 1
	for k := 1; k <= 2; k++ {
		if sk := c.Skill(k, -1); sk.Cost > 0 && sk.Target == TSelf {
			a.UseSkill(h.SID, k)
			break
		}
	}
	if h.Hero.overheat <= 0 {
		t.Fatal("au maximum, la Vapeur bloque l'armure")
	}
}
