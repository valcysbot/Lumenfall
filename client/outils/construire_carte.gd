extends SceneTree
## Transforme un plan de carte (tools/maps/<id>.plan.json) en scène Godot :
## res://cartes/<id>.tscn, avec deux calques de tuiles (le sol, et les terrains
## à bords : chemins, pavés, eau, herbe grise) et les décors.
##
## Après ça, la carte se retouche à la souris dans l'éditeur de Godot :
## peindre les terrains (onglet TileMap › Terrains), glisser des décors depuis
## res://cartes/decors/. Puis on lance exporter_carte.gd pour le serveur.
##
##   godot --headless --path client --script res://outils/construire_carte.gd -- glennmor

const SOL := "res://assets/tuiles/"
const DECORS := "res://assets/decors/"
const SORTIE := "res://cartes/"
const TERRAINS := ["chemin", "paves", "eau", "gris"]
const SOL_DE := {".": "herbe", ",": "herbe_fleurie", ":": "sous_bois", "f": "plancher", "v": "vide", "b": "plancher"}
const TERRAIN_DE := {"p": "chemin", "c": "paves", "W": "eau", "g": "gris"}
const VARIANTES := {"herbe": 4, "herbe_fleurie": 2, "sous_bois": 2, "plancher": 2}


func _init() -> void:
	var id := "glennmor"
	var args := OS.get_cmdline_user_args()
	if args.size() > 0:
		id = args[0]
	var plan_path := ProjectSettings.globalize_path("res://").path_join("../tools/maps/%s.plan.json" % id)
	var plan: Dictionary = JSON.parse_string(FileAccess.get_file_as_string(plan_path))
	var sol_meta: Dictionary = JSON.parse_string(FileAccess.get_file_as_string(SOL + "sol.json"))
	var decor_meta: Dictionary = JSON.parse_string(FileAccess.get_file_as_string(DECORS + "decors.json"))
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(SORTIE + "decors"))

	var tileset := _tileset(sol_meta)
	ResourceSaver.save(tileset, SORTIE + "sol.tres")
	tileset = load(SORTIE + "sol.tres")
	var scenes := _decor_scenes(decor_meta)

	var root := Node2D.new()
	root.name = plan.id.capitalize()
	root.set_meta("nom", plan.nom)
	root.set_meta("interieurs", int(plan.get("interieurs", 1 << 20)))
	var sol := TileMapLayer.new()
	sol.name = "Sol"
	sol.tile_set = tileset
	root.add_child(sol)
	sol.owner = root
	var terr := TileMapLayer.new()
	terr.name = "Terrains"
	terr.tile_set = tileset
	root.add_child(terr)
	terr.owner = root
	# Les calques de tuiles se calent sur nos coordonnées : la case (x, y) a son
	# centre en Iso.to_screen(x + 0.5, y + 0.5).
	var step := sol.map_to_local(Vector2i(1, 0)) - sol.map_to_local(Vector2i.ZERO)
	assert(step.is_equal_approx(Vector2(32, 16)), "orientation des tuiles inattendue : %s" % step)
	sol.position = Iso.to_screen(Vector2(0.5, 0.5)) - sol.map_to_local(Vector2i.ZERO)
	terr.position = sol.position

	var by_terrain := {}
	for t in TERRAINS:
		by_terrain[t] = []
	var rows: Array = plan.sol
	for y in rows.size():
		var row: String = rows[y]
		for x in row.length():
			var c := row[x]
			var base: String = SOL_DE.get(c, "herbe")
			var n: int = VARIANTES.get(base, 1)
			var name: String = base if n == 1 else "%s_%d" % [base, (x * 7 + y * 13 + (x * y) % 5) % n]
			var at: Array = sol_meta.tuiles[name].atlas
			sol.set_cell(Vector2i(x, y), 0, Vector2i(at[0], at[1]))
			if TERRAIN_DE.has(c):
				by_terrain[TERRAIN_DE[c]].append(Vector2i(x, y))
	for t in TERRAINS:
		terr.set_cells_terrain_connect(by_terrain[t], 0, TERRAINS.find(t), false)

	var decors := Node2D.new()
	decors.name = "Decors"
	decors.y_sort_enabled = true
	root.add_child(decors)
	decors.owner = root
	for d in plan.decors:
		var m: Dictionary = decor_meta[d.d]
		var node: Node2D = scenes[d.d].instantiate()
		node.position = Iso.to_screen(Vector2(d.x + m.emprise[0] / 2.0, d.y + m.emprise[1] / 2.0)).round()
		decors.add_child(node)
		node.owner = root

	var spawn := Marker2D.new()
	spawn.name = "Apparition"
	spawn.position = Iso.to_screen(Vector2(plan.apparition[0], plan.apparition[1]))
	root.add_child(spawn)
	spawn.owner = root
	var doors := Node2D.new()
	doors.name = "Portes"
	root.add_child(doors)
	doors.owner = root
	for i in plan.portes.size():
		var p: Dictionary = plan.portes[i]
		var mk := Marker2D.new()
		mk.name = "Porte%d" % (i + 1)
		mk.position = Iso.to_screen(Vector2(p.de[0] + 0.5, p.de[1] + 0.5))
		mk.set_meta("vers", Vector2(p.vers[0], p.vers[1]))
		doors.add_child(mk)
		mk.owner = root

	var packed := PackedScene.new()
	packed.pack(root)
	var err := ResourceSaver.save(packed, SORTIE + plan.id + ".tscn")
	print("Carte écrite : ", SORTIE + plan.id + ".tscn", " (", plan.decors.size(), " décors) ", error_string(err))
	root.free()
	quit(0 if err == OK else 1)


func _tileset(meta: Dictionary) -> TileSet:
	var ts := TileSet.new()
	ts.tile_shape = TileSet.TILE_SHAPE_ISOMETRIC
	ts.tile_layout = TileSet.TILE_LAYOUT_DIAMOND_DOWN
	ts.tile_size = Vector2i(64, 32)
	ts.add_custom_data_layer()
	ts.set_custom_data_layer_name(0, "bloque")
	ts.set_custom_data_layer_type(0, TYPE_BOOL)
	ts.add_terrain_set()
	ts.set_terrain_set_mode(0, TileSet.TERRAIN_MODE_MATCH_SIDES)
	var colors := {"chemin": Color("a8834f"), "paves": Color("9a9aa8"), "eau": Color("3a78b8"), "gris": Color("7d867a")}
	for t in TERRAINS:
		var k := TERRAINS.find(t)
		ts.add_terrain(0)
		ts.set_terrain_name(0, k, t)
		ts.set_terrain_color(0, k, colors[t])
	var src := TileSetAtlasSource.new()
	src.texture = load(SOL + "sol.png")
	src.texture_region_size = Vector2i(64, 32)
	ts.add_source(src, 0)

	# Quel côté du losange donne sur quelle case voisine ?
	var probe := TileMapLayer.new()
	probe.tile_set = ts
	var sides := {}
	for n in [TileSet.CELL_NEIGHBOR_TOP_RIGHT_SIDE, TileSet.CELL_NEIGHBOR_BOTTOM_RIGHT_SIDE,
			TileSet.CELL_NEIGHBOR_BOTTOM_LEFT_SIDE, TileSet.CELL_NEIGHBOR_TOP_LEFT_SIDE]:
		var off := probe.get_neighbor_cell(Vector2i.ZERO, n)
		for side in meta.cotes:
			if Vector2i(meta.cotes[side][0], meta.cotes[side][1]) == off:
				sides[side] = n
	probe.free()
	assert(sides.size() == 4, "côtés des tuiles introuvables")

	for name in meta.tuiles:
		var info: Dictionary = meta.tuiles[name]
		var at := Vector2i(info.atlas[0], info.atlas[1])
		src.create_tile(at)
		var td := src.get_tile_data(at, 0)
		td.set_custom_data("bloque", info.get("bloque", false))
		if info.has("terrain"):
			var k := TERRAINS.find(info.terrain)
			td.terrain_set = 0
			td.terrain = k
			for side in sides:
				td.set_terrain_peering_bit(sides[side], k if side in info.relie else -1)
	return ts


## Une petite scène par décor, à glisser dans la carte depuis l'éditeur.
func _decor_scenes(meta: Dictionary) -> Dictionary:
	var out := {}
	for name in meta:
		var m: Dictionary = meta[name]
		var s := Sprite2D.new()
		s.name = name.capitalize().replace(" ", "")
		s.texture = load(DECORS + name + ".png")
		s.centered = false
		s.offset = -Vector2(m.ancre[0], m.ancre[1])
		s.texture_filter = CanvasItem.TEXTURE_FILTER_NEAREST
		s.set_meta("decor", name)
		s.set_meta("emprise", Vector2i(m.emprise[0], m.emprise[1]))
		if m.get("passable", false):
			s.set_meta("passable", true)
		if m.has("lumiere"):
			var l := PointLight2D.new()
			l.name = "Lumiere"
			l.position = Vector2(m.lumiere[0], m.lumiere[1])
			l.enabled = false
			l.add_to_group("lumieres", true)
			s.add_child(l)
			l.owner = s
		var p := PackedScene.new()
		p.pack(s)
		var path: String = SORTIE + "decors/" + name + ".tscn"
		ResourceSaver.save(p, path)
		out[name] = load(path)
		s.free()
	return out
