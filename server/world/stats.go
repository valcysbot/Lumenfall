package world

import (
	"errors"
	"math"
)

// Les 9 attributs (Acte IV) : 5 points à placer par niveau.

// Attr : un attribut.
type Attr int

const (
	Force Attr = iota
	Intelligence
	Agilite
	Vitesse
	Esquive
	Vitalite
	Chance
	Endurance
	Precision
	nAttrs
)

// Attrs : les points d'un héros, dans l'ordre des attributs.
type Attrs [nAttrs]int

var AttrNames = [...]string{"force", "intelligence", "agilite", "vitesse", "esquive", "vitalite", "chance", "endurance", "precision"}

// PointsPerLevel : points d'attributs gagnés à chaque niveau.
const PointsPerLevel = 5

// FreeResetLevel : jusqu'à ce niveau, on peut tout redistribuer gratuitement.
const FreeResetLevel = 10

// Points : le total des points placés.
func (at Attrs) Points() int {
	n := 0
	for _, v := range at {
		n += v
	}
	return n
}

// PointsAt : les points disponibles au niveau l.
func PointsAt(l int) int { return PointsPerLevel * (l - 1) }

// ValidateAttrs vérifie une nouvelle répartition, par rapport à l'ancienne.
func ValidateAttrs(old, nw Attrs, level int) error {
	for i, v := range nw {
		if v < 0 {
			return errors.New("Un attribut ne peut pas être négatif.")
		}
		if level > FreeResetLevel && v < old[i] {
			return errors.New("Après le niveau 10, on ne peut plus reprendre ses points.")
		}
	}
	if nw.Points() > PointsAt(level) {
		return errors.New("Tu n'as pas autant de points à placer.")
	}
	return nil
}

// Recommend : la « répartition conseillée » d'une classe pour n points.
func Recommend(c *Class, n int) Attrs {
	var out Attrs
	total := 0
	for _, w := range c.Attr {
		total += w
	}
	if total == 0 || n <= 0 {
		return out
	}
	// Méthode des plus forts restes : chaque point va à l'attribut le plus en retard.
	for p := 0; p < n; p++ {
		best, bestGap := 0, -1.0
		for i, w := range c.Attr {
			gap := float64(w)*float64(p+1)/float64(total) - float64(out[i])
			if gap > bestGap {
				best, bestGap = i, gap
			}
		}
		out[best]++
	}
	return out
}

// Race : ce qui compte d'un peuple pour le combat (petit bonus de la bible).
type Race struct {
	ID      string
	Faction string
	Bonus   Attrs
	Resist  Element // −10 % de dégâts reçus de cet élément (Physique : aucun)
}

func bonus(at Attr) Attrs {
	var b Attrs
	b[at] = 1
	return b
}

// Races : les 15 peuples d'Erydor.
var Races = map[string]Race{
	"humain":     {ID: "humain", Faction: "concorde", Bonus: bonus(Chance)},
	"nain":       {ID: "nain", Faction: "concorde", Resist: Terre},
	"halfelin":   {ID: "halfelin", Faction: "concorde", Bonus: bonus(Chance)},
	"gnome":      {ID: "gnome", Faction: "concorde", Bonus: bonus(Intelligence)},
	"golem":      {ID: "golem", Faction: "concorde", Bonus: bonus(Endurance)},
	"elfe":       {ID: "elfe", Faction: "pacte", Bonus: bonus(Precision)},
	"fee":        {ID: "fee", Faction: "pacte", Bonus: bonus(Vitesse)},
	"felin":      {ID: "felin", Faction: "pacte", Bonus: bonus(Esquive)},
	"homme_bete": {ID: "homme_bete", Faction: "pacte", Bonus: bonus(Vitalite)},
	"sirene":     {ID: "sirene", Faction: "pacte", Resist: Eau},
	"orque":      {ID: "orque", Faction: "ligue", Bonus: bonus(Force)},
	"drakeide":   {ID: "drakeide", Faction: "ligue", Resist: Feu},
	"revenant":   {ID: "revenant", Faction: "ligue", Resist: Terre},
	"elfe_noir":  {ID: "elfe_noir", Faction: "ligue", Bonus: bonus(Intelligence)},
	"lumin":      {ID: "lumin", Faction: "ligue", Resist: Ombre},
}

// Gods : les dieux qu'un héros peut servir (les six actifs et les trois Silencieux).
var Gods = []string{"solenn", "maelis", "brannoc", "caelia", "fergal", "seren", "aurel", "selis", "cernos"}

// ValidGod dit si le dieu existe.
func ValidGod(id string) bool {
	for _, g := range Gods {
		if g == id {
			return true
		}
	}
	return false
}

// Stats : ce que les attributs, le peuple et la voie donnent au héros.
type Stats struct {
	Phys, Mag, Heal   float64 // multiplicateurs de dégâts et de soins
	Crit, CritMul     float64
	Dodge             float64
	Taken             float64 // multiplicateur de dégâts reçus
	MaxHP             int
	Speed             float64 // multiplicateur de vitesse
	Threat, Pet       float64
	Shields, Lifelink float64
	Resist            Element
	Armor             float64            // l'armure portée (Acte V)
	Res               [nElements]float64 // résistances des objets, repas et élixirs
}

// ComputeStats calcule les statistiques d'un héros.
func ComputeStats(c *Class, spec int, race string, at Attrs, level int, b Bonus) Stats {
	r := Races[race]
	var t Attrs
	for i := range at {
		t[i] = at[i] + r.Bonus[i]
	}
	m := Mods{}
	if spec >= 0 && spec < 3 && c != nil {
		m = c.Specs[spec].Mods
	}
	one := func(v float64) float64 {
		if v == 0 {
			return 1
		}
		return v
	}
	f := func(a Attr) float64 { return float64(t[a]) }
	main := f(Force)
	if c != nil && c.Range > 0 && !c.Magic {
		main = f(Agilite) // les armes à distance (arc, arquebuse, fioles)
	}
	s := Stats{
		Phys:     (1 + 0.02*main + 0.005*f(Agilite)) * one(m.Dmg) * (1 + b.Dmg),
		Mag:      (1 + 0.02*f(Intelligence)) * one(m.Dmg) * (1 + b.Dmg),
		Heal:     (1 + 0.015*f(Intelligence)) * one(m.Heal) * (1 + b.Heal),
		Crit:     HeroCrit + 0.004*(f(Chance)+f(Precision)) + 0.002*f(Agilite) + m.Crit + b.Crit,
		CritMul:  CritMul + 0.01*f(Precision),
		Dodge:    math.Min(0.35, 0.004*f(Esquive)+m.Dodge+b.Dodge),
		Taken:    math.Max(0.6, 1-0.006*f(Endurance)) * one(m.Taken) * (1 - b.Taken),
		Speed:    1 + math.Min(0.25, 0.008*f(Vitesse)) + m.Speed + b.Speed,
		Threat:   one(m.Threat) * (1 + b.Threat),
		Pet:      one(m.Pet) * (1 + b.Pet),
		Shields:  one(m.Shields) * (1 + b.Shields),
		Lifelink: m.Lifelink + b.Lifelink,
		Resist:   r.Resist,
	}
	s.MaxHP = int(math.Round(float64(HeroMaxHP(level)+8*t[Vitalite]) * one(m.HP) * (1 + b.HP)))
	return s
}
