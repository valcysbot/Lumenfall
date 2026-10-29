class_name Prop
extends Node2D
## Décor provisoire de l'Acte I : arbre ou rocher dessiné simplement.

var kind := "T"
var seed := 0.0


func _draw() -> void:
	if kind == "T":
		_tree()
	else:
		_rock()


func _tree() -> void:
	var s := 0.85 + seed * 0.35
	draw_set_transform(Vector2.ZERO, 0, Vector2(1, 0.5))
	draw_circle(Vector2(4, 2), 22 * s, Color(0, 0, 0, 0.28))
	draw_set_transform(Vector2.ZERO, 0, Vector2.ONE)
	draw_rect(Rect2(-4 * s, -26 * s, 8 * s, 26 * s), Color("5a4030"))
	var greens := [Color("24492a"), Color("2d5a31"), Color("3a6e3a"), Color("4f8a46")]
	var h := 0.0 if seed < 0.5 else 0.06
	draw_circle(Vector2(0, -40) * s, 24 * s, greens[0].lerp(Color("1f3f3a"), h))
	draw_circle(Vector2(-9, -48) * s, 17 * s, greens[1])
	draw_circle(Vector2(8, -50) * s, 16 * s, greens[2].lerp(greens[1], seed))
	draw_circle(Vector2(-2, -60) * s, 13 * s, greens[2])
	draw_circle(Vector2(-6, -64) * s, 6 * s, Color(greens[3], 0.7))


func _rock() -> void:
	draw_set_transform(Vector2.ZERO, 0, Vector2(1, 0.5))
	draw_circle(Vector2(2, 2), 18, Color(0, 0, 0, 0.25))
	draw_set_transform(Vector2.ZERO, 0, Vector2.ONE)
	var s := 0.9 + seed * 0.3
	var pts := PackedVector2Array([Vector2(-18, 0), Vector2(-14, -16), Vector2(-3, -24), Vector2(10, -20), Vector2(18, -6), Vector2(14, 4), Vector2(-10, 5)])
	for i in pts.size():
		pts[i] *= s
	draw_colored_polygon(pts, Color("7d7a86"))
	var top := PackedVector2Array([Vector2(-14, -16), Vector2(-3, -24), Vector2(10, -20), Vector2(2, -12)])
	for i in top.size():
		top[i] *= s
	draw_colored_polygon(top, Color("a19eab"))
	draw_polyline(pts + PackedVector2Array([pts[0]]), Color("4a4752"), 1.5, true)
