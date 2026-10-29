package world

import "math"

// Vitesses d'un héros, en cases par seconde.
const (
	WalkSpeed = 4.0
	RunSpeed  = 6.5
)

// Body est la partie « physique » d'un héros dans la simulation.
// Le client n'envoie que des intentions (une direction ou une destination) :
// la position est toujours calculée ici, par le serveur.
type Body struct {
	X, Y float64
	// Dir est la dernière direction regardée, de 0 à 7 (0 = est, sens horaire
	// à l'écran en vue de dessus : 2 = sud, 4 = ouest, 6 = nord).
	Dir int
	// Run : le joueur court (touche Maj ou bouton de course).
	Run bool

	inputX, inputY float64      // direction demandée au clavier ou au joystick
	path           [][2]float64 // trajet demandé par un clic
}

// SetInput enregistre une direction (clavier, joystick). Une direction nulle
// arrête le héros. Une direction annule un trajet en cours.
func (b *Body) SetInput(dx, dy float64) {
	if math.IsNaN(dx) || math.IsNaN(dy) || math.IsInf(dx, 0) || math.IsInf(dy, 0) {
		dx, dy = 0, 0
	}
	if l := math.Hypot(dx, dy); l > 1 {
		dx, dy = dx/l, dy/l
	} else if l < 0.15 {
		dx, dy = 0, 0
	}
	b.inputX, b.inputY = dx, dy
	if dx != 0 || dy != 0 {
		b.path = nil
	}
}

// SetPath remplace le trajet suivi (clic sur la carte).
func (b *Body) SetPath(p [][2]float64) {
	b.path = p
	b.inputX, b.inputY = 0, 0
}

// Moving indique si le héros a une intention de déplacement.
func (b *Body) Moving() bool {
	return b.inputX != 0 || b.inputY != 0 || len(b.path) > 0
}

// Step fait avancer le héros de dt secondes sur la carte m. Renvoie vrai si
// sa position ou sa direction a changé.
func (b *Body) Step(m *Map, dt float64) bool {
	return b.StepAmong(m, nil, dt)
}

// StepAmong fait avancer le héros comme Step, sans traverser les autres
// héros de la foule (crowd peut être nil).
func (b *Body) StepAmong(m *Map, crowd *Crowd, dt float64) bool {
	ox, oy, od := b.X, b.Y, b.Dir
	budget := b.Speed() * dt
	if b.inputX != 0 || b.inputY != 0 {
		b.move(m, crowd, b.inputX*budget, b.inputY*budget)
	} else {
		for budget > 1e-9 && len(b.path) > 0 {
			tx, ty := b.path[0][0], b.path[0][1]
			dx, dy := tx-b.X, ty-b.Y
			d := math.Hypot(dx, dy)
			if d <= budget {
				px, py := b.X, b.Y
				b.move(m, crowd, dx, dy)
				budget -= d
				b.path = b.path[1:]
				if math.Hypot(b.X-px, b.Y-py) < d*0.5 {
					// Bloqué en chemin : on abandonne le trajet.
					b.path = nil
				}
			} else {
				b.move(m, crowd, dx/d*budget, dy/d*budget)
				budget = 0
			}
		}
	}
	return b.X != ox || b.Y != oy || b.Dir != od
}

// Speed renvoie la vitesse actuelle du héros.
func (b *Body) Speed() float64 {
	if b.Run {
		return RunSpeed
	}
	return WalkSpeed
}

// move applique un petit déplacement en glissant le long des obstacles.
func (b *Body) move(m *Map, crowd *Crowd, dx, dy float64) {
	if dx == 0 && dy == 0 {
		return
	}
	b.Dir = DirOf(dx, dy)
	// On découpe en petits pas pour ne jamais traverser un mur.
	n := int(math.Ceil(math.Hypot(dx, dy)/0.2)) + 1
	sx, sy := dx/float64(n), dy/float64(n)
	free := func(x, y float64) bool {
		return m.Free(x, y) && (crowd == nil || !crowd.Blocks(b, x, y))
	}
	for i := 0; i < n; i++ {
		if free(b.X+sx, b.Y+sy) {
			b.X += sx
			b.Y += sy
		} else if free(b.X+sx, b.Y) {
			b.X += sx
		} else if free(b.X, b.Y+sy) {
			b.Y += sy
		}
	}
}

// DirOf convertit un vecteur en l'une des 8 directions.
func DirOf(dx, dy float64) int {
	a := math.Atan2(dy, dx)
	d := int(math.Round(a/(math.Pi/4))) % 8
	if d < 0 {
		d += 8
	}
	return d
}
