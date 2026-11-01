class_name DuelRing
extends Node2D
## Le cercle d'un duel, tracé au sol autour d'un drapeau. En sortir, c'est
## abandonner. Les coordonnées sont celles du monde, en cases.

var center := Vector2.ZERO
var r := 12.0
var id := 0
var _t := 0.0


func _process(delta: float) -> void:
	_t += delta
	queue_redraw()


func _draw() -> void:
	var pts := PackedVector2Array()
	for i in 73:
		var a := TAU * i / 72.0
		pts.append(Iso.to_screen(center + Vector2(cos(a), sin(a)) * r))
	var k := 0.55 + 0.25 * sin(_t * 3)
	draw_polyline(pts, Color(UI.LUMEN, 0.25 * k), 9.0, true)
	draw_polyline(pts, Color(UI.LUMEN, k), 2.5, true)
	# Le drapeau planté au milieu.
	var base := Iso.to_screen(center)
	draw_line(base, base + Vector2(0, -46), Color("5a4632"), 3.0)
	var wave := sin(_t * 4) * 3
	draw_colored_polygon(PackedVector2Array([base + Vector2(1, -46), base + Vector2(26, -39 + wave), base + Vector2(1, -31)]), UI.EMBER)
	draw_circle(base + Vector2(0, -47), 2.5, UI.LUMEN)
