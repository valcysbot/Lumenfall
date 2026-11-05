extends Node
## Chef d'orchestre : passe de l'écran de connexion au choix du héros, puis au monde.
##
## Mode capture (captures d'écran automatiques, sans joueur) :
##   godot --path client -- --capture=world --out=/chemin/capture.png
##   (options : --aller=x,y --marche=s --salut --chat=… --parler=mabon --choix=sanglier
##    --gigue=ms (réseau irrégulier) --mesure=s --trajet=x,y (fluidité) --fluidite --touches
##    --cible=sanglier --competences=1,2,roulade --defier=Brann --accepter --reste=s
##    --capture=heroes --effacer --confirmer
##    Acte V : --objet=a:…,… (actions envoyées au serveur, séparées par « ; »)
##    --recolter (ramasse ou récolte le plus proche) --panneau=boutique|maitre:doran|atelier:forge|coffre|miroir
##    --sac[=metiers] --choisir=i (un objet du sac) --echanger=Nom --accepter_echange)

const LoginScreen := preload("res://scripts/login_screen.gd")
const HeroScreen := preload("res://scripts/hero_screen.gd")
const WorldScreen := preload("res://scripts/world_screen.gd")

var ui_root: CanvasLayer
var current: Node
var args := {}
var _journal_t := 0.0
var _journal_min := 0


func _ready() -> void:
	for a in OS.get_cmdline_user_args():
		var kv: PackedStringArray = a.trim_prefix("--").split("=", true, 1)
		args[kv[0]] = kv[1] if kv.size() > 1 else "true"
	_journal_debut()
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


## Le journal de bord (godot.log) : de quoi comprendre un plantage ou un gel
## après coup. Une ligne au lancement, une par minute, une à chaque saccade.
func _journal_debut() -> void:
	print("[jeu] Lumenfall %s · %s · %s · %s" % [ProjectSettings.get_setting("application/config/version"),
		OS.get_name(), RenderingServer.get_video_adapter_name(), RenderingServer.get_current_rendering_driver_name()])


func _journal_etat() -> String:
	return "écran %s, %d ips, %d Mo, %d nœuds, %d objets" % [current.get_script().resource_path.get_file().get_basename() if current != null and current.get_script() != null else "?",
		Engine.get_frames_per_second(), int(Performance.get_monitor(Performance.MEMORY_STATIC) / 1048576.0),
		int(Performance.get_monitor(Performance.OBJECT_NODE_COUNT)), int(Performance.get_monitor(Performance.OBJECT_COUNT))]


func _process(delta: float) -> void:
	if delta > 0.4:
		print("[saccade] image de %.1f s (%s)" % [delta, _journal_etat()])
	_journal_t += delta
	if _journal_t >= 60.0:
		_journal_t = 0.0
		_journal_min += 1
		print("[jeu] %d min · %s" % [_journal_min, _journal_etat()])


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
	print("[jeu] déconnecté : %s (%s)" % [reason, _journal_etat()])
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
			await get_tree().process_frame
			var m: Miroir = current.get_child(current.get_child_count() - 1)
			m.form.race = args.get("race", "elfe")
			m.form.classe = args.get("classe", "druide")
			m.form.name = "D'Arvel"
			m._show(args.get("onglet", "peuple"))
			m._refresh()
			await get_tree().create_timer(float(args.get("wait", "1"))).timeout
		if args.has("effacer") and not current.heroes.is_empty():
			# Ouvre la demande de confirmation et tape le nom, comme un joueur.
			current._ask_delete(current.heroes[0])
			await get_tree().process_frame
			var field: LineEdit = current.find_child("Effacer", true, false).find_children("*", "LineEdit", true, false)[0]
			field.text = current.heroes[0].name if args.effacer == "true" else args.effacer
			field.text_changed.emit(field.text)
			await get_tree().create_timer(0.5).timeout
			if args.has("confirmer"):
				var yes: Button = current.find_child("Effacer", true, false).find_children("*", "Button", true, false)[-1]
				yes.pressed.emit()
				await get_tree().create_timer(2.0).timeout
		await _shoot(out)
		return
	var list := await Net.hero_list()
	var heroes: Array = list.get("heroes", [])
	var hero: Dictionary
	if heroes.is_empty():
		var lk := Heros.allure_au_hasard(args.get("race", "humain"))
		lk.tu = int(args.get("color", "6"))
		hero = await Net.hero_create({"slot": 0, "name": args.get("name", "Elowyn"), "race": args.get("race", "humain"),
			"classe": args.get("classe", "paladin"), "dieu": args.get("dieu", "solenn"), "hardcore": false, "look": lk})
	else:
		hero = heroes[0]
	enter_world(hero)
	if args.has("gigue"):
		current.gigue = float(args.gigue) / 1000.0
	current._fluidite = args.has("fluidite")
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
	if args.has("defier"):
		# Défie en duel le héros de ce nom, comme un joueur (bouton « Défier »).
		for sid in current.pawns:
			if current.pawns[sid].hero_name == args.defier:
				current._set_target(sid)
				current.combat.challenge.emit()
		await get_tree().create_timer(float(args.get("attente", "5"))).timeout
	if args.has("accepter"):
		# Accepte le premier défi reçu (bouton « J'accepte »).
		for i in 600:
			if current.duel_card.visible:
				await get_tree().create_timer(1.0).timeout
				Net.send(Net.OP_DUEL_REPLY, {"ok": true})
				current.duel_card.visible = false
				break
			await get_tree().create_timer(0.1).timeout
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
	if args.has("cible") or args.has("defier"):
		for k in args.get("competences", "").split(",", false):
			await get_tree().create_timer(float(args.get("pause", "1.6"))).timeout
			if k == "roulade":
				current._roll()
			elif k == "garde":
				current._set_block(true)
			else:
				current._use_skill(int(k))
		await get_tree().create_timer(float(args.get("combat", "2"))).timeout
	if args.has("zoom"):
		current._zoom_by(float(args.zoom))
	if args.has("conseil"):
		# « Répartition conseillée » puis « Valider », comme un joueur.
		current._open_sheet()
		current.sheet._render()
		var c := Heros.classe(str(current._me.get("cl", "")))
		current.sheet.attrs.emit(HeroSheet.conseil(c.conseil, int(current._me.get("pt", 0))))
		await get_tree().create_timer(1.0).timeout
		current._close_sheet()
	if args.has("fiche"):
		current._open_sheet()
		current.sheet._tab = args.fiche
		current.sheet._render()
		if args.has("defile"):
			await get_tree().process_frame
			var sc: ScrollContainer = current.sheet.find_children("*", "ScrollContainer", true, false)[0]
			sc.scroll_vertical = int(args.defile)
	if args.has("touches"):
		var p := preload("res://scripts/touches_panel.gd").new()
		current.settings.get_parent().add_child(p)
	# Acte V : le sac, les métiers, les panneaux, l'échange.
	if args.has("objet"):
		for a in args.objet.split(";", false):
			var msg := {}
			for kv in a.split(",", false):
				var p: PackedStringArray = kv.split(":", true, 1)
				msg[p[0]] = int(p[1]) if p[1].is_valid_int() else p[1]
			Net.send(Net.OP_OBJET, msg)
			await get_tree().create_timer(0.4).timeout
	if args.has("recolter"):
		for i in int(args.recolter) if args.recolter.is_valid_int() else 1:
			current.objets.act_nearest()
			await get_tree().create_timer(float(args.get("travail", "4"))).timeout
	if args.has("echanger"):
		for sid in current.pawns:
			if current.pawns[sid].hero_name == args.echanger:
				current._set_target(sid)
				current.combat.trade.emit()
		await get_tree().create_timer(float(args.get("attente", "3"))).timeout
	if args.has("accepter_echange"):
		for i in 300:
			if current.objets._demande != null:
				await get_tree().create_timer(1.0).timeout
				Net.send(Net.OP_OBJET, {"a": "echange_reponse", "ok": true})
				break
			await get_tree().create_timer(0.1).timeout
		await get_tree().create_timer(1.0).timeout
	if args.has("panneau"):
		var pp: PackedStringArray = args.panneau.split(":")
		Net.send(Net.OP_OBJET, {"a": "panneau", "b": pp[0], "r": pp[1] if pp.size() > 1 else ""})
		await get_tree().create_timer(1.0).timeout
	if args.has("sac"):
		if current.objets.sac_win == null:
			current.objets.open_sac("")
		if args.sac == "metiers":
			current.objets.sac_win._tab = "metiers"
		if args.has("choisir"):
			current.objets.sac_win._sel = {"sac": int(args.choisir)}
		current.objets.sac_win._render()
	if args.has("reste"):
		await get_tree().create_timer(float(args.reste)).timeout
	if args.has("mesure"):
		# Mesure la fluidité : images par seconde, pire image, appels de dessin.
		var frames := 0
		var pire := 0.0
		var appels := 0
		var calcul := 0.0
		# --trajet=x,y : le héros marche pendant la mesure, et on compte ses à-coups.
		var me: Pawn = current.pawns.get(current.my_sid)
		if args.has("trajet"):
			var tj: PackedStringArray = args.trajet.split(",")
			Net.send(Net.OP_MOVE_TO, {"t": [float(tj[0]), float(tj[1])], "r": false})
			await get_tree().create_timer(0.5).timeout
		var arrets := 0
		var pas := 0
		var vitesses: Array[float] = []
		var dernier_pas := Time.get_ticks_usec()
		var avant: Vector2 = me.position if me != null else Vector2.ZERO
		var t0 := Time.get_ticks_usec()
		var dernier := t0
		while Time.get_ticks_usec() - t0 < int(float(args.mesure) * 1e6):
			await get_tree().process_frame
			var now := Time.get_ticks_usec()
			pire = maxf(pire, (now - dernier) / 1000.0)
			dernier = now
			frames += 1
			appels = maxi(appels, int(Performance.get_monitor(Performance.RENDER_TOTAL_DRAW_CALLS_IN_FRAME)))
			calcul += Performance.get_monitor(Performance.TIME_PROCESS) * 1000.0
			if me != null and me.walking:
				pas += 1
				if me.position.distance_to(avant) < 0.01:
					arrets += 1
				vitesses.append(me.position.distance_to(avant) / maxf(0.001, (now - dernier_pas) / 1e6))
			dernier_pas = now
			if me != null:
				avant = me.position
		print("Mesure : %.1f images/s, pire image %.1f ms, calcul %.2f ms par image, %d appels de dessin, %d nœuds" % [
			frames / float(args.mesure), pire, calcul / maxf(frames, 1), appels, int(Performance.get_monitor(Performance.OBJECT_NODE_COUNT))])
		if pas > 0:
			var moy := 0.0
			for v in vitesses:
				moy += v / vitesses.size()
			var ecart := 0.0
			for v in vitesses:
				ecart += (v - moy) * (v - moy) / vitesses.size()
			print("Marche : %d images, %d figées, vitesse à l'écran irrégulière à %.0f %%" % [pas, arrets, 100.0 * sqrt(ecart) / maxf(moy, 0.001)])
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
