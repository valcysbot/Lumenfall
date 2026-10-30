package world

import "math"

// Le combat (Acte III) : éléments, effets d'état, points de vie, expérience.
// Tout se calcule ici, sur le serveur ; le client n'envoie que des intentions
// (« je vise ce loup », « compétence 2 », « roulade ») et affiche le résultat.

// Element : les 6 éléments d'Erydor, plus le physique (sans élément).
type Element int

const (
	Physique Element = iota
	Feu
	Eau
	Terre
	Air
	Lumiere
	Ombre
)

var ElementNames = [...]string{"physique", "feu", "eau", "terre", "air", "lumiere", "ombre"}

// strongAgainst : la roue des éléments. Feu > air > terre > eau > feu, et la
// lumière et l'ombre se blessent l'une l'autre.
var strongAgainst = map[Element]Element{Feu: Air, Air: Terre, Terre: Eau, Eau: Feu, Lumiere: Ombre, Ombre: Lumiere}

// ElementBonus : +50 % quand l'attaque frappe la faiblesse de la cible.
const ElementBonus = 1.5

// ElementMul renvoie le multiplicateur d'une attaque d'élément atk contre une
// cible d'élément def.
func ElementMul(atk, def Element) float64 {
	if w, ok := strongAgainst[atk]; ok && w == def {
		return ElementBonus
	}
	return 1
}

// Effect : les effets d'état. Les 4 premiers sont ceux du Conseil des Armes ;
// les autres sont des bonus et malus propres au Paladin et à la mort.
type Effect int

const (
	Brulure Effect = iota
	Poison
	Etourdi
	Ralenti
	Faiblesse // après la mort : −25 % de dégâts infligés
	Aura      // Aura de Solenn : −15 % de dégâts reçus, +10 % infligés
	nEffects
)

var EffectNames = [...]string{"brulure", "poison", "etourdi", "ralenti", "faiblesse", "aura"}

// Durées et forces des effets.
const (
	FaiblesseTime = 120.0 // secondes
	FaiblesseMul  = 0.75
	AuraTakenMul  = 0.85
	AuraDealtMul  = 1.10
	RalentiMul    = 0.6
)

type effect struct {
	left  float64 // secondes restantes
	tick  float64 // prochaine brûlure ou piqûre de poison
	power int     // dégâts par seconde (brûlure, poison)
	src   int     // qui l'a posé (pour la menace et l'expérience)
}

// Fighter est tout ce qui se bat : un héros ou un monstre.
type Fighter struct {
	SID   int
	Name  string
	Body  *Body
	Level int
	HP    int
	MaxHP int
	Elem  Element
	Dead  bool

	Hero *HeroState // nil pour un monstre
	Mon  *Monster   // nil pour un héros

	fx    [nEffects]effect
	dirty bool // points de vie ou effets changés : à renvoyer aux joueurs
}

// Has indique si l'effet est actif.
func (f *Fighter) Has(e Effect) bool { return f.fx[e].left > 0 }

// FxBits résume les effets actifs en un nombre (1 bit par effet), pour le client.
func (f *Fighter) FxBits() int {
	n := 0
	for e := Effect(0); e < nEffects; e++ {
		if f.Has(e) {
			n |= 1 << e
		}
	}
	return n
}

// Bits : les effets, plus le bouclier levé (bit 6), l'incantation (bit 7)
// et la phase 2 d'un boss (bit 8).
func (f *Fighter) Bits() int {
	n := f.FxBits()
	if f.Hero != nil && f.Hero.Blocking {
		n |= 1 << 6
	}
	if f.Hero != nil && f.Hero.Cast != 0 {
		n |= 1 << 7
	}
	if f.Mon != nil && f.Mon.Phase == 2 {
		n |= 1 << 8
	}
	return n
}

// Apply pose (ou renouvelle) un effet.
func (f *Fighter) Apply(e Effect, dur float64, power, src int) {
	if f.Dead {
		return
	}
	cur := &f.fx[e]
	if cur.left <= 0 {
		cur.tick = 1
	}
	cur.left = math.Max(cur.left, dur)
	if power > cur.power || cur.left <= dur {
		cur.power = power
	}
	cur.src = src
	f.dirty = true
}

// Clear retire tous les effets.
func (f *Fighter) Clear() {
	f.fx = [nEffects]effect{}
	f.dirty = true
}

// Pct renvoie la part de vie restante, de 0 à 1.
func (f *Fighter) Pct() float64 {
	if f.MaxHP <= 0 {
		return 0
	}
	return float64(f.HP) / float64(f.MaxHP)
}

// Dirty dit si la vie ou les effets ont changé depuis le dernier envoi, et remet le drapeau à zéro.
func (f *Fighter) Dirty() bool {
	d := f.dirty
	f.dirty = false
	return d
}

// --- Progression ----------------------------------------------------------------

// XPForLevel : l'expérience à gagner pour passer du niveau l au suivant.
// Pas de niveau maximum : la courbe continue.
func XPForLevel(l int) int { return 50 * l * (l + 1) }

// HeroMaxHP : la vie d'un héros selon son niveau.
func HeroMaxHP(l int) int { return 110 + 22*(l-1) }

// HeroWeapon : les dégâts de base de l'arme d'un héros selon son niveau.
func HeroWeapon(l int) float64 { return 7 + 2.2*float64(l) }

// KillXP : l'expérience d'un monstre de niveau ml pour un héros de niveau hl.
// Un monstre bien plus faible que soi ne rapporte presque plus rien.
func KillXP(ml, hl int, mul float64) int {
	xp := float64(15+10*ml) * mul
	if d := hl - ml; d > 2 {
		xp *= math.Max(0.1, 1-0.2*float64(d-2))
	}
	return int(math.Round(xp))
}
