package world

import (
	"math"
	"testing"
)

// L'Acte VI : groupes, PvP, duels de groupe, Cercle des Braves, gloire, guildes, outils.

// champ : une case libre hors du village, loin des monstres.
func champ(t *testing.T, a *Arena) (float64, float64) {
	t.Helper()
	for y := 20; y < a.M.IndoorsFrom-10; y++ {
		for x := 20; x < a.M.Width-20; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			if math.Hypot(fx-a.Village[0], fy-a.Village[1]) < VillageRadius+8 {
				continue
			}
			ok := true
			for dx := -3; dx <= 3 && ok; dx++ {
				ok = a.M.Free(fx+float64(dx), fy) && a.M.Free(fx+float64(dx), fy+1)
			}
			for _, m := range a.Monsters {
				if math.Hypot(m.Body.X-fx, m.Body.Y-fy) < 30 {
					ok = false
				}
			}
			for _, c := range a.Camps {
				if math.Hypot(c.At[0]-fx, c.At[1]-fy) < c.Radius+30 {
					ok = false
				}
			}
			if ok {
				return fx, fy
			}
		}
	}
	t.Fatal("aucune case tranquille hors du village")
	return 0, 0
}

func heroDe(race string, lv int) HeroInit {
	hi := paladin(lv)
	hi.Race = race
	return hi
}

func groupe(a *Arena, chef int, autres ...int) *Group {
	for _, x := range autres {
		a.GroupInvite(chef, x)
		a.GroupAnswer(x, true)
	}
	return a.GroupOf(chef)
}

func TestUnGroupeSeFormeEtSeTransmet(t *testing.T) {
	a, h := arene(t)
	b := a.AddHero(2, "Brann", &Body{X: 112.5, Y: 98.5}, paladin(1))
	c := a.AddHero(3, "Cybil", &Body{X: 113.5, Y: 98.5}, paladin(1))
	g := groupe(a, h.SID, b.SID, c.SID)
	if g == nil || len(g.Members) != 3 || g.Leader != h.SID || !a.SameGroup(b.SID, c.SID) {
		t.Fatalf("groupe mal formé : %+v", g)
	}
	a.GroupInvite(b.SID, h.SID) // seul le chef invite
	if a.groupInv[h.SID].from == b.SID {
		t.Fatal("un membre a pu inviter")
	}
	a.GroupLeave(h.SID)
	if g.Leader == h.SID || a.GroupOf(h.SID) != nil || len(g.Members) != 2 {
		t.Fatalf("le chef parti, le groupe doit changer de chef : %+v", g)
	}
	a.GroupLeave(b.SID)
	if a.GroupOf(c.SID) != nil || len(a.Groups) != 0 {
		t.Fatal("un groupe d'un seul héros doit se dissoudre")
	}
}

func TestLeGroupeEstLimite(t *testing.T) {
	a, h := arene(t)
	var sids []int
	for i := 2; i <= GroupMax+1; i++ {
		sids = append(sids, a.AddHero(i, "H"+itoa(int64(i)), &Body{X: 111.5, Y: 99.5}, paladin(1)).SID)
	}
	g := groupe(a, h.SID, sids...)
	if len(g.Members) != GroupMax {
		t.Fatalf("le groupe a %d membres (max %d)", len(g.Members), GroupMax)
	}
}

func TestLExperienceSePartageDansLeGroupe(t *testing.T) {
	a, h := arene(t)
	b := a.AddHero(2, "Brann", &Body{X: 112.5, Y: 98.5}, paladin(1))
	m := premier(a, "sanglier")
	// Seul : toute l'expérience.
	m.Mon.threat = map[int]float64{h.SID: 1}
	xp0 := h.Hero.XP
	a.reward(m)
	seul := h.Hero.XP - xp0
	// À deux, B près de lui : chacun (1 + 10 %) / 2.
	groupe(a, h.SID, b.SID)
	b.Body.Teleport(m.Body.X+1, m.Body.Y)
	h.Body.Teleport(m.Body.X-1, m.Body.Y)
	xa, xb := h.Hero.XP, b.Hero.XP
	m.Mon.threat = map[int]float64{h.SID: 1}
	a.reward(m)
	ga, gb := h.Hero.XP-xa, b.Hero.XP-xb
	want := int(math.Round(float64(seul) * 1.1 / 2))
	if gb == 0 || abs(ga-want) > 1 || abs(gb-want) > 1 {
		t.Fatalf("seul %d, à deux %d et %d (attendu ≈ %d chacun)", seul, ga, gb, want)
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestLeDrapeauDeGuerre(t *testing.T) {
	a, h := arene(t)
	b := a.AddHero(2, "Ourk", &Body{X: 112.5, Y: 98.5}, heroDe("orque", 1))
	c := a.AddHero(3, "Nilda", &Body{X: 112.5, Y: 99.5}, heroDe("nain", 1))
	if a.foe(h, b) {
		t.Fatal("sans drapeau, on ne frappe pas un héros")
	}
	a.ToggleFlag(h.SID)
	a.ToggleFlag(b.SID)
	a.ToggleFlag(c.SID)
	if a.foe(h, b) {
		t.Fatal("on ne se bat pas au village")
	}
	x, y := champ(t, a)
	h.Body.Teleport(x, y)
	b.Body.Teleport(x+1, y)
	c.Body.Teleport(x-1, y)
	if !a.foe(h, b) || !a.foe(b, h) {
		t.Fatal("deux drapeaux levés de factions ennemies, hors du village : la guerre")
	}
	if a.foe(h, c) {
		t.Fatal("on ne frappe pas sa propre faction")
	}
	groupe(a, h.SID, b.SID)
	if a.foe(h, b) {
		t.Fatal("on ne frappe pas son groupe, même d'une autre faction")
	}
	a.GroupLeave(b.SID)
	// Un coup : le drapeau reste levé un moment.
	a.pvpHit(h, b)
	a.ToggleFlag(h.SID)
	if !h.Hero.Flag {
		t.Fatal("on ne baisse pas son drapeau juste après un combat")
	}
	combat(a, FlagLowerWait+1)
	a.ToggleFlag(h.SID)
	if h.Hero.Flag {
		t.Fatal("le drapeau devrait se baisser après le délai")
	}
}

func TestLesControlesSAffaiblissentEntreHeros(t *testing.T) {
	a, h := arene(t)
	b := a.AddHero(2, "Ourk", &Body{X: 112.5, Y: 98.5}, heroDe("orque", 1))
	var durs []float64
	for i := 0; i < 4; i++ {
		b.fx[Etourdi] = effect{}
		a.applyCC(h, b, Etourdi, 4, 0)
		durs = append(durs, b.fx[Etourdi].left)
	}
	want := []float64{4, 2, 1, 0}
	for i := range want {
		if math.Abs(durs[i]-want[i]) > 1e-9 {
			t.Fatalf("durées %v, attendu %v", durs, want)
		}
	}
	a.tickPvP(b, DRReset+0.1)
	b.fx[Etourdi] = effect{}
	a.applyCC(h, b, Etourdi, 4, 0)
	if b.fx[Etourdi].left != 4 {
		t.Fatal("après un moment, le contrôle retrouve toute sa durée")
	}
	// Sur un monstre, rien ne change.
	m := premier(a, "sanglier")
	for i := 0; i < 3; i++ {
		m.fx[Etourdi] = effect{}
		a.applyCC(h, m, Etourdi, 4, 0)
		if m.fx[Etourdi].left != 4 {
			t.Fatal("les contrôles sur les monstres ne s'affaiblissent pas")
		}
	}
}

func TestLesSoinsBaissentEnPvP(t *testing.T) {
	a, h := arene(t)
	b := a.AddHero(2, "Ourk", &Body{X: 112.5, Y: 98.5}, heroDe("orque", 1))
	h.HP = 1
	a.heal(h, h, 100, 0)
	paix := h.HP - 1
	h.HP = 1
	a.pvpHit(b, h)
	a.heal(h, h, 100, 0)
	if guerre := h.HP - 1; guerre != int(math.Round(float64(paix)*PvPHealMul)) {
		t.Fatalf("soin en paix %d, en PvP %d (attendu −30 %%)", paix, guerre)
	}
}

func TestTomberSousUnHerosNeCouteRien(t *testing.T) {
	a, h := arene(t)
	hi := heroDe("orque", 1)
	hi.Hardcore = true
	b := a.AddHero(2, "Ourk", &Body{X: 112.5, Y: 98.5}, hi)
	a.kill(b, h)
	if !b.Hero.PvPDeath() {
		t.Fatal("la mort devrait compter comme PvP")
	}
	for _, e := range a.Events {
		if e.Kind == EvDeath && e.Flags&FlagHardcore != 0 {
			t.Fatal("un héros hardcore tombé sous un héros ne doit pas être effacé")
		}
	}
	combat(a, RespawnHero+0.5)
	if b.Dead {
		t.Fatal("le héros hardcore doit se relever")
	}
	if b.Has(Faiblesse) {
		t.Fatal("pas de Faiblesse après une mort en PvP")
	}
}

func TestUnDuelDeGroupe(t *testing.T) {
	a, h := arene(t)
	h2 := a.AddHero(2, "Hana", &Body{X: 111.5, Y: 99.5}, paladin(1))
	b := a.AddHero(3, "Brann", &Body{X: 113.5, Y: 98.5}, paladin(1))
	b2 := a.AddHero(4, "Bea", &Body{X: 113.5, Y: 99.5}, paladin(1))
	groupe(a, h.SID, h2.SID)
	groupe(a, b.SID, b2.SID)
	a.Challenge(h2.SID, b.SID) // pas le chef
	if a.invites[b.SID].from == h2.SID {
		t.Fatal("seul le chef défie au nom de son groupe")
	}
	a.Challenge(h.SID, b.SID)
	a.AnswerDuel(b.SID, true)
	d := a.DuelOf(h2.SID)
	if d == nil || !d.Group() || d.Side(h2.SID) != 0 || d.Side(b2.SID) != 1 {
		t.Fatalf("les deux groupes doivent être du duel : %+v", d)
	}
	combat(a, DuelCountdown+0.1)
	if !a.foe(h2, b2) || a.foe(h, h2) {
		t.Fatal("on frappe le camp d'en face, pas le sien")
	}
	b2.HP = 0
	a.duelBlow(h2, b2)
	if !d.Out[b2.SID] || a.DuelOf(h.SID) == nil {
		t.Fatal("Bea est hors du duel, le duel continue")
	}
	a.Events = nil
	b.HP = 0
	a.duelBlow(h, b)
	if a.DuelOf(h.SID) != nil {
		t.Fatal("le dernier du camp tombé, le duel est fini")
	}
	fin := duelEvents(a, "fin")
	if len(fin) != 1 || fin[0].Src != h.SID {
		t.Fatalf("le chef vainqueur doit être annoncé : %+v", fin)
	}
}

func cercleArene(t *testing.T) (*Arena, *Fighter, *Fighter) {
	t.Helper()
	a, h := arene(t)
	d, err := LoadCercle(a.M)
	if err != nil || d == nil {
		t.Fatalf("cercle : %v", err)
	}
	a.Cercle.Data = d
	h.Body.Teleport(d.Anneau[0], d.Anneau[1])
	b := a.AddHero(2, "Brann", &Body{X: d.Anneau[0] + 1, Y: d.Anneau[1]}, paladin(1))
	h.Hero.Gloire, b.Hero.Gloire = NewGloire(), NewGloire()
	return a, h, b
}

func TestLeCercleDesBraves(t *testing.T) {
	a, h, b := cercleArene(t)
	c := a.Cercle
	x0, y0 := h.Body.X, h.Body.Y
	c.Join(h.SID, 1)
	if c.QueueSize(1) != 1 {
		t.Fatal("Elowyn devrait attendre dans la file")
	}
	c.Join(b.SID, 1)
	combat(a, 1.5)
	bt := c.BoutOf(h.SID)
	if bt == nil || bt != c.BoutOf(b.SID) || c.QueueSize(1) != 0 {
		t.Fatal("deux héros dans la file 1 c. 1 : un combat")
	}
	if a.InVillage(h) == false || h.Body.Y < float64(a.M.IndoorsFrom) {
		t.Fatal("les combattants doivent être dans une salle du Cercle")
	}
	if a.foe(h, b) {
		t.Fatal("pas de coup avant le signal")
	}
	combat(a, CercleCountdown+0.1)
	if !a.foe(h, b) {
		t.Fatal("après le signal, on se bat")
	}
	a.Events = nil
	b.HP = 0
	a.duelBlow(h, b)
	if c.BoutOf(h.SID) != nil {
		t.Fatal("le combat doit être fini")
	}
	gh, gb := h.Hero.Gloire, b.Hero.Gloire
	if gh.Cote["1"] != CoteStart+CoteK/2 || gb.Cote["1"] != CoteStart-CoteK/2 {
		t.Fatalf("cotes %d et %d", gh.Cote["1"], gb.Cote["1"])
	}
	if gh.Insignes != InsWin || gb.Insignes != InsLose {
		t.Fatalf("insignes %d et %d", gh.Insignes, gb.Insignes)
	}
	if h.Body.X != x0 || h.Body.Y != y0 || b.HP != b.MaxHP {
		t.Fatal("les combattants reviennent à leur place, remis sur pied")
	}
	var fin []Event
	for _, e := range a.Events {
		if e.Kind == EvCercle && e.Text == "fin" {
			fin = append(fin, e)
		}
	}
	if len(fin) != 2 || fin[0].Value != 1 {
		t.Fatalf("fin du combat : %+v", fin)
	}
}

func TestLeCercleEnEquipes(t *testing.T) {
	a, h, b := cercleArene(t)
	d := a.Cercle.Data
	h2 := a.AddHero(3, "Hana", &Body{X: d.Anneau[0], Y: d.Anneau[1] + 1}, paladin(1))
	b2 := a.AddHero(4, "Bea", &Body{X: d.Anneau[0] + 1, Y: d.Anneau[1] + 1}, paladin(1))
	groupe(a, h.SID, h2.SID)
	a.Cercle.Join(h2.SID, 2) // pas le chef
	if a.Cercle.QueueSize(2) != 0 {
		t.Fatal("seul le chef inscrit le groupe")
	}
	a.Cercle.Join(h.SID, 2)
	a.Cercle.Join(b.SID, 2)
	a.Cercle.Join(b2.SID, 2)
	combat(a, 1.5)
	bt := a.Cercle.BoutOf(h.SID)
	if bt == nil || bt.Side(h.SID) != bt.Side(h2.SID) || bt.Side(b.SID) != bt.Side(b2.SID) || bt.Side(h.SID) == bt.Side(b.SID) {
		t.Fatalf("équipes mal faites : %+v", bt)
	}
}

func TestLesHautsFaits(t *testing.T) {
	a, h := arene(t)
	h.Hero.Gloire = NewGloire()
	a.Events = nil
	a.Count(h, "tue", 1)
	got := false
	for _, e := range a.Events {
		got = got || (e.Kind == EvGloire && e.Text == "premier_sang")
	}
	if !got || h.Hero.Gloire.HF["premier_sang"] == 0 {
		t.Fatal("le premier monstre vaincu donne « Premier sang »")
	}
	a.Count(h, "tue", 99)
	if !h.Hero.Gloire.HasTitre("chasseur") {
		t.Fatal("100 monstres : le titre de Chasseur du val")
	}
	a.SetTitre(h.SID, "chasseur")
	if h.Hero.Gloire.Titre != "chasseur" {
		t.Fatal("on peut porter un titre gagné")
	}
	a.SetTitre(h.SID, "fleau")
	if h.Hero.Gloire.Titre != "chasseur" {
		t.Fatal("on ne porte pas un titre qu'on n'a pas")
	}
	if h.Hero.Gloire.Points() < 15 {
		t.Fatalf("points : %d", h.Hero.Gloire.Points())
	}
	ids := map[string]bool{}
	for _, x := range HautsFaits {
		if ids[x.ID] {
			t.Fatalf("haut fait en double : %s", x.ID)
		}
		ids[x.ID] = true
	}
	if len(HautsFaits) < 50 {
		t.Fatalf("seulement %d hauts faits", len(HautsFaits))
	}
}

func TestLaGuilde(t *testing.T) {
	a, h := areneSac(t, "paladin", 5)
	g := NewGuilde("Les Lanternes", Blason{})
	if !g.Can(0, DroitCoffre) || g.Can(4, DroitCoffre) || g.Can(4, DroitInviter) {
		t.Fatal("droits par défaut : le chef peut tout, la recrue rien")
	}
	// Déposer, puis reprendre selon le rang.
	i := -1
	for k, it := range h.Hero.Inv.Sac {
		if it != nil && !it.Lie {
			i = k
			break
		}
	}
	if i < 0 {
		t.Skip("pas d'objet à déposer")
	}
	if err := a.GuildeDepot(h.SID, i, 0, g); err != nil || g.Onglets[0][0] == nil {
		t.Fatalf("dépôt : %v", err)
	}
	if err := a.GuildeDepot(h.SID, 0, 2, g); err != ErrGuildeOnglet {
		t.Fatal("le troisième onglet attend le niveau 5")
	}
	if err := a.GuildeRetrait(h.SID, 0, 0, 4, g); err != ErrGuildeDroit {
		t.Fatal("une recrue ne vide pas le coffre")
	}
	if err := a.GuildeRetrait(h.SID, 0, 0, 1, g); err != nil || g.Onglets[0][0] != nil {
		t.Fatalf("un officier reprend l'objet : %v", err)
	}
	h.Hero.Gold = 500
	_ = a.GuildeArgent(h.SID, 300, 4, g)
	if g.Argent != 300 || h.Hero.Gold != 200 {
		t.Fatal("tout le monde peut donner de l'argent")
	}
	if len(g.Journal) != 3 {
		t.Fatalf("le journal doit tout noter : %d lignes", len(g.Journal))
	}
	// Les niveaux.
	if up := g.AddXP(GuildeXP(1)); up != 2 || g.Niveau != 2 {
		t.Fatalf("niveau %d", g.Niveau)
	}
	if _, err := ValidateGuildeNom("x"); err == nil {
		t.Fatal("un nom trop court")
	}
	if _, err := ValidateGuildeNom("Les Lanternes"); err != nil {
		t.Fatal("noms de guilde mal vérifiés")
	}
}

func TestLesOutilsDeRecolte(t *testing.T) {
	a, _ := arene(t)
	gatherTime := a.gatherTime
	inv := NewInventaire()
	if gatherTime(inv, "minage") != GatherTime {
		t.Fatal("sans outil, le temps normal")
	}
	inv.Add(MakeStack(a.rng, "pioche", 1))
	if gatherTime(inv, "minage") >= GatherTime || gatherTime(inv, "herboristerie") != GatherTime {
		t.Fatal("la pioche aide le minage, et seulement lui")
	}
}

func TestLIntendanceDeBravoure(t *testing.T) {
	a, h := areneSac(t, "paladin", 12)
	h.Hero.Gloire = NewGloire()
	if err := a.BuyBravoure(h.SID, "titre_brave"); err != ErrInsignes {
		t.Fatal("sans Insignes, rien")
	}
	h.Hero.Gloire.Insignes = 1000
	if err := a.BuyBravoure(h.SID, "titre_brave"); err != nil || !h.Hero.Gloire.HasTitre("brave") || h.Hero.Gloire.Insignes != 850 {
		t.Fatalf("titre : %v", err)
	}
	if err := a.BuyBravoure(h.SID, "titre_brave"); err != ErrDejaTitre {
		t.Fatal("un titre ne s'achète qu'une fois")
	}
	n := 0
	for _, it := range h.Hero.Inv.Sac {
		if it != nil {
			n++
		}
	}
	if err := a.BuyBravoure(h.SID, "coffret_arme"); err != nil {
		t.Fatalf("coffret : %v", err)
	}
	var got *Item
	k := 0
	for _, it := range h.Hero.Inv.Sac {
		if it != nil {
			k++
			if it.B().Slot == ArmeS && it.Lie && it.R == Rare && it.Lv == 12 {
				got = it
			}
		}
	}
	if k != n+1 || got == nil || !UsableBy(got.B(), "paladin") {
		t.Fatal("le coffret doit donner une arme rare, liée, de son niveau, pour sa classe")
	}
}
