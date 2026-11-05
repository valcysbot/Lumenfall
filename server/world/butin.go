package world

import (
	"errors"
	"math"
	"sort"
	"strconv"
)

// Le butin, la récolte, l'atelier et les provisions dans l'arène (Acte V).
// En groupe, le butin est commun : chaque objet qui tombe est tiré aux dés
// (besoin / cupidité / passer) entre les héros qui ont combattu. L'or et les
// matériaux vont tout seuls dans le sac de chacun.

// Réglages du butin.
const (
	GroundLife   = 180.0 // un objet reste 3 minutes au sol
	DiceTime     = 20.0  // secondes pour choisir aux dés
	PickReach    = 3.0   // cases pour ramasser
	SkillWork    = 12    // EvCast : une récolte, une fabrication, un parchemin
	PlanRareP    = 0.015 // plan rare sur un Effacé
	PlanEpiqueP  = 0.2   // plan épique sur le Grand Sanglier
	LumineuxP    = 0.002 // l'Éclat du Lumenfall, sur un Effacé ou le boss
	MabonP       = 0.03  // les Défenses de Mabon, sur le Grand Sanglier
	BossEpiqueP  = 0.2   // la pièce du boss est épique (sinon rare)
	RechargeTime = 1.0
)

// Les votes aux dés.
const (
	VoteAttente = iota
	VoteBesoin
	VoteCupidite
	VotePasser
)

// LootRoll : un tirage aux dés en cours.
type LootRoll struct {
	Votes map[int]int // héros → vote
	Dice  map[int]int // héros → résultat du dé (1 à 100)
	Left  float64
}

// GroundItem : un objet qui brille au sol.
type GroundItem struct {
	ID    int
	It    *Item
	X, Y  float64
	Owner int          // le héros qui peut le ramasser (0 : n'importe lequel des Eligible)
	Elig  map[int]bool // les héros qui ont combattu
	Left  float64
	Roll  *LootRoll
}

// Work : ce que fait un héros qui ne se bat pas.
type Work struct {
	Kind   string // recolte, depece, fabrique, retour
	Node   int
	Mon    int
	Recipe *Recipe
	Choice string
	Base   string // le parchemin utilisé
	Left   float64
	Total  float64
	Repeat int // fabrications qui restent après celle-ci
}

// Ledger : l'or qui entre dans le monde et qui en sort, par raison
// (Conseil des Objets : « un suivi de l'or qui entre et sort pour régler »).
type Ledger struct {
	In, Out map[string]int
}

func NewLedger() *Ledger { return &Ledger{In: map[string]int{}, Out: map[string]int{}} }

// Earn : un héros gagne de l'argent (qui entre dans le monde).
func (a *Arena) Earn(h *Fighter, n int, why string) {
	if n <= 0 {
		return
	}
	h.Hero.Gold += n
	a.Ledger.In[why] += n
	h.Hero.me = true
}

// Spend : un héros dépense de l'argent (qui sort du monde). false : pas assez.
func (a *Arena) Spend(h *Fighter, n int, why string) bool {
	if n < 0 || h.Hero.Gold < n {
		return false
	}
	h.Hero.Gold -= n
	a.Ledger.Out[why] += n
	h.Hero.me = true
	return true
}

// Money écrit une somme : « 2 or 15 argent 40 cuivre ».
func Money(n int) string {
	if n == 0 {
		return "0 cuivre"
	}
	out := ""
	add := func(v int, u string) {
		if v > 0 {
			if out != "" {
				out += " "
			}
			out += strconv.Itoa(v) + " " + u
		}
	}
	add(n/Or, "or")
	add(n%Or/Argent, "argent")
	add(n%Argent, "cuivre")
	return out
}

// --- Le butin ------------------------------------------------------------------

// rollRarity : la rareté d'un objet tombé d'un monstre.
func (a *Arena) rollRarity(k *MonsterKind) int {
	table := []float64{55, 30, 12, 2.7, 0.3} // commun → légendaire
	if k.Elem == Ombre {                     // les Effacés de la Lisière
		table = []float64{40, 35, 19, 5, 1}
	}
	x := a.rng.Float64() * 100
	for r, p := range table {
		if x < p {
			return r
		}
		x -= p
	}
	return Commun
}

// dropChance : la chance qu'un monstre lâche une pièce d'équipement.
func dropChance(k *MonsterKind) float64 {
	switch {
	case k.Boss:
		return 1
	case k.Flee:
		return 1
	case k.Elem == Ombre:
		return 0.16
	}
	return 0.11
}

// randomBase : une pièce d'équipement au hasard, souvent utile à l'un des héros.
func (a *Arena) randomBase(classes []string) string {
	var all, useful []string
	for id, b := range Bases {
		if b.Kind != KEquipement {
			continue
		}
		all = append(all, id)
		for _, cl := range classes {
			if UsableBy(b, cl) {
				useful = append(useful, id)
				break
			}
		}
	}
	sort.Strings(all)
	sort.Strings(useful)
	if len(useful) > 0 && a.rng.Float64() < 0.65 {
		return useful[a.rng.Intn(len(useful))]
	}
	return all[a.rng.Intn(len(all))]
}

// lootFor : le butin d'un monstre (objets qui tombent au sol pour le groupe).
func (a *Arena) dropItems(m *Fighter, heroes []*Fighter) {
	k := m.Mon.Kind
	var classes []string
	for _, h := range heroes {
		classes = append(classes, h.Hero.Class.ID)
	}
	cl := ""
	if len(classes) > 0 {
		cl = classes[a.rng.Intn(len(classes))]
	}
	var items []*Item
	lv := m.Level
	switch {
	case k.Boss:
		// Une pièce rare (parfois épique) pour chacun, tirée aux dés comme le reste.
		for range heroes {
			r := Rare
			if a.rng.Float64() < BossEpiqueP {
				r = Epique
			}
			items = append(items, MakeEquip(a.rng, a.randomBase(classes), lv, r, cl))
		}
		if a.rng.Float64() < MabonP {
			items = append(items, MakeUnique(a.rng, "defenses_de_mabon"))
		}
		if a.rng.Float64() < PlanEpiqueP {
			items = append(items, a.makePlan("epique"))
		}
	case k.Flee:
		items = append(items, MakeEquip(a.rng, a.randomBase(classes), lv, Rare, cl))
	default:
		if a.rng.Float64() < dropChance(k) {
			items = append(items, MakeEquip(a.rng, a.randomBase(classes), lv+a.rng.Intn(3)-1, a.rollRarity(k), cl))
		}
		if k.Elem == Ombre && a.rng.Float64() < PlanRareP {
			items = append(items, a.makePlan("rare"))
		}
	}
	if (k.Elem == Ombre || k.Boss) && a.rng.Float64() < LumineuxP {
		items = append(items, MakeUnique(a.rng, "eclat_du_lumenfall"))
	}
	for _, it := range items {
		if it != nil {
			a.drop(it, m.Body.X, m.Body.Y, heroes)
		}
	}
}

// makePlan : un plan rare ou épique, pour une recette au hasard.
func (a *Arena) makePlan(kind string) *Item {
	var ids []string
	for id, r := range Recipes {
		if r.Plan == kind {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return nil
	}
	it := MakeStack(a.rng, "plan_"+kind, 1)
	it.Rec = ids[a.rng.Intn(len(ids))]
	it.R = Rare
	if kind == "epique" {
		it.R = Epique
	}
	return it
}

// drop pose un objet au sol. Seul : il est à toi. En groupe : les dés.
func (a *Arena) drop(it *Item, x, y float64, heroes []*Fighter) {
	a.groundID++
	g := &GroundItem{ID: a.groundID, It: it, Left: GroundLife, Elig: map[int]bool{}}
	g.X, g.Y = spotNear(a.M, x, y, 1.2, a.rng)
	for _, h := range heroes {
		g.Elig[h.SID] = true
	}
	switch {
	case len(heroes) == 1:
		g.Owner = heroes[0].SID
	case len(heroes) > 1:
		g.Roll = &LootRoll{Votes: map[int]int{}, Dice: map[int]int{}, Left: DiceTime}
		for _, h := range heroes {
			g.Roll.Votes[h.SID] = VoteAttente
		}
	}
	a.Ground[g.ID] = g
	a.emit(Event{Kind: EvGround, Value: g.ID, Text: "pose"})
	if g.Roll != nil {
		a.emit(Event{Kind: EvGround, Value: g.ID, Text: "des"})
	}
}

// Vote : un héros choisit aux dés (besoin, cupidité, passer).
func (a *Arena) Vote(sid, gid, vote int) {
	g := a.Ground[gid]
	h := a.Heroes[sid]
	if g == nil || g.Roll == nil || h == nil {
		return
	}
	if v, ok := g.Roll.Votes[sid]; !ok || v != VoteAttente || vote < VoteBesoin || vote > VotePasser {
		return
	}
	if vote == VoteBesoin && !UsableBy(g.It.B(), h.Hero.Class.ID) {
		vote = VoteCupidite // « besoin » seulement pour ce que ta classe peut porter
	}
	g.Roll.Votes[sid] = vote
	if vote != VotePasser {
		g.Roll.Dice[sid] = 1 + a.rng.Intn(100)
	}
	for _, v := range g.Roll.Votes {
		if v == VoteAttente {
			return
		}
	}
	a.settle(g)
}

// settle : tout le monde a voté (ou le temps est écoulé). Besoin bat cupidité,
// puis le plus haut dé gagne.
func (a *Arena) settle(g *GroundItem) {
	best, bestVote, bestDie := 0, VotePasser, -1
	sids := make([]int, 0, len(g.Roll.Votes))
	for sid := range g.Roll.Votes {
		sids = append(sids, sid)
	}
	sort.Ints(sids)
	for _, sid := range sids {
		v := g.Roll.Votes[sid]
		if v == VoteAttente {
			v = VotePasser
			g.Roll.Votes[sid] = v
		}
		if v == VotePasser {
			continue
		}
		d := g.Roll.Dice[sid]
		if v < bestVote || (v == bestVote && d > bestDie) {
			best, bestVote, bestDie = sid, v, d
		}
	}
	g.Owner = best // 0 : tout le monde a passé, le premier venu le prend
	a.emit(Event{Kind: EvGround, Value: g.ID, Text: "gagne", Src: best})
	g.Roll = nil
}

// Pick : ramasser un objet au sol.
func (a *Arena) Pick(sid, gid int) {
	h := a.Heroes[sid]
	g := a.Ground[gid]
	if h == nil || h.Dead || h.Hero.Inv == nil {
		return
	}
	if g == nil {
		a.refuse(h, "Quelqu'un a été plus rapide !")
		return
	}
	switch {
	case g.Roll != nil:
		a.refuse(h, "Les dés roulent encore !")
		return
	case g.Owner != 0 && g.Owner != sid:
		a.refuse(h, "Cet objet a été gagné par un autre héros.")
		return
	case g.Owner == 0 && len(g.Elig) > 0 && !g.Elig[sid]:
		a.refuse(h, "Ce butin revient à ceux qui ont combattu.")
		return
	case math.Hypot(h.Body.X-g.X, h.Body.Y-g.Y) > PickReach:
		a.refuse(h, "Trop loin pour le ramasser.")
		return
	}
	if rest := h.Hero.Inv.Add(g.It); rest != nil {
		g.It = rest
		a.refuse(h, ErrSacPlein.Error())
		return
	}
	delete(a.Ground, gid)
	a.emit(Event{Kind: EvGround, Value: gid, Text: "prend", Src: sid, Tgt: sid})
	a.emit(Event{Kind: EvItem, Tgt: sid, Text: g.It.Name(), Value: g.It.R, Flags: g.It.Count()})
}

// tickGround : les objets oubliés disparaissent, les dés se terminent.
func (a *Arena) tickGround(dt float64) {
	for id, g := range a.Ground {
		if g.Roll != nil {
			if g.Roll.Left -= dt; g.Roll.Left <= 0 {
				a.settle(g)
			}
		}
		if g.Left -= dt; g.Left <= 0 {
			delete(a.Ground, id)
			a.emit(Event{Kind: EvGround, Value: id, Text: "fin"})
		}
	}
}

// give : un objet va directement dans le sac (or et matériaux) ; s'il ne tient
// pas, il tombe au sol, pour ce seul héros.
func (a *Arena) give(h *Fighter, it *Item) {
	if h.Hero.Inv == nil || it == nil {
		return
	}
	name, r, n := it.Name(), it.R, it.Count()
	if rest := h.Hero.Inv.Add(it); rest != nil {
		a.drop(rest, h.Body.X, h.Body.Y, []*Fighter{h})
		a.refuse(h, "Ton sac est plein : "+rest.Name()+" est tombé à tes pieds.")
		return
	}
	a.emit(Event{Kind: EvItem, Tgt: h.SID, Text: name, Value: r, Flags: n})
}

// --- La récolte et le dépeçage -------------------------------------------------

// SetMetiers place les points de récolte et les ateliers de la carte.
func (a *Arena) SetMetiers(d *MetiersData) error {
	a.Nodes, a.Ateliers = nil, nil
	for i, n := range d.Noeuds {
		k := NodeKinds[n.Sorte]
		if k == nil {
			return errors.New("point de récolte inconnu : " + n.Sorte)
		}
		if !a.M.Free(n.At[0], n.At[1]) {
			return errors.New("point de récolte dans un mur : " + n.Sorte + " " + strconv.Itoa(i))
		}
		a.Nodes = append(a.Nodes, &Node{ID: i + 1, Kind: k, X: n.At[0], Y: n.At[1], Charges: k.Charges})
	}
	for i, at := range d.Ateliers {
		if Metiers[at.Metier] == nil || Metiers[at.Metier].Recolte {
			return errors.New("atelier inconnu : " + at.Metier)
		}
		a.Ateliers = append(a.Ateliers, &Atelier{ID: i + 1, Metier: at.Metier, X: at.At[0], Y: at.At[1]})
	}
	return nil
}

// Node : un point de récolte par son numéro.
func (a *Arena) Node(id int) *Node {
	if id < 1 || id > len(a.Nodes) {
		return nil
	}
	return a.Nodes[id-1]
}

func (a *Arena) canWork(h *Fighter) bool {
	if h == nil || h.Dead || h.Hero.Inv == nil {
		return false
	}
	if h.Hero.InCombat() {
		a.refuse(h, "Pas en plein combat !")
		return false
	}
	return true
}

func (a *Arena) startWork(h *Fighter, w *Work, label string) {
	h.Body.Stop()
	h.Hero.Work = w
	w.Total = w.Left
	a.emit(Event{Kind: EvCast, Src: h.SID, Skill: SkillWork, Value: int(w.Left * 1000), Text: label})
}

// stopWork interrompt ce que fait le héros (il a bougé, ou on l'a frappé).
func (a *Arena) stopWork(h *Fighter, why string) {
	if h.Hero == nil || h.Hero.Work == nil {
		return
	}
	h.Hero.Work = nil
	a.emit(Event{Kind: EvCast, Src: h.SID, Skill: 0})
	if why != "" {
		a.refuse(h, why)
	}
}

// Gather : récolter un point (minerai, plante, arbre).
func (a *Arena) Gather(sid, id int) {
	h := a.Heroes[sid]
	if !a.canWork(h) {
		return
	}
	n := a.Node(id)
	inv := h.Hero.Inv
	switch {
	case n == nil || !n.Up():
		a.refuse(h, "Il n'y a plus rien à récolter ici.")
	case math.Hypot(h.Body.X-n.X, h.Body.Y-n.Y) > GatherReach:
		a.refuse(h, "Approche-toi d'abord.")
	case inv.Metiers[n.Kind.Metier].Lv < n.Kind.Req:
		a.refuse(h, "Il faut le niveau "+strconv.Itoa(n.Kind.Req)+" en "+Metiers[n.Kind.Metier].Name+".")
	case inv.Room(n.Kind.Gives) == 0:
		a.refuse(h, ErrSacPlein.Error())
	default:
		h.Body.Face(n.X, n.Y)
		a.startWork(h, &Work{Kind: "recolte", Node: id, Left: GatherTime}, n.Kind.Name)
	}
}

// Skin : dépecer le corps d'une bête.
func (a *Arena) Skin(sid, msid int) {
	h := a.Heroes[sid]
	if !a.canWork(h) {
		return
	}
	m := a.Get(msid)
	if m == nil || m.Mon == nil || !m.Dead || m.Mon.gone || m.Mon.skinned {
		a.refuse(h, "Il n'y a rien à dépecer.")
		return
	}
	sk, ok := Skins[m.Mon.Kind.ID]
	switch {
	case !ok:
		a.refuse(h, "Ce corps ne se dépèce pas.")
	case dist(h, m) > GatherReach+m.Mon.Kind.Radius:
		a.refuse(h, "Approche-toi d'abord.")
	case h.Hero.Inv.Metiers["depecage"].Lv < sk.Req:
		a.refuse(h, "Il faut le niveau "+strconv.Itoa(sk.Req)+" en Dépeçage.")
	default:
		h.Body.Face(m.Body.X, m.Body.Y)
		a.startWork(h, &Work{Kind: "depece", Mon: msid, Left: GatherTime}, "Dépeçage")
	}
}

// Craft : fabriquer une recette à l'atelier (repeat fois de suite).
func (a *Arena) Craft(sid int, recipe, choice string, repeat int) {
	h := a.Heroes[sid]
	if !a.canWork(h) {
		return
	}
	r := Recipes[recipe]
	if r == nil {
		return
	}
	if err := h.Hero.Inv.CanCraft(r); err != nil {
		a.refuse(h, err.Error())
		return
	}
	if a.atelierNear(h, r.Metier) == nil {
		a.refuse(h, "Il faut être à l'atelier : "+Metiers[r.Metier].Name+".")
		return
	}
	a.startWork(h, &Work{Kind: "fabrique", Recipe: r, Choice: choice, Left: CraftTime, Repeat: max(0, min(repeat, 50)-1)}, r.Name())
}

func (a *Arena) atelierNear(h *Fighter, metier string) *Atelier {
	for _, at := range a.Ateliers {
		if at.Metier == metier && math.Hypot(h.Body.X-at.X, h.Body.Y-at.Y) <= AtelierReach {
			return at
		}
	}
	return nil
}

// tickWork : le travail avance ; bouger l'interrompt.
func (a *Arena) tickWork(h *Fighter, dt float64) {
	hs := h.Hero
	w := hs.Work
	if w == nil {
		return
	}
	if h.Body.Moving() || h.Has(Etourdi) {
		a.stopWork(h, "")
		return
	}
	if w.Left -= dt; w.Left > 0 {
		return
	}
	hs.Work = nil
	inv := hs.Inv
	switch w.Kind {
	case "recolte":
		n := a.Node(w.Node)
		if n == nil || !n.Up() {
			a.refuse(h, "Quelqu'un a été plus rapide !")
			break
		}
		k := n.Kind
		a.give(h, MakeStack(a.rng, k.Gives, k.Min+a.rng.Intn(k.Max-k.Min+1)))
		if k.Bonus != "" && a.rng.Float64() < k.BonusP {
			a.give(h, MakeStack(a.rng, k.Bonus, 1))
		}
		n.Charges--
		if n.Charges <= 0 {
			n.wait = k.Respawn[0] + a.rng.Float64()*(k.Respawn[1]-k.Respawn[0])
			a.emit(Event{Kind: EvNode, Value: n.ID, Flags: 0})
		}
		a.metierUp(h, k.Metier, k.Req, 8)
	case "depece":
		m := a.Get(w.Mon)
		if m == nil || m.Mon == nil || m.Mon.skinned || m.Mon.gone {
			a.refuse(h, "Quelqu'un a été plus rapide !")
			break
		}
		sk := Skins[m.Mon.Kind.ID]
		m.Mon.skinned = true
		a.give(h, MakeStack(a.rng, sk.Hide, sk.N[0]+a.rng.Intn(sk.N[1]-sk.N[0]+1)))
		if sk.Meat != "" {
			a.give(h, MakeStack(a.rng, sk.Meat, 1+a.rng.Intn(2)))
		}
		a.metierUp(h, "depecage", sk.Req, 8)
	case "fabrique":
		r := w.Recipe
		if a.atelierNear(h, r.Metier) == nil {
			a.refuse(h, "Tu t'es éloigné de l'atelier.")
			break
		}
		it, lucky, err := inv.Craft(a.rng, r, hs.Class.ID, h.Name, w.Choice)
		if err != nil {
			a.refuse(h, err.Error())
			break
		}
		flags := it.Count()
		if lucky {
			flags |= 1 << 16 // la chance a souri
		}
		a.emit(Event{Kind: EvItem, Tgt: h.SID, Text: it.Name(), Value: it.R, Flags: flags, Skill: 1})
		xp := 10
		if Bases[r.Out].Kind == KMateriau {
			xp = 5
		}
		a.metierUp(h, r.Metier, r.Req, xp)
		if w.Repeat > 0 && inv.CanCraft(r) == nil {
			a.startWork(h, &Work{Kind: "fabrique", Recipe: r, Choice: w.Choice, Left: CraftTime, Repeat: w.Repeat - 1}, r.Name())
			return
		}
	case "retour":
		if !inv.Take(w.Base, 1) {
			break
		}
		x, y := spotNear(a.M, a.Temple[0], a.Temple[1], 1.5, a.rng)
		h.Body.Teleport(x, y)
		hs.lastX, hs.lastY = x, y
		h.dirty = true
	}
	a.emit(Event{Kind: EvCast, Src: h.SID, Skill: 0, Value: 1})
}

func (a *Arena) metierUp(h *Fighter, id string, req, xp int) {
	if lv := h.Hero.Inv.GainMetier(id, req, xp); lv > 0 {
		a.emit(Event{Kind: EvMetier, Tgt: h.SID, Text: id, Value: lv})
	}
}

// tickNodes : les points épuisés repoussent.
func (a *Arena) tickNodes(dt float64) {
	for _, n := range a.Nodes {
		if n.Charges > 0 {
			continue
		}
		if n.wait -= dt; n.wait <= 0 {
			n.Charges = n.Kind.Charges
			a.emit(Event{Kind: EvNode, Value: n.ID, Flags: 1})
		}
	}
}

// --- Le sac --------------------------------------------------------------------

func (a *Arena) heroInv(sid int) (*Fighter, *Inventaire) {
	h := a.Heroes[sid]
	if h == nil || h.Hero.Inv == nil {
		return nil, nil
	}
	return h, h.Hero.Inv
}

// Equip : porter l'objet d'une case du sac.
func (a *Arena) Equip(sid, i int, want Slot) {
	h, inv := a.heroInv(sid)
	if h == nil {
		return
	}
	if h.Hero.InCombat() && h.Hero.Work == nil {
		// On peut se changer en combat, mais pas l'arme : un peu de bon sens.
		if it := inv.At(i); it != nil && it.B().Slot == ArmeS {
			a.refuse(h, "Pas le temps de changer d'arme en plein combat !")
			return
		}
	}
	if a.inTrade(h, inv.At(i)) {
		return
	}
	if err := inv.Equip(i, h.Hero.Class.ID, h.Level, want); err != nil {
		a.refuse(h, err.Error())
		return
	}
	h.Restat()
	h.dirty = true
	a.emit(Event{Kind: EvLook, Src: sid})
}

// Unequip : retirer une pièce.
func (a *Arena) Unequip(sid int, s Slot) {
	h, inv := a.heroInv(sid)
	if h == nil {
		return
	}
	if err := inv.Unequip(s); err != nil {
		a.refuse(h, err.Error())
		return
	}
	h.Restat()
	a.emit(Event{Kind: EvLook, Src: sid})
}

// Destroy : jeter un objet (il disparaît).
func (a *Arena) Destroy(sid, i int) {
	h, inv := a.heroInv(sid)
	if h == nil || inv.At(i) == nil || a.inTrade(h, inv.At(i)) {
		return
	}
	inv.Sac[i] = nil
	inv.Touch()
}

// MoveItem : ranger le sac.
func (a *Arena) MoveItem(sid, from, to int) {
	if _, inv := a.heroInv(sid); inv != nil {
		inv.Move(from, to)
	}
}

// SortBag : trier le sac.
func (a *Arena) SortBag(sid int) {
	if h, inv := a.heroInv(sid); inv != nil && a.Trades[h.SID] == nil {
		inv.Sort()
	}
}

// SetQuick : un objet dans un emplacement rapide (k = 0 ou 1), ou la potion de la touche R (k = 2).
func (a *Arena) SetQuick(sid, k int, base string) {
	h, inv := a.heroInv(sid)
	if h == nil || (base != "" && Bases[base] == nil) {
		return
	}
	switch k {
	case 0, 1:
		inv.Rapides[k] = base
	case 2:
		if b := Bases[base]; base == "" || (b.Use != nil && b.Use.Heal > 0) {
			inv.Potion = base
		}
	}
	inv.Touch()
}

// UseBase : utiliser le premier objet de cette base (emplacement rapide).
func (a *Arena) UseBase(sid int, base string) {
	h, inv := a.heroInv(sid)
	if h == nil {
		return
	}
	for i, it := range inv.Sac {
		if it != nil && it.Base == base {
			a.UseItem(sid, i)
			return
		}
	}
	a.refuse(h, "Tu n'en as plus.")
}

// UseItem : utiliser l'objet d'une case (potion, plat, élixir, parchemin, sac, plan).
func (a *Arena) UseItem(sid, i int) {
	h, inv := a.heroInv(sid)
	if h == nil || h.Dead {
		return
	}
	it := inv.At(i)
	if it == nil || a.inTrade(h, it) {
		return
	}
	b := it.B()
	hs := h.Hero
	switch b.Kind {
	case KEquipement:
		a.Equip(sid, i, -1)
	case KSac:
		if err := inv.Grow(b.Places); err != nil {
			a.refuse(h, err.Error())
			return
		}
		inv.Take(it.Base, 1)
		a.refuse(h, "Ton sac a maintenant "+strconv.Itoa(len(inv.Sac))+" places !")
	case KRecette:
		r := Recipes[it.Rec]
		switch {
		case r == nil:
			a.refuse(h, "Ce plan est illisible.")
		case !inv.Knows(r.Metier):
			a.refuse(h, "Il faut pratiquer le métier : "+Metiers[r.Metier].Name+".")
		case inv.Recettes[r.ID]:
			a.refuse(h, "Tu connais déjà cette recette.")
		default:
			inv.Recettes[r.ID] = true
			inv.Sac[i] = nil
			inv.Touch()
			a.emit(Event{Kind: EvMetier, Tgt: sid, Text: r.Metier, Value: -1})
		}
	case KConso:
		u := b.Use
		switch {
		case u == nil:
		case u.Heal > 0:
			a.drink(h, it.Base)
		case u.Buff != "":
			inv.Buffs[u.Buff] = &Buff{Base: it.Base, Left: u.Minutes * 60}
			inv.Take(it.Base, 1)
			h.Restat()
			a.emit(Event{Kind: EvAnim, Src: sid, Text: "potion"})
		case u.Recall:
			if !a.canWork(h) {
				return
			}
			a.startWork(h, &Work{Kind: "retour", Base: it.Base, Left: u.Cast}, b.Name)
		}
	}
	_ = hs
}

// drink : boire une potion (base vide : celle de Sœur Clémence).
func (a *Arena) drink(h *Fighter, base string) bool {
	hs := h.Hero
	switch {
	case hs.PotionCD > 0:
		return false
	case h.HP >= h.MaxHP:
		a.refuse(h, "Tu es déjà en pleine forme.")
		return false
	}
	heal := PotionHeal
	if base == "" {
		if hs.Potions <= 0 {
			return false
		}
		hs.Potions--
	} else {
		b := Bases[base]
		if b == nil || b.Use == nil || b.Use.Heal <= 0 || !hs.Inv.Take(base, 1) {
			return false
		}
		heal = b.Use.Heal
	}
	hs.PotionCD = PotionCD
	a.heal(h, h, int(math.Round(float64(h.MaxHP)*heal)), SkillPotion)
	a.emit(Event{Kind: EvAnim, Src: h.SID, Text: "potion"})
	hs.me = true
	return true
}

// potions : la touche R. La potion choisie, sinon celle de Clémence, sinon la plus petite du sac.
func (a *Arena) potionR(h *Fighter) {
	hs := h.Hero
	if hs.PotionCD > 0 {
		return
	}
	inv := hs.Inv
	if inv.Potion != "" && inv.Count(inv.Potion) > 0 {
		a.drink(h, inv.Potion)
		return
	}
	if hs.Potions > 0 {
		a.drink(h, "")
		return
	}
	for _, id := range []string{"petite_potion", "potion", "grande_potion"} {
		if inv.Count(id) > 0 {
			a.drink(h, id)
			return
		}
	}
	a.refuse(h, "Plus de potions ! Sœur Clémence en remplit 3 au village, et Aelis en vend.")
}
