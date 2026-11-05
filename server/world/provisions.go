package world

// Les provisions (Acte V) : potions, plats, élixirs, parchemins et sacs.
// Sœur Clémence remplit 3 potions gratuites ; les autres s'achètent chez
// Aelis ou se fabriquent. Un plat et un élixir à la fois, 30 minutes chacun.

func conso(b *Base) {
	b.Kind, b.Armor = KConso, -1
	if b.Stack == 0 {
		b.Stack = 20
	}
	addBase(b)
}

func init() {
	conso(&Base{ID: "petite_potion", Name: "Petite potion de soin", Icon: "🧪", Price: 40, Value: 10,
		Desc: "Rend 25 % de ta vie.", Use: &Usage{Heal: 0.25}})
	conso(&Base{ID: "potion", Name: "Potion de soin", Icon: "🧪", Price: 120, Value: 30,
		Desc: "Rend 40 % de ta vie.", Use: &Usage{Heal: 0.4}})
	conso(&Base{ID: "grande_potion", Name: "Grande potion de soin", Icon: "🧪", Value: 70,
		Desc: "Rend 60 % de ta vie.", Use: &Usage{Heal: 0.6}})

	conso(&Base{ID: "brochette", Name: "Brochette de sanglier", Icon: "🍢", Value: 12,
		Desc: "Un plat : +40 de vie pendant 30 minutes.", Use: &Usage{Buff: "repas", Minutes: 30, Aff: []Affix{{"vie", 40}}}})
	conso(&Base{ID: "galette_sauge", Name: "Galette à la sauge", Icon: "🫓", Value: 8, Price: 60,
		Desc: "Un plat : +3 d'Endurance pendant 30 minutes.", Use: &Usage{Buff: "repas", Minutes: 30, Aff: []Affix{{"endurance", 3}}}})
	conso(&Base{ID: "ragout", Name: "Ragoût de la Chope", Icon: "🍲", Value: 25,
		Desc: "Un plat : +3 de Force, d'Intelligence et d'Agilité pendant 30 minutes.",
		Use:  &Usage{Buff: "repas", Minutes: 30, Aff: []Affix{{"force", 3}, {"intelligence", 3}, {"agilite", 3}}}})
	conso(&Base{ID: "tourte_chasseur", Name: "Tourte du chasseur", Icon: "🥧", Value: 45,
		Desc: "Un plat : +3 % de critique et +60 de vie pendant 30 minutes.",
		Use:  &Usage{Buff: "repas", Minutes: 30, Aff: []Affix{{"crit", 0.03}, {"vie", 60}}}})
	conso(&Base{ID: "infusion_lune", Name: "Infusion de lune", Icon: "🍵", Value: 45,
		Desc: "Un plat : +6 % de soins et +8 % de résistance à l'ombre pendant 30 minutes.",
		Use:  &Usage{Buff: "repas", Minutes: 30, Aff: []Affix{{"soins", 0.06}, {"res_ombre", 0.08}}}})

	conso(&Base{ID: "elixir_force", Name: "Élixir du sanglier", Icon: "🍷", Value: 40,
		Desc: "Un élixir : +5 de Force et d'Endurance pendant 30 minutes.",
		Use:  &Usage{Buff: "elixir", Minutes: 30, Aff: []Affix{{"force", 5}, {"endurance", 5}}}})
	conso(&Base{ID: "elixir_esprit", Name: "Élixir de clairvoyance", Icon: "🍸", Value: 40,
		Desc: "Un élixir : +5 d'Intelligence et de Précision pendant 30 minutes.",
		Use:  &Usage{Buff: "elixir", Minutes: 30, Aff: []Affix{{"intelligence", 5}, {"precision", 5}}}})
	conso(&Base{ID: "elixir_lisiere", Name: "Élixir de la Lisière", Icon: "🥃", Value: 90,
		Desc: "Un élixir : +15 % de résistance à l'ombre et +4 % de dégâts pendant 30 minutes.",
		Use:  &Usage{Buff: "elixir", Minutes: 30, Aff: []Affix{{"res_ombre", 0.15}, {"degats", 0.04}}}})

	conso(&Base{ID: "parchemin_retour", Name: "Parchemin de retour", Icon: "📜", Price: 150, Value: 30,
		Desc: "Après 5 secondes immobile, te ramène au temple de Solenn.", Use: &Usage{Recall: true, Cast: 5}})

	addBase(&Base{ID: "sac_toile", Name: "Sac de toile", Kind: KSac, Icon: "🎒", Armor: -1, Stack: 5, Places: 6, Value: 60, Price: 500,
		Desc: "À utiliser : +6 places dans ton sac, pour toujours."})
	addBase(&Base{ID: "sac_voyage", Name: "Sac de voyage", Kind: KSac, Icon: "🎒", Armor: -1, Stack: 5, Places: 10, Value: 150,
		Desc: "À utiliser : +10 places dans ton sac, pour toujours."})
	addBase(&Base{ID: "sac_aventurier", Name: "Sac d'aventurier", Kind: KSac, Icon: "🎒", Armor: -1, Stack: 5, Places: 14, Value: 400,
		Desc: "À utiliser : +14 places dans ton sac, pour toujours."})
}

// Shop : ce qu'Aelis vend, dans l'ordre de son étal.
var Shop = []string{"petite_potion", "potion", "galette_sauge", "parchemin_retour", "sac_toile",
	"fil", "flux", "fiole", "sel", "tanin"}

// BuffAffixes : ce que donnent le repas et l'élixir en cours.
func (inv *Inventaire) BuffAffixes() []Affix {
	var out []Affix
	for _, b := range inv.Buffs {
		if base := Bases[b.Base]; base != nil && base.Use != nil && b.Left > 0 {
			out = append(out, base.Use.Aff...)
		}
	}
	return out
}

// TickBuffs fait passer le temps des bonus. Il dit si l'un d'eux s'est terminé.
func (inv *Inventaire) TickBuffs(dt float64) bool {
	ended := false
	for k, b := range inv.Buffs {
		b.Left -= dt
		if b.Left <= 0 {
			delete(inv.Buffs, k)
			ended = true
			inv.dirty = true
		}
	}
	return ended
}
