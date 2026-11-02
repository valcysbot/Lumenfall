package world

import "testing"

// duellistes : deux héros côte à côte sur la place, assez loin des monstres.
func duellistes(t *testing.T) (*Arena, *Fighter, *Fighter) {
	t.Helper()
	a, h := arene(t)
	b := a.AddHero(2, "Brann", &Body{X: 113.5, Y: 98.5}, paladin(1))
	return a, h, b
}

func duelEvents(a *Arena, text string) []Event {
	var out []Event
	for _, e := range a.Events {
		if e.Kind == EvDuel && e.Text == text {
			out = append(out, e)
		}
	}
	return out
}

func TestOnNeFrappePasUnHerosSansDuel(t *testing.T) {
	a, h, b := duellistes(t)
	face(h, b)
	a.SetTarget(h.SID, b.SID)
	a.UseSkill(h.SID, 1)
	combat(a, 3)
	if b.HP != b.MaxHP {
		t.Fatalf("Brann a perdu %d points de vie sans duel", b.MaxHP-b.HP)
	}
}

func TestUnDuelSeGagneSansMourir(t *testing.T) {
	a, h, b := duellistes(t)
	a.Challenge(h.SID, b.SID)
	if len(duelEvents(a, "defi")) != 1 {
		t.Fatal("Brann n'a pas reçu le défi")
	}
	a.AnswerDuel(b.SID, true)
	d := a.DuelOf(h.SID)
	if d == nil || d != a.DuelOf(b.SID) || d.Started() {
		t.Fatal("le duel doit commencer par un compte à rebours")
	}
	face(h, b)
	a.SetTarget(h.SID, b.SID)
	a.UseSkill(h.SID, 1) // trop tôt : refusé
	if b.HP != b.MaxHP {
		t.Fatal("un coup avant la fin du compte à rebours")
	}
	combat(a, DuelCountdown+0.1)
	if !d.Started() {
		t.Fatal("le compte à rebours ne finit pas")
	}
	b.HP = 3
	a.Events = nil
	a.UseSkill(h.SID, 1)
	if b.Dead || b.HP != 1 {
		t.Fatalf("le perdant doit rester à 1 point de vie (vie %d, mort %v)", b.HP, b.Dead)
	}
	fin := duelEvents(a, "fin")
	if len(fin) != 1 || fin[0].Src != h.SID || fin[0].Tgt != b.SID {
		t.Fatalf("le vainqueur n'est pas annoncé : %+v", fin)
	}
	if a.DuelOf(h.SID) != nil || h.Hero.Target == b.SID {
		t.Fatal("le duel doit être fini, et l'adversaire n'est plus une cible")
	}
	combat(a, 4)
	if b.HP < 1 || b.Dead {
		t.Fatal("après le duel, plus aucun coup ne doit passer")
	}
}

func TestSortirDuCercleCestAbandonner(t *testing.T) {
	a, h, b := duellistes(t)
	a.Challenge(h.SID, b.SID)
	a.AnswerDuel(b.SID, true)
	combat(a, DuelCountdown+0.1)
	a.Events = nil
	b.Body.Teleport(a.Duels[0].X+DuelRadius+2, a.Duels[0].Y)
	combat(a, 0.2)
	fin := duelEvents(a, "fin")
	if len(fin) != 1 || fin[0].Src != h.SID || fin[0].Flags != duelEnds[DuelFled] {
		t.Fatalf("Brann a fui : Elowyn gagne par abandon (%+v)", fin)
	}
}

func TestUnDefiRefuseOuTropLoin(t *testing.T) {
	a, h, b := duellistes(t)
	a.Challenge(h.SID, b.SID)
	a.AnswerDuel(b.SID, false)
	if a.DuelOf(h.SID) != nil {
		t.Fatal("un défi refusé ne lance pas de duel")
	}
	b.Body.Teleport(h.Body.X+DuelReach+3, h.Body.Y)
	a.Challenge(h.SID, b.SID)
	a.AnswerDuel(b.SID, true)
	if a.DuelOf(h.SID) != nil {
		t.Fatal("on ne défie pas un héros trop loin")
	}
	// Un défi oublié expire.
	b.Body.Teleport(h.Body.X+2, h.Body.Y)
	a.Challenge(h.SID, b.SID)
	combat(a, DuelInviteTime+1)
	a.AnswerDuel(b.SID, true)
	if a.DuelOf(h.SID) != nil {
		t.Fatal("un défi expiré ne lance pas de duel")
	}
}

func TestLaBruluresEteintEnFinDeDuel(t *testing.T) {
	a, h, b := duellistes(t)
	a.Challenge(h.SID, b.SID)
	a.AnswerDuel(b.SID, true)
	combat(a, DuelCountdown+0.1)
	b.Apply(Brulure, 10, 50, h.SID) // une brûlure énorme
	b.HP = 2
	combat(a, 3)
	if b.Dead || b.Has(Brulure) {
		t.Fatalf("la brûlure du duel doit s'éteindre avec lui (mort %v)", b.Dead)
	}
}

func TestToutesLesCompetencesEnDuel(t *testing.T) {
	a, h, b := duellistes(t)
	a.Challenge(h.SID, b.SID)
	a.AnswerDuel(b.SID, true)
	combat(a, DuelCountdown+0.1)
	face(h, b)
	a.SetTarget(h.SID, b.SID)
	h.Level, b.Level = 9, 9
	h.Restat()
	b.Restat()
	b.HP = b.MaxHP
	for k := 1; k <= 7; k++ {
		h.Hero.Res = 100
		h.Hero.GCD = 0
		a.UseSkill(h.SID, k)
		combat(a, 1.6)
	}
	if b.HP == b.MaxHP {
		t.Fatal("aucun coup n'a porté")
	}
	if b.Has(Aura) {
		t.Fatal("l'aura ne doit pas renforcer l'adversaire")
	}
}
