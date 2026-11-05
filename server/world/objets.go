package world

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

// Les objets (Acte V, Conseil des Objets) : 12 emplacements, 4 types
// d'armure, 6 raretés, une base fixe et des bonus tirés au hasard. Un objet
// est un petit enregistrement (Item) ; tout ce qu'on en montre (nom, chiffres,
// prix) se calcule à partir de sa base, de son niveau et de sa rareté.

// Monnaie : tout se compte en cuivres. 100 cuivres = 1 argent, 100 argents = 1 or.
const (
	Argent = 100
	Or     = 100 * Argent
)

// Slot : un emplacement d'équipement.
type Slot int

const (
	Tete Slot = iota
	Epaules
	Torse
	Mains
	Jambes
	Pieds
	Dos
	ArmeS
	MainGauche
	Cou
	Anneau1
	Anneau2
	NSlots
)

// SlotIDs : les noms des emplacements, dans l'ordre (le client les connaît).
var SlotIDs = [NSlots]string{"tete", "epaules", "torse", "mains", "jambes", "pieds", "dos", "arme", "main_gauche", "cou", "anneau1", "anneau2"}

// Armure : le type d'armure d'une classe.
type Armure int

const (
	Tissu Armure = iota
	Cuir
	Mailles
	Plaques
)

var ArmureIDs = [4]string{"tissu", "cuir", "mailles", "plaques"}
var armureNoms = [4]string{"de tissu", "de cuir", "de mailles", "de plaques"}

// ClassArmor : le type d'armure de chaque classe.
var ClassArmor = map[string]Armure{
	"guerrier": Plaques, "paladin": Plaques, "chevalier_dragon": Plaques,
	"berserker": Mailles, "lancier": Mailles, "ingenieur": Mailles, "chaman": Mailles,
	"moine": Cuir, "alchimiste": Cuir, "druide": Cuir, "barde": Cuir, "voleur": Cuir, "archer": Cuir,
	"mage": Tissu, "necromancien": Tissu, "invocateur": Tissu, "pretre": Tissu,
}

// ClassWeapon : l'arme de chaque classe (le dessin de la forge des peuples).
var ClassWeapon = map[string]string{
	"guerrier": "epee", "paladin": "epee", "berserker": "hache", "lancier": "lance", "chevalier_dragon": "epee_longue",
	"moine": "baton", "mage": "baton_mage", "necromancien": "faux", "invocateur": "orbe", "alchimiste": "fioles",
	"ingenieur": "arquebuse", "pretre": "masse", "druide": "baton_noueux", "chaman": "totem", "barde": "rapiere",
	"voleur": "dagues", "archer": "arc",
}

// ClassOffhand : ce que chaque classe tient de l'autre main. Les autres classes
// portent un talisman (un objet de bonus qui ne se voit pas).
var ClassOffhand = map[string]string{
	"guerrier": "bouclier_rond", "paladin": "bouclier", "barde": "luth", "archer": "carquois",
}

// Rareté.
const (
	Commun = iota
	PeuCommun
	Rare
	Epique
	Legendaire
	Lumineux
	NRaretes
)

var RareteIDs = [NRaretes]string{"commun", "peu_commun", "rare", "epique", "legendaire", "lumineux"}
var RareteNoms = [NRaretes]string{"Commun", "Peu commun", "Rare", "Épique", "Légendaire", "Lumineux"}

// rareteMul : la force de la base (armure, arme) selon la rareté.
var rareteMul = [NRaretes]float64{1, 1.06, 1.12, 1.2, 1.28, 1.36}

// rareteBonus : le nombre de bonus tirés au hasard.
var rareteBonus = [NRaretes]int{0, 1, 2, 3, 3, 4}

// rareteValeur : la valeur d'un objet, selon sa rareté.
var rareteValeur = [NRaretes]float64{1, 2, 5, 12, 30, 60}

// Les poids d'armure de chaque pièce (le torse protège le plus).
var slotArmure = map[Slot]float64{Torse: 1, Jambes: 0.75, Tete: 0.6, Epaules: 0.5, Mains: 0.4, Pieds: 0.4}

// Kind : la nature d'une base.
type Kind int

const (
	KEquipement Kind = iota
	KMateriau
	KConso
	KRecette
	KSac
)

// Base : un modèle d'objet.
type Base struct {
	ID     string
	Name   string
	Kind   Kind
	Slot   Slot   // équipement
	Armor  Armure // pièces d'armure (-1 sinon)
	Weapon string // arme ou main gauche : le dessin (epee, bouclier…), "talisman"
	Icon   string
	Stack  int    // objets qui s'empilent : jusqu'à combien
	Price  int    // matériaux et provisions : prix chez Aelis (0 : pas en vente)
	Value  int    // valeur des matériaux et provisions (rachat : un quart)
	Tier   int    // matériaux : palier (1, 2, 3)
	Desc   string // une phrase
	Use    *Usage // provisions
	Places int    // sacs : places en plus
	Recipe string // recettes : la recette apprise
}

// Bases : tous les modèles connus, par identifiant.
var Bases = map[string]*Base{}

func addBase(b *Base) *Base {
	if _, ok := Bases[b.ID]; ok {
		panic("base en double : " + b.ID)
	}
	if b.Stack == 0 {
		b.Stack = 1
	}
	Bases[b.ID] = b
	return b
}

// Les noms des pièces, par type d'armure.
var pieceNoms = map[Slot][4]string{
	Tete:    {"Capuche", "Chaperon", "Coiffe", "Heaume"},
	Epaules: {"Mantelet", "Spallières", "Épaulières", "Spalières"},
	Torse:   {"Robe", "Pourpoint", "Haubert", "Cuirasse"},
	Mains:   {"Mitaines", "Gants", "Gantelets", "Gantelets"},
	Jambes:  {"Chausses", "Jambières", "Jambières", "Cuissards"},
	Pieds:   {"Sandales", "Bottes", "Solerets", "Sabatons"},
}
var pieceIcones = map[Slot]string{Tete: "🪖", Epaules: "🎽", Torse: "🥋", Mains: "🧤", Jambes: "👖", Pieds: "🥾"}

var armeNoms = map[string]string{
	"epee": "Épée", "hache": "Hache", "lance": "Lance", "epee_longue": "Épée longue", "baton": "Bâton",
	"baton_mage": "Bâton de mage", "faux": "Faux", "orbe": "Orbe", "fioles": "Fioles", "arquebuse": "Arquebuse",
	"masse": "Masse", "baton_noueux": "Bâton noueux", "totem": "Totem", "rapiere": "Rapière", "dagues": "Dagues",
	"arc": "Arc", "bouclier": "Écu", "bouclier_rond": "Bouclier rond", "luth": "Luth", "carquois": "Carquois",
	"talisman": "Talisman",
}
var armeIcones = map[string]string{
	"epee": "🗡️", "hache": "🪓", "lance": "🔱", "epee_longue": "⚔️", "baton": "🦯", "baton_mage": "🪄", "faux": "☠️",
	"orbe": "🔮", "fioles": "⚗️", "arquebuse": "🔫", "masse": "🔨", "baton_noueux": "🌿", "totem": "🗿", "rapiere": "🤺",
	"dagues": "🔪", "arc": "🏹", "bouclier": "🛡️", "bouclier_rond": "🛡️", "luth": "🪕", "carquois": "🎯", "talisman": "🧿",
}

func init() {
	for s, noms := range pieceNoms {
		for a := Tissu; a <= Plaques; a++ {
			addBase(&Base{ID: SlotIDs[s] + "_" + ArmureIDs[a], Name: noms[a] + " " + armureNoms[a], Kind: KEquipement,
				Slot: s, Armor: a, Icon: pieceIcones[s]})
		}
	}
	addBase(&Base{ID: "dos_cape", Name: "Cape", Kind: KEquipement, Slot: Dos, Armor: -1, Icon: "🧣"})
	addBase(&Base{ID: "cou_amulette", Name: "Amulette", Kind: KEquipement, Slot: Cou, Armor: -1, Icon: "📿"})
	addBase(&Base{ID: "anneau", Name: "Anneau", Kind: KEquipement, Slot: Anneau1, Armor: -1, Icon: "💍"})
	seen := map[string]bool{}
	for _, w := range ClassWeapon {
		if !seen[w] {
			seen[w] = true
			addBase(&Base{ID: "arme_" + w, Name: armeNoms[w], Kind: KEquipement, Slot: ArmeS, Armor: -1, Weapon: w, Icon: armeIcones[w]})
		}
	}
	for _, w := range []string{"bouclier", "bouclier_rond", "luth", "carquois", "talisman"} {
		addBase(&Base{ID: "mg_" + w, Name: armeNoms[w], Kind: KEquipement, Slot: MainGauche, Armor: -1, Weapon: w, Icon: armeIcones[w]})
	}
}

// Affix : un bonus d'objet.
type Affix struct {
	Stat string  `json:"s"`
	V    float64 `json:"v"`
}

// Les bonus possibles. Les attributs sont des points ; les autres, des
// pourcentages (0,03 = 3 %) ou de la vie.
var AffixStats = []string{"force", "intelligence", "agilite", "vitesse", "esquive", "vitalite", "chance", "endurance", "precision",
	"vie", "crit", "vol", "vit", "degats", "soins", "res_feu", "res_eau", "res_terre", "res_air", "res_lumiere", "res_ombre"}

// AffixNoms : comment le client écrit chaque bonus.
var AffixNoms = map[string]string{
	"force": "Force", "intelligence": "Intelligence", "agilite": "Agilité", "vitesse": "Vitesse", "esquive": "Esquive",
	"vitalite": "Vitalité", "chance": "Chance", "endurance": "Endurance", "precision": "Précision", "vie": "Vie",
	"crit": "Critique", "vol": "Vol de vie", "vit": "Vitesse de course", "degats": "Dégâts", "soins": "Soins",
	"res_feu": "Résistance au feu", "res_eau": "Résistance à l'eau", "res_terre": "Résistance à la terre",
	"res_air": "Résistance à l'air", "res_lumiere": "Résistance à la lumière", "res_ombre": "Résistance à l'ombre",
}

// AffixText : « +3 Force », « +1,2 % Critique ».
func AffixText(a Affix) string {
	switch a.Stat {
	case "crit", "vol", "vit", "degats", "soins":
		return "+" + strings.Replace(strconv.FormatFloat(math.Round(a.V*1000)/10, 'f', -1, 64), ".", ",", 1) + " % " + AffixNoms[a.Stat]
	case "res_feu", "res_eau", "res_terre", "res_air", "res_lumiere", "res_ombre":
		return "+" + strconv.Itoa(int(math.Round(a.V*100))) + " % " + AffixNoms[a.Stat]
	}
	return "+" + strconv.Itoa(int(math.Round(a.V))) + " " + AffixNoms[a.Stat]
}

// Les suffixes des noms : « Heaume de plaques du sanglier ».
var affixSuffixes = map[string]string{
	"force": "de l'ours", "intelligence": "du hibou", "agilite": "du renard", "vitesse": "du lièvre", "esquive": "du chat",
	"vitalite": "du chêne", "chance": "du trèfle", "endurance": "du sanglier", "precision": "du faucon", "vie": "de la source",
	"crit": "de l'éclair", "vol": "de la sangsue", "vit": "du vent", "degats": "de la colère", "soins": "de Solenn",
	"res_feu": "de la braise", "res_eau": "de la marée", "res_terre": "de la roche", "res_air": "des nuées",
	"res_lumiere": "de l'aube", "res_ombre": "du crépuscule",
}

// AffixValue : la valeur d'un bonus d'un objet de niveau lv.
func AffixValue(stat string, lv int) float64 {
	l := float64(lv)
	switch stat {
	case "vie":
		return math.Round(6 + 3*l)
	case "crit", "vit":
		return round3(0.01 + 0.001*l)
	case "vol":
		return round3(0.008 + 0.0008*l)
	case "degats", "soins":
		return round3(0.015 + 0.0015*l)
	}
	if strings.HasPrefix(stat, "res_") {
		return round3(0.03 + 0.003*l)
	}
	return math.Round(1 + 0.4*l) // un attribut
}

func round3(v float64) float64 { return math.Round(v*1000) / 1000 }

// LegendEffect : ce que fait un légendaire à une compétence.
type LegendEffect struct {
	Skill int    `json:"k"` // 1 à 7
	Kind  string `json:"e"` // puissance, recharge, cout, duree, zone
}

// LegendKinds : les effets des légendaires.
var LegendKinds = []string{"puissance", "recharge", "cout", "duree", "zone"}

// LegendText : la phrase d'un effet de légendaire.
func LegendText(e LegendEffect, skillName string) string {
	switch e.Kind {
	case "puissance":
		return skillName + " : +30 % de dégâts et de soins"
	case "recharge":
		return skillName + " : recharge 25 % plus courte"
	case "cout":
		return skillName + " : coûte 30 % de moins"
	case "duree":
		return skillName + " : ses effets durent 40 % de plus"
	case "zone":
		return skillName + " : touche aussi les ennemis à 1,5 case autour"
	}
	return skillName
}

func (e LegendEffect) talent() Talent {
	t := Talent{}
	switch e.Kind {
	case "puissance":
		t.Power, t.HealUp = 0.3, 0.3
	case "recharge":
		t.CD = 0.25
	case "cout":
		t.Cost = 0.3
	case "duree":
		t.DurUp = 0.4
	case "zone":
		t.Radius = 1.5
	}
	return t
}

// Item : un objet, dans un sac, au sol, porté ou au coffre.
type Item struct {
	ID   int64          `json:"id"`
	Base string         `json:"b"`
	Lv   int            `json:"lv,omitempty"`
	R    int            `json:"r,omitempty"`
	Aff  []Affix        `json:"a,omitempty"`
	Leg  []LegendEffect `json:"lg,omitempty"`
	Dur  int            `json:"d,omitempty"`  // usure : 100 = neuf, 0 = cassé
	Lie  bool           `json:"l,omitempty"`  // lié au héros
	Fab  string         `json:"f,omitempty"`  // fabriqué par
	Look string         `json:"lk,omitempty"` // apparence copiée au miroir : « base:rareté »
	N    int            `json:"n,omitempty"`  // pile
	Uniq string         `json:"u,omitempty"`  // objet unique, avec son nom
	Rec  string         `json:"rc,omitempty"` // plans : la recette apprise
}

// B : la base de l'objet (nil si elle n'existe plus).
func (it *Item) B() *Base { return Bases[it.Base] }

// Count : le nombre d'objets de la pile.
func (it *Item) Count() int {
	if it.N <= 0 {
		return 1
	}
	return it.N
}

// Broken : un objet cassé ne donne plus rien.
func (it *Item) Broken() bool {
	b := it.B()
	return b != nil && b.Kind == KEquipement && it.Dur <= 0
}

// Uniques : les objets uniques, nommés, avec leurs bonus fixes.
type Unique struct {
	ID, Name, Base, Lore string
	R, Lv                int
	Aff                  []Affix
	Leg                  []LegendEffect
}

var Uniques = map[string]*Unique{
	"defenses_de_mabon": {ID: "defenses_de_mabon", Name: "Les Défenses de Mabon", Base: "cou_amulette", R: Legendaire, Lv: 10,
		Lore: "Deux défenses taillées dans le Grand Sanglier lui-même. Mabon jure qu'elles sont « un peu plus petites que dans son histoire ».",
		Aff:  []Affix{{"vitalite", 6}, {"endurance", 6}, {"vie", 50}},
		Leg:  []LegendEffect{{Skill: 2, Kind: "puissance"}}},
	"eclat_du_lumenfall": {ID: "eclat_du_lumenfall", Name: "Éclat du Lumenfall", Base: "anneau", R: Lumineux, Lv: 8,
		Lore: "Un éclat de la lumière qui tomba du ciel. Il est chaud, même la nuit, même au fond d'une poche.",
		Aff:  []Affix{{"chance", 6}, {"crit", 0.03}, {"res_ombre", 0.08}, {"degats", 0.04}},
		Leg:  []LegendEffect{{Skill: 1, Kind: "puissance"}, {Skill: 3, Kind: "recharge"}}},
}

// NewID : un numéro d'objet (unique en pratique : 62 bits au hasard).
func NewID(rng *rand.Rand) int64 { return rng.Int63()>>1 | 1 }

// MakeEquip fabrique une pièce d'équipement de base b, de niveau lv et de
// rareté r, avec ses bonus tirés au hasard (penchant vers la classe cl).
func MakeEquip(rng *rand.Rand, base string, lv, r int, cl string) *Item {
	it := &Item{ID: NewID(rng), Base: base, Lv: max(1, lv), R: r, Dur: 100}
	b := it.B()
	if b == nil || b.Kind != KEquipement {
		return nil
	}
	it.Aff = rollAffixes(rng, rareteBonus[r], it.Lv, cl, nil)
	if r >= Legendaire {
		n := 1
		if r == Lumineux {
			n = 2
		}
		for i := 0; i < n; i++ {
			it.Leg = append(it.Leg, LegendEffect{Skill: 1 + rng.Intn(7), Kind: LegendKinds[rng.Intn(len(LegendKinds))]})
		}
	}
	return it
}

// MakeUnique fabrique un objet unique.
func MakeUnique(rng *rand.Rand, id string) *Item {
	u := Uniques[id]
	if u == nil {
		return nil
	}
	return &Item{ID: NewID(rng), Base: u.Base, Lv: u.Lv, R: u.R, Dur: 100, Uniq: id,
		Aff: append([]Affix(nil), u.Aff...), Leg: append([]LegendEffect(nil), u.Leg...)}
}

// MakeStack fabrique n objets empilables (matériaux, provisions, recettes).
func MakeStack(rng *rand.Rand, base string, n int) *Item {
	if Bases[base] == nil {
		return nil
	}
	return &Item{ID: NewID(rng), Base: base, N: n}
}

// classPool : les bonus que la classe aime (d'après sa répartition conseillée).
func classPool(cl string) []string {
	c := ClassByID(cl)
	if c == nil {
		return nil
	}
	type w struct {
		s string
		v int
	}
	var ws []w
	for i, v := range c.Attr {
		if v > 0 {
			ws = append(ws, w{AttrNames[i], v})
		}
	}
	sort.Slice(ws, func(i, j int) bool { return ws[i].v > ws[j].v })
	out := []string{"vie", "crit"}
	for i, x := range ws {
		if i < 3 {
			out = append(out, x.s)
		}
	}
	if c.Magic {
		out = append(out, "degats")
	}
	if c.ID == "pretre" || c.ID == "druide" || c.ID == "chaman" || c.ID == "paladin" || c.ID == "moine" {
		out = append(out, "soins")
	}
	return out
}

// rollAffixes tire n bonus différents ; la moitié du temps dans ce que la
// classe aime. forced : des bonus choisis (l'artisan en choisit un).
func rollAffixes(rng *rand.Rand, n, lv int, cl string, forced []string) []Affix {
	var out []Affix
	used := map[string]bool{}
	for _, s := range forced {
		if len(out) >= n || used[s] || AffixNoms[s] == "" {
			continue
		}
		used[s] = true
		out = append(out, Affix{s, AffixValue(s, lv)})
	}
	pool := classPool(cl)
	for tries := 0; len(out) < n && tries < 100; tries++ {
		var s string
		if len(pool) > 0 && rng.Float64() < 0.5 {
			s = pool[rng.Intn(len(pool))]
		} else {
			s = AffixStats[rng.Intn(len(AffixStats))]
		}
		if used[s] {
			continue
		}
		used[s] = true
		v := AffixValue(s, lv)
		// Un peu de hasard dans la valeur : de 80 % à 110 %.
		k := 0.8 + 0.3*rng.Float64()
		if v >= 1 {
			v = math.Max(1, math.Round(v*k))
		} else {
			v = round3(v * k)
		}
		out = append(out, Affix{s, v})
	}
	return out
}

// ArmorValue : l'armure d'une pièce.
func (it *Item) ArmorValue() float64 {
	b := it.B()
	if b == nil || b.Kind != KEquipement || b.Armor < 0 || it.Broken() {
		return 0
	}
	typ := float64(b.Armor+1) / 4
	v := typ * slotArmure[b.Slot] * 0.048 * (40 + 12*float64(it.Lv)) * rareteMul[it.R]
	if it.Fab != "" {
		v *= CraftBonus
	}
	return math.Round(v*10) / 10
}

// WeaponPower : la puissance d'une arme (0 si ce n'en est pas une).
func (it *Item) WeaponPower() float64 {
	b := it.B()
	if b == nil || b.Slot != ArmeS || b.Kind != KEquipement || it.Broken() {
		return 0
	}
	v := HeroWeapon(it.Lv) * rareteMul[it.R]
	if it.Fab != "" {
		v *= CraftBonus
	}
	return math.Round(v*10) / 10
}

// CraftBonus : un objet fabriqué est un peu meilleur que le butin de même rareté
// (précision d'Elowan au Conseil des Objets).
const CraftBonus = 1.08

// Affixes : les bonus, avec le petit plus des objets fabriqués.
func (it *Item) Affixes() []Affix {
	if it.Fab == "" {
		return it.Aff
	}
	out := make([]Affix, len(it.Aff))
	for i, a := range it.Aff {
		v := a.V * CraftBonus
		if a.V >= 1 {
			v = math.Ceil(v)
		} else {
			v = round3(v)
		}
		out[i] = Affix{a.Stat, v}
	}
	return out
}

// Name : le nom de l'objet.
func (it *Item) Name() string {
	b := it.B()
	if b == nil {
		return "Objet oublié"
	}
	if u := Uniques[it.Uniq]; u != nil {
		return u.Name
	}
	if b.Kind == KRecette {
		if r := Recipes[it.Rec]; r != nil {
			return b.Name + " : " + r.Name()
		}
		return b.Name
	}
	if b.Kind != KEquipement || len(it.Aff) == 0 {
		return b.Name
	}
	// Le bonus le plus fort (en proportion) donne son suffixe.
	best, bestV := "", -1.0
	for _, a := range it.Aff {
		if r := a.V / AffixValue(a.Stat, it.Lv); r > bestV {
			best, bestV = a.Stat, r
		}
	}
	return b.Name + " " + affixSuffixes[best]
}

// Value : la valeur d'un objet (Aelis en rachète un quart).
func (it *Item) Value() int {
	b := it.B()
	if b == nil {
		return 0
	}
	if b.Kind != KEquipement {
		return b.Value * it.Count()
	}
	l := float64(it.Lv)
	return int(math.Round((20 + 6*l*l) * rareteValeur[it.R]))
}

// SellPrice : ce qu'Aelis en donne.
func (it *Item) SellPrice() int { return max(1, it.Value()/4) }

// RepairPrice : ce que Doran demande pour remettre l'objet à neuf.
func (it *Item) RepairPrice() int {
	b := it.B()
	if b == nil || b.Kind != KEquipement || it.Dur >= 100 {
		return 0
	}
	return max(1, int(math.Round(float64(it.Value())*0.2*float64(100-it.Dur)/100)))
}

// UsableBy dit si une classe peut porter la pièce (type d'armure, arme de la classe).
func UsableBy(b *Base, cl string) bool {
	if b == nil || b.Kind != KEquipement {
		return false
	}
	if b.Armor >= 0 {
		return ClassArmor[cl] == b.Armor
	}
	switch b.Slot {
	case ArmeS:
		return ClassWeapon[cl] == b.Weapon
	case MainGauche:
		if oh, ok := ClassOffhand[cl]; ok {
			return oh == b.Weapon
		}
		return b.Weapon == "talisman"
	}
	return true
}

// Score : ce que vaut une pièce pour une classe (la note ⭐ du client).
func Score(it *Item, cl string) float64 {
	if it == nil || it.Broken() {
		return 0
	}
	b := it.B()
	if b == nil || b.Kind != KEquipement {
		return 0
	}
	l := float64(it.Lv)
	s := it.ArmorValue() / (0.048 * (40 + 12*l)) * 4
	if p := it.WeaponPower(); p > 0 {
		s += p / HeroWeapon(it.Lv) * 8
	}
	liked := map[string]bool{}
	for _, x := range classPool(cl) {
		liked[x] = true
	}
	for _, a := range it.Affixes() {
		w := 0.4
		if liked[a.Stat] {
			w = 1
		}
		s += w * a.V / AffixValue(a.Stat, max(1, it.Lv))
	}
	s += 2 * float64(len(it.Leg))
	return math.Round(s*100) / 100
}

// Stars : une note de 1 à 5 étoiles pour la classe.
func Stars(it *Item, cl string) int {
	if it == nil || it.B() == nil || it.B().Kind != KEquipement {
		return 0
	}
	if !UsableBy(it.B(), cl) {
		return 0
	}
	sc := Score(it, cl)
	ref := 1.0 + 0.8*float64(rareteBonus[Rare]) // une pièce rare, bien tirée
	if it.B().Slot == ArmeS {
		ref += 8
	} else if it.B().Armor >= 0 {
		ref += slotArmure[it.B().Slot] * float64(ClassArmor[cl]+1)
	}
	return max(1, min(5, int(math.Round(sc/ref*3))))
}

// Gear : ce que l'équipement porté ajoute au héros.
type Gear struct {
	Attrs  Attrs
	Bonus  Bonus
	HP     int
	Armor  float64
	Weapon float64 // 0 : pas d'arme
	Res    [nElements]float64
	Legend []LegendEffect
}

// SumGear additionne l'équipement porté.
func SumGear(eq *[NSlots]*Item) Gear {
	var g Gear
	if eq == nil {
		return g
	}
	for _, it := range eq {
		if it == nil || it.Broken() {
			continue
		}
		g.Armor += it.ArmorValue()
		if p := it.WeaponPower(); p > 0 {
			g.Weapon = p
		}
		for _, a := range it.Affixes() {
			g.add(a)
		}
		g.Legend = append(g.Legend, it.Leg...)
	}
	return g
}

func (g *Gear) add(a Affix) {
	for i, n := range AttrNames {
		if n == a.Stat {
			g.Attrs[i] += int(a.V)
			return
		}
	}
	switch a.Stat {
	case "vie":
		g.HP += int(a.V)
	case "crit":
		g.Bonus.Crit += a.V
	case "vol":
		g.Bonus.Lifelink += a.V
	case "vit":
		g.Bonus.Speed += a.V
	case "degats":
		g.Bonus.Dmg += a.V
	case "soins":
		g.Bonus.Heal += a.V
	default:
		if strings.HasPrefix(a.Stat, "res_") {
			if e, ok := ElementByID(strings.TrimPrefix(a.Stat, "res_")); ok {
				g.Res[e] += a.V
			}
		}
	}
}

// ArmorReduction : la part des dégâts physiques que l'armure arrête.
func ArmorReduction(armor float64, level int) float64 {
	if armor <= 0 {
		return 0
	}
	return math.Min(0.5, armor/(armor+40+12*float64(level)))
}

// ItemView : ce que le client voit d'un objet.
type ItemView struct {
	ID     string         `json:"id"` // en texte : les nombres de 62 bits ne tiennent pas dans un double
	Base   string         `json:"b"`
	Name   string         `json:"nom"`
	Icon   string         `json:"ic"`
	Kind   string         `json:"k"`
	Slot   string         `json:"sl,omitempty"`
	Armor  string         `json:"ar,omitempty"`
	Weapon string         `json:"w,omitempty"`
	Lv     int            `json:"lv,omitempty"`
	R      int            `json:"r"`
	Aff    [][]any        `json:"a,omitempty"` // [nom, valeur affichée]
	Leg    []string       `json:"lg,omitempty"` // les effets des légendaires, en phrases
	ArmorV float64        `json:"arm,omitempty"`
	Power  float64        `json:"pw,omitempty"`
	Dur    int            `json:"d"`
	Lie    bool           `json:"l,omitempty"`
	Fab    string         `json:"f,omitempty"`
	N      int            `json:"n,omitempty"`
	Sell   int            `json:"px"`
	Repair int            `json:"rep,omitempty"`
	Use    bool           `json:"u,omitempty"`
	Usable bool           `json:"ok"` // le héros peut le porter (type d'armure, arme)
	Stars  int            `json:"st,omitempty"`
	Desc   string         `json:"ds,omitempty"`
	Look   string         `json:"lk,omitempty"`
	Score  float64        `json:"sc,omitempty"`
}

// View : l'objet tel que le héros de classe cl le voit.
func (it *Item) View(cl string) ItemView {
	b := it.B()
	v := ItemView{ID: fmt.Sprint(it.ID), Base: it.Base, Name: it.Name(), R: it.R, Lv: it.Lv, Dur: it.Dur, Lie: it.Lie,
		Fab: it.Fab, Sell: it.SellPrice(), Look: it.Look}
	if b == nil {
		return v
	}
	v.Icon, v.Desc = b.Icon, b.Desc
	if u := Uniques[it.Uniq]; u != nil {
		v.Desc = u.Lore
	}
	switch b.Kind {
	case KEquipement:
		v.Kind = "equipement"
		v.Slot = SlotIDs[b.Slot]
		if b.Slot == Anneau1 {
			v.Slot = "anneau"
		}
		if b.Armor >= 0 {
			v.Armor = ArmureIDs[b.Armor]
		}
		v.Weapon = b.Weapon
		v.ArmorV = it.ArmorValue()
		v.Power = it.WeaponPower()
		for _, a := range it.Affixes() {
			v.Aff = append(v.Aff, []any{a.Stat, a.V, AffixText(a)})
		}
		for _, e := range it.Leg {
			name := "Compétence " + strconv.Itoa(e.Skill)
			if c := ClassByID(cl); c != nil && e.Skill >= 1 && e.Skill <= 7 {
				name = c.Skills[e.Skill-1].Name
			}
			v.Leg = append(v.Leg, LegendText(e, name))
		}
		v.Repair = it.RepairPrice()
		v.Usable = UsableBy(b, cl)
		v.Stars = Stars(it, cl)
		v.Score = Score(it, cl)
	case KMateriau:
		v.Kind = "materiau"
		v.N = it.Count()
	case KConso:
		v.Kind = "provision"
		v.N = it.Count()
		v.Use = true
	case KRecette:
		v.Kind = "recette"
		v.Use = true
		v.N = it.Count()
	case KSac:
		v.Kind = "sac"
		v.Use = true
		v.N = it.Count()
	}
	return v
}

const nElements = 7

// ElementByID retrouve un élément par son nom.
func ElementByID(id string) (Element, bool) {
	for i, n := range ElementNames {
		if n == id {
			return Element(i), true
		}
	}
	return Physique, false
}
