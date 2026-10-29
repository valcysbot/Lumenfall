package world

import "math"

// Separate écarte doucement les héros qui se chevauchent : chacun recule de
// la moitié du chevauchement, sans jamais entrer dans un obstacle. Les héros
// se bousculent donc sans pouvoir se traverser, mais personne ne peut en
// bloquer un autre contre un mur. Renvoie l'indice des héros déplacés.
func Separate(m *Map, bodies []*Body) []int {
	const minDist = 2 * HeroRadius
	// Rangement par case pour ne comparer que les voisins proches.
	grid := make(map[[2]int][]int, len(bodies))
	for i, b := range bodies {
		k := [2]int{int(math.Floor(b.X)), int(math.Floor(b.Y))}
		grid[k] = append(grid[k], i)
	}
	pushX := make([]float64, len(bodies))
	pushY := make([]float64, len(bodies))
	for i, a := range bodies {
		cx, cy := int(math.Floor(a.X)), int(math.Floor(a.Y))
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				for _, j := range grid[[2]int{cx + dx, cy + dy}] {
					if j <= i {
						continue
					}
					b := bodies[j]
					vx, vy := b.X-a.X, b.Y-a.Y
					d := math.Hypot(vx, vy)
					if d >= minDist {
						continue
					}
					push := (minDist - d) / 2
					if d < 1e-6 {
						// Exactement au même endroit : on choisit une direction stable.
						ang := float64(i*7+j*13) * 0.618
						vx, vy, d = math.Cos(ang), math.Sin(ang), 1
					}
					nx, ny := vx/d*push, vy/d*push
					pushX[i] -= nx
					pushY[i] -= ny
					pushX[j] += nx
					pushY[j] += ny
				}
			}
		}
	}
	var moved []int
	for i, b := range bodies {
		if pushX[i] == 0 && pushY[i] == 0 {
			continue
		}
		ox, oy := b.X, b.Y
		if m.Free(b.X+pushX[i], b.Y+pushY[i]) {
			b.X += pushX[i]
			b.Y += pushY[i]
		} else if m.Free(b.X+pushX[i], b.Y) {
			b.X += pushX[i]
		} else if m.Free(b.X, b.Y+pushY[i]) {
			b.Y += pushY[i]
		}
		if b.X != ox || b.Y != oy {
			moved = append(moved, i)
		}
	}
	return moved
}

// Crowd sait où sont les héros, pour qu'un héros en marche ne puisse pas
// en traverser un autre.
type Crowd struct {
	grid map[[2]int][]*Body
}

// NewCrowd range les héros par case.
func NewCrowd(bodies []*Body) *Crowd {
	c := &Crowd{grid: make(map[[2]int][]*Body, len(bodies))}
	for _, b := range bodies {
		k := [2]int{int(math.Floor(b.X)), int(math.Floor(b.Y))}
		c.grid[k] = append(c.grid[k], b)
	}
	return c
}

// Blocks indique si le héros self, en allant en (x, y), se rapprocherait
// d'un autre héros déjà trop proche. S'éloigner reste toujours permis, pour
// que deux héros collés puissent se séparer.
func (c *Crowd) Blocks(self *Body, x, y float64) bool {
	const minDist = 2 * HeroRadius
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			for _, o := range c.grid[[2]int{cx + dx, cy + dy}] {
				if o == self {
					continue
				}
				nd := math.Hypot(x-o.X, y-o.Y)
				if nd < minDist && nd < math.Hypot(self.X-o.X, self.Y-o.Y) {
					return true
				}
			}
		}
	}
	return false
}
