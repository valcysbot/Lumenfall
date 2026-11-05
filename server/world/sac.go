package world

import (
	"errors"
	"math/rand"
	"strconv"
)

// Le sac et l'équipement d'un héros (Acte V). Tout est enregistré en base
// dans une seule colonne JSON (lf_heroes.inventaire).

// Réglages du sac.
const (
	SacDepart    = 30 // places au départ
	SacMax       = 90 // jusqu'à 90 avec des sacs plus grands
	DurPerDeath  = 10 // usure de chaque pièce portée à chaque mort
	FreePotions  = 3  // Sœur Clémence remplit 3 potions gratuites
	MaxRepasMin  = 30 // un plat dure 30 minutes
	RestRegenMul = 3.0
)

// Usage : ce que fait un objet qu'on utilise.
type Usage struct {
	Heal    float64 // potion : part de la vie maximale
	Buff    string  // « repas » ou « elixir » : un seul de chaque à la fois
	Minutes float64 // durée du bonus
	Aff     []Affix // ce que donne le bonus
	Recall  bool    // parchemin de retour au temple
	Cast    float64 // secondes à attendre (le retour)
}

// Buff : un repas ou un élixir en cours.
type Buff struct {
	Base string  `json:"b"`
	Left float64 `json:"t"` // secondes restantes (le temps passe seulement en jeu)
}

// Metier : un niveau de métier.
type Metier struct {
	Lv int `json:"lv"`
	XP int `json:"xp"`
}

// Inventaire : tout ce qu'un héros possède.
type Inventaire struct {
	Sac        []*Item            `json:"sac"`
	Eq         [NSlots]*Item      `json:"eq"`
	Metiers    map[string]*Metier `json:"me"`
	Artisanats []string           `json:"art"` // les 2 métiers d'artisanat choisis (la cuisine est pour tous)
	Recettes   map[string]bool    `json:"rec"`
	Apparences map[string]bool    `json:"app"` // « base:rareté » des pièces déjà portées
	Rapides    [2]string          `json:"rap"` // les 2 emplacements rapides : une base
	Potion     string             `json:"po"`  // la potion de la touche R (vide : celle de Clémence)
	Buffs      map[string]*Buff   `json:"bu"`  // « repas », « elixir »
	Rachat     []*Item            `json:"-"`   // ce qu'on a vendu à Aelis (on peut le racheter)
	dirty      bool
}

// NewInventaire : un sac vide de 30 places.
func NewInventaire() *Inventaire {
	inv := &Inventaire{Sac: make([]*Item, SacDepart)}
	inv.fix()
	return inv
}

// fix complète ce qui manque (données anciennes ou abîmées).
func (inv *Inventaire) fix() {
	if len(inv.Sac) < SacDepart {
		inv.Sac = append(inv.Sac, make([]*Item, SacDepart-len(inv.Sac))...)
	}
	if len(inv.Sac) > SacMax {
		inv.Sac = inv.Sac[:SacMax]
	}
	if inv.Metiers == nil {
		inv.Metiers = map[string]*Metier{}
	}
	for _, id := range MetierIDs {
		if inv.Metiers[id] == nil {
			inv.Metiers[id] = &Metier{Lv: 0}
		}
		if Metiers[id].Recolte && inv.Metiers[id].Lv < 1 {
			inv.Metiers[id].Lv = 1 // toutes les récoltes, pour tous
		}
	}
	if inv.Metiers["cuisine"].Lv < 1 {
		inv.Metiers["cuisine"].Lv = 1 // la cuisine, pour tous
	}
	if inv.Recettes == nil {
		inv.Recettes = map[string]bool{}
	}
	if inv.Apparences == nil {
		inv.Apparences = map[string]bool{}
	}
	if inv.Buffs == nil {
		inv.Buffs = map[string]*Buff{}
	}
	for i, it := range inv.Sac {
		if it != nil && it.B() == nil {
			inv.Sac[i] = nil
		}
	}
	for s, it := range inv.Eq {
		if it != nil && it.B() == nil {
			inv.Eq[s] = nil
		}
	}
	// Les recettes de départ des métiers connus.
	for _, id := range append([]string{"cuisine"}, inv.Artisanats...) {
		for _, r := range RecipesOf(id) {
			if r.Start {
				inv.Recettes[r.ID] = true
			}
		}
	}
}

// Fix : à appeler après lecture en base.
func (inv *Inventaire) Fix() { inv.fix() }

// Dirty dit si le sac a changé depuis le dernier envoi (et l'oublie).
func (inv *Inventaire) TakeDirty() bool {
	d := inv.dirty
	inv.dirty = false
	return d
}

// Touch : le sac a changé.
func (inv *Inventaire) Touch() { inv.dirty = true }

// Free : le nombre de places libres.
func (inv *Inventaire) Free() int {
	n := 0
	for _, it := range inv.Sac {
		if it == nil {
			n++
		}
	}
	return n
}

// Room dit combien d'objets de cette base le sac peut encore prendre.
func (inv *Inventaire) Room(base string) int {
	b := Bases[base]
	if b == nil {
		return 0
	}
	n := 0
	for _, it := range inv.Sac {
		if it == nil {
			n += b.Stack
		} else if b.Stack > 1 && it.Base == base {
			n += b.Stack - it.Count()
		}
	}
	return n
}

// Add met un objet dans le sac (en complétant les piles). Il renvoie ce qui
// n'a pas tenu (nil : tout est rangé).
func (inv *Inventaire) Add(it *Item) *Item {
	if it == nil {
		return nil
	}
	b := it.B()
	if b == nil {
		return nil
	}
	inv.dirty = true
	if b.Stack > 1 {
		n := it.Count()
		for _, o := range inv.Sac {
			if n == 0 {
				break
			}
			if o != nil && o.Base == it.Base && o.Count() < b.Stack {
				k := min(n, b.Stack-o.Count())
				o.N = o.Count() + k
				n -= k
			}
		}
		if n == 0 {
			return nil
		}
		it.N = n
	}
	for i, o := range inv.Sac {
		if o == nil {
			if b.Stack > 1 && it.Count() > b.Stack {
				rest := *it
				rest.N = it.Count() - b.Stack
				it.N = b.Stack
				inv.Sac[i] = it
				it = &rest
				continue
			}
			inv.Sac[i] = it
			return nil
		}
	}
	return it
}

// Count : combien d'objets de cette base dans le sac.
func (inv *Inventaire) Count(base string) int {
	n := 0
	for _, it := range inv.Sac {
		if it != nil && it.Base == base {
			n += it.Count()
		}
	}
	return n
}

// Take retire n objets de cette base du sac (tout ou rien).
func (inv *Inventaire) Take(base string, n int) bool {
	if inv.Count(base) < n {
		return false
	}
	for i := len(inv.Sac) - 1; i >= 0 && n > 0; i-- {
		it := inv.Sac[i]
		if it == nil || it.Base != base {
			continue
		}
		k := min(n, it.Count())
		n -= k
		if it.Count() == k {
			inv.Sac[i] = nil
		} else {
			it.N = it.Count() - k
		}
	}
	inv.dirty = true
	return true
}

// At : l'objet de la case i (nil si vide ou hors du sac).
func (inv *Inventaire) At(i int) *Item {
	if i < 0 || i >= len(inv.Sac) {
		return nil
	}
	return inv.Sac[i]
}

// Find : la case d'un objet, par son numéro (-1 : pas dans le sac).
func (inv *Inventaire) Find(id int64) int {
	for i, it := range inv.Sac {
		if it != nil && it.ID == id {
			return i
		}
	}
	return -1
}

// ParseID lit un numéro d'objet envoyé en texte par le client.
func ParseID(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// Move range un objet d'une case à une autre (échange, ou complète une pile).
func (inv *Inventaire) Move(from, to int) {
	a, b := inv.At(from), inv.At(to)
	if a == nil || to < 0 || to >= len(inv.Sac) || from == to {
		return
	}
	inv.dirty = true
	if b != nil && b.Base == a.Base && a.B().Stack > 1 {
		k := min(a.Count(), a.B().Stack-b.Count())
		b.N = b.Count() + k
		if a.Count() == k {
			inv.Sac[from] = nil
		} else {
			a.N = a.Count() - k
		}
		return
	}
	inv.Sac[from], inv.Sac[to] = b, a
}

// Sort range le sac : équipement par rareté, puis le reste.
func (inv *Inventaire) Sort() {
	var items []*Item
	for _, it := range inv.Sac {
		if it != nil {
			items = append(items, it)
		}
	}
	key := func(it *Item) (int, int, string) {
		b := it.B()
		return int(b.Kind), -it.R, b.ID
	}
	for i := 1; i < len(items); i++ {
		for j := i; j > 0; j-- {
			ka, ra, ia := key(items[j])
			kb, rb, ib := key(items[j-1])
			if ka < kb || (ka == kb && (ra < rb || (ra == rb && ia < ib))) {
				items[j], items[j-1] = items[j-1], items[j]
			} else {
				break
			}
		}
	}
	for i := range inv.Sac {
		inv.Sac[i] = nil
	}
	copy(inv.Sac, items)
	inv.dirty = true
}

// Les erreurs du sac (montrées telles quelles au joueur).
var (
	ErrSacPlein    = errors.New("Ton sac est plein !")
	ErrPasLa       = errors.New("Cet objet n'est plus là.")
	ErrPasPortable = errors.New("Ta classe ne peut pas porter ça.")
)

// Equip porte l'objet de la case i. La pièce qu'il remplace retourne dans le sac.
func (inv *Inventaire) Equip(i int, cl string, level int, want Slot) error {
	it := inv.At(i)
	if it == nil {
		return ErrPasLa
	}
	b := it.B()
	if b.Kind != KEquipement {
		return errors.New("Ça ne se porte pas.")
	}
	if !UsableBy(b, cl) {
		return ErrPasPortable
	}
	if it.Lv > level {
		return errors.New("Il faut être niveau " + strconv.Itoa(it.Lv) + " pour porter ça.")
	}
	s := b.Slot
	if s == Anneau1 {
		s = Anneau1
		if want == Anneau2 || (want != Anneau1 && inv.Eq[Anneau1] != nil && inv.Eq[Anneau2] == nil) {
			s = Anneau2
		}
	}
	inv.Sac[i] = inv.Eq[s] // l'ancienne pièce prend sa place dans le sac
	inv.Eq[s] = it
	it.Lie = true // un objet porté devient lié
	inv.Apparences[it.Base+":"+strconv.Itoa(it.R)] = true
	inv.dirty = true
	return nil
}

// Unequip retire la pièce de l'emplacement s et la range dans le sac.
func (inv *Inventaire) Unequip(s Slot) error {
	if s < 0 || s >= NSlots || inv.Eq[s] == nil {
		return ErrPasLa
	}
	for i, o := range inv.Sac {
		if o == nil {
			inv.Sac[i] = inv.Eq[s]
			inv.Eq[s] = nil
			inv.dirty = true
			return nil
		}
	}
	return ErrSacPlein
}

// Wear : l'usure d'une mort sur tout ce qui est porté.
func (inv *Inventaire) Wear() {
	for _, it := range inv.Eq {
		if it != nil && it.Dur > 0 {
			it.Dur = max(0, it.Dur-DurPerDeath)
			inv.dirty = true
		}
	}
}

// Grow agrandit le sac.
func (inv *Inventaire) Grow(n int) error {
	if len(inv.Sac) >= SacMax {
		return errors.New("Ton sac est déjà le plus grand d'Erydor (90 places).")
	}
	n = min(n, SacMax-len(inv.Sac))
	inv.Sac = append(inv.Sac, make([]*Item, n)...)
	inv.dirty = true
	return nil
}

// Starter : l'équipement commun de départ d'un héros, de son niveau et de sa
// classe (Conseil des Objets : les héros de l'Acte IV le reçoivent aussi).
func (inv *Inventaire) Starter(rng *rand.Rand, cl string, level int) {
	a, ok := ClassArmor[cl]
	if !ok {
		a = Cuir
	}
	put := func(s Slot, base string) {
		if inv.Eq[s] == nil {
			if it := MakeEquip(rng, base, level, Commun, cl); it != nil {
				it.Lie = true
				inv.Eq[s] = it
				inv.Apparences[it.Base+":0"] = true
			}
		}
	}
	for _, s := range []Slot{Tete, Epaules, Torse, Mains, Jambes, Pieds} {
		put(s, SlotIDs[s]+"_"+ArmureIDs[a])
	}
	put(ArmeS, "arme_"+ClassWeapon[cl])
	if oh, ok := ClassOffhand[cl]; ok {
		put(MainGauche, "mg_"+oh)
	}
	if HasCape(cl) {
		put(Dos, "dos_cape")
	}
	inv.dirty = true
}

// HasCape : les classes qui portaient déjà une cape à l'Acte IV.
func HasCape(cl string) bool {
	switch cl {
	case "paladin", "chevalier_dragon", "necromancien", "barde", "voleur":
		return true
	}
	return false
}

// Look : ce que l'équipement donne à voir aux autres (« base:rareté » par
// emplacement visible, l'apparence copiée au miroir passant avant).
func (inv *Inventaire) Look() map[string]string {
	out := map[string]string{}
	for _, s := range []Slot{Tete, Epaules, Torse, Mains, Jambes, Pieds, Dos, ArmeS, MainGauche} {
		it := inv.Eq[s]
		if it == nil {
			continue
		}
		v := it.Base + ":" + strconv.Itoa(it.R)
		if it.Look != "" {
			v = it.Look
		}
		out[SlotIDs[s]] = v
	}
	return out
}
