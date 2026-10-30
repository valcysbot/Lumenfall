extends Node
## Chef d'orchestre : passe de l'écran de connexion au choix du héros, puis au monde.
##
## Mode capture (captures d'écran automatiques, sans joueur) :
##   godot --path client -- --capture=world --out=/chemin/capture.png
##   (options : --aller=x,y --marche=s --salut --chat=… --parler=mabon --choix=sanglier)

const LoginScreen := preload("res://scripts/login_screen.gd")
const HeroScreen := preload("res://scripts/hero_screen.gd")
const WorldScreen := preload("res://scripts/world_screen.gd")

var ui_root: CanvasLayer
var current: Node
var args := {}


func _ready() -> void:
	for a in OS.get_cmdline_user_args():
		var kv: PackedStringArray = a.trim_prefix("--").split("=", true, 1)
		args[kv[0]] = kv[1] if kv.size() > 1 else "true"
	get_window().title = "Lumenfall, les Chroniques d'Erydor"
	get_window().min_size = Vector2i(640, 360)
	RenderingServer.set_default_clear_color(UI.NIGHT)
	Net.disconnected.connect(_on_disconnected)
	if args.has("capture"):
		_capture.call_deferred()
	elif Net.resume():
		show_heroes()
	else:
		show_login()


func _swap(node: Node) -> void:
	if current != null:
		current.queue_free()
	current = node
	add_child(node)


func show_login(message: String = "") -> void:
	var s = LoginScreen.new()
	s.logged_in.connect(show_heroes)
	_swap(s)
	if message != "":
		s.set_status(message, true)


func show_heroes() -> void:
	var s = HeroScreen.new()
	s.play.connect(enter_world)
	s.logout.connect(func(): Net.logout(); show_login())
	_swap(s)


func enter_world(hero: Dictionary) -> void:
	var w = WorldScreen.new()
	w.hero = hero
	w.quit_world.connect(func(): Net.leave_world(); show_heroes())
	_swap(w)


func _on_disconnected(reason: String) -> void:
	if current is WorldScreen:
		show_login(reason + " Reconnecte-toi pour reprendre l'aventure.")


# --- Captures d'écran automatiques -------------------------------------------

func _capture() -> void:
	var what: String = args.get("capture", "world")
	var out: String = args.get("out", "user://capture.png")
	if what == "login":
		show_login()
		await _shoot(out)
		return
	var err := await Net.login_test(args.get("device", "capture-lumenfall-0001"))
	if err != "":
		push_error(err)
		get_tree().quit(1)
		return
	if what == "heroes" or what == "create":
		show_heroes()
		await get_tree().create_timer(1.5).timeout
		if what == "create":
			current._open_form(1)
			current.name_edit.text = "D'Arvel"
			current.form_color = 0
			current._mark_swatch()
			current._dress_preview()
		await _shoot(out)
		return
	var list := await Net.hero_list()
	var heroes: Array = list.get("heroes", [])
	var hero: Dictionary
	if heroes.is_empty():
		hero = await Net.hero_create(0, args.get("name", "Elowyn"), int(args.get("color", "6")))
	else:
		hero = heroes[0]
	enter_world(hero)
	await get_tree().create_timer(float(args.get("wait", "4"))).timeout
	if args.has("aller"):
		var t: PackedStringArray = args.aller.split(",")
		Net.send(Net.OP_MOVE_TO, {"t": [float(t[0]), float(t[1])], "r": true})
		await get_tree().create_timer(float(args.get("marche", "6"))).timeout
	if args.has("salut"):
		current._wave()
	if args.has("parler"):
		# Clique sur un habitant présent, comme un joueur : on marche jusqu'à lui.
		for sid in current.pawns:
			var pawn: Pawn = current.pawns[sid]
			if pawn.npc == args.parler:
				current._click_npc(pawn.position + Vector2(0, -28))
		await get_tree().create_timer(float(args.get("marche", "6"))).timeout
		if args.has("choix"):
			current.dialogue._show(args.choix)
	if args.has("cible"):
		# Vise le monstre le plus proche de cette sorte et l'attaque, comme un joueur.
		var best := -1
		var best_d := 1e9
		var me: Pawn = current.pawns.get(current.my_sid)
		for sid in current.pawns:
			var pawn: Pawn = current.pawns[sid]
			if pawn.kind == args.cible and not pawn.dead and me != null:
				var d := me.position.distance_to(pawn.position)
				if d < best_d:
					best_d = d
					best = sid
		if best != -1:
			current._click_fighter(current.pawns[best].position + Vector2(0, -16))
		for k in args.get("competences", "").split(",", false):
			await get_tree().create_timer(float(args.get("pause", "1.6"))).timeout
			if k == "roulade":
				current._roll()
			elif k == "garde":
				current._set_block(true)
			else:
				current._use_skill(int(k))
		await get_tree().create_timer(float(args.get("combat", "2"))).timeout
	if args.has("chat"):
		Net.send(Net.OP_CHAT, {"m": args.chat})
		await get_tree().create_timer(1.0).timeout
	await _shoot(out)


func _shoot(path: String) -> void:
	await get_tree().create_timer(0.5).timeout
	await RenderingServer.frame_post_draw
	var img := get_viewport().get_texture().get_image()
	img.save_png(path)
	print("Capture enregistrée : ", path)
	get_tree().quit()
