class_name Telegraph
extends Node2D
## Une attaque annoncée au sol : une zone rouge qui se remplit jusqu'à
## l'impact. Cercle (piétinement, éclats gris) ou ligne (charge). Les
## coordonnées sont celles du monde, en cases ; on les projette en 3/4.

var line := false
var a := Vector2.ZERO     # centre du cercle, ou départ de la ligne
var b := Vector2.ZERO     # arrivée de la ligne
var r := 1.0              # rayon, ou demi-largeur
var total := 1.0          # secondes avant l'impact
var _t := 0.0


func _process(delta: float) -> void:
	_t += delta
	if _t > total + 0.25:
		queue_free()
	queue_redraw()


func _shape(k: float) -> PackedVector2Array:
	var pts := PackedVector2Array()
	if not line:
		for i in 40:
			var ang := TAU * i / 40.0
			pts.append(Iso.to_screen(a + Vector2(cos(ang), sin(ang)) * r * k))
		return pts
	var d := b - a
	var n := Vector2(-d.y, d.x).normalized() * r
	var e := a + d * k
	for p in [a + n, e + n, e - n, a - n]:
		pts.append(Iso.to_screen(p))
	return pts


func _draw() -> void:
	if line and a.distance_to(b) < 0.01:
		return
	var k := clampf(_t / maxf(0.01, total), 0, 1)
	if _t > total:
		# L'impact : un éclair blanc.
		draw_colored_polygon(_shape(1.0), Color(1, 0.9, 0.8, 0.55 * (1 - (_t - total) / 0.25)))
		return
	var pulse := 0.5 + 0.5 * sin(_t * 14)
	var outline := _shape(1.0)
	draw_colored_polygon(outline, Color(0.85, 0.12, 0.08, 0.16 + 0.06 * pulse))
	draw_colored_polygon(_shape(k), Color(0.95, 0.2, 0.1, 0.38))
	outline.append(outline[0])
	draw_polyline(outline, Color(1, 0.35, 0.2, 0.9), 2.0, true)
