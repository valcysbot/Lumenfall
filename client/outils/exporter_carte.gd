extends SceneTree
## Lit une carte Godot (res://cartes/<id>.tscn), éventuellement retouchée dans
## l'éditeur, et écrit ce que le serveur et le client doivent savoir : les
## cases bloquées (eau, vide, emprise des décors), le point d'apparition et
## les portes.
##
##   godot --headless --path client --script res://outils/exporter_carte.gd -- glennmor
##
## Écrit server/world/maps/<id>.json et client/data/maps/<id>.json (identiques).

func _init() -> void:
	var id := "glennmor"
	var args := OS.get_cmdline_user_args()
	if args.size() > 0:
		id = args[0]
	var root: Node2D = load("res://cartes/%s.tscn" % id).instantiate()
	var sol: TileMapLayer = root.get_node("Sol")
	var terr: TileMapLayer = root.get_node("Terrains")
	var used := sol.get_used_rect()
	var w := used.end.x
	var h := used.end.y
	var blocked := PackedByteArray()
	blocked.resize(w * h)
	for layer in [sol, terr]:
		for c in layer.get_used_cells():
			var td: TileData = layer.get_cell_tile_data(c)
			if td != null and td.get_custom_data("bloque") and c.x >= 0 and c.y >= 0 and c.x < w and c.y < h:
				blocked[c.y * w + c.x] = 1
	var n := 0
	for d in root.get_node("Decors").get_children():
		if not d.has_meta("emprise") or d.get_meta("passable", false):
			continue
		n += 1
		var e: Vector2i = d.get_meta("emprise")
		var center := Iso.to_world(d.position)
		var x0 := roundi(center.x - e.x / 2.0)
		var y0 := roundi(center.y - e.y / 2.0)
		for y in range(y0, y0 + e.y):
			for x in range(x0, x0 + e.x):
				if x >= 0 and y >= 0 and x < w and y < h:
					blocked[y * w + x] = 1
	var doors := []
	for p in root.get_node("Portes").get_children():
		var at := Iso.to_world(p.position).floor()
		var to: Vector2 = p.get_meta("vers")
		doors.append({"from": [int(at.x), int(at.y)], "to": [to.x, to.y]})
		blocked[int(at.y) * w + int(at.x)] = 0
	var rows := []
	for y in h:
		var r := ""
		for x in w:
			r += "#" if blocked[y * w + x] else "."
		rows.append(r)
	var spawn := Iso.to_world(root.get_node("Apparition").position)
	var out := {
		"id": id, "name": root.get_meta("nom", id), "width": w, "height": h,
		"spawn": [snappedf(spawn.x, 0.01), snappedf(spawn.y, 0.01)],
		"blocking": "#", "rows": rows, "doors": doors,
		"indoors_from": root.get_meta("interieurs", h),
	}
	var text := JSON.stringify(out)
	var base := ProjectSettings.globalize_path("res://")
	for path in [base.path_join("../server/world/maps/%s.json" % id), base.path_join("data/maps/%s.json" % id)]:
		var f := FileAccess.open(path, FileAccess.WRITE)
		f.store_string(text)
		f.close()
	print("Carte exportée : %s, %d × %d, %d décors, %d portes" % [id, w, h, n, doors.size()])
	root.free()
	quit()
