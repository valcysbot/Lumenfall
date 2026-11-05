extends SceneTree
## Reporte une partie du plan (tools/maps/<id>.plan.json) dans la scène déjà
## retouchée (res://cartes/<id>.tscn), sans toucher au reste : le sol, les
## terrains et les décors d'un rectangle de cases. Sert à ajouter une pièce
## (les salles du Cercle des Braves, par exemple) sans tout reconstruire.
##
##   godot --headless --path client --script res://outils/reporter_plan.gd -- glennmor x0 y0 x1 y1
##
## Puis exporter_carte.gd, comme d'habitude.

const SOL := "res://assets/tuiles/"
const TERRAINS := ["chemin", "paves", "eau", "gris"]
const SOL_DE := {".": "herbe", ",": "herbe_fleurie", ":": "sous_bois", "f": "plancher", "v": "vide", "b": "plancher"}
const TERRAIN_DE := {"p": "chemin", "c": "paves", "W": "eau", "g": "gris"}
const VARIANTES := {"herbe": 4, "herbe_fleurie": 2, "sous_bois": 2, "plancher": 2}


func _init() -> void:
	var args := OS.get_cmdline_user_args()
	if args.size() < 5:
		push_error("usage : -- <carte> x0 y0 x1 y1")
		quit(1)
		return
	var id: String = args[0]
	var r := Rect2i(int(args[1]), int(args[2]), int(args[3]) - int(args[1]), int(args[4]) - int(args[2]))
	var plan: Dictionary = JSON.parse_string(FileAccess.get_file_as_string(
		ProjectSettings.globalize_path("res://").path_join("../tools/maps/%s.plan.json" % id)))
	var sol_meta: Dictionary = JSON.parse_string(FileAccess.get_file_as_string(SOL + "sol.json"))
	var root: Node2D = load("res://cartes/%s.tscn" % id).instantiate()
	var sol: TileMapLayer = root.get_node("Sol")
	var terr: TileMapLayer = root.get_node("Terrains")
	var by_terrain := {}
	for t in TERRAINS:
		by_terrain[t] = []
	var rows: Array = plan.sol
	for y in range(r.position.y, r.end.y):
		var row: String = rows[y]
		for x in range(r.position.x, r.end.x):
			var c := row[x]
			var base: String = SOL_DE.get(c, "herbe")
			var n: int = VARIANTES.get(base, 1)
			var name: String = base if n == 1 else "%s_%d" % [base, (x * 7 + y * 13 + (x * y) % 5) % n]
			var at: Array = sol_meta.tuiles[name].atlas
			sol.set_cell(Vector2i(x, y), 0, Vector2i(at[0], at[1]))
			terr.erase_cell(Vector2i(x, y))
			if TERRAIN_DE.has(c):
				by_terrain[TERRAIN_DE[c]].append(Vector2i(x, y))
	for t in TERRAINS:
		if not by_terrain[t].is_empty():
			terr.set_cells_terrain_connect(by_terrain[t], 0, TERRAINS.find(t), false)

	var decor_meta: Dictionary = JSON.parse_string(FileAccess.get_file_as_string("res://assets/decors/decors.json"))
	var decors: Node2D = root.get_node("Decors")
	# Les décors déjà là, par position, pour ne pas les poser deux fois.
	var deja := {}
	for d in decors.get_children():
		deja[Vector2i(d.position.round())] = true
	var n_new := 0
	for d in plan.decors:
		if not r.has_point(Vector2i(d.x, d.y)):
			continue
		var m: Dictionary = decor_meta[d.d]
		var pos := Iso.to_screen(Vector2(d.x + m.emprise[0] / 2.0, d.y + m.emprise[1] / 2.0)).round()
		if deja.has(Vector2i(pos)):
			continue
		var node: Node2D = (load("res://cartes/decors/%s.tscn" % d.d) as PackedScene).instantiate()
		node.position = pos
		node.name = "%s_%d_%d" % [d.d.capitalize().replace(" ", ""), d.x, d.y]
		decors.add_child(node)
		node.owner = root
		n_new += 1

	var packed := PackedScene.new()
	packed.pack(root)
	var err := ResourceSaver.save(packed, "res://cartes/%s.tscn" % id)
	print("Plan reporté dans %s : %s, %d décors ajoutés (%s)" % [id, r, n_new, error_string(err)])
	root.free()
	quit(0 if err == OK else 1)
