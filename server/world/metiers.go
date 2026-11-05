package world

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
)

// Les métiers (Acte V) : 4 de récolte (pour tous) et 6 d'artisanat (2 au
// choix, la cuisine en plus pour tous). Doran, Nessa, Gwenn, Mabon et Aubin
// le bûcheron les enseignent ; on fabrique à l'atelier du métier, au village.

// MetierDef : un métier.
type MetierDef struct {
	ID, Name, Icon, Master string
	Recolte                bool
}

// MetierIDs : l'ordre des métiers (récoltes d'abord).
var MetierIDs = []string{"minage", "herboristerie", "depecage", "bucheronnage", "forge", "couture", "cuir", "alchimie", "joaillerie", "cuisine"}

var Metiers = map[string]*MetierDef{
	"minage":        {ID: "minage", Name: "Minage", Icon: "⛏️", Master: "doran", Recolte: true},
	"herboristerie": {ID: "herboristerie", Name: "Herboristerie", Icon: "🌿", Master: "nessa", Recolte: true},
	"depecage":      {ID: "depecage", Name: "Dépeçage", Icon: "🔪", Master: "gwenn", Recolte: true},
	"bucheronnage":  {ID: "bucheronnage", Name: "Bûcheronnage", Icon: "🪓", Master: "aubin", Recolte: true},
	"forge":         {ID: "forge", Name: "Forge", Icon: "⚒️", Master: "doran"},
	"couture":       {ID: "couture", Name: "Couture", Icon: "🧵", Master: "gwenn"},
	"cuir":          {ID: "cuir", Name: "Travail du cuir", Icon: "🧶", Master: "gwenn"},
	"alchimie":      {ID: "alchimie", Name: "Alchimie", Icon: "⚗️", Master: "nessa"},
	"joaillerie":    {ID: "joaillerie", Name: "Joaillerie", Icon: "💎", Master: "doran"},
	"cuisine":       {ID: "cuisine", Name: "Cuisine", Icon: "🍲", Master: "mabon"},
}

// Réglages des métiers.
const (
	MetierMax      = 50   // niveau maximum d'un métier
	ArtisanatsMax  = 2    // métiers d'artisanat par héros (la cuisine en plus)
	GatherTime     = 2.0  // secondes pour récolter
	CraftTime      = 2.5  // secondes pour fabriquer
	GatherReach    = 2.2  // cases
	AtelierReach   = 3.5  // cases
	SkinTime       = 25.0 // un corps de bête reste 25 s, le temps de le dépecer
	LuckBase       = 0.10 // chance qu'un objet fabriqué gagne une rareté
	LuckPerLevel   = 0.005
	LuckMax        = 0.25
	ForgetLevelMin = 0
	EchecBase      = 0.05 // chance de rater une récolte ou une fabrication (moitié de LuckBase)
	EchecPerLevel  = 0.005
	EchecMin       = 0.01
)

// echecsActifs : les tests qui comptent des objets fabriqués les coupent.
var echecsActifs = true

// EchecChance : la chance de rater, au niveau lv pour une tâche du niveau req.
// 5 % au niveau demandé, un peu moins à chaque niveau au-dessus, jamais moins de 1 %.
func EchecChance(lv, req int) float64 {
	return math.Max(EchecMin, EchecBase-EchecPerLevel*float64(lv-req))
}

// Rate dit si la tâche est ratée cette fois.
func (inv *Inventaire) Rate(rng *rand.Rand, id string, req int) bool {
	if !echecsActifs {
		return false
	}
	lv := 1
	if me := inv.Metiers[id]; me != nil && me.Lv > 0 {
		lv = me.Lv
	}
	return rng.Float64() < EchecChance(lv, req)
}

// Gache : une fabrication ratée gâche la moitié des matériaux (au moins un).
func (inv *Inventaire) Gache(r *Recipe) {
	lost := 0
	for _, n := range r.Needs {
		if k := n.N / 2; k > 0 && inv.Take(n.Base, k) {
			lost += k
		}
	}
	if lost == 0 && len(r.Needs) > 0 {
		inv.Take(r.Needs[0].Base, 1)
	}
	inv.dirty = true
}

// MetierXP : l'expérience pour passer du niveau l au suivant.
func MetierXP(l int) int { return 10 + 4*l }

// --- Les matériaux ------------------------------------------------------------

func mat(id, name, icon string, tier, value, price int, desc string) {
	addBase(&Base{ID: id, Name: name, Kind: KMateriau, Icon: icon, Stack: 99, Tier: tier, Value: value, Price: price, Desc: desc, Armor: -1})
}

// TrophyMat : les trophées de l'Acte III deviennent des matériaux (Conseil des Objets).
var TrophyMat = map[string]string{
	"Défense de sanglier": "defense_sanglier", "Croc de loup": "croc_loup", "Croc sans couleur": "croc_sans_couleur",
	"Plume grise": "plume_grise", "Défense dorée": "defense_doree", "Défense du Grand Sanglier": "defense_grand_sanglier",
}

func init() {
	// La récolte.
	mat("minerai_cuivre", "Minerai de cuivre", "🪨", 1, 6, 0, "Il se fond à la forge en lingots.")
	mat("minerai_fer", "Minerai de fer", "🪨", 2, 14, 0, "Le fer des collines du nord. Doran n'aime pas son reflet gris.")
	mat("quartz", "Quartz brut", "🔹", 1, 20, 0, "Un caillou brillant. Braise en a une collection.")
	mat("ambre", "Ambre brut", "🟠", 2, 40, 0, "Il y a parfois un insecte dedans. Il a l'air surpris.")
	mat("sauge", "Sauge", "🌿", 1, 5, 0, "Pour les potions et pour le ragoût.")
	mat("lin", "Lin", "🌾", 1, 4, 0, "Il se tisse en toile.")
	mat("lysimaque", "Lysimaque", "🌼", 1, 8, 0, "Une fleur jaune qui calme les blessures.")
	mat("chardon_gris", "Chardon gris", "🥀", 2, 12, 0, "Il ne pousse que dans la Lisière. Ses fibres font une toile solide.")
	mat("fleur_de_lune", "Fleur de lune", "🌸", 2, 30, 0, "Elle s'ouvre pour la lumière, même celle d'une torche.")
	mat("cuir_brut", "Cuir brut", "🟫", 1, 6, 0, "À tanner avant d'en faire quoi que ce soit.")
	mat("cuir_terne", "Cuir terne", "⬛", 2, 14, 0, "La peau d'une bête effacée. Elle n'a plus de couleur.")
	mat("viande", "Viande de gibier", "🥩", 1, 4, 0, "Mabon sait quoi en faire.")
	mat("bois_chene", "Bois de chêne", "🪵", 1, 5, 0, "Du bon bois du val.")
	mat("bois_gris", "Bois gris", "🪵", 2, 12, 0, "Un bois de la Lisière, dur comme de la pierre.")
	// Les trophées.
	mat("defense_sanglier", "Défense de sanglier", "🦷", 1, 30, 0, "Un trophée du val, recherché par les artisans.")
	mat("croc_loup", "Croc de loup", "🦷", 1, 40, 0, "Un trophée du val, recherché par les artisans.")
	mat("croc_sans_couleur", "Croc sans couleur", "🦴", 2, 80, 0, "Il sert aux plans rares.")
	mat("plume_grise", "Plume grise", "🪶", 2, 80, 0, "Elle sert aux plans rares.")
	mat("defense_doree", "Défense dorée", "✨", 2, 600, 0, "Le trophée du Sanglier doré.")
	mat("soie_doree", "Soie dorée", "🧵", 2, 400, 0, "Les poils d'or du Sanglier doré. Les artisans se l'arrachent.")
	mat("defense_grand_sanglier", "Défense du Grand Sanglier", "🐗", 2, 500, 0, "Elle sert aux plans épiques.")
	// Chez Aelis.
	mat("fil", "Bobine de fil", "🧵", 1, 2, 10, "Vendu par Aelis.")
	mat("flux", "Flux de forge", "🧂", 1, 3, 15, "Vendu par Aelis.")
	mat("fiole", "Fiole vide", "🧪", 1, 2, 10, "Vendue par Aelis.")
	mat("sel", "Sel et épices", "🧂", 1, 1, 5, "Vendus par Aelis.")
	mat("tanin", "Tanin", "🫙", 1, 3, 15, "Vendu par Aelis.")
	// Les matériaux travaillés.
	mat("lingot_cuivre", "Lingot de cuivre", "🟧", 1, 15, 0, "Fondu à la forge.")
	mat("lingot_fer", "Lingot de fer", "⬜", 2, 32, 0, "Fondu à la forge.")
	mat("toile_lin", "Toile de lin", "📜", 1, 10, 0, "Tissée à l'atelier de couture.")
	mat("toile_grise", "Toile grise", "📜", 2, 26, 0, "Tissée à l'atelier de couture.")
	mat("cuir_tanne", "Cuir tanné", "🟤", 1, 12, 0, "Tanné à l'atelier du cuir.")
	mat("cuir_gris", "Cuir gris tanné", "🟤", 2, 30, 0, "Tanné à l'atelier du cuir.")
	mat("gemme_quartz", "Quartz taillé", "💠", 1, 45, 0, "Taillé à l'établi du joaillier.")
	mat("gemme_ambre", "Ambre taillé", "🔶", 2, 90, 0, "Taillé à l'établi du joaillier.")

	addBase(&Base{ID: "plan_rare", Name: "Plan rare", Kind: KRecette, Icon: "📘", Armor: -1, Value: 200,
		Desc: "Un plan d'artisan. À lire pour apprendre une recette rare."})
	addBase(&Base{ID: "plan_epique", Name: "Plan épique", Kind: KRecette, Icon: "📕", Armor: -1, Value: 800,
		Desc: "Un plan d'artisan. À lire pour apprendre une recette épique."})
	buildRecipes()
}

// --- Les recettes -------------------------------------------------------------

// Need : un matériau demandé par une recette.
type Need struct {
	Base string `json:"b"`
	N    int    `json:"n"`
}

// Recipe : une recette.
type Recipe struct {
	ID     string
	Metier string
	Out    string // la base fabriquée
	Lv     int    // niveau de l'objet (équipement)
	R      int    // rareté de l'objet (équipement)
	N      int    // nombre fabriqué (matériaux, provisions)
	Req    int    // niveau de métier demandé
	Needs  []Need
	Start  bool   // connue dès qu'on apprend le métier
	Price  int    // prix chez le maître (0 : plan ou recette de départ)
	Plan   string // "rare" ou "epique" : la recette s'apprend avec un plan
}

// Recipes : toutes les recettes, par identifiant.
var Recipes = map[string]*Recipe{}
var recipesBy = map[string][]*Recipe{}

func addRecipe(r *Recipe) {
	if r.N == 0 {
		r.N = 1
	}
	if _, ok := Recipes[r.ID]; ok {
		panic("recette en double : " + r.ID)
	}
	Recipes[r.ID] = r
	recipesBy[r.Metier] = append(recipesBy[r.Metier], r)
}

// RecipesOf : les recettes d'un métier, par niveau demandé.
func RecipesOf(metier string) []*Recipe { return recipesBy[metier] }

// Les trois paliers d'objets du val : niveau de l'objet → niveau de métier.
var craftLevels = []struct{ Lv, Req, Tier int }{{1, 1, 1}, {5, 5, 1}, {10, 10, 2}}

// Les matériaux de chaque palier.
var tierMat = map[string][3]string{
	"lingot": {"", "lingot_cuivre", "lingot_fer"},
	"toile":  {"", "toile_lin", "toile_grise"},
	"cuir":   {"", "cuir_tanne", "cuir_gris"},
	"bois":   {"", "bois_chene", "bois_gris"},
	"gemme":  {"", "gemme_quartz", "gemme_ambre"},
	"herbe":  {"", "sauge", "chardon_gris"},
}

func pieceN(s Slot) int { return int(math.Round(2 + 3*slotArmure[s])) }

// RecipePrice : le prix d'une recette chez le maître.
func RecipePrice(req int) int {
	if req <= 1 {
		return 0
	}
	return req * req * 20
}

func equipRecipe(metier, base string, needs func(tier int) []Need) {
	for _, cl := range craftLevels {
		r := &Recipe{ID: fmt.Sprintf("%s:%s:%d", metier, base, cl.Lv), Metier: metier, Out: base, Lv: cl.Lv, R: PeuCommun,
			Req: cl.Req, Needs: needs(cl.Tier), Start: cl.Req <= 1, Price: RecipePrice(cl.Req)}
		addRecipe(r)
		// Les plans rares et épiques : les mêmes matériaux, plus des trophées.
		rare := *r
		rare.ID, rare.R, rare.Start, rare.Price, rare.Plan = r.ID+"@rare", Rare, false, 0, "rare"
		rare.Needs = append(append([]Need(nil), r.Needs...), Need{[2]string{"croc_sans_couleur", "plume_grise"}[len(base)%2], 2})
		addRecipe(&rare)
		if cl.Lv == 10 {
			ep := *r
			ep.ID, ep.R, ep.Start, ep.Price, ep.Plan = r.ID+"@epique", Epique, false, 0, "epique"
			ep.Needs = append(append([]Need(nil), r.Needs...), Need{"defense_grand_sanglier", 1}, Need{"soie_doree", 1})
			addRecipe(&ep)
		}
	}
}

func simpleRecipe(metier, id, out string, n, req int, needs ...Need) {
	addRecipe(&Recipe{ID: metier + ":" + id, Metier: metier, Out: out, N: n, Req: req, Needs: needs,
		Start: req <= 1, Price: RecipePrice(req)})
}

func buildRecipes() {
	// Les matériaux travaillés.
	simpleRecipe("forge", "lingot_cuivre", "lingot_cuivre", 1, 1, Need{"minerai_cuivre", 2})
	simpleRecipe("forge", "lingot_fer", "lingot_fer", 1, 10, Need{"minerai_fer", 2}, Need{"flux", 1})
	simpleRecipe("couture", "toile_lin", "toile_lin", 1, 1, Need{"lin", 2})
	simpleRecipe("couture", "toile_grise", "toile_grise", 1, 10, Need{"chardon_gris", 2}, Need{"fil", 1})
	simpleRecipe("cuir", "cuir_tanne", "cuir_tanne", 1, 1, Need{"cuir_brut", 1}, Need{"tanin", 1})
	simpleRecipe("cuir", "cuir_gris", "cuir_gris", 1, 10, Need{"cuir_terne", 1}, Need{"tanin", 1})
	simpleRecipe("joaillerie", "gemme_quartz", "gemme_quartz", 1, 1, Need{"quartz", 1})
	simpleRecipe("joaillerie", "gemme_ambre", "gemme_ambre", 1, 10, Need{"ambre", 1})

	m := func(kind string, tier int) string { return tierMat[kind][tier] }
	// Les armures.
	for s := range pieceNoms {
		s := s
		n := pieceN(s)
		equipRecipe("couture", SlotIDs[s]+"_tissu", func(t int) []Need { return []Need{{m("toile", t), n}, {"fil", 1}} })
		equipRecipe("cuir", SlotIDs[s]+"_cuir", func(t int) []Need { return []Need{{m("cuir", t), n}, {"fil", 1}} })
		equipRecipe("forge", SlotIDs[s]+"_mailles", func(t int) []Need { return []Need{{m("lingot", t), n}, {m("cuir", t), 1}} })
		equipRecipe("forge", SlotIDs[s]+"_plaques", func(t int) []Need { return []Need{{m("lingot", t), n + 1}, {"flux", 1}} })
	}
	// Les armes.
	metal := []string{"epee", "hache", "lance", "epee_longue", "faux", "arquebuse", "masse", "rapiere", "dagues"}
	for _, w := range metal {
		equipRecipe("forge", "arme_"+w, func(t int) []Need { return []Need{{m("lingot", t), 4}, {m("bois", t), 1}} })
	}
	for _, w := range []string{"bouclier", "bouclier_rond"} {
		equipRecipe("forge", "mg_"+w, func(t int) []Need { return []Need{{m("lingot", t), 3}, {m("bois", t), 2}} })
	}
	for _, w := range []string{"baton", "baton_mage", "baton_noueux", "totem", "orbe"} {
		equipRecipe("joaillerie", "arme_"+w, func(t int) []Need { return []Need{{m("bois", t), 3}, {m("gemme", t), 1}} })
	}
	equipRecipe("joaillerie", "mg_talisman", func(t int) []Need { return []Need{{m("lingot", t), 1}, {m("gemme", t), 1}} })
	equipRecipe("joaillerie", "anneau", func(t int) []Need { return []Need{{m("lingot", t), 1}, {m("gemme", t), 1}} })
	equipRecipe("joaillerie", "cou_amulette", func(t int) []Need { return []Need{{m("lingot", t), 1}, {m("gemme", t), 2}} })
	equipRecipe("cuir", "arme_arc", func(t int) []Need { return []Need{{m("bois", t), 3}, {"fil", 2}} })
	equipRecipe("cuir", "mg_carquois", func(t int) []Need { return []Need{{m("cuir", t), 3}, {"fil", 1}} })
	equipRecipe("couture", "dos_cape", func(t int) []Need { return []Need{{m("toile", t), 3}, {"fil", 1}} })
	equipRecipe("couture", "mg_luth", func(t int) []Need { return []Need{{m("bois", t), 3}, {"fil", 2}} })
	equipRecipe("alchimie", "arme_fioles", func(t int) []Need { return []Need{{"fiole", 4}, {m("herbe", t), 3}} })

	// Les sacs (couture).
	simpleRecipe("couture", "sac_toile", "sac_toile", 1, 1, Need{"toile_lin", 4}, Need{"fil", 2})
	simpleRecipe("couture", "sac_voyage", "sac_voyage", 1, 5, Need{"toile_lin", 6}, Need{"cuir_tanne", 2}, Need{"fil", 3})
	simpleRecipe("couture", "sac_aventurier", "sac_aventurier", 1, 10, Need{"toile_grise", 6}, Need{"cuir_gris", 2}, Need{"fil", 4})

	// L'alchimie : potions et élixirs.
	simpleRecipe("alchimie", "petite_potion", "petite_potion", 2, 1, Need{"sauge", 2}, Need{"fiole", 2})
	simpleRecipe("alchimie", "potion", "potion", 2, 5, Need{"lysimaque", 2}, Need{"sauge", 1}, Need{"fiole", 2})
	simpleRecipe("alchimie", "grande_potion", "grande_potion", 2, 10, Need{"fleur_de_lune", 1}, Need{"chardon_gris", 2}, Need{"fiole", 2})
	simpleRecipe("alchimie", "elixir_force", "elixir_force", 1, 3, Need{"sauge", 2}, Need{"defense_sanglier", 1}, Need{"fiole", 1})
	simpleRecipe("alchimie", "elixir_esprit", "elixir_esprit", 1, 3, Need{"lysimaque", 2}, Need{"quartz", 1}, Need{"fiole", 1})
	simpleRecipe("alchimie", "elixir_lisiere", "elixir_lisiere", 1, 10, Need{"fleur_de_lune", 1}, Need{"plume_grise", 1}, Need{"fiole", 1})

	// La cuisine de Mabon.
	simpleRecipe("cuisine", "brochette", "brochette", 2, 1, Need{"viande", 2}, Need{"sel", 1})
	simpleRecipe("cuisine", "galette_sauge", "galette_sauge", 2, 1, Need{"sauge", 2}, Need{"sel", 1})
	simpleRecipe("cuisine", "ragout", "ragout", 2, 5, Need{"viande", 2}, Need{"sauge", 1}, Need{"lysimaque", 1}, Need{"sel", 1})
	simpleRecipe("cuisine", "tourte_chasseur", "tourte_chasseur", 2, 10, Need{"viande", 3}, Need{"chardon_gris", 1}, Need{"sel", 1})
	simpleRecipe("cuisine", "infusion_lune", "infusion_lune", 2, 10, Need{"fleur_de_lune", 1}, Need{"lysimaque", 1})

	for _, rs := range recipesBy {
		sort.SliceStable(rs, func(i, j int) bool {
			if rs[i].Req != rs[j].Req {
				return rs[i].Req < rs[j].Req
			}
			return rs[i].ID < rs[j].ID
		})
	}
}

// RecipeName : le nom d'une recette (celui de ce qu'elle fabrique).
func (r *Recipe) Name() string {
	b := Bases[r.Out]
	if b == nil {
		return r.ID
	}
	if b.Kind == KEquipement {
		return fmt.Sprintf("%s (niv. %d)", b.Name, r.Lv)
	}
	return b.Name
}

// --- La récolte ---------------------------------------------------------------

// NodeKind : une sorte de point de récolte.
type NodeKind struct {
	ID, Name, Metier string
	Req              int
	Gives            string
	Min, Max         int
	Bonus            string  // une trouvaille en plus, parfois
	BonusP           float64 // sa chance
	Charges          int
	Respawn          [2]float64 // secondes avant de repousser (min, max)
}

var NodeKinds = map[string]*NodeKind{
	"filon_cuivre":  {ID: "filon_cuivre", Name: "Filon de cuivre", Metier: "minage", Req: 1, Gives: "minerai_cuivre", Min: 1, Max: 2, Bonus: "quartz", BonusP: 0.12, Charges: 3, Respawn: [2]float64{120, 200}},
	"filon_fer":     {ID: "filon_fer", Name: "Filon de fer", Metier: "minage", Req: 10, Gives: "minerai_fer", Min: 1, Max: 2, Bonus: "ambre", BonusP: 0.1, Charges: 3, Respawn: [2]float64{150, 240}},
	"sauge":         {ID: "sauge", Name: "Touffe de sauge", Metier: "herboristerie", Req: 1, Gives: "sauge", Min: 1, Max: 3, Charges: 2, Respawn: [2]float64{90, 160}},
	"lin":           {ID: "lin", Name: "Lin sauvage", Metier: "herboristerie", Req: 1, Gives: "lin", Min: 2, Max: 3, Charges: 2, Respawn: [2]float64{90, 160}},
	"lysimaque":     {ID: "lysimaque", Name: "Lysimaque", Metier: "herboristerie", Req: 4, Gives: "lysimaque", Min: 1, Max: 2, Charges: 2, Respawn: [2]float64{100, 180}},
	"chardon_gris":  {ID: "chardon_gris", Name: "Chardon gris", Metier: "herboristerie", Req: 10, Gives: "chardon_gris", Min: 1, Max: 3, Charges: 2, Respawn: [2]float64{120, 200}},
	"fleur_de_lune": {ID: "fleur_de_lune", Name: "Fleur de lune", Metier: "herboristerie", Req: 12, Gives: "fleur_de_lune", Min: 1, Max: 1, Charges: 1, Respawn: [2]float64{240, 400}},
	"chene":         {ID: "chene", Name: "Chêne à couper", Metier: "bucheronnage", Req: 1, Gives: "bois_chene", Min: 1, Max: 3, Charges: 3, Respawn: [2]float64{120, 200}},
	"arbre_gris":    {ID: "arbre_gris", Name: "Arbre gris", Metier: "bucheronnage", Req: 10, Gives: "bois_gris", Min: 1, Max: 2, Charges: 3, Respawn: [2]float64{150, 240}},
}

// Skins : ce que donne le dépeçage de chaque bête (niveau demandé, cuir, viande).
var Skins = map[string]struct {
	Req        int
	Hide, Meat string
	N          [2]int
}{
	"sanglier":       {1, "cuir_brut", "viande", [2]int{1, 2}},
	"loup":           {1, "cuir_brut", "", [2]int{1, 2}},
	"loup_efface":    {10, "cuir_terne", "", [2]int{1, 2}},
	"sanglier_dore":  {1, "soie_doree", "viande", [2]int{1, 1}},
	"grand_sanglier": {10, "cuir_terne", "viande", [2]int{4, 6}},
}

// Node : un point de récolte de la carte (partagé : le premier arrivé se sert).
type Node struct {
	ID      int
	Kind    *NodeKind
	X, Y    float64
	Charges int
	wait    float64 // secondes avant de repousser
}

// Up dit si le point peut être récolté.
func (n *Node) Up() bool { return n.Charges > 0 }

// Atelier : là où l'on fabrique.
type Atelier struct {
	ID     int
	Metier string
	X, Y   float64
}

// MetiersData : les points de récolte et les ateliers d'une carte
// (maps/<carte>.metiers.json).
type MetiersData struct {
	Noeuds []struct {
		Sorte string     `json:"sorte"`
		At    [2]float64 `json:"at"`
	} `json:"noeuds"`
	Ateliers []struct {
		Metier string     `json:"metier"`
		At     [2]float64 `json:"at"`
	} `json:"ateliers"`
}

// GainMetier ajoute de l'expérience à un métier. Il renvoie le nouveau niveau
// s'il a monté (0 sinon). Une tâche bien plus facile que son niveau ne rapporte rien.
func (inv *Inventaire) GainMetier(id string, req, xp int) int {
	me := inv.Metiers[id]
	if me == nil || me.Lv <= 0 || me.Lv >= MetierMax {
		return 0
	}
	if d := me.Lv - req; d >= 10 {
		return 0
	} else if d >= 5 {
		xp /= 2
	}
	me.XP += xp
	up := 0
	for me.Lv < MetierMax && me.XP >= MetierXP(me.Lv) {
		me.XP -= MetierXP(me.Lv)
		me.Lv++
		up = me.Lv
	}
	inv.dirty = true
	return up
}

// Learn apprend un métier d'artisanat (2 au plus, la cuisine en plus).
func (inv *Inventaire) Learn(id string) error {
	d := Metiers[id]
	if d == nil || d.Recolte || id == "cuisine" {
		return errors.New("Ce métier ne s'apprend pas ainsi.")
	}
	for _, a := range inv.Artisanats {
		if a == id {
			return errors.New("Tu connais déjà ce métier.")
		}
	}
	if len(inv.Artisanats) >= ArtisanatsMax {
		return errors.New("Tu connais déjà deux métiers d'artisanat. Il faut en oublier un d'abord.")
	}
	inv.Artisanats = append(inv.Artisanats, id)
	inv.Metiers[id] = &Metier{Lv: 1}
	for _, r := range RecipesOf(id) {
		if r.Start {
			inv.Recettes[r.ID] = true
		}
	}
	inv.dirty = true
	return nil
}

// Forget oublie un métier d'artisanat (son niveau et ses recettes sont perdus).
func (inv *Inventaire) Forget(id string) error {
	for i, a := range inv.Artisanats {
		if a == id {
			inv.Artisanats = append(inv.Artisanats[:i], inv.Artisanats[i+1:]...)
			inv.Metiers[id] = &Metier{}
			for _, r := range RecipesOf(id) {
				delete(inv.Recettes, r.ID)
			}
			inv.dirty = true
			return nil
		}
	}
	return errors.New("Tu ne connais pas ce métier.")
}

// Knows dit si le héros pratique ce métier.
func (inv *Inventaire) Knows(id string) bool {
	me := inv.Metiers[id]
	return me != nil && me.Lv > 0
}

// CanCraft vérifie qu'une recette est connue et que les matériaux sont là.
func (inv *Inventaire) CanCraft(r *Recipe) error {
	if !inv.Knows(r.Metier) {
		return errors.New("Tu ne pratiques pas ce métier.")
	}
	if !inv.Recettes[r.ID] {
		return errors.New("Tu ne connais pas cette recette.")
	}
	if inv.Metiers[r.Metier].Lv < r.Req {
		return errors.New("Il faut le niveau " + strconv.Itoa(r.Req) + " en " + Metiers[r.Metier].Name + ".")
	}
	for _, n := range r.Needs {
		if inv.Count(n.Base) < n.N {
			return errors.New("Il manque : " + Bases[n.Base].Name + ".")
		}
	}
	return nil
}

// Craft fabrique : les matériaux sont pris, l'objet rangé. choice : le bonus
// choisi par l'artisan. Il renvoie l'objet, et si la chance a souri.
func (inv *Inventaire) Craft(rng *rand.Rand, r *Recipe, cl, maker, choice string) (*Item, bool, error) {
	if err := inv.CanCraft(r); err != nil {
		return nil, false, err
	}
	out := Bases[r.Out]
	var it *Item
	lucky := false
	if out.Kind == KEquipement {
		rr := r.R
		luck := math.Min(LuckMax, LuckBase+LuckPerLevel*float64(inv.Metiers[r.Metier].Lv-r.Req))
		if rng.Float64() < luck && rr < Legendaire {
			rr++
			lucky = true
		}
		it = &Item{ID: NewID(rng), Base: r.Out, Lv: r.Lv, R: rr, Dur: 100, Fab: maker}
		var forced []string
		if AffixNoms[choice] != "" {
			forced = []string{choice}
		}
		it.Aff = rollAffixes(rng, rareteBonus[rr], r.Lv, cl, forced)
		if rr >= Legendaire {
			it.Leg = []LegendEffect{{Skill: 1 + rng.Intn(7), Kind: LegendKinds[rng.Intn(len(LegendKinds))]}}
		}
		if inv.Free() == 0 {
			return nil, false, ErrSacPlein
		}
	} else {
		it = MakeStack(rng, r.Out, r.N)
		if inv.Room(r.Out) < r.N {
			return nil, false, ErrSacPlein
		}
	}
	for _, n := range r.Needs {
		inv.Take(n.Base, n.N)
	}
	inv.Add(it)
	return it, lucky, nil
}

// LoadMetiers lit les points de récolte et les ateliers d'une carte
// (rien si la carte n'en a pas).
func LoadMetiers(m *Map) (*MetiersData, error) {
	raw, err := mapFiles.ReadFile("maps/" + m.ID + ".metiers.json")
	if err != nil {
		return &MetiersData{}, nil
	}
	var d MetiersData
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("métiers de %q: %w", m.ID, err)
	}
	for i, n := range d.Noeuds {
		if NodeKinds[n.Sorte] == nil {
			return nil, fmt.Errorf("métiers de %q: sorte inconnue %q", m.ID, n.Sorte)
		}
		if !m.Free(n.At[0], n.At[1]) {
			return nil, fmt.Errorf("métiers de %q: le point %d (%s) est dans un mur", m.ID, i+1, n.Sorte)
		}
	}
	for _, at := range d.Ateliers {
		if !m.Free(at.At[0], at.At[1]) {
			return nil, fmt.Errorf("métiers de %q: l'atelier %s est dans un mur", m.ID, at.Metier)
		}
	}
	return &d, nil
}
