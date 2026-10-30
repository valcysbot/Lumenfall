package world

import (
	"math/rand"
	"testing"
)

// arene : le Val de Glennmor avec ses monstres, et un héros au village.
func arene(t *testing.T) (*Arena, *Fighter) {
	t.Helper()
	m, err := LoadMap("glennmor")
	if err != nil {
		t.Fatal(err)
	}
	camps, err := LoadCamps(m)
	if err != nil || len(camps) == 0 {
		t.Fatalf("camps: %v", err)
	}
	a := NewArena(m, camps, rand.New(rand.NewSource(7)))
	a.Temple = [2]float64{111.5, 98.5}
	a.Village = [2]float64{95.5, 100.5}
	h := a.AddHero(1, "Elowyn", &Body{X: 111.5, Y: 98.5}, 1, 0, 0, MaxPotions, nil)
	return a, h
}

// premier : le premier monstre d'une sorte.
func premier(a *Arena, sorte string) *Fighter {
	for _, f := range a.Monsters {
		if f.Mon.Kind.ID == sorte {
			return f
		}
	}
	return nil
}

// face place le héros juste à côté du monstre.
func face(h, m *Fighter) {
	h.Body.Teleport(m.Body.X+1, m.Body.Y)
	h.Body.Face(m.Body.X, m.Body.Y)
}

func TestRoueDesElements(t *testing.T) {
	if ElementMul(Lumiere, Ombre) != 1.5 || ElementMul(Ombre, Lumiere) != 1.5 {
		t.Fatal("lumière et ombre se blessent l'une l'autre")
	}
	if ElementMul(Feu, Air) != 1.5 || ElementMul(Air, Terre) != 1.5 || ElementMul(Terre, Eau) != 1.5 || ElementMul(Eau, Feu) != 1.5 {
		t.Fatal("la roue feu > air > terre > eau > feu")
	}
	if ElementMul(Air, Feu) != 1 || ElementMul(Physique, Ombre) != 1 {
		t.Fatal("pas de bonus hors de la roue")
	}
}

func TestLeBestiaireEstEnPlace(t *testing.T) {
	a, _ := arene(t)
	if len(a.Monsters) != 34 {
		t.Fatalf("%d monstres", len(a.Monsters))
	}
	for _, f := range a.Monsters {
		if !a.M.Free(f.Body.X, f.Body.Y) || f.HP <= 0 || f.Level < 1 {
			t.Fatalf("%s mal installé", f.Name)
		}
	}
}

// combat fait durer le combat n secondes.
func combat(a *Arena, secondes float64) {
	for i := 0; i < int(secondes*10+0.5); i++ {
		for _, h := range a.Heroes {
			h.Body.Step(a.M, 0.1)
		}
		a.Tick(0.1)
	}
}

func TestUnSanglierSeBatEtMeurt(t *testing.T) {
	a, h := arene(t)
	s := premier(a, "sanglier")
	face(h, s)
	a.SetTarget(h.SID, s.SID)
	morts := 0
	for i := 0; i < 600 && !s.Dead; i++ {
		a.Tick(0.1)
		if i%5 == 0 {
			a.UseSkill(h.SID, 1)
		}
		face(h, s)
	}
	if !s.Dead {
		t.Fatalf("le sanglier est toujours debout (%d/%d)", s.HP, s.MaxHP)
	}
	gold, xp := 0, 0
	for _, e := range a.Events {
		switch e.Kind {
		case EvDeath:
			morts++
		case EvLoot:
			gold, xp = e.Value, e.XP
		}
	}
	if morts != 1 || gold < 1 || xp <= 0 || h.Hero.Gold != gold {
		t.Fatalf("morts %d, or %d, xp %d", morts, gold, xp)
	}
	if h.Hero.Ferveur <= 0 {
		t.Fatal("frapper doit donner de la Ferveur")
	}
	// Le corps disparaît, puis le sanglier revient.
	combat(a, RespawnTime+1)
	if s.Dead || s.HP != s.MaxHP {
		t.Fatal("le sanglier aurait dû réapparaître")
	}
}

func TestLaMenaceEtLeCoupDeBouclier(t *testing.T) {
	a, h := arene(t)
	h2 := a.AddHero(2, "Ami", &Body{}, 1, 0, 0, 5, nil)
	s := premier(a, "sanglier")
	face(h2, s)
	a.hit(h2, s, 10, Physique, 0, 0, 1)
	face(h, s)
	a.SetTarget(h.SID, s.SID)
	a.UseSkill(h.SID, 2)
	if !s.Has(Etourdi) {
		t.Fatal("le coup de bouclier étourdit")
	}
	if s.Mon.threat[h.SID] <= s.Mon.threat[h2.SID] {
		t.Fatalf("menace %v", s.Mon.threat)
	}
	combat(a, 2.5)
	if s.Mon.target != h.SID {
		t.Fatal("le sanglier doit se tourner vers le Paladin")
	}
}

func TestLaRouladeEsquiveEtSEssouffle(t *testing.T) {
	a, h := arene(t)
	a.Roll(h.SID, 1, 0)
	a.Roll(h.SID, 0, 0) // refusée : la première roulade n'est pas finie
	combat(a, 0.4)
	a.Roll(h.SID, 1, 0)
	if h.Hero.Endurance >= 1 {
		t.Fatalf("endurance %v après deux roulades", h.Hero.Endurance)
	}
	s := premier(a, "sanglier")
	if a.hit(s, h, 50, Physique, 0, 0, 1) != 0 {
		t.Fatal("en pleine roulade, on esquive")
	}
	combat(a, 0.4)
	a.Roll(h.SID, 1, 0)
	if !h.Body.Dashing() && h.Hero.Endurance < 1 {
		// essoufflé : refus attendu
	} else if h.Hero.Endurance >= 0.5 {
		t.Fatal("une troisième roulade aussitôt ne devrait pas passer")
	}
	combat(a, EnduranceTime*2)
	if h.Hero.Endurance != MaxEndurance {
		t.Fatal("l'endurance remonte")
	}
}

func TestLeBouclierProtegeDeFace(t *testing.T) {
	a, h := arene(t)
	s := premier(a, "loup")
	face(h, s)
	a.Block(h.SID, true)
	a.rng = rand.New(rand.NewSource(1))
	de := a.hit(s, h, 100, Physique, 0, 0, 1)
	h.Body.Face(s.Body.X+2, s.Body.Y+40) // dos tourné
	dos := a.hit(s, h, 100, Physique, 0, 0, 1)
	if de > 40 || dos < 80 {
		t.Fatalf("de face %d, de dos %d", de, dos)
	}
}

func TestMortRetourAuTempleAvecFaiblesse(t *testing.T) {
	a, h := arene(t)
	s := premier(a, "loup")
	a.hit(s, h, 10000, Physique, 0, 0, 1)
	if !h.Dead {
		t.Fatal("le héros devrait être tombé")
	}
	combat(a, RespawnHero+0.2)
	if h.Dead || h.HP != h.MaxHP || !h.Has(Faiblesse) {
		t.Fatal("retour au temple, en pleine forme mais affaibli")
	}
	if dx, dy := h.Body.X-a.Temple[0], h.Body.Y-a.Temple[1]; dx*dx+dy*dy > 4 {
		t.Fatal("pas au temple")
	}
	avant := a.rng
	a.rng = rand.New(rand.NewSource(3))
	n := a.hit(h, s, 100, Physique, 0, 0, 1)
	a.rng = avant
	if n > 85 {
		t.Fatalf("la Faiblesse réduit les dégâts (%d)", n)
	}
}

func TestLesNiveauxMontent(t *testing.T) {
	a, h := arene(t)
	a.GainXP(h, XPForLevel(1)+XPForLevel(2)+5)
	if h.Level != 3 || h.Hero.XP != 5 || h.MaxHP != HeroMaxHP(3) {
		t.Fatalf("niveau %d, xp %d", h.Level, h.Hero.XP)
	}
	if KillXP(1, 9, 1) >= KillXP(1, 1, 1) {
		t.Fatal("un monstre trop faible rapporte moins")
	}
}

func TestLeMonstreRentreChezLui(t *testing.T) {
	a, h := arene(t)
	s := premier(a, "sanglier")
	face(h, s)
	a.hit(h, s, 5, Physique, 0, 0, 1)
	h.Body.Teleport(a.Temple[0], a.Temple[1]) // on s'enfuit au village
	combat(a, 40)
	if s.Mon.evading || len(s.Mon.threat) != 0 || s.HP != s.MaxHP {
		t.Fatal("le sanglier aurait dû rentrer chez lui et guérir")
	}
}

func TestLePoisonDuCorbeau(t *testing.T) {
	a, h := arene(t)
	c := premier(a, "corbeau_efface")
	h.Apply(Poison, 3, 4, c.SID)
	hp := h.HP
	combat(a, 3.2)
	if h.HP != hp-12 || h.Has(Poison) {
		t.Fatalf("poison : %d → %d", hp, h.HP)
	}
}

func TestLesPotions(t *testing.T) {
	a, h := arene(t)
	h.Body.Teleport(178, 60) // loin du village
	h.HP = 20
	a.Potion(h.SID)
	if h.HP <= 20 || h.Hero.Potions != MaxPotions-1 {
		t.Fatal("la potion soigne")
	}
	h.HP = 20
	a.Potion(h.SID)
	if h.HP != 20 {
		t.Fatal("la potion a une recharge")
	}
	h.Body.Teleport(a.Temple[0], a.Temple[1])
	combat(a, CombatTime+0.5)
	if h.Hero.Potions != MaxPotions {
		t.Fatal("Sœur Clémence remplit la besace au village")
	}
}

func TestLeSoinSIncanteEtSInterrompt(t *testing.T) {
	a, h := arene(t)
	h.Hero.Ferveur = 100
	h.HP = 30
	a.UseSkill(h.SID, 3)
	combat(a, 0.5)
	h.Body.SetInput(1, 0)
	combat(a, 0.2)
	if h.Hero.Cast != 0 || h.HP > 40 {
		t.Fatal("bouger interrompt le soin")
	}
	h.Body.SetInput(0, 0)
	combat(a, GCD)
	a.UseSkill(h.SID, 3)
	combat(a, 1.6)
	if h.HP < 30+h.MaxHP/4 {
		t.Fatalf("soin incanté : %d", h.HP)
	}
}

func TestLeGrandSanglier(t *testing.T) {
	a, h := arene(t)
	b := premier(a, "grand_sanglier")
	if b == nil || !b.Mon.Kind.Boss || b.Level != 10 {
		t.Fatal("le Grand Sanglier manque à l'appel")
	}
	h.Level, h.MaxHP, h.HP = 10, HeroMaxHP(10), HeroMaxHP(10)
	// Le héros se poste à 6 cases : le boss doit annoncer sa charge.
	h.Body.Teleport(b.Body.X-6, b.Body.Y)
	if !a.M.Free(h.Body.X, h.Body.Y) {
		t.Fatal("la bauge doit être dégagée")
	}
	a.hit(h, b, 10, Physique, 0, 0, 1)
	b.Mon.chargeCD = 0
	var charge *Telegraph
	for i := 0; i < 80 && charge == nil; i++ {
		a.Tick(0.1)
		for _, tl := range a.Teles {
			if tl.Line {
				charge = tl
			}
		}
	}
	if charge == nil {
		t.Fatal("pas de charge annoncée")
	}
	if !charge.Inside(h.Body.X, h.Body.Y) {
		t.Fatal("la charge vise le héros")
	}
	// On roule hors de la ligne : la charge passe à côté.
	h.Body.Teleport(h.Body.X, h.Body.Y+4)
	hp := h.HP
	combat(a, charge.Left+0.1)
	if h.HP < hp-40 {
		t.Fatal("hors de la zone, la charge ne touche pas")
	}
	// Sous la moitié de sa vie : phase 2, il devient un Effacé.
	b.HP = b.MaxHP/2 - 1
	h.Body.Teleport(b.Body.X+1, b.Body.Y)
	a.hit(h, b, 1, Physique, 0, 0, 1)
	combat(a, 0.3)
	if b.Mon.Phase != 2 || b.Elem != Ombre || b.Bits()&(1<<8) == 0 {
		t.Fatal("phase 2 attendue")
	}
	// Abandonné, il rentre chez lui et redevient lui-même.
	h.Body.Teleport(a.Temple[0], a.Temple[1])
	combat(a, 30)
	if b.Mon.Phase != 1 || b.HP != b.MaxHP || len(a.Teles) != 0 {
		t.Fatalf("remise à zéro : phase %d, %d/%d", b.Mon.Phase, b.HP, b.MaxHP)
	}
}
