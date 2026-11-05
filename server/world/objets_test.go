package world

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

// areneSac : le val, ses points de récolte, et un héros qui a un sac.
func areneSac(t *testing.T, cl string, lv int) (*Arena, *Fighter) {
	t.Helper()
	a, _ := arene(t)
	d, err := LoadMetiers(a.M)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.SetMetiers(d); err != nil {
		t.Fatal(err)
	}
	inv := NewInventaire()
	inv.Starter(a.rng, cl, lv)
	c := ClassByID(cl)
	h := a.AddHero(2, "Brann", &Body{X: 95.5, Y: 104.5}, HeroInit{Level: lv, Potions: 3, Class: cl, Spec: -1, Race: "nain",
		Attrs: Recommend(c, PointsAt(lv)), Inv: inv})
	return a, h
}

func TestBasesEtRecettes(t *testing.T) {
	if n := len(Bases); n < 90 {
		t.Fatalf("seulement %d bases", n)
	}
	for id, r := range Recipes {
		if Bases[r.Out] == nil {
			t.Fatalf("%s fabrique une base inconnue %q", id, r.Out)
		}
		for _, n := range r.Needs {
			if Bases[n.Base] == nil {
				t.Fatalf("%s demande %q, inconnu", id, n.Base)
			}
		}
		if Metiers[r.Metier] == nil || Metiers[r.Metier].Recolte {
			t.Fatalf("%s : métier %q", id, r.Metier)
		}
	}
	// Chaque classe peut se faire fabriquer son arme et toutes ses pièces d'armure.
	made := map[string]bool{}
	for _, r := range Recipes {
		made[r.Out] = true
	}
	for cl := range ClassArmor {
		for _, s := range []Slot{Tete, Epaules, Torse, Mains, Jambes, Pieds} {
			if !made[SlotIDs[s]+"_"+ArmureIDs[ClassArmor[cl]]] {
				t.Fatalf("%s : pas de recette pour %s", cl, SlotIDs[s])
			}
		}
		if !made["arme_"+ClassWeapon[cl]] {
			t.Fatalf("%s : pas de recette pour son arme", cl)
		}
	}
	// Chaque matériau de recette s'obtient : récolte, dépeçage, butin, Aelis ou atelier.
	src := map[string]bool{}
	for _, k := range NodeKinds {
		src[k.Gives], src[k.Bonus] = true, true
	}
	for _, s := range Skins {
		src[s.Hide], src[s.Meat] = true, true
	}
	for _, m := range TrophyMat {
		src[m] = true
	}
	for _, b := range Shop {
		src[b] = true
	}
	for _, r := range Recipes {
		src[r.Out] = true
	}
	src["soie_doree"] = true
	for id, r := range Recipes {
		for _, n := range r.Needs {
			if !src[n.Base] {
				t.Fatalf("%s demande %q, introuvable dans le monde", id, n.Base)
			}
		}
	}
}

func TestEquipementDeDepart(t *testing.T) {
	for cl := range ClassArmor {
		inv := NewInventaire()
		inv.Starter(rand.New(rand.NewSource(1)), cl, 4)
		if inv.Eq[ArmeS] == nil || inv.Eq[Torse] == nil {
			t.Fatalf("%s : pas d'arme ou de torse", cl)
		}
		for _, it := range inv.Eq {
			if it != nil && (!UsableBy(it.B(), cl) || it.R != Commun || it.Lv != 4) {
				t.Fatalf("%s : pièce de départ incorrecte %+v", cl, it)
			}
		}
	}
}

func TestEquiperChangeLesStats(t *testing.T) {
	a, h := areneSac(t, "guerrier", 5)
	hp, arm := h.MaxHP, h.Hero.Stats.Armor
	if arm <= 0 || h.Weapon() != HeroWeapon(5) {
		t.Fatalf("armure %.1f, arme %.1f", arm, h.Weapon())
	}
	it := MakeEquip(a.rng, "torse_plaques", 5, Epique, "guerrier")
	it.Aff = []Affix{{"vie", 100}, {"force", 5}, {"vitalite", 5}}
	inv := h.Hero.Inv
	inv.Add(it)
	a.Equip(h.SID, inv.Find(it.ID), -1)
	if inv.Eq[Torse] != it || !it.Lie {
		t.Fatal("le plastron n'est pas porté (ou pas lié)")
	}
	if h.MaxHP < hp+100+40 || h.Hero.Stats.Armor <= arm {
		t.Fatalf("vie %d → %d, armure %.1f → %.1f", hp, h.MaxHP, arm, h.Hero.Stats.Armor)
	}
	// Un Mage ne porte pas de plaques ; un niveau 5 ne porte pas un niveau 9.
	robe := MakeEquip(a.rng, "torse_tissu", 5, Commun, "mage")
	inv.Add(robe)
	a.Equip(h.SID, inv.Find(robe.ID), -1)
	if inv.Eq[Torse] == robe {
		t.Fatal("un guerrier porte une robe")
	}
	gros := MakeEquip(a.rng, "tete_plaques", 9, Commun, "guerrier")
	inv.Add(gros)
	a.Equip(h.SID, inv.Find(gros.ID), -1)
	if inv.Eq[Tete] == gros {
		t.Fatal("un niveau 5 porte une pièce de niveau 9")
	}
	// À mains nues, la moitié de la force.
	a.Unequip(h.SID, ArmeS)
	if h.Weapon() != HeroWeapon(5)*0.5 {
		t.Fatalf("à mains nues : %.1f", h.Weapon())
	}
}

func TestLArmureProtege(t *testing.T) {
	a, h := areneSac(t, "paladin", 3)
	sans, _ := arene(t)
	nu := sans.Heroes[1]
	loupA := premier(a, "loup")
	loupB := premier(sans, "loup")
	total := func(ar *Arena, m, t *Fighter) int {
		n := 0
		for i := 0; i < 200; i++ {
			t.HP = t.MaxHP
			n += ar.hit(m, t, 20, Physique, 0, 0, 1)
		}
		return n
	}
	d1, d2 := total(a, loupA, h), total(sans, loupB, nu)
	if d1 >= d2 {
		t.Fatalf("en plaques : %d dégâts, sans armure : %d", d1, d2)
	}
}

func TestSacEtPiles(t *testing.T) {
	inv := NewInventaire()
	rng := rand.New(rand.NewSource(3))
	if rest := inv.Add(MakeStack(rng, "lin", 150)); rest != nil {
		t.Fatal("150 lins ne tiennent pas dans un sac vide")
	}
	if inv.Count("lin") != 150 || inv.Free() != SacDepart-2 {
		t.Fatalf("lin %d, places libres %d", inv.Count("lin"), inv.Free())
	}
	inv.Add(MakeStack(rng, "lin", 48)) // complète la pile de 51
	if inv.Free() != SacDepart-2 {
		t.Fatalf("la pile n'a pas été complétée (%d libres)", inv.Free())
	}
	if !inv.Take("lin", 100) || inv.Count("lin") != 98 {
		t.Fatal("retrait des piles")
	}
	if inv.Take("lin", 99) {
		t.Fatal("on ne retire pas plus qu'on n'a")
	}
	for inv.Free() > 0 {
		inv.Add(MakeEquip(rng, "anneau", 1, Commun, ""))
	}
	if rest := inv.Add(MakeEquip(rng, "anneau", 1, Commun, "")); rest == nil {
		t.Fatal("un sac plein accepte encore un objet")
	}
	if err := inv.Grow(6); err != nil || len(inv.Sac) != 36 {
		t.Fatal("le sac ne grandit pas")
	}
}

func sansEchecs(t *testing.T) {
	echecsActifs = false
	t.Cleanup(func() { echecsActifs = true })
}

func TestRecolteEtFabrication(t *testing.T) {
	sansEchecs(t)
	a, h := areneSac(t, "guerrier", 3)
	inv := h.Hero.Inv
	if err := a.LearnMetier(h.SID, "forge"); err != nil {
		t.Fatal(err)
	}
	if a.LearnMetier(h.SID, "couture") != nil || a.LearnMetier(h.SID, "cuir") == nil {
		t.Fatal("deux métiers d'artisanat, pas trois")
	}
	// Récolter un filon de cuivre.
	var filon *Node
	for _, n := range a.Nodes {
		if n.Kind.ID == "filon_cuivre" {
			filon = n
			break
		}
	}
	h.Body.Teleport(filon.X+1, filon.Y)
	h.Hero.combat = CombatTime
	for filon.Up() {
		a.Gather(h.SID, filon.ID)
		if h.Hero.Work == nil {
			t.Fatalf("pas de récolte : %v", a.Events)
		}
		for i := 0; i < 25; i++ {
			a.tickWork(h, 0.1)
		}
	}
	if inv.Count("minerai_cuivre") < 3 || filon.wait <= 0 {
		t.Fatalf("minerai %d, repousse dans %.0f s", inv.Count("minerai_cuivre"), filon.wait)
	}
	if inv.Metiers["minage"].XP == 0 && inv.Metiers["minage"].Lv == 1 {
		t.Fatal("le minage n'a rien appris")
	}
	// Fondre un lingot, loin de la forge : refusé ; à la forge : fait.
	a.Craft(h.SID, "forge:lingot_cuivre", "", 1)
	if h.Hero.Work != nil {
		t.Fatal("on fabrique loin de l'atelier")
	}
	forge := a.Ateliers[0]
	h.Body.Teleport(forge.X+1, forge.Y)
	a.Craft(h.SID, "forge:lingot_cuivre", "", 1)
	for i := 0; i < 30; i++ {
		a.tickWork(h, 0.1)
	}
	if inv.Count("lingot_cuivre") != 1 {
		t.Fatalf("lingots : %d", inv.Count("lingot_cuivre"))
	}
	// Un plastron de plaques, avec le bonus choisi.
	inv.Add(MakeStack(a.rng, "lingot_cuivre", 10))
	inv.Add(MakeStack(a.rng, "flux", 2))
	r := Recipes["forge:torse_plaques:1"]
	it, _, err := inv.Craft(a.rng, r, "guerrier", "Brann", "endurance")
	if err != nil {
		t.Fatal(err)
	}
	if it.R < PeuCommun || it.Fab != "Brann" || it.Aff[0].Stat != "endurance" {
		t.Fatalf("objet fabriqué : %+v", it)
	}
	butin := &Item{Base: it.Base, Lv: it.Lv, R: it.R, Dur: 100}
	if it.ArmorValue() <= butin.ArmorValue() {
		t.Fatal("un objet fabriqué devrait être un peu meilleur que le butin")
	}
}

func TestDesBesoinCupidite(t *testing.T) {
	a, h := areneSac(t, "guerrier", 3)
	inv2 := NewInventaire()
	inv2.Starter(a.rng, "mage", 3)
	mage := a.AddHero(3, "Ilse", &Body{X: 96.5, Y: 104.5}, HeroInit{Level: 3, Class: "mage", Spec: -1, Race: "elfe", Inv: inv2})
	plastron := MakeEquip(a.rng, "torse_plaques", 3, Rare, "guerrier")
	a.drop(plastron, h.Body.X, h.Body.Y, []*Fighter{h, mage})
	var g *GroundItem
	for _, x := range a.Ground {
		g = x
	}
	if g.Roll == nil {
		t.Fatal("en groupe, il faut lancer les dés")
	}
	a.Pick(h.SID, g.ID)
	if a.Ground[g.ID] == nil {
		t.Fatal("ramassé avant la fin des dés")
	}
	a.Vote(mage.SID, g.ID, VoteBesoin) // le mage n'a pas besoin de plaques : cupidité
	if g.Roll.Votes[mage.SID] != VoteCupidite {
		t.Fatal("besoin accepté pour une pièce que la classe ne porte pas")
	}
	a.Vote(h.SID, g.ID, VoteBesoin)
	if g.Roll != nil || g.Owner != h.SID {
		t.Fatalf("besoin doit battre cupidité (gagnant %d)", g.Owner)
	}
	a.Pick(mage.SID, g.ID)
	if a.Ground[g.ID] == nil {
		t.Fatal("le perdant a ramassé l'objet")
	}
	h.Body.Teleport(g.X, g.Y)
	a.Pick(h.SID, g.ID)
	if a.Ground[g.ID] != nil || h.Hero.Inv.Find(plastron.ID) < 0 {
		t.Fatal("le gagnant n'a pas pu ramasser")
	}
}

func TestMonstreTueDonneButin(t *testing.T) {
	a, h := areneSac(t, "guerrier", 3)
	l := premier(a, "loup")
	before := h.Hero.Gold
	for i := 0; i < 400; i++ {
		l.Mon.threat = map[int]float64{h.SID: 1}
		a.kill(l, h)
		l.Dead = false
	}
	if h.Hero.Gold-before < 400*2*Argent {
		t.Fatalf("argent gagné : %d cuivres", h.Hero.Gold-before)
	}
	n := len(a.Ground)
	if n < 15 || n > 90 {
		t.Fatalf("%d objets au sol pour 400 loups", n)
	}
	for _, g := range a.Ground {
		if g.Owner != h.SID || g.Roll != nil {
			t.Fatal("seul, le butin est à toi, sans dés")
		}
	}
	if a.Ledger.In["butin"] <= 0 {
		t.Fatal("l'or du butin n'est pas compté")
	}
}

func TestEchange(t *testing.T) {
	a, h := areneSac(t, "guerrier", 3)
	inv2 := NewInventaire()
	ilse := a.AddHero(3, "Ilse", &Body{X: 96.5, Y: 104.5}, HeroInit{Level: 3, Class: "mage", Spec: -1, Race: "elfe", Inv: inv2})
	h.Hero.Gold, ilse.Hero.Gold = 500, 1000
	anneau := MakeEquip(a.rng, "anneau", 3, Rare, "")
	h.Hero.Inv.Add(anneau)
	lie := h.Hero.Inv.Eq[Torse] // lié : il ne s'échange pas
	a.TradeAsk(h.SID, ilse.SID)
	a.TradeAnswer(ilse.SID, true)
	a.TradeOffer(h.SID, []string{itoa(anneau.ID), itoa(lie.ID)}, 0)
	a.TradeOffer(ilse.SID, nil, 300)
	a.TradeOK(h.SID)
	a.TradeOffer(ilse.SID, nil, 250) // un changement remet les validations à zéro
	a.TradeOK(ilse.SID)
	if a.Trades[h.SID] == nil {
		t.Fatal("l'échange s'est fait sans la nouvelle validation de Brann")
	}
	a.TradeOK(h.SID)
	if a.Trades[h.SID] != nil || inv2.Find(anneau.ID) < 0 || h.Hero.Gold != 750 || ilse.Hero.Gold != 750 {
		t.Fatalf("échange : anneau %d, or %d / %d", inv2.Find(anneau.ID), h.Hero.Gold, ilse.Hero.Gold)
	}
}

func TestCommerceEtReparation(t *testing.T) {
	a, h := areneSac(t, "archer", 4)
	inv := h.Hero.Inv
	h.Hero.Gold = 1000
	a.Buy(h.SID, "petite_potion", 5)
	if inv.Count("petite_potion") != 5 || h.Hero.Gold != 800 {
		t.Fatalf("achat : %d potions, %d cuivres", inv.Count("petite_potion"), h.Hero.Gold)
	}
	a.Buy(h.SID, "grande_potion", 1) // pas à l'étal
	if inv.Count("grande_potion") != 0 {
		t.Fatal("Aelis vend ce qu'elle n'a pas")
	}
	i := inv.Find(inv.Sac[0].ID)
	a.Sell(h.SID, i)
	if len(inv.Rachat) != 1 || h.Hero.Gold <= 800 {
		t.Fatal("vente")
	}
	a.Buyback(h.SID, 0)
	if inv.Count("petite_potion") != 5 || h.Hero.Gold != 800 {
		t.Fatalf("rachat : %d potions, %d cuivres", inv.Count("petite_potion"), h.Hero.Gold)
	}
	// Mourir use l'équipement ; Doran répare.
	a.kill(h, premier(a, "loup"))
	if inv.Eq[Torse].Dur != 100-DurPerDeath || inv.RepairCost() == 0 {
		t.Fatal("pas d'usure à la mort")
	}
	a.Repair(h.SID)
	if inv.Eq[Torse].Dur != 100 || h.Hero.Gold >= 800 {
		t.Fatal("réparation")
	}
	// Une pièce cassée ne donne plus rien.
	inv.Eq[Torse].Dur = 0
	h.Restat()
	if inv.Eq[Torse].ArmorValue() != 0 {
		t.Fatal("une pièce cassée protège encore")
	}
}

func TestPotionsEtRepas(t *testing.T) {
	a, h := areneSac(t, "pretre", 3)
	inv := h.Hero.Inv
	h.HP = 10
	a.Potion(h.SID)
	if h.Hero.Potions != 2 || h.HP <= 10 {
		t.Fatal("la potion de Clémence d'abord")
	}
	inv.Add(MakeStack(a.rng, "ragout", 1))
	hp, force := h.Hero.Stats.Phys, h.Hero.Stats.Mag
	a.UseBase(h.SID, "ragout")
	if inv.Buffs["repas"] == nil || h.Hero.Stats.Mag <= force || h.Hero.Stats.Phys <= hp {
		t.Fatal("le ragoût ne donne rien")
	}
	inv.TickBuffs(31 * 60)
	if inv.Buffs["repas"] != nil {
		t.Fatal("le repas dure plus de 30 minutes")
	}
}

func TestLegendaireChangeUneCompetence(t *testing.T) {
	a, h := areneSac(t, "mage", 5)
	before := h.Hero.Skill(1).Power
	it := MakeEquip(a.rng, "anneau", 5, Legendaire, "mage")
	it.Leg = []LegendEffect{{Skill: 1, Kind: "puissance"}}
	h.Hero.Inv.Add(it)
	a.Equip(h.SID, h.Hero.Inv.Find(it.ID), -1)
	if got := h.Hero.Skill(1).Power; got < before*1.29 {
		t.Fatalf("puissance %.2f → %.2f", before, got)
	}
}

func TestNomsEtArgent(t *testing.T) {
	if Money(12345) != "1 or 23 argent 45 cuivre" || Money(5) != "5 cuivre" {
		t.Fatalf("Money: %q", Money(12345))
	}
	it := &Item{Base: "tete_plaques", Lv: 3, R: Rare, Aff: []Affix{{"force", 2}, {"crit", 0.03}}}
	if n := it.Name(); !strings.HasPrefix(n, "Heaume de plaques") {
		t.Fatalf("nom : %q", n)
	}
}

func itoa(v int64) string { return strconv.FormatInt(v, 10) }

func TestOublierRemetAZero(t *testing.T) {
	a, h := areneSac(t, "guerrier", 3)
	inv := h.Hero.Inv
	a.LearnMetier(h.SID, "forge")
	inv.Metiers["forge"].Lv = 17
	inv.Recettes["forge:torse_plaques:1"] = true
	if err := a.ForgetMetier(h.SID, "forge"); err != nil {
		t.Fatal(err)
	}
	if inv.Knows("forge") || inv.Recettes["forge:torse_plaques:1"] {
		t.Fatal("oublier un métier efface son niveau et ses recettes")
	}
	a.LearnMetier(h.SID, "forge")
	if inv.Metiers["forge"].Lv != 1 {
		t.Fatalf("la forge réapprise repart du niveau 1 : %d", inv.Metiers["forge"].Lv)
	}
}

func TestEchecs(t *testing.T) {
	if EchecChance(1, 1) != EchecBase || EchecChance(30, 1) != EchecMin || EchecChance(5, 1) >= EchecBase {
		t.Fatal("5 % au niveau demandé, moins au-dessus, 1 % au moins")
	}
	if EchecBase > LuckBase {
		t.Fatal("rater ne doit pas être plus fréquent qu'une rareté gagnée")
	}
	inv := NewInventaire()
	rng := rand.New(rand.NewSource(1))
	inv.Add(MakeStack(rng, "minerai_cuivre", 10))
	r := &Recipe{Needs: []Need{{Base: "minerai_cuivre", N: 4}}}
	inv.Gache(r)
	if inv.Count("minerai_cuivre") != 8 {
		t.Fatalf("une fabrication ratée gâche la moitié : reste %d", inv.Count("minerai_cuivre"))
	}
	r.Needs[0].N = 1
	inv.Gache(r)
	if inv.Count("minerai_cuivre") != 7 {
		t.Fatal("au moins un matériau gâché")
	}
	ratees := 0
	for i := 0; i < 10000; i++ {
		if inv.Rate(rng, "minage", 1) {
			ratees++
		}
	}
	if ratees < 350 || ratees > 650 {
		t.Fatalf("environ 5 %% de récoltes ratées au niveau 1 : %d sur 10000", ratees)
	}
}
