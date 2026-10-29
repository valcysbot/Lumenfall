// Package world contient la simulation du monde de Lumenfall : cartes,
// collisions, recherche de chemin et déplacement des héros. Il ne dépend
// pas de Nakama, pour pouvoir être testé seul.
package world

import (
	"container/heap"
	"embed"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

//go:embed maps/*.json
var mapFiles embed.FS

// Map est une carte en cases. Les coordonnées du monde sont en cases :
// la case (x, y) couvre [x, x+1[ × [y, y+1[.
type Map struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Width    int        `json:"width"`
	Height   int        `json:"height"`
	Spawn    [2]float64 `json:"spawn"`
	Blocking string     `json:"blocking"`
	Rows     []string   `json:"rows"`
	Doors    []Door     `json:"doors"`

	blocked []bool
	doors   map[[2]int]*Door
}

// Door est une porte : marcher sur la case From emmène le héros au point To
// (par exemple, de la rue à l'intérieur de l'auberge).
type Door struct {
	From [2]int     `json:"from"`
	To   [2]float64 `json:"to"`
}

// LoadMap charge une carte embarquée par son identifiant.
func LoadMap(id string) (*Map, error) {
	raw, err := mapFiles.ReadFile("maps/" + id + ".json")
	if err != nil {
		return nil, fmt.Errorf("carte %q introuvable: %w", id, err)
	}
	return ParseMap(raw)
}

// ParseMap lit une carte au format JSON et vérifie sa cohérence.
func ParseMap(raw []byte) (*Map, error) {
	m := &Map{}
	if err := json.Unmarshal(raw, m); err != nil {
		return nil, err
	}
	if len(m.Rows) != m.Height {
		return nil, fmt.Errorf("carte %q: %d lignes au lieu de %d", m.ID, len(m.Rows), m.Height)
	}
	m.blocked = make([]bool, m.Width*m.Height)
	for y, row := range m.Rows {
		r := []rune(row)
		if len(r) != m.Width {
			return nil, fmt.Errorf("carte %q: ligne %d de largeur %d au lieu de %d", m.ID, y, len(r), m.Width)
		}
		for x, c := range r {
			m.blocked[y*m.Width+x] = strings.ContainsRune(m.Blocking, c)
		}
	}
	m.doors = map[[2]int]*Door{}
	for i := range m.Doors {
		d := &m.Doors[i]
		if m.BlockedCell(d.From[0], d.From[1]) || !m.Walkable(d.To[0], d.To[1]) {
			return nil, fmt.Errorf("carte %q: porte %d bloquée", m.ID, i+1)
		}
		m.doors[d.From] = d
	}
	if !m.Walkable(m.Spawn[0], m.Spawn[1]) {
		return nil, fmt.Errorf("carte %q: le point d'apparition est bloqué", m.ID)
	}
	return m, nil
}

// DoorAt renvoie la porte de la case où se trouve le point (x, y), ou nil.
func (m *Map) DoorAt(x, y float64) *Door {
	return m.doors[[2]int{int(math.Floor(x)), int(math.Floor(y))}]
}

// BlockedCell indique si la case (cx, cy) bloque le passage. Hors carte = bloqué.
func (m *Map) BlockedCell(cx, cy int) bool {
	if cx < 0 || cy < 0 || cx >= m.Width || cy >= m.Height {
		return true
	}
	return m.blocked[cy*m.Width+cx]
}

// Walkable indique si un point du monde est sur une case praticable.
func (m *Map) Walkable(x, y float64) bool {
	return !m.BlockedCell(int(math.Floor(x)), int(math.Floor(y)))
}

// HeroRadius est le rayon de collision d'un héros, en cases.
const HeroRadius = 0.3

// Free indique si un héros centré en (x, y) ne touche aucune case bloquante.
func (m *Map) Free(x, y float64) bool {
	r := HeroRadius
	for _, p := range [4][2]float64{{x - r, y - r}, {x + r, y - r}, {x - r, y + r}, {x + r, y + r}} {
		if !m.Walkable(p[0], p[1]) {
			return false
		}
	}
	return true
}

// FindPath cherche un chemin (A*, 8 directions, sans couper les coins)
// entre deux points et renvoie la liste des centres de cases à suivre,
// sans la case de départ. Renvoie nil si la destination est inaccessible
// ou si le chemin dépasse maxNodes cases explorées.
func (m *Map) FindPath(fx, fy, tx, ty float64, maxNodes int) [][2]float64 {
	sx, sy := int(math.Floor(fx)), int(math.Floor(fy))
	gx, gy := int(math.Floor(tx)), int(math.Floor(ty))
	if m.BlockedCell(gx, gy) || m.BlockedCell(sx, sy) {
		return nil
	}
	if sx == gx && sy == gy {
		return [][2]float64{{tx, ty}}
	}
	idx := func(x, y int) int { return y*m.Width + x }
	h := func(x, y int) float64 {
		dx, dy := math.Abs(float64(x-gx)), math.Abs(float64(y-gy))
		return dx + dy + (math.Sqrt2-2)*math.Min(dx, dy)
	}
	gScore := map[int]float64{idx(sx, sy): 0}
	from := map[int]int{}
	open := &nodeHeap{{x: sx, y: sy, f: h(sx, sy)}}
	closed := map[int]bool{}
	explored := 0
	for open.Len() > 0 {
		cur := heap.Pop(open).(node)
		ci := idx(cur.x, cur.y)
		if closed[ci] {
			continue
		}
		if cur.x == gx && cur.y == gy {
			var cells [][2]float64
			for i := ci; i != idx(sx, sy); i = from[i] {
				cells = append(cells, [2]float64{float64(i%m.Width) + 0.5, float64(i/m.Width) + 0.5})
			}
			for l, r := 0, len(cells)-1; l < r; l, r = l+1, r-1 {
				cells[l], cells[r] = cells[r], cells[l]
			}
			cells[len(cells)-1] = [2]float64{tx, ty}
			return m.smooth(fx, fy, cells)
		}
		closed[ci] = true
		explored++
		if explored > maxNodes {
			return nil
		}
		for _, d := range dirs8 {
			nx, ny := cur.x+d[0], cur.y+d[1]
			if m.BlockedCell(nx, ny) {
				continue
			}
			if d[0] != 0 && d[1] != 0 && (m.BlockedCell(cur.x+d[0], cur.y) || m.BlockedCell(cur.x, cur.y+d[1])) {
				continue
			}
			ni := idx(nx, ny)
			cost := 1.0
			if d[0] != 0 && d[1] != 0 {
				cost = math.Sqrt2
			}
			g := gScore[ci] + cost
			if old, ok := gScore[ni]; ok && g >= old {
				continue
			}
			gScore[ni] = g
			from[ni] = ci
			heap.Push(open, node{x: nx, y: ny, f: g + h(nx, ny)})
		}
	}
	return nil
}

// smooth retire les étapes intermédiaires quand une ligne droite libre
// permet de les sauter, pour des trajets plus naturels.
func (m *Map) smooth(fx, fy float64, pts [][2]float64) [][2]float64 {
	out := make([][2]float64, 0, len(pts))
	ax, ay := fx, fy
	i := 0
	for i < len(pts) {
		j := len(pts) - 1
		for j > i && !m.clearLine(ax, ay, pts[j][0], pts[j][1]) {
			j--
		}
		out = append(out, pts[j])
		ax, ay = pts[j][0], pts[j][1]
		i = j + 1
	}
	return out
}

// clearLine vérifie qu'un héros peut aller en ligne droite de a à b.
func (m *Map) clearLine(ax, ay, bx, by float64) bool {
	dist := math.Hypot(bx-ax, by-ay)
	steps := int(dist/0.1) + 1
	for s := 0; s <= steps; s++ {
		t := float64(s) / float64(steps)
		if !m.Free(ax+(bx-ax)*t, ay+(by-ay)*t) {
			return false
		}
	}
	return true
}

var dirs8 = [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}

type node struct {
	x, y int
	f    float64
}

type nodeHeap []node

func (h nodeHeap) Len() int            { return len(h) }
func (h nodeHeap) Less(i, j int) bool  { return h[i].f < h[j].f }
func (h nodeHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *nodeHeap) Push(x interface{}) { *h = append(*h, x.(node)) }
func (h *nodeHeap) Pop() interface{} {
	old := *h
	n := old[len(old)-1]
	*h = old[:len(old)-1]
	return n
}
