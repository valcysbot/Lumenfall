class_name IsoMap
extends Node2D
## Dessine le sol d'une carte en 3/4 isométrique et place les décors
## (arbres, rochers) dans le calque trié en profondeur.

var data := {}
var width := 0
var height := 0
var rows: Array = []
var blocking := ""


func load_map(id: String) -> bool:
	var path := "res://data/maps/%s.json" % id
	if not FileAccess.file_exists(path):
		return false
	data = JSON.parse_string(FileAccess.get_file_as_string(path))
	width = int(data.width)
	height = int(data.height)
	rows = data.rows
	blocking = data.blocking
	queue_redraw()
	return true


func cell(x: int, y: int) -> String:
	if x < 0 or y < 0 or x >= width or y >= height:
		return "T"
	return rows[y][x]


func walkable(p: Vector2) -> bool:
	return not blocking.contains(cell(floori(p.x), floori(p.y)))


func _hash(x: int, y: int) -> float:
	var h := (x * 374761393 + y * 668265263) & 0x7fffffff
	h = (h ^ (h >> 13)) * 1274126177 & 0x7fffffff
	return float(h % 1000) / 1000.0


func _draw() -> void:
	# La forêt continue au-delà des bords de la carte.
	var big := PackedVector2Array([Iso.to_screen(Vector2(-60, -60)), Iso.to_screen(Vector2(width + 60, -60)),
		Iso.to_screen(Vector2(width + 60, height + 60)), Iso.to_screen(Vector2(-60, height + 60))])
	draw_colored_polygon(big, Color("1f3a24"))
	var grass_a := Color("3f6b3a")
	var grass_b := Color("4a7a40")
	for y in height:
		for x in width:
			var c := cell(x, y)
			var n := _hash(x, y)
			var col := grass_a.lerp(grass_b, n)
			match c:
				"p": col = Color("a08a5c").lerp(Color("b09868"), n)
				"W": col = Color("2f5f86").lerp(Color("2a5579"), n)
				"T": col = Color("2f5230").lerp(Color("355a33"), n)
			_diamond(x, y, col)
			if c == "f":
				_flowers(x, y, n)
			elif c == "W":
				var center := Iso.to_screen(Vector2(x + 0.5, y + 0.5))
				if n > 0.6:
					draw_line(center + Vector2(-10, 0), center + Vector2(8, 0), Color(1, 1, 1, 0.18), 2)
				_shore(x, y)
			elif c == "." and n > 0.8:
				var p := Iso.to_screen(Vector2(x + 0.3 + n * 0.4, y + 0.6))
				draw_line(p, p + Vector2(-1.5, -4), Color("5a8a4a"), 1.2)
				draw_line(p + Vector2(2, 0), p + Vector2(3, -5), Color("5a8a4a"), 1.2)


func _diamond(x: int, y: int, col: Color) -> void:
	var pts := PackedVector2Array([
		Iso.to_screen(Vector2(x, y)), Iso.to_screen(Vector2(x + 1, y)),
		Iso.to_screen(Vector2(x + 1, y + 1)), Iso.to_screen(Vector2(x, y + 1)),
	])
	draw_colored_polygon(pts, col)
	# Grille très légère pour lire les cases.
	draw_polyline(PackedVector2Array([pts[0], pts[1], pts[2]]), Color(0, 0, 0, 0.06), 1.0)


func _shore(x: int, y: int) -> void:
	var edges := [[Vector2(0, -1), Vector2(x, y), Vector2(x + 1, y)], [Vector2(-1, 0), Vector2(x, y), Vector2(x, y + 1)],
		[Vector2(1, 0), Vector2(x + 1, y), Vector2(x + 1, y + 1)], [Vector2(0, 1), Vector2(x, y + 1), Vector2(x + 1, y + 1)]]
	for e in edges:
		if cell(x + int(e[0].x), y + int(e[0].y)) != "W":
			draw_line(Iso.to_screen(e[1]), Iso.to_screen(e[2]), Color("c9d9a0", 0.55), 2.0)


func _flowers(x: int, y: int, n: float) -> void:
	var cols := [Color("f2d16b"), Color("e88aa8"), Color("f2f0ea"), Color("b49ae8")]
	for i in 4:
		var p := Iso.to_screen(Vector2(x + 0.2 + fmod(n * (i + 3) * 1.7, 0.6), y + 0.2 + fmod(n * (i + 5) * 2.3, 0.6)))
		draw_circle(p, 2.2, cols[(i + int(n * 10)) % cols.size()])


## Ajoute un décor par case bloquante (sauf l'eau) dans le calque trié.
func spawn_props(layer: Node2D) -> void:
	for y in height:
		for x in width:
			var c := cell(x, y)
			if c == "T" or c == "R":
				var p := Prop.new()
				p.kind = c
				p.seed = _hash(x, y)
				p.position = Iso.to_screen(Vector2(x + 0.5, y + 0.5))
				layer.add_child(p)
