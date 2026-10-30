package world

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
)

// Les monstres : les bêtes du val et les Effacés de la Lisière Grise. Chaque
// sorte a ses chiffres ; chaque camp (maps/<carte>.monstres.json) dit où
// vivent combien de bêtes, et de quel niveau.

// MonsterKind décrit une sorte de monstre.
type MonsterKind struct {
	ID       string
	Name     string
	HP, HPLv float64 // vie au niveau 0, et par niveau
	Dmg      float64 // dégâts au niveau 0
	DmgLv    float64
	Every    float64 // secondes entre deux attaques
	Reach    float64 // portée d'attaque, en cases
	Speed    float64 // vitesse de poursuite (la flânerie est deux fois plus lente)
	Aggro    float64 // distance à laquelle il remarque un héros
	Elem     Element
	Pack     bool    // chasse en meute : ses voisins du camp le suivent
	Flee     bool    // fuit les héros au lieu de se battre (le Doré)
	XPMul    float64 // multiplicateur d'expérience
	Gold     [2]int  // or lâché (min, max)
	Trophy   string  // trophée rare
	TrophyP  float64 // chance de trophée
	OnHit    Effect  // effet posé par ses morsures
	OnHitP   float64 // chance de le poser (0 : jamais)
	OnHitDur float64
	OnHitPow float64 // dégâts par seconde (poison), par niveau
	Elite    bool
	Radius   float64 // taille au sol (portée de corps à corps des héros)
	Lifetime float64 // secondes avant de disparaître tout seul (0 : jamais)
	Boss     bool    // a ses propres attaques annoncées au sol (boss.go)
	Respawn  float64 // secondes avant de revenir (0 : RespawnTime)
}

// MonsterKinds : le bestiaire de l'Acte III.
var MonsterKinds = map[string]*MonsterKind{
	"sanglier": {ID: "sanglier", Name: "Sanglier", HP: 40, HPLv: 18, Dmg: 4, DmgLv: 1.6, Every: 2.2, Reach: 1.5, Speed: 4.6,
		Aggro: 4, XPMul: 1, Gold: [2]int{1, 4}, Trophy: "Défense de sanglier", TrophyP: 0.06, Radius: 0.55},
	"loup": {ID: "loup", Name: "Loup", HP: 35, HPLv: 13, Dmg: 4.5, DmgLv: 1.5, Every: 1.8, Reach: 1.5, Speed: 5.0,
		Aggro: 5.5, Pack: true, XPMul: 1, Gold: [2]int{2, 6}, Trophy: "Croc de loup", TrophyP: 0.06, Radius: 0.5},
	"loup_efface": {ID: "loup_efface", Name: "Loup effacé", HP: 50, HPLv: 15, Dmg: 5, DmgLv: 1.5, Every: 1.8, Reach: 1.5, Speed: 5.2,
		Aggro: 6, Elem: Ombre, Pack: true, XPMul: 1.1, Gold: [2]int{4, 10}, Trophy: "Croc sans couleur", TrophyP: 0.05,
		OnHit: Ralenti, OnHitP: 0.3, OnHitDur: 3, Radius: 0.5},
	"corbeau_efface": {ID: "corbeau_efface", Name: "Corbeau effacé", HP: 35, HPLv: 12, Dmg: 5, DmgLv: 1.5, Every: 1.6, Reach: 1.6, Speed: 5.5,
		Aggro: 7, Elem: Ombre, XPMul: 1.1, Gold: [2]int{4, 10}, Trophy: "Plume grise", TrophyP: 0.05,
		OnHit: Poison, OnHitP: 0.35, OnHitDur: 6, OnHitPow: 0.6, Radius: 0.4},
	"sanglier_dore": {ID: "sanglier_dore", Name: "Sanglier doré", HP: 120, HPLv: 0, Dmg: 0, Every: 99, Reach: 0, Speed: 5.0,
		Aggro: 6, Flee: true, XPMul: 3, Gold: [2]int{50, 80}, Trophy: "Défense dorée", TrophyP: 1, Elite: true, Radius: 0.55,
		Lifetime: 180},
	// Le Grand Sanglier de Mabon : l'histoire de l'aubergiste était vraie.
	"grand_sanglier": {ID: "grand_sanglier", Name: "Grand Sanglier", HP: 4500, Dmg: 22, Every: 2.4, Reach: 2.4, Speed: 4.4,
		Aggro: 7, XPMul: 8, Gold: [2]int{60, 120}, Trophy: "Défense du Grand Sanglier", TrophyP: 1, Elite: true, Boss: true,
		Radius: 1.2, Respawn: 300},
}

// GoldenChance : la chance qu'un sanglier qui réapparaît soit le Doré.
const GoldenChance = 0.03

// Camp : un groupe de monstres d'une même sorte autour d'un point.
type Camp struct {
	Kind   string     `json:"sorte"`
	At     [2]float64 `json:"centre"`
	Radius float64    `json:"rayon"`
	Count  int        `json:"nombre"`
	Levels [2]int     `json:"niveaux"`
}

// Comportement des monstres.
const (
	LeashRadius  = 20.0 // au-delà, il abandonne et rentre chez lui
	RespawnTime  = 45.0 // secondes avant qu'un monstre tué revienne
	CorpseTime   = 3.0  // le corps reste un peu avant de disparaître
	PackCall     = 8.0  // la meute entend l'appel jusqu'ici
	NightAggro   = 3.0  // la nuit, les Effacés voient les torches de plus loin
	EvadeHealPct = 1.0
)

// Monster est un monstre vivant (ou en attente de réapparition).
type Monster struct {
	Kind   *MonsterKind
	Camp   int
	Home   [2]float64
	Golden bool // un Doré, apparu à la place d'un sanglier

	threat   map[int]float64
	target   int
	attackCD float64
	wander   float64
	evading  bool
	respawn  float64 // secondes avant de revenir, une fois mort
	gone     bool    // corps retiré, invisible
	age      float64
	repath   float64

	// Le boss.
	Phase    int
	chargeCD float64
	stompCD  float64
	shardCD  float64
	windup   float64 // il prépare une attaque annoncée : il ne bouge plus
}

// LoadCamps lit les camps de monstres d'une carte. Pas de fichier : pas de monstres.
func LoadCamps(m *Map) ([]Camp, error) {
	raw, err := mapFiles.ReadFile("maps/" + m.ID + ".monstres.json")
	if err != nil {
		return nil, nil
	}
	var f struct {
		Camps []Camp `json:"camps"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("monstres de %q: %w", m.ID, err)
	}
	for i, c := range f.Camps {
		if MonsterKinds[c.Kind] == nil {
			return nil, fmt.Errorf("monstres de %q: sorte inconnue %q", m.ID, c.Kind)
		}
		if !m.Free(c.At[0], c.At[1]) {
			return nil, fmt.Errorf("monstres de %q: le camp %d est dans un mur", m.ID, i+1)
		}
		if c.Count < 1 || c.Levels[0] < 1 || c.Levels[1] < c.Levels[0] {
			return nil, fmt.Errorf("monstres de %q: camp %d mal réglé", m.ID, i+1)
		}
	}
	return f.Camps, nil
}

// spotNear trouve une case libre au hasard près d'un point.
func spotNear(m *Map, x, y, r float64, rng *rand.Rand) (float64, float64) {
	for try := 0; try < 20; try++ {
		a, d := rng.Float64()*2*math.Pi, math.Sqrt(rng.Float64())*r
		px, py := x+math.Cos(a)*d, y+math.Sin(a)*d
		if m.Free(px, py) {
			return px, py
		}
	}
	return x, y
}

// monsterStats règle la vie d'un monstre selon sa sorte et son niveau.
func monsterStats(k *MonsterKind, lv int) int {
	return int(math.Round(k.HP + k.HPLv*float64(lv)))
}

func monsterDmg(k *MonsterKind, lv int) float64 { return k.Dmg + k.DmgLv*float64(lv) }
