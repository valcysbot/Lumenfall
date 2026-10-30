package world

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Les habitants (PNJ) : chacun a un lieu de travail, où il flâne le jour, et
// une maison, où il rentre dormir. Les noctambules font l'inverse. Le serveur
// ne connaît que leurs déplacements ; leurs tenues et leurs dialogues sont
// côté client.

// NPCPathNodes borne la recherche de trajet d'un habitant : ils traversent
// tout le village, bien plus loin qu'un clic de joueur.
const NPCPathNodes = 40000

// NPC est un habitant de la carte.
type NPC struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Work      [2]float64 `json:"work"`
	Home      [2]float64 `json:"home"`
	Wander    float64    `json:"wander"`   // rayon de flânerie autour du travail
	Nocturnal bool       `json:"nocturne"` // éveillé la nuit
	Running   bool       `json:"run"`      // toujours en train de courir (enfants)

	Body    Body `json:"-"`
	Visible bool `json:"-"` // dehors (ou à son poste), donc vu des joueurs

	rest int // secondes avant la prochaine flânerie
}

// Awake indique si l'habitant est debout à l'instant t.
func (n *NPC) Awake(t time.Time) bool { return n.Nocturnal == IsNight(t) }

// LoadNPCs charge les habitants d'une carte (maps/<id>.pnj.json) et vérifie
// que leurs lieux sont praticables et reliés entre eux. Pas de fichier : pas
// d'habitants.
func LoadNPCs(m *Map) ([]*NPC, error) {
	raw, err := mapFiles.ReadFile("maps/" + m.ID + ".pnj.json")
	if err != nil {
		return nil, nil
	}
	var f struct {
		PNJ []*NPC `json:"pnj"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("habitants de %q: %w", m.ID, err)
	}
	seen := map[string]bool{}
	for _, n := range f.PNJ {
		if n.ID == "" || seen[n.ID] {
			return nil, fmt.Errorf("habitants de %q: identifiant %q vide ou en double", m.ID, n.ID)
		}
		seen[n.ID] = true
		if !m.Free(n.Work[0], n.Work[1]) || !m.Free(n.Home[0], n.Home[1]) {
			return nil, fmt.Errorf("habitants de %q: %s travaille ou dort dans un mur", m.ID, n.ID)
		}
		if m.FindPath(n.Home[0], n.Home[1], n.Work[0], n.Work[1], NPCPathNodes) == nil {
			return nil, fmt.Errorf("habitants de %q: %s ne peut pas aller de chez lui au travail", m.ID, n.ID)
		}
		n.Body = Body{X: n.Home[0], Y: n.Home[1], Dir: 2, Run: n.Running}
	}
	return f.PNJ, nil
}

// Start place l'habitant au démarrage du monde : à son poste s'il est debout,
// chez lui sinon.
func (n *NPC) Start(now time.Time) {
	if n.Awake(now) {
		n.Body.Teleport(n.Work[0], n.Work[1])
		n.Visible = true
	}
}

// NPC events renvoyés par Think.
const (
	NPCNothing = iota
	NPCAppear  // sort de chez lui : les joueurs doivent le voir arriver
	NPCVanish  // rentré chez lui : il disparaît
)

// Think décide de ce que fait l'habitant ; à appeler une fois par seconde.
func (n *NPC) Think(m *Map, now time.Time, rng *rand.Rand) int {
	if n.Awake(now) {
		if !n.Visible {
			n.Visible = true
			n.Body.Teleport(n.Home[0], n.Home[1])
			n.goTo(m, n.Work[0], n.Work[1])
			return NPCAppear
		}
		if n.Body.Moving() {
			return NPCNothing
		}
		if n.rest > 0 {
			n.rest--
			return NPCNothing
		}
		n.rest = 2 + rng.Intn(7)
		if math.Hypot(n.Body.X-n.Work[0], n.Body.Y-n.Work[1]) > n.Wander+1 {
			n.goTo(m, n.Work[0], n.Work[1])
			return NPCNothing
		}
		for try := 0; try < 6; try++ {
			a, r := rng.Float64()*2*math.Pi, math.Sqrt(rng.Float64())*n.Wander
			x, y := n.Work[0]+math.Cos(a)*r, n.Work[1]+math.Sin(a)*r
			if m.Free(x, y) && n.goTo(m, x, y) {
				break
			}
		}
		return NPCNothing
	}
	if !n.Visible {
		return NPCNothing
	}
	if math.Hypot(n.Body.X-n.Home[0], n.Body.Y-n.Home[1]) < 0.8 {
		n.Visible = false
		n.Body.SetPath(nil)
		return NPCVanish
	}
	if !n.Body.Moving() && !n.goTo(m, n.Home[0], n.Home[1]) {
		// Coincé : il rentre quand même, à l'abri des regards.
		n.Body.Teleport(n.Home[0], n.Home[1])
	}
	return NPCNothing
}

func (n *NPC) goTo(m *Map, x, y float64) bool {
	p := m.FindPath(n.Body.X, n.Body.Y, x, y, NPCPathNodes)
	if p == nil {
		return false
	}
	n.Body.SetPath(p)
	return true
}
