package world

import (
	"errors"
	"sort"
)

// L'intendance de bravoure (Acte VI) : au Cercle des Braves, les Insignes
// gagnés dans les combats s'échangent contre des titres, et contre des coffrets
// d'équipement aussi forts que le butin (choix d'Elowan). Les teintures
// viendront avec les maisons (Acte VIII).

// Article : ce que l'intendance propose.
type Article struct {
	ID    string `json:"id"`
	Nom   string `json:"nom"`
	Desc  string `json:"ds"`
	Ic    string `json:"ic"`
	Prix  int    `json:"px"` // en Insignes
	Titre string `json:"ti"` // un titre (identifiant), ou ""
	Slots []Slot `json:"-"`  // un coffret : les emplacements possibles
}

// Bravoure : la liste, dans l'ordre affiché.
var Bravoure = []Article{
	{ID: "titre_brave", Nom: "Titre « le Brave »", Ic: "🎖️", Prix: 150, Titre: "brave",
		Desc: "Pour qui a déjà goûté au sable du Cercle."},
	{ID: "titre_veteran", Nom: "Titre « Vétéran du Cercle »", Ic: "🏟️", Prix: 450, Titre: "veteran_cercle",
		Desc: "Des dizaines de combats, et toujours debout."},
	{ID: "titre_lame", Nom: "Titre « Lame d'Erydor »", Ic: "⚔️", Prix: 1200, Titre: "lame_erydor",
		Desc: "Le plus beau titre de l'intendance. On le murmure sur la place."},
	{ID: "coffret_arme", Nom: "Coffret de bravoure : une arme", Ic: "🗡️", Prix: 260, Slots: []Slot{ArmeS},
		Desc: "Une arme de ton niveau, pour ta classe, rare (épique dès le niveau 20). Liée à toi."},
	{ID: "coffret_main", Nom: "Coffret de bravoure : main gauche", Ic: "🛡️", Prix: 200, Slots: []Slot{MainGauche},
		Desc: "Un bouclier, un talisman ou une seconde arme de ton niveau. Liée à toi."},
	{ID: "coffret_armure", Nom: "Coffret de bravoure : une pièce d'armure", Ic: "🥋", Prix: 200,
		Slots: []Slot{Tete, Epaules, Torse, Mains, Jambes, Pieds, Dos},
		Desc:  "Une pièce d'armure de ton niveau, de ton type d'armure. Liée à toi."},
	{ID: "coffret_bijou", Nom: "Coffret de bravoure : un bijou", Ic: "💍", Prix: 220, Slots: []Slot{Cou, Anneau1},
		Desc: "Une amulette ou un anneau de ton niveau. Lié à toi."},
}

func init() {
	Titres["brave"] = "le Brave"
	Titres["veteran_cercle"] = "Vétéran du Cercle"
	Titres["lame_erydor"] = "Lame d'Erydor"
}

// Les erreurs de l'intendance.
var (
	ErrInsignes   = errors.New("Il te manque des Insignes de bravoure : gagne-les au Cercle !")
	ErrDejaTitre  = errors.New("Tu portes déjà ce titre.")
	ErrBravoureNA = errors.New("L'intendance ne vend pas ça.")
)

// BuyBravoure : sid échange ses Insignes contre un article.
func (a *Arena) BuyBravoure(sid int, id string) error {
	h, inv := a.heroInv(sid)
	if h == nil || h.Hero.Gloire == nil {
		return nil
	}
	g := h.Hero.Gloire
	var art *Article
	for i := range Bravoure {
		if Bravoure[i].ID == id {
			art = &Bravoure[i]
		}
	}
	switch {
	case art == nil:
		return ErrBravoureNA
	case art.Titre != "" && g.HasTitre(art.Titre):
		return ErrDejaTitre
	case g.Insignes < art.Prix:
		return ErrInsignes
	}
	if art.Titre != "" {
		g.Insignes -= art.Prix
		g.AddTitre(art.Titre)
		g.dirty = true
		a.emit(Event{Kind: EvGloire, Tgt: sid, Text: ""})
		return nil
	}
	it := a.coffret(h, art)
	if it == nil {
		return ErrBravoureNA
	}
	if inv.Add(it) != nil {
		return ErrSacPlein
	}
	g.Insignes -= art.Prix
	g.dirty = true
	inv.Touch()
	a.emit(Event{Kind: EvItem, Tgt: sid, Text: it.Name(), Value: it.R, Flags: 1})
	a.emit(Event{Kind: EvGloire, Tgt: sid, Text: ""})
	return nil
}

// coffret : une pièce tirée pour la classe du héros, à son niveau.
func (a *Arena) coffret(h *Fighter, art *Article) *Item {
	cl := h.Hero.Class.ID
	var pool []string
	for id, b := range Bases {
		if b.Kind != KEquipement || !UsableBy(b, cl) {
			continue
		}
		for _, s := range art.Slots {
			if b.Slot == s {
				pool = append(pool, id)
			}
		}
	}
	if len(pool) == 0 {
		return nil
	}
	sort.Strings(pool)
	r := Rare
	if h.Level >= 20 {
		r = Epique
	}
	it := MakeEquip(a.rng, pool[a.rng.Intn(len(pool))], h.Level, r, cl)
	if it != nil {
		it.Lie = true
	}
	return it
}
