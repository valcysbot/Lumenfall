class_name Iso
## Conversion entre les coordonnées du monde (en cases) et l'écran (3/4 isométrique).

const TILE_W := 64.0
const TILE_H := 32.0


static func to_screen(p: Vector2) -> Vector2:
	return Vector2((p.x - p.y) * TILE_W / 2, (p.x + p.y) * TILE_H / 2)


static func to_world(s: Vector2) -> Vector2:
	var a := s.x / (TILE_W / 2)
	var b := s.y / (TILE_H / 2)
	return Vector2((a + b) / 2, (b - a) / 2)


## Direction d'écran (ex. clavier, joystick) vers direction du monde.
static func screen_dir_to_world(d: Vector2) -> Vector2:
	if d.length() < 0.01:
		return Vector2.ZERO
	var w := Vector2(d.x / 2 + d.y, d.y - d.x / 2)
	return w.normalized() * min(1.0, d.length())


## Direction du monde (0 à 7) vers vecteur d'écran unitaire.
static func dir_to_screen(d: int) -> Vector2:
	var a := d * PI / 4
	return to_screen(Vector2(cos(a), sin(a))).normalized()
