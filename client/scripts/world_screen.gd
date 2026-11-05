extends Node2D
## Le monde : la carte, les héros (le tien et ceux des autres), les
## habitants, Braise, les commandes (clic, ZQSD, joystick) et le chat.
##
## Le client n'envoie que des intentions. Le serveur calcule les positions
## et les renvoie 10 fois par seconde ; on les adoucit à l'écran.

signal quit_world

const INTERP_DELAY := 0.17  # secondes de retard volontaire pour un mouvement fluide
const TICK := 0.1            # le serveur envoie le monde 10 fois par seconde
var _fluidite := false        # réglage : montrer images/s et régularité du réseau
var _dernier_delta := -1.0
var _ecart_reseau := 0.0      # écart moyen entre deux images du monde et le rythme attendu (s)
var gigue := 0.0             # pour les essais : retard aléatoire ajouté aux messages (secondes)
const RUN_SEEN := 5.2       # au-dessus de cette vitesse (cases/s), on montre la course
const ZOOM_MIN := 0.8
const ZOOM_MAX := 2.2
const TALK_REACH := 2.2     # cases : assez près pour parler à un habitant
const BUBBLE_REACH := 4.5   # cases : un habitant te lance une bulle en passant
const BUBBLE_EVERY := 25.0  # secondes entre deux bulles d'un même habitant
const MELEE_REACH := 1.6    # cases : assez près pour frapper (le serveur décide)
const TARGET_RANGE := 16.0  # cases : Tab cherche une cible jusque-là

var hero := {}

var iso_map: IsoMap
var sky: WorldSky
var entities: Node2D
var camera: Camera2D
var marker: Node2D
var pawns := {}       # sid -> Pawn
var buffers := {}     # sid -> Array de [temps, position monde]
var my_sid := -1
var map_id := ""

var dialogue: DialogueBox
var braise: Braise
var _talk_to := -1          # habitant vers qui on marche pour lui parler
var _bubble_at := {}        # sid -> heure de la dernière bulle
var _near_t := 0.0
var _chase_t := 0.0         # dernier trajet demandé vers l'habitant
var _talk_at := Vector2.ZERO  # où l'on était en commençant à parler

var hud: CanvasLayer
var combat: CombatHud
var fx_layer: Node2D
var tele_layer: Node2D
var target_sid := 0
var _assaillants := {}       # sid → dernière fois qu'il t'a frappé
var _hunt := 0              # monstre vers qui l'on marche pour le frapper
var _hunt_t := 0.0
var _shake := 0.0
var _slowmo := 0.0
var _blocking := false
var _fight_t := -1e9       # dernier coup donné ou reçu
var tutorial: CombatTutorial
var duel_card: PanelContainer
var _duel_text: Label
var _duel_t := 0.0          # secondes pour répondre au défi
var _duel_rings := {}       # numéro du duel -> DuelRing
var _duel_foe := 0          # ton adversaire en duel (0 : aucun)
var _duel_id := 0
const TUTO_REACH := 10.0    # cases : le tutoriel commence quand un monstre est à cette distance
var info: Label
var chat_log: VBoxContainer
var chat_edit: LineEdit
var joystick: TouchStick
var run_btn: Button
var settings: PanelContainer
var toast: Label
var sheet: HeroSheet        # la fiche du héros ouverte (C), ou nil
var miroir_btn: Button
var _me := {}               # le dernier état privé du héros (opMe)
var objectif: Label          # « 🎯 la prochaine étape », sous les boutons
var sheet_btn: Button       # « 📜 Héros » : un point lumineux quand des talents attendent
var _miroir: Miroir
var objets: MondeObjets       # l'Acte V : sac, butin, métiers, commerce
var sac_btn: Button
const MIROIR_REACH := 7.0   # cases : assez près de Sœur Clémence pour le miroir de Solenn

var _last_dir := Vector2.ZERO
var _last_run := false
var _run_toggle := false
var _last_dir_sent := 0.0
var _clock := 0.0
var _marker_t := 0.0
var _info_t := 0.0

const WEATHER_TEXT := {
	"clair": "Braise dit : le ciel se dégage !",
	"pluie": "La pluie se met à tomber sur la région.",
	"brouillard": "Un brouillard épais se lève…",
	"orage": "L'orage gronde ! Braise se cache sous ta cape.",
}


func _ready() -> void:
	iso_map = IsoMap.new()
	add_child(iso_map)
	tele_layer = Node2D.new()   # les zones rouges, au ras du sol
	add_child(tele_layer)
	marker = Node2D.new()
	marker.draw.connect(_draw_marker)
	marker.visible = false
	add_child(marker)
	entities = Node2D.new()
	entities.y_sort_enabled = true
	add_child(entities)
	fx_layer = Node2D.new()
	fx_layer.z_index = 50
	add_child(fx_layer)
	camera = Camera2D.new()
	camera.position_smoothing_enabled = true
	camera.position_smoothing_speed = 8.0
	add_child(camera)
	camera.make_current()
	sky = WorldSky.new()
	add_child(sky)
	_build_hud()

	camera.position = Iso.to_screen(Vector2(hero.x, hero.y))
	Net.match_state.connect(_on_state_gigue)
	_show_toast("En route pour Erydor…")
	var err := await Net.join_world(hero.id)
	if err != "":
		_show_toast(err)
		await get_tree().create_timer(3).timeout
		quit_world.emit()


# --- Interface --------------------------------------------------------------

func _build_hud() -> void:
	hud = CanvasLayer.new()
	add_child(hud)
	var root := Control.new()
	root.set_anchors_preset(Control.PRESET_FULL_RECT)
	root.mouse_filter = Control.MOUSE_FILTER_IGNORE
	root.theme = UI.make_theme()
	hud.add_child(root)

	var top := PanelContainer.new()
	top.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.7), Color(UI.LINE, 0.8), 12, 1, 8))
	top.position = Vector2(14, 14)
	info = UI.label("", 17)
	info.autowrap_mode = TextServer.AUTOWRAP_OFF
	top.add_child(info)
	root.add_child(top)

	# La prochaine étape : quoi faire, et une flèche autour du héros pour y aller.
	var obj := PanelContainer.new()
	obj.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.7), Color(UI.LINE, 0.8), 12, 1, 8))
	obj.set_anchors_and_offsets_preset(Control.PRESET_TOP_RIGHT, Control.PRESET_MODE_MINSIZE, 14)
	obj.grow_horizontal = Control.GROW_DIRECTION_BEGIN
	obj.position.y += 64
	obj.mouse_filter = Control.MOUSE_FILTER_IGNORE
	objectif = UI.label("", 15, UI.PARCH)
	objectif.custom_minimum_size = Vector2(0, 0)
	objectif.autowrap_mode = TextServer.AUTOWRAP_OFF
	obj.add_child(objectif)
	obj.visible = false
	root.add_child(obj)

	var buttons := HBoxContainer.new()
	buttons.add_theme_constant_override("separation", 8)
	buttons.set_anchors_and_offsets_preset(Control.PRESET_TOP_RIGHT, Control.PRESET_MODE_MINSIZE, 14)
	buttons.grow_horizontal = Control.GROW_DIRECTION_BEGIN
	var wave_btn := UI.button("👋 Saluer")
	wave_btn.tooltip_text = "Touche H"
	wave_btn.pressed.connect(_wave)
	buttons.add_child(wave_btn)
	sheet_btn = UI.button("📜 Héros")
	sheet_btn.tooltip_text = "Tes attributs, ta voie et tes talents (touche C)"
	sheet_btn.pressed.connect(_open_sheet)
	buttons.add_child(sheet_btn)
	sac_btn = UI.button("🎒 Sac")
	sac_btn.tooltip_text = "Ton sac, ton équipement et tes métiers (touche I)"
	sac_btn.pressed.connect(func(): objets.toggle_sac())
	buttons.add_child(sac_btn)
	miroir_btn = UI.button("🪞 Miroir")
	miroir_btn.tooltip_text = "Le miroir de Solenn : change ton allure"
	miroir_btn.pressed.connect(_open_miroir)
	miroir_btn.visible = false
	buttons.add_child(miroir_btn)
	var chat_btn := UI.button("💬 Parler")
	chat_btn.pressed.connect(_open_chat)
	buttons.add_child(chat_btn)
	var gear := UI.button("⚙")
	gear.tooltip_text = "Réglages"
	gear.pressed.connect(func(): settings.visible = not settings.visible)
	buttons.add_child(gear)
	var quit := UI.button("Quitter")
	quit.pressed.connect(func(): quit_world.emit())
	buttons.add_child(quit)
	root.add_child(buttons)
	# Espace sert à rouler : aucun bouton ne doit garder le focus clavier.
	for btn in buttons.get_children():
		btn.focus_mode = Control.FOCUS_NONE
	_build_settings(root)

	dialogue = DialogueBox.new()
	dialogue.hero_name = hero.get("name", "")
	dialogue.visible = false
	dialogue.set_anchors_and_offsets_preset(Control.PRESET_CENTER_BOTTOM, Control.PRESET_MODE_MINSIZE, 24)
	dialogue.grow_horizontal = Control.GROW_DIRECTION_BOTH
	dialogue.grow_vertical = Control.GROW_DIRECTION_BEGIN
	root.add_child(dialogue)

	# --tactile : la disposition téléphone sur PC (pour les captures).
	var touch := DisplayServer.is_touchscreen_available() or "--tactile" in OS.get_cmdline_user_args()
	combat = CombatHud.new()
	combat.touch = touch
	root.add_child(combat)
	combat.skill.connect(_use_skill)
	combat.potion.connect(_potion)
	combat.roll.connect(_roll)
	combat.next_target.connect(_next_target)
	combat.block.connect(_set_block)
	combat.challenge.connect(func(): Net.send(Net.OP_CHALLENGE, {"t": target_sid}))
	combat.fiche.connect(func(): Net.send(Net.OP_FICHE, {"s": target_sid}))
	chat_log = VBoxContainer.new()
	chat_log.mouse_filter = Control.MOUSE_FILTER_IGNORE
	chat_log.set_anchors_and_offsets_preset(Control.PRESET_BOTTOM_LEFT, Control.PRESET_MODE_MINSIZE, 14)
	chat_log.grow_vertical = Control.GROW_DIRECTION_BEGIN
	chat_log.custom_minimum_size = Vector2(380, 0)
	chat_log.position.y -= 190 if touch else 20
	root.add_child(chat_log)

	chat_edit = LineEdit.new()
	chat_edit.placeholder_text = "Écris ton message, puis Entrée (Échap pour fermer)"
	chat_edit.max_length = 120
	chat_edit.custom_minimum_size = Vector2(520, 46)
	chat_edit.set_anchors_and_offsets_preset(Control.PRESET_CENTER_BOTTOM, Control.PRESET_MODE_MINSIZE, 14)
	chat_edit.grow_horizontal = Control.GROW_DIRECTION_BOTH
	chat_edit.grow_vertical = Control.GROW_DIRECTION_BEGIN
	chat_edit.visible = false
	chat_edit.text_submitted.connect(_send_chat)
	root.add_child(chat_edit)

	if touch:
		joystick = TouchStick.new()
		joystick.set_anchors_and_offsets_preset(Control.PRESET_BOTTOM_LEFT, Control.PRESET_MODE_MINSIZE, 24)
		joystick.grow_vertical = Control.GROW_DIRECTION_BEGIN
		root.add_child(joystick)
		var side := VBoxContainer.new()
		side.add_theme_constant_override("separation", 10)
		# À droite, au-dessus de l'éventail de combat.
		side.set_anchors_and_offsets_preset(Control.PRESET_CENTER_RIGHT, Control.PRESET_MODE_MINSIZE, 24)
		side.grow_horizontal = Control.GROW_DIRECTION_BEGIN
		side.grow_vertical = Control.GROW_DIRECTION_BOTH
		side.position.y -= 110
		for z in [["＋", 1.25], ["－", 0.8]]:
			var zb := UI.button(z[0])
			zb.custom_minimum_size = Vector2(64, 56)
			zb.pressed.connect(_zoom_by.bind(z[1]))
			side.add_child(zb)
		run_btn = UI.button("🏃 Courir")
		run_btn.toggle_mode = true
		run_btn.custom_minimum_size = Vector2(120, 72)
		run_btn.toggled.connect(func(on): _run_toggle = on)
		run_btn.focus_mode = Control.FOCUS_NONE
		side.add_child(run_btn)
		root.add_child(side)

	toast = UI.label("", 20, UI.LUMEN)
	toast.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	toast.autowrap_mode = TextServer.AUTOWRAP_OFF
	toast.set_anchors_and_offsets_preset(Control.PRESET_CENTER_TOP, Control.PRESET_MODE_MINSIZE, 150)
	toast.grow_horizontal = Control.GROW_DIRECTION_BOTH
	toast.add_theme_color_override("font_outline_color", UI.NIGHT)
	toast.add_theme_constant_override("outline_size", 6)
	root.add_child(toast)

	_build_duel_card(root)
	objets = MondeObjets.new()
	add_child(objets)
	objets.setup(self, root, touch)
	objets.add_apparences_button(miroir_btn)
	dialogue.action.connect(objets.npc_action)
	combat.trade.connect(func(): Net.send(Net.OP_OBJET, {"a": "echange_demande", "t": target_sid}))
	tutorial = CombatTutorial.new()
	tutorial.touch = touch
	tutorial.glow.connect(_glow)
	tutorial.finished.connect(func(): _set_tuto_done(true))
	root.add_child(tutorial)
	_update_info()


func _build_settings(root: Control) -> void:
	settings = PanelContainer.new()
	settings.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.92), UI.LUMEN, 12, 1, 14))
	settings.set_anchors_and_offsets_preset(Control.PRESET_TOP_RIGHT, Control.PRESET_MODE_MINSIZE, 14)
	settings.grow_horizontal = Control.GROW_DIRECTION_BEGIN
	settings.position.y += 64
	settings.visible = false
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 8)
	settings.add_child(col)
	col.add_child(UI.label("Réglages", 20, UI.LUMEN))
	var l := UI.label("Luminosité de la nuit", 16, UI.MUTED)
	l.autowrap_mode = TextServer.AUTOWRAP_OFF
	col.add_child(l)
	var slider := HSlider.new()
	slider.custom_minimum_size = Vector2(240, 32)
	slider.min_value = 0.0
	slider.max_value = 1.0
	slider.step = 0.05
	slider.value = sky.night_brightness
	slider.value_changed.connect(sky.set_night_brightness)
	col.add_child(slider)
	var hint := UI.label("Plus à gauche : nuits noires, la torche devient précieuse.", 14, UI.MUTED)
	hint.custom_minimum_size = Vector2(240, 0)
	col.add_child(hint)
	var follow := CheckButton.new()
	follow.text = "Braise me suit"
	follow.button_pressed = _braise_wanted()
	follow.toggled.connect(_set_braise)
	col.add_child(follow)
	var fl := CheckButton.new()
	fl.text = "Afficher la fluidité"
	fl.tooltip_text = "Images par seconde de ton écran, et la régularité du réseau"
	fl.toggled.connect(func(on: bool):
		_fluidite = on
		_update_info())
	col.add_child(fl)
	var keys := UI.button("⌨ Changer mes touches")
	keys.focus_mode = Control.FOCUS_NONE
	keys.pressed.connect(func():
		settings.visible = false
		var p := preload("res://scripts/touches_panel.gd").new()
		p.changed.connect(combat.refresh_keys)
		settings.get_parent().add_child(p))
	col.add_child(keys)
	var tuto := UI.button("🐉 Revoir le tutoriel de combat")
	tuto.focus_mode = Control.FOCUS_NONE
	tuto.pressed.connect(func():
		settings.visible = false
		_start_tutorial())
	col.add_child(tuto)
	root.add_child(settings)


## Charge la carte annoncée par le serveur : la scène Godot (sol, décors) si
## elle existe, et toujours les cases praticables pour les clics.
func _load_map(id: String) -> bool:
	if not iso_map.load_map(id):
		return false
	map_id = id
	var path := "res://cartes/%s.tscn" % id
	if ResourceLoader.exists(path):
		var scene: Node2D = load(path).instantiate()
		iso_map.add_child(scene)
		iso_map.ground_visible = false
		# Les décors rejoignent les héros dans le calque trié en profondeur,
		# pour qu'on puisse passer derrière une maison ou devant un arbre.
		for d in scene.get_node("Decors").get_children():
			d.reparent(entities, false)
		sky.lamps = get_tree().get_nodes_in_group("lumieres")
	else:
		iso_map.spawn_props(entities)
	return true


func _map_name() -> String:
	return iso_map.data.get("name", "Erydor")


func _update_info() -> void:
	var n := 0
	for sid in pawns:
		if pawns[sid].npc == "" and pawns[sid].kind == "":
			n += 1
	info.text = "%s · %s · %s · %d %s" % [hero.get("name", ""), _map_name(), sky.describe(), n, "aventurier en ligne" if n <= 1 else "aventuriers en ligne"]
	if _fluidite:
		info.text += " · %d im/s · réseau ±%d ms" % [Engine.get_frames_per_second(), roundi(_ecart_reseau * 1000)]


func _show_toast(text: String) -> void:
	toast.text = text
	toast.modulate.a = 1.0
	var tw := create_tween()
	tw.tween_interval(2.5)
	tw.tween_property(toast, "modulate:a", 0.0, 0.6)


func _open_chat() -> void:
	chat_edit.visible = true
	chat_edit.grab_focus()


func _send_chat(text: String) -> void:
	text = text.strip_edges()
	if text != "":
		Net.send(Net.OP_CHAT, {"m": text})
	chat_edit.text = ""
	chat_edit.visible = false
	chat_edit.release_focus()


func _add_chat_line(who: String, text: String, system: bool = false) -> void:
	var l := RichTextLabel.new()
	l.bbcode_enabled = true
	l.fit_content = true
	l.scroll_active = false
	l.mouse_filter = Control.MOUSE_FILTER_IGNORE
	l.custom_minimum_size = Vector2(420, 0)
	l.add_theme_font_size_override("normal_font_size", 16)
	l.add_theme_constant_override("outline_size", 5)
	l.add_theme_color_override("font_outline_color", Color(0, 0, 0, 0.9))
	var safe := text.replace("[", "[lb]")
	if system:
		l.text = "[color=#%s]%s[/color]" % [UI.LUMEN.to_html(false), safe]
	else:
		l.text = "[color=#%s]%s[/color] : %s" % [UI.EMBER.to_html(false), who.replace("[", "[lb]"), safe]
	chat_log.add_child(l)
	while chat_log.get_child_count() > 7:
		chat_log.get_child(0).queue_free()
		chat_log.remove_child(chat_log.get_child(0))
	var tw := l.create_tween()
	tw.tween_interval(20)
	tw.tween_property(l, "modulate:a", 0.0, 2)


# --- Messages du serveur ------------------------------------------------------

## Pour les essais (--gigue=ms) : imite un réseau irrégulier, comme chez des amis
## loin du serveur. Les messages gardent leur ordre, mais arrivent en retard.
var _gigue_fin := 0.0
func _on_state_gigue(op: int, d: Dictionary) -> void:
	if gigue <= 0.0:
		_on_state(op, d)
		return
	_gigue_fin = maxf(_gigue_fin, _clock + randf() * gigue)
	get_tree().create_timer(_gigue_fin - _clock).timeout.connect(_on_state.bind(op, d))


func _on_state(op: int, d: Dictionary) -> void:
	if objets.on_state(op, d):
		return
	match op:
		Net.OP_SNAPSHOT:
			my_sid = int(d.you)
			if not _load_map(d.get("map", "glennmor")):
				_show_toast("Cette région manque dans ton jeu. Mets-le à jour !")
				return
			if d.has("sky"):
				sky.set_sky(d.sky)
			for p in d.players:
				_add_pawn(p)
			if pawns.has(my_sid) and _braise_wanted():
				_set_braise(true)
			var where := _map_name()
			for art in ["Le ", "La ", "Les "]:
				if where.begins_with(art):
					where = art.to_lower() + where.substr(art.length())
			_add_chat_line("", "Bienvenue dans %s ! Braise te souhaite une belle balade." % where, true)
			_show_toast(_map_name())
		Net.OP_JOIN:
			_add_pawn(d)
			if not d.has("npc") and not d.has("m"):
				_add_chat_line("", "%s arrive dans la région." % d.n, true)
		Net.OP_LEAVE:
			var sid := int(d.s)
			if pawns.has(sid):
				if sid == target_sid:
					_set_target(0)
				if pawns[sid].npc == "" and pawns[sid].kind == "":
					_add_chat_line("", "%s est parti." % pawns[sid].hero_name, true)
				elif dialogue.npc == pawns[sid].npc:
					dialogue.close()
				pawns[sid].queue_free()
				pawns.erase(sid)
				buffers.erase(sid)
				_assaillants.erase(sid)
				_update_info()
		Net.OP_DELTA:
			if _dernier_delta >= 0.0 and _clock - _dernier_delta < 0.5:   # sinon, rien ne bougeait
				_ecart_reseau = lerpf(_ecart_reseau, absf(_clock - _dernier_delta - TICK), 0.05)
			_dernier_delta = _clock
			for e in d.p:
				var sid := int(e[0])
				if buffers.has(sid):
					var p := Vector2(e[1], e[2]) / 100.0
					var buf: Array = buffers[sid]
					if buf[-1][1].distance_to(p) > 3.0:
						# Une porte : on saute directement de l'autre côté.
						buf.clear()
						buf.append([_clock - 1.0, p, int(e[3])])
						if sid == my_sid:
							_through_door()
							if braise != null:
								braise.snap(p)
					else:
						buf.append([_stamp(buf), p, int(e[3])])
		Net.OP_CHAT_OUT:
			var sid := int(d.s)
			_add_chat_line(d.n, d.m)
			if pawns.has(sid):
				pawns[sid].say(d.m)
		Net.OP_NOTICE:
			_add_chat_line("", d.m, true)
			_show_toast(d.m)
		Net.OP_WEATHER:
			sky.set_weather(d.w)
			_add_chat_line("", WEATHER_TEXT.get(d.w, ""), true)
			_update_info()
		Net.OP_EMOTE_OUT:
			var sid := int(d.s)
			if pawns.has(sid):
				pawns[sid].play(d.e)
		Net.OP_HITS:
			for h in d.h:
				_on_hit(int(h[0]), int(h[1]), int(h[2]), int(h[3]), int(h[4]))
		Net.OP_VITALS:
			for v in d.v:
				var sid := int(v[0])
				if pawns.has(sid):
					pawns[sid].set_vitals(int(v[1]), int(v[2]), int(v[3]), int(v[4]))
				if sid == my_sid:
					combat.set_vitals(int(v[1]), int(v[2]), int(v[3]), int(v[4]))
		Net.OP_ME:
			_me.merge(d, true)
			combat.set_me(d)
			_glow_sheet()
			if sheet != null and is_instance_valid(sheet) and sheet.public.is_empty():
				sheet.update_me(_me)
		Net.OP_FICHE:
			_close_sheet()
			sheet = HeroSheet.new()
			sheet.public = d
			sheet.closed.connect(func(): sheet = null)
			hud.get_child(0).add_child(sheet)
		Net.OP_VOIE_OUT:
			_on_voie(d)
		Net.OP_CAST:
			if int(d.s) == my_sid:
				combat.start_cast(int(d.k), int(d.ms), str(d.get("t", "")))
		Net.OP_ANIM:
			for a in d.a:
				_on_anim(int(a[0]), str(a[1]))
		Net.OP_DEATH:
			_on_death(int(d.s), int(d.k), int(d.f))
		Net.OP_LEVEL:
			_on_level(int(d.s), int(d.lv))
		Net.OP_LOOT:
			_on_loot(d)
		Net.OP_REFUSE:
			_show_toast(d.m)
		Net.OP_DUEL:
			_on_duel(d)
		Net.OP_TELE:
			var t := Telegraph.new()
			t.line = bool(d.l)
			t.a = Vector2(d.a[0], d.a[1]) / 100.0
			t.b = Vector2(d.b[0], d.b[1]) / 100.0
			t.r = float(d.r) / 100.0
			t.total = float(d.ms) / 1000.0
			tele_layer.add_child(t)
			if str(d.get("n", "")) != "":
				_float(int(d.s), str(d.n), Color("ff8a3c"), true)   # « Charge ! » : le coup qui arrive


func _add_pawn(p: Dictionary) -> void:
	var sid := int(p.s)
	if pawns.has(sid):
		# Un héros déjà là a changé d'allure (miroir de Solenn) ou de voie.
		var old: Pawn = pawns[sid]
		if p.has("r") and old.race != "":
			old.allure = p.get("lk", {})
			old.equipement = p.get("eq")
			old.voie = int(p.get("v", -1))
			old.stature = Heros.echelle(old.race, old.allure)
			old.dress_hero()
		return
	var pawn := Pawn.new()
	if p.has("r"):
		pawn.race = str(p.r)
		pawn.classe = str(p.get("cl", ""))
		pawn.voie = int(p.get("v", -1))
		pawn.dieu = str(p.get("dieu", ""))
		pawn.hardcore = bool(p.get("hc", false))
		pawn.allure = p.get("lk", {})
		pawn.equipement = p.get("eq")
	if p.has("npc"):
		var who := DialogueBox.info(map_id, p.npc)
		pawn.npc = p.npc
		pawn.look = who.get("peuple", who.get("look", pawn.look))
		pawn.stature = float(who.get("taille", 1.0))
	if p.has("m"):
		pawn.kind = p.m
		pawn.elite = bool(p.get("el", false))
	pawn.hero_name = p.n
	pawn.level = int(p.get("lv", 1))
	pawn.hp = int(p.get("hp", 1))
	pawn.hp_max = int(p.get("hpm", 1))
	pawn.fx = int(p.get("fx", 0))
	pawn.dead = p.has("hp") and int(p.hp) <= 0
	pawn.dir = int(p.d)
	pawn.is_me = sid == my_sid
	var wp := Vector2(p.x, p.y) / 100.0
	pawn.position = Iso.to_screen(wp)
	entities.add_child(pawn)
	pawns[sid] = pawn
	buffers[sid] = [[_clock - 1.0, wp, int(p.d)]]
	if pawn.is_me:
		camera.position = pawn.position
		camera.reset_smoothing()
		combat.set_vitals(pawn.hp, pawn.hp_max, pawn.fx, pawn.level)
		combat.hardcore = pawn.hardcore
		combat.accord = int(pawn.allure.get("a", 0))
		if pawn.classe != "":
			tutorial.shield = bool(Heros.classe(pawn.classe).bouclier)
	if sid == target_sid:
		pawn.targeted = true
	_update_info()


# --- Boucle -----------------------------------------------------------------

func _process(delta: float) -> void:
	_clock += delta
	_interpolate()
	_read_direction()
	if pawns.has(my_sid):
		sky.indoors = Iso.to_world(pawns[my_sid].position).y >= float(iso_map.data.get("indoors_from", 1e9))
	sky.update_torches(pawns, my_sid)
	_update_npcs(delta)
	_update_hunt()
	objets.process(delta)
	if duel_card.visible:
		_duel_t -= delta
		_duel_text.text = "⚔️ %s" % duel_card.get_meta("text") + "\n(%d s pour répondre)" % ceili(maxf(0, _duel_t))
		if _duel_t <= 0:
			duel_card.visible = false
	_info_t += delta
	if _info_t > 1.0:
		_info_t = 0.0
		miroir_btn.visible = _near_npc("clemence", MIROIR_REACH)
		_update_info()
		_update_objectif()
		_maybe_tutorial()
	if my_sid != -1 and pawns.has(my_sid):
		camera.position = pawns[my_sid].position
	# Un coup critique fait trembler l'écran ; le coup fatal d'un élite ralentit le temps (sur ton écran seulement).
	if _shake > 0:
		_shake = maxf(0, _shake - delta / maxf(0.05, Engine.time_scale))
		camera.position += Vector2(randf_range(-1, 1), randf_range(-1, 1)) * 6.0 * _shake / 0.25
	if _slowmo > 0:
		_slowmo -= delta / maxf(0.05, Engine.time_scale)
		Engine.time_scale = 0.3 if _slowmo > 0 else 1.0
	# Pendant un dialogue, on remonte la vue pour que la boîte ne cache pas la scène.
	camera.offset = camera.offset.lerp(Vector2(0, 120) if dialogue.visible else Vector2.ZERO, minf(1.0, delta * 6))
	if marker.visible:
		_marker_t += delta
		marker.modulate.a = clampf(1.2 - _marker_t, 0, 1)
		marker.queue_redraw()
		if _marker_t > 1.2:
			marker.visible = false


## L'heure d'une nouvelle position. Le réseau livre les images du monde un peu
## n'importe quand ; on les replace à intervalles réguliers (un tick), en se
## recalant doucement sur l'heure d'arrivée. Sans ça, un message en retard de
## quelques millisecondes faisait s'arrêter puis sauter les héros.
func _stamp(buf: Array) -> float:
	var prev: float = buf[-1][0]
	if _clock - prev > 2.5 * TICK:
		return _clock   # il ne bougeait plus : on repart de maintenant
	return clampf(lerpf(prev + TICK, _clock, 0.1), prev + TICK * 0.5, _clock + TICK)


func _interpolate() -> void:
	var t := _clock - INTERP_DELAY
	for sid in buffers:
		var buf: Array = buffers[sid]
		while buf.size() > 2 and buf[1][0] <= t:
			buf.pop_front()
		var pawn: Pawn = pawns[sid]
		var a = buf[0]
		var pos: Vector2 = a[1]
		var moving := false
		if buf.size() >= 2 and t > a[0]:
			var b = buf[1]
			var k := clampf((t - a[0]) / maxf(0.001, b[0] - a[0]), 0.0, 1.0)
			moving = a[1].distance_to(b[1]) > 0.001
			pos = a[1].lerp(b[1], k)
			if moving:
				pawn.running = a[1].distance_to(b[1]) / maxf(0.05, b[0] - a[0]) > RUN_SEEN
			pawn.dir = b[2]
		elif buf.size() == 1:
			pos = a[1]
			pawn.dir = a[2]
		pawn.walking = moving or (buf.size() >= 2 and t < buf[-1][0] + 0.15)
		pawn.position = Iso.to_screen(pos)


func _read_direction() -> void:
	var d := Vector2.ZERO
	if not chat_edit.has_focus():
		if Touches.pressed("haut") or Input.is_key_pressed(KEY_UP): d.y -= 1
		if Touches.pressed("bas") or Input.is_key_pressed(KEY_DOWN): d.y += 1
		if Touches.pressed("gauche") or Input.is_key_pressed(KEY_LEFT): d.x -= 1
		if Touches.pressed("droite") or Input.is_key_pressed(KEY_RIGHT): d.x += 1
		# Le stick gauche d'une manette.
		for j in Input.get_connected_joypads():
			var st := Vector2(Input.get_joy_axis(j, JOY_AXIS_LEFT_X), Input.get_joy_axis(j, JOY_AXIS_LEFT_Y))
			if st.length() > 0.3:
				d = st.limit_length(1.0)
	if joystick != null and joystick.value.length() > 0.2:
		d = joystick.value
	var w := Iso.screen_dir_to_world(d.normalized() if d.length() > 1 else d)
	w = (w * 20).round() / 20
	var run := _running()
	if (w != _last_dir or (run != _last_run and w != Vector2.ZERO)) and (_clock - _last_dir_sent > 0.05 or w == Vector2.ZERO):
		_last_dir = w
		_last_run = run
		_last_dir_sent = _clock
		Net.send(Net.OP_MOVE_DIR, {"d": [w.x, w.y], "r": run})


## Maj sur PC, ou le bouton « Courir » sur téléphone.
## Petit fondu au noir quand on passe une porte.
func _through_door() -> void:
	var fade := ColorRect.new()
	fade.color = Color(0, 0, 0, 1)
	fade.set_anchors_preset(Control.PRESET_FULL_RECT)
	fade.mouse_filter = Control.MOUSE_FILTER_IGNORE
	hud.add_child(fade)
	var tw := fade.create_tween()
	tw.tween_property(fade, "color:a", 0.0, 0.45)
	tw.tween_callback(fade.queue_free)
	camera.reset_smoothing.call_deferred()


func _running() -> bool:
	return _run_toggle or (not chat_edit.has_focus() and Touches.pressed("courir"))


func _wave() -> void:
	Net.send(Net.OP_EMOTE, {"e": "salut"})


func _zoom_by(k: float) -> void:
	var z := clampf(camera.zoom.x * k, ZOOM_MIN, ZOOM_MAX)
	camera.zoom = Vector2(z, z)


func _unhandled_input(event: InputEvent) -> void:
	if event is InputEventKey and event.pressed and not event.echo:
		if event.keycode == KEY_ENTER or event.keycode == KEY_KP_ENTER:
			if not chat_edit.visible:
				_open_chat()
				get_viewport().set_input_as_handled()
		elif event.keycode == KEY_ESCAPE and chat_edit.visible:
			chat_edit.visible = false
			chat_edit.release_focus()
		elif not chat_edit.has_focus() and _combat_key(event):
			get_viewport().set_input_as_handled()
			return
	elif (event is InputEventMouseButton or event is InputEventJoypadButton) and event.pressed \
			and not chat_edit.has_focus() and _combat_key(event):
		get_viewport().set_input_as_handled()
		return
	# Le bouclier (clic droit par défaut) : levé tant que le bouton est tenu.
	if not event.is_echo() and Touches.action_for_event(event) == "garde":
		_set_block(event.is_pressed())
		return
	if event is InputEventMouseButton and event.pressed:
		if event.button_index == MOUSE_BUTTON_WHEEL_UP:
			_zoom_by(1.1)
		elif event.button_index == MOUSE_BUTTON_WHEEL_DOWN:
			_zoom_by(1 / 1.1)
	if event is InputEventMagnifyGesture:
		_zoom_by(event.factor)
	var target = null
	if event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_LEFT:
		target = get_global_mouse_position()
	if target != null and objets.click(target):
		_talk_to = -1
		_hunt = 0
		return
	if target != null and _click_fighter(target):
		return
	if target != null and _click_npc(target):
		return
	if target != null:
		_talk_to = -1
		_hunt = 0
		var wp := Iso.to_world(target)
		if iso_map.walkable(wp):
			Net.send(Net.OP_MOVE_TO, {"t": [snappedf(wp.x, 0.01), snappedf(wp.y, 0.01)], "r": _running()})
			marker.position = target
			marker.visible = true
			_marker_t = 0.0
		else:
			_show_toast(["Braise dit : on ne marche pas sur l'eau. Pas encore.", "Braise dit : ce chemin est bloqué !"][randi() % 2])


func _draw_marker() -> void:
	var r := 6 + _marker_t * 14
	marker.draw_set_transform(Vector2.ZERO, 0, Vector2(1, 0.5))
	marker.draw_arc(Vector2.ZERO, r, 0, TAU, 32, UI.LUMEN, 2.5, true)
	marker.draw_set_transform(Vector2.ZERO, 0, Vector2.ONE)


# --- Les habitants et Braise ---------------------------------------------------

## Un clic sur un habitant : on lui parle, en allant d'abord vers lui s'il est loin.
func _click_npc(at: Vector2) -> bool:
	var best := -1
	var best_d := 30.0
	for sid in pawns:
		var pawn: Pawn = pawns[sid]
		if pawn.npc == "":
			continue
		# On vise le corps, pas les pieds.
		var d := at.distance_to(pawn.position + Vector2(0, -28 * pawn.stature))
		if d < best_d:
			best_d = d
			best = sid
	if best == -1:
		return false
	_talk_to = best
	_chase_t = -1e9
	get_viewport().set_input_as_handled()
	return true


func _near(sid: int, reach: float) -> bool:
	if not pawns.has(my_sid) or not pawns.has(sid):
		return false
	return Iso.to_world(pawns[my_sid].position).distance_to(Iso.to_world(pawns[sid].position)) <= reach


func _update_npcs(delta: float) -> void:
	if _talk_to != -1:
		_hunt = 0
		if not pawns.has(_talk_to):
			_talk_to = -1
		elif _near(_talk_to, TALK_REACH):
			var pawn: Pawn = pawns[_talk_to]
			_talk_to = -1
			Net.send(Net.OP_MOVE_DIR, {"d": [0, 0], "r": false})
			_talk_at = Iso.to_world(pawns[my_sid].position)
			dialogue.open(map_id, pawn.npc, pawn.hero_name)
		elif _clock - _chase_t > 1.0:
			# L'habitant bouge : on recalcule le trajet de temps en temps.
			_chase_t = _clock
			var wp := Iso.to_world(pawns[_talk_to].position)
			Net.send(Net.OP_MOVE_TO, {"t": [snappedf(wp.x, 0.01), snappedf(wp.y, 0.01)], "r": _running()})
	# Le dialogue se ferme si tu t'éloignes (l'habitant, lui, peut flâner un peu).
	if dialogue.visible and (not pawns.has(my_sid) or Iso.to_world(pawns[my_sid].position).distance_to(_talk_at) > 3.0):
		dialogue.close()
	if pawns.has(my_sid) and braise != null:
		braise.follow(Iso.to_world(pawns[my_sid].position), pawns[my_sid].dir, delta)
		if braise.wants_to_talk(delta):
			braise.say(_braise_remark())
	_near_t += delta
	if _near_t < 0.5:
		return
	_near_t = 0.0
	# Les habitants te lancent une bulle quand tu passes près d'eux.
	for sid in pawns:
		var pawn: Pawn = pawns[sid]
		if pawn.npc == "" or dialogue.npc == pawn.npc or not _near(sid, BUBBLE_REACH):
			continue
		if _clock - float(_bubble_at.get(sid, -1e9)) < BUBBLE_EVERY:
			continue
		var lines: Array = DialogueBox.info(map_id, pawn.npc).get("bulles", [])
		if not lines.is_empty():
			_bubble_at[sid] = _clock
			pawn.say(lines.pick_random())


## Ce que Braise trouve à dire, selon l'endroit et le moment.
func _braise_remark() -> String:
	var lines: Dictionary = DialogueBox.data(map_id).get("braise", {})
	var me := Iso.to_world(pawns[my_sid].position)
	var kind := "village"
	if sky.indoors:
		kind = "interieur"
	elif sky.weather in ["pluie", "orage"]:
		kind = "pluie"
	elif sky.is_night():
		kind = "nuit"
	elif _clock - _fight_t < 8.0:
		kind = "combat"
	elif me.x > iso_map.data.get("width", 200) * 0.8:
		kind = "lisiere" if randf() < 0.5 else "effaces"
	else:
		for sid in pawns:
			if pawns[sid].npc != "" and _near(sid, 5.0):
				kind = "pnj"
		if kind == "village" and me.distance_to(Vector2(95, 100)) > 30:
			kind = "foret"
	var pick: Array = lines.get(kind, lines.get("village", ["Braise bâille."]))
	return pick.pick_random()


func _braise_wanted() -> bool:
	var cfg := ConfigFile.new()
	cfg.load(WorldSky.SETTINGS)
	return bool(cfg.get_value("jeu", "braise", true))


func _set_braise(on: bool) -> void:
	var cfg := ConfigFile.new()
	cfg.load(WorldSky.SETTINGS)
	cfg.set_value("jeu", "braise", on)
	cfg.save(WorldSky.SETTINGS)
	if on and braise == null and pawns.has(my_sid):
		braise = Braise.new()
		entities.add_child(braise)
		braise.snap(Iso.to_world(pawns[my_sid].position))
		braise.say("Braise : « Me revoilà ! »" if _clock > 5 else "Braise : « En route, %s ! »" % hero.get("name", ""))
	elif not on and braise != null:
		braise.queue_free()
		braise = null


# --- Le combat ------------------------------------------------------------------

## Le tutoriel de combat : une seule fois, au premier monstre qui approche.
func _maybe_tutorial() -> void:
	if tutorial.running() or _tuto_done() or not pawns.has(my_sid) or dialogue.visible:
		return
	for sid in pawns:
		var p: Pawn = pawns[sid]
		if p.kind != "" and not p.dead and _near(sid, TUTO_REACH):
			_start_tutorial()
			return


func _start_tutorial() -> void:
	tutorial.start(pawns.has(target_sid) and pawns[target_sid].kind != "")
	if braise != null:
		braise.say("Braise : « Attention, un monstre ! Je t'explique, fais comme moi. »")


func _tuto(id: String) -> void:
	if tutorial != null and tutorial.did(id) and braise != null and tutorial.step == CombatTutorial.STEPS.size() - 1:
		braise.say("Braise fait un looping : « Tu sais te battre ! »")


func _glow(id) -> void:
	for b in combat.buttons:
		combat.buttons[b].glow = id != null and str(b) == str(id)


func _tuto_done() -> bool:
	var cfg := ConfigFile.new()
	cfg.load(WorldSky.SETTINGS)
	return bool(cfg.get_value("jeu", "tuto_combat", false))


func _set_tuto_done(on: bool) -> void:
	var cfg := ConfigFile.new()
	cfg.load(WorldSky.SETTINGS)
	cfg.set_value("jeu", "tuto_combat", on)
	cfg.save(WorldSky.SETTINGS)

## Les touches de combat (par défaut 1 à 8, R, Espace, Tab, C ; à changer dans ⌨ Touches,
## clavier, souris ou manette) et Échap.
func _combat_key(event: InputEvent) -> bool:
	var act := Touches.action_for_event(event)
	if act == "saluer":
		_wave()
		return true
	if dialogue.visible:
		return false   # 1 à 4 choisissent une réponse
	var k := Touches.code_of(event) if event is InputEventKey else 0
	if act.begins_with("comp_"):
		_use_skill(int(act.substr(5)))
	elif k >= KEY_KP_1 and k <= KEY_KP_8:
		_use_skill(k - KEY_KP_1 + 1)
	elif act == "fiche":
		if sheet != null:
			_close_sheet()
		else:
			_open_sheet()
	elif act == "potion":
		_potion()
	elif act == "sac":
		objets.toggle_sac()
	elif act == "rapide_1" or act == "rapide_2":
		objets.quick(int(act.substr(7)) - 1)
	elif act == "ramasser":
		objets.act_nearest()
	elif act == "roulade":
		_roll()
	elif act == "cible":
		_next_target()
	elif k == KEY_ESCAPE and target_sid != 0 and not dialogue.visible:
		_set_target(0)
	else:
		return false
	return true


func _use_skill(k: int) -> void:
	if pawns.has(my_sid) and pawns[my_sid].dead:
		return
	if k == 1:
		_tuto("frappe")
	# Une compétence au corps à corps sur un monstre trop loin : on y va d'abord.
	var sk := Heros.competence(str(_me.get("cl", "")), k, int(_me.get("v", -1)))
	if not sk.is_empty() and sk.cible == "ennemi" and float(sk.portee) == 0 and _hostile(target_sid) and not _in_reach(target_sid):
		_hunt = target_sid
		_hunt_t = -1e9
	Net.send(Net.OP_SKILL, {"k": k})


func _roll() -> void:
	_hunt = 0
	_tuto("roulade")
	Net.send(Net.OP_ROLL, {"d": [_last_dir.x, _last_dir.y]})


func _set_block(on: bool) -> void:
	if on == _blocking:
		return
	_blocking = on
	Net.send(Net.OP_BLOCK, {"b": on})
	if on:
		_tuto("garde")


func _potion() -> void:
	_tuto("potion")
	Net.send(Net.OP_POTION, {})


func _set_target(sid: int) -> void:
	if pawns.has(target_sid):
		pawns[target_sid].targeted = false
	target_sid = sid
	if pawns.has(sid):
		pawns[sid].targeted = true
		combat.target = pawns[sid]
		if pawns[sid].kind != "":
			_tuto("cible")
	else:
		target_sid = 0
		combat.target = null
	Net.send(Net.OP_TARGET, {"t": target_sid})


## Tab : le monstre vivant le plus proche, puis le suivant à chaque appui.
func _next_target() -> void:
	if not pawns.has(my_sid):
		return
	var me := Iso.to_world(pawns[my_sid].position)
	var list := []
	for sid in pawns:
		var p: Pawn = pawns[sid]
		if p.kind != "" and not p.dead:
			var d := me.distance_to(Iso.to_world(p.position))
			if d <= TARGET_RANGE:
				list.append([d, sid])
	if list.is_empty():
		_show_toast("Braise dit : aucun monstre en vue.")
		return
	list.sort_custom(func(a, b): return a[0] < b[0])
	var pick: int = list[0][1]
	for i in list.size():
		if list[i][1] == target_sid:
			pick = list[(i + 1) % list.size()][1]
	_set_target(pick)


func _in_reach(sid: int) -> bool:
	return _near(sid, MELEE_REACH)


## Un clic sur un monstre (on le vise et on y court) ou sur un autre héros (pour le soigner).
func _click_fighter(at: Vector2) -> bool:
	var best := -1
	var best_d := 34.0
	for sid in pawns:
		var p: Pawn = pawns[sid]
		if p.npc != "" or p.dead or sid == my_sid:
			continue
		var d := at.distance_to(p.position + Vector2(0, -16 if p.kind != "" else -28))
		if d < best_d:
			best_d = d
			best = sid
	if best == -1:
		return false
	_talk_to = -1
	_set_target(best)
	if _hostile(best):
		_hunt = best
		_hunt_t = -1e9
	get_viewport().set_input_as_handled()
	return true


## On court vers le monstre visé jusqu'à être à portée ; l'attaque de base fait le reste.
func _update_hunt() -> void:
	if _hunt == 0:
		return
	if not pawns.has(_hunt) or pawns[_hunt].dead or _last_dir != Vector2.ZERO or not pawns.has(my_sid):
		_hunt = 0
		return
	if _in_reach(_hunt):
		_hunt = 0
		Net.send(Net.OP_MOVE_DIR, {"d": [0, 0], "r": false})
	elif _clock - _hunt_t > 0.5:
		_hunt_t = _clock
		var wp := Iso.to_world(pawns[_hunt].position)
		Net.send(Net.OP_MOVE_TO, {"t": [snappedf(wp.x, 0.01), snappedf(wp.y, 0.01)], "r": true})


func _float(sid: int, text: String, color: Color, big: bool = false) -> void:
	if not pawns.has(sid):
		return
	var f := FloatText.new()
	f.text = text
	f.color = color
	f.big = big
	f.position = pawns[sid].position + Vector2(randf_range(-10, 10), -pawns[sid]._head() + 6)
	fx_layer.add_child(f)


func _on_hit(src: int, tgt: int, value: int, flags: int, skill: int) -> void:
	var crit := flags & 1 != 0
	var heal := flags & 2 != 0
	if flags & 8:
		_float(tgt, "Esquivé !", UI.MOSS)
		return
	if heal:
		if value > 0:
			_float(tgt, "+%d" % value, Color("7fe08a"), crit)
		return
	if pawns.has(tgt):
		pawns[tgt].flash()
	var color := Color.WHITE
	if tgt == my_sid:
		color = Color("ff6a5a")
	elif flags & 16:
		color = Color("f0a03c")   # brûlure ou poison
	elif skill == 1 or skill == 5:
		color = Color("ffe9a0")   # la lumière
	if crit:
		color = Color("ffd23a")
	var txt := str(value) + ("!" if crit else "")
	if flags & 4:
		txt += " 🛡"
	_float(tgt, txt, color, crit)
	if src == my_sid or tgt == my_sid:
		_fight_t = _clock
	if tgt == my_sid and _hostile(src):
		# Celui qui te frappe devient ta cible, sauf si ta cible actuelle se bat
		# déjà contre toi (sinon on sauterait de l'un à l'autre à chaque coup).
		_assaillants[src] = _clock
		if src != target_sid and not (_hostile(target_sid) and _t_attaque(target_sid)):
			_set_target(src)
	if crit and (src == my_sid or tgt == my_sid):
		_shake = 0.25
	if tgt == my_sid and braise != null and randf() < 0.06:
		braise.say(["Braise : « Aïe ! Je souffle sur la plaie ! »", "Braise : « Lève ton bouclier ! %s ! »" % Touches.label("garde"), "Braise : « Roule ! Roule ! »"].pick_random())


func _on_anim(sid: int, anim: String) -> void:
	if not pawns.has(sid):
		return
	var p: Pawn = pawns[sid]
	match anim:
		"releve":
			p.revive()
			if sid == my_sid:
				combat.show_death(false)
				_show_toast("💔 Faiblesse : −25 % de dégâts pendant 2 minutes.")
				if braise != null:
					braise.say("Braise : « Te revoilà ! Sœur Clémence dit de faire attention. »")
		"aura":
			p.play("sort")
		"potion":
			_float(sid, "🧪", UI.MOSS)
		_:
			p.play(anim)


func _on_death(sid: int, killer: int, flags: int) -> void:
	if not pawns.has(sid):
		return
	var p: Pawn = pawns[sid]
	p.die()
	_assaillants.erase(sid)
	if sid == target_sid and p.kind != "":
		_set_target(_prochain_assaillant())
	if flags & 32 and (killer == my_sid or _near(sid, 12)):
		_slowmo = 0.6
		_shake = 0.3
	var accord := int(p.allure.get("a", 0))
	var tombe: String = ["tombé", "tombée", "tombé·e"][clampi(accord, 0, 2)]
	if sid == my_sid:
		combat.show_death(true)
		_hunt = 0
		if flags & 64:
			# Hardcore : le héros est effacé ; retour à l'écran des héros.
			if braise != null:
				braise.say("Braise pleure une petite flamme : « Je me souviendrai de toi. »")
			get_tree().create_timer(6.0).timeout.connect(func(): quit_world.emit())
		elif braise != null:
			braise.say("Braise : « Non ! Je reste avec toi jusqu'au temple. »")
	elif p.kind == "" and not flags & 64:
		_add_chat_line("", "%s est %s au combat." % [p.hero_name, tombe], true)


func _on_level(sid: int, lv: int) -> void:
	if not pawns.has(sid):
		return
	pawns[sid].level = lv
	pawns[sid]._refresh_name()
	_float(sid, "Niveau %d !" % lv, UI.LUMEN, true)
	if sid == my_sid:
		_show_toast("✨ Niveau %d ! Tu te sens plus fort." % lv)
		if Heros.palier_au_niveau(lv) and (lv < int(Heros.data().voie) + int(Heros.data().talents.pas) or int(_me.get("v", -1)) >= 0):
			_show_toast("🌳 Un nouveau palier de talents s'ouvre ! Fiche du héros (C), onglet Talents.")
			if braise != null:
				braise.say("Braise : « NIVEAU %d ! Et un nouveau palier de talents, regarde ta fiche ! »" % lv)
		elif braise != null:
			braise.say("Braise fait un looping : « NIVEAU %d ! »" % lv)
	else:
		_add_chat_line("", "%s passe au niveau %d !" % [pawns[sid].hero_name, lv], true)


func _on_loot(d: Dictionary) -> void:
	var line := "+%d XP · +%s" % [int(d.xp), Objets.monnaie(int(d.get("or", 0)), true)]
	_add_chat_line("", line, true)
	var tro := str(d.get("tr", ""))
	if tro != "":
		_show_toast("🏆 Trophée : %s !" % tro)
		_add_chat_line("", "🏆 Tu ramasses : %s." % tro, true)
		if braise != null:
			braise.say("Braise : « Oh, un %s ! On l'accroche où ? »" % tro.to_lower())
	if pawns.has(int(d.m)):
		_float(int(d.m), "+" + Objets.monnaie(int(d.get("or", 0))), UI.LUMEN)


# --- Les duels ------------------------------------------------------------------

## Un monstre, ou ton adversaire en duel : on peut lui courir dessus et le frapper.
## Le monstre qui t'a frappé le plus récemment (ces 6 dernières secondes), ou 0.
func _prochain_assaillant() -> int:
	var best := 0
	var quand := _clock - 6.0
	for sid in _assaillants:
		if _assaillants[sid] > quand and _hostile(sid):
			best = sid
			quand = _assaillants[sid]
	return best


## Ce monstre t'a-t-il frappé ces 6 dernières secondes ?
func _t_attaque(sid: int) -> bool:
	return float(_assaillants.get(sid, -1e9)) > _clock - 6.0


func _hostile(sid: int) -> bool:
	return pawns.has(sid) and not pawns[sid].dead and (pawns[sid].kind != "" or sid == _duel_foe)


func _build_duel_card(root: Control) -> void:
	duel_card = PanelContainer.new()
	duel_card.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.94), UI.LUMEN, 14, 2, 16))
	duel_card.set_anchors_and_offsets_preset(Control.PRESET_CENTER_TOP, Control.PRESET_MODE_MINSIZE, 190)
	duel_card.grow_horizontal = Control.GROW_DIRECTION_BOTH
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 10)
	duel_card.add_child(col)
	_duel_text = UI.label("", 19, UI.LUMEN)
	_duel_text.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_duel_text.custom_minimum_size = Vector2(380, 0)
	col.add_child(_duel_text)
	var row := HBoxContainer.new()
	row.alignment = BoxContainer.ALIGNMENT_CENTER
	row.add_theme_constant_override("separation", 12)
	col.add_child(row)
	for b in [["⚔️ J'accepte", true], ["Non merci", false]]:
		var btn := UI.button(b[0], b[1])
		btn.focus_mode = Control.FOCUS_NONE
		btn.pressed.connect(func():
			Net.send(Net.OP_DUEL_REPLY, {"ok": b[1]})
			duel_card.visible = false)
		row.add_child(btn)
	duel_card.visible = false
	root.add_child(duel_card)


func _on_duel(d: Dictionary) -> void:
	var a := int(d.get("a", 0))
	var b := int(d.get("b", 0))
	var id := int(d.get("id", 0))
	var mine := a == my_sid or b == my_sid
	match str(d.k):
		"defi":
			duel_card.set_meta("text", "%s te défie en duel !" % d.get("n", "Un héros"))
			_duel_t = float(d.get("sec", 30))
			duel_card.visible = true
			if braise != null:
				braise.say("Braise : « Un duel ? Montre-lui ce que vaut un Paladin ! »")
		"debut":
			var ring := DuelRing.new()
			ring.id = id
			ring.center = Vector2(d.c[0], d.c[1]) / 100.0
			ring.r = float(d.r) / 100.0
			tele_layer.add_child(ring)
			_duel_rings[id] = ring
			if mine:
				_duel_foe = b if a == my_sid else a
				_duel_id = id
				combat.in_duel = true
				duel_card.visible = false
				if pawns.has(_duel_foe):
					pawns[_duel_foe].foe = true
					_set_target(_duel_foe)
				_show_toast("⚔️ Duel ! Reste dans le cercle… %d" % int(d.get("sec", 3)))
			elif pawns.has(a) and pawns.has(b):
				_add_chat_line("", "⚔️ %s et %s commencent un duel !" % [pawns[a].hero_name, pawns[b].hero_name], true)
		"compte":
			if mine:
				var n := int(d.get("n", 0))
				_show_toast("⚔️ Combattez !" if n <= 0 else "⚔️ %d…" % n)
		"fin":
			if _duel_rings.has(id):
				_duel_rings[id].queue_free()
				_duel_rings.erase(id)
			var w := a
			var l := b
			var wn: String = pawns[w].hero_name if pawns.has(w) else "?"
			var ln: String = pawns[l].hero_name if pawns.has(l) else "?"
			var line := ""
			match int(d.get("f", 0)):
				0: line = "⚔️ %s remporte le duel contre %s !" % [wn, ln]
				1: line = "⚔️ %s sort du cercle : %s gagne par abandon." % [ln, wn]
				2: line = "⚔️ Le temps est écoulé : match nul !"
				_: line = "⚔️ Le duel est interrompu."
			_add_chat_line("", line, true)
			if id == _duel_id and _duel_foe != 0:
				if pawns.has(_duel_foe):
					pawns[_duel_foe].foe = false
				if target_sid == _duel_foe:
					_set_target(0)
				_duel_foe = 0
				_duel_id = 0
				combat.in_duel = false
				if w == my_sid:
					_show_toast("🏆 Victoire !")
					if braise != null:
						braise.say("Braise fait un looping : « On a gagné ! On a gagné ! »")
				elif l == my_sid:
					_show_toast("Défaite… tu feras mieux la prochaine fois.")
					if braise != null:
						braise.say("Braise : « Pas grave. On s'entraîne sur les sangliers ? »")
				else:
					_show_toast(line)


# --- Le héros : fiche, voie, miroir de Solenn (Acte IV) --------------------------

func _open_sheet() -> void:
	_close_sheet()
	sheet = HeroSheet.new()
	sheet.me = _me
	if pawns.has(my_sid):
		sheet.hero = {"race": pawns[my_sid].race}
	sheet.attrs.connect(func(a): Net.send(Net.OP_ATTRS, {"a": a}))
	sheet.voie.connect(func(i): Net.send(Net.OP_VOIE, {"v": i}))
	sheet.talents.connect(func(t): Net.send(Net.OP_TALENTS, {"t": t}))
	sheet.closed.connect(func(): sheet = null)
	hud.get_child(0).add_child(sheet)


## La prochaine étape conseillée (res://data/objectifs/<carte>.json).
func _update_objectif() -> void:
	var path := "res://data/objectifs/%s.json" % map_id
	var data: Dictionary = {}
	if FileAccess.file_exists(path):
		data = JSON.parse_string(FileAccess.get_file_as_string(path))
	var texte := ""
	if not data.is_empty() and not _me.is_empty():
		var lv := int(_me.get("lv", 1))
		var v := int(_me.get("v", -1))
		var etape: Dictionary = {}
		if lv >= int(Heros.data().voie):
			if v < 0 and int(_me.get("q", 0)) < int(_me.get("qm", 5)):
				etape = data.epreuve
				texte = str(etape.texte) % [int(_me.get("q", 0)), int(_me.get("qm", 5))]
			elif v < 0:
				etape = data.voie
			else:
				etape = data.boss
		else:
			for e in data.etapes:
				if lv <= int(e.jusqu_au):
					etape = e
					break
		if texte == "" and etape.has("texte"):
			texte = str(etape.texte)
	objectif.text = "🎯 " + texte
	objectif.get_parent().visible = texte != ""


## Le point lumineux sur « 📜 Héros » : des points de talent attendent d'être placés.
func _glow_sheet() -> void:
	if sheet_btn == null:
		return
	var lv := int(_me.get("lv", 1))
	var tree := Heros.arbre(str(_me.get("cl", "")), int(_me.get("v", -1)))
	var ta := Heros.talents_ints(_me.get("ta", []))
	var room := 0
	for i in tree.size():
		if lv >= int(tree[i].niveau):
			for j in 3:
				room += int(tree[i].talents[j].max) - int(ta[i][j])
	var waiting := mini(room, Heros.points_de_talent(lv) - Heros.points_places(ta)) > 0
	sheet_btn.text = "📜 Héros ✦" if waiting else "📜 Héros"
	if waiting:
		sheet_btn.add_theme_color_override("font_color", UI.LUMEN)
	else:
		sheet_btn.remove_theme_color_override("font_color")


func _close_sheet() -> void:
	if sheet != null and is_instance_valid(sheet):
		sheet.queue_free()
	sheet = null


func _near_npc(id: String, reach: float) -> bool:
	for sid in pawns:
		if pawns[sid].npc == id:
			return _near(sid, reach)
	return false


func _open_miroir() -> void:
	if _miroir != null or not pawns.has(my_sid):
		return
	var me: Pawn = pawns[my_sid]
	_miroir = Miroir.new()
	_miroir.mode = "solenn"
	_miroir.price = 0 if int(_me.get("mir", 0)) == 0 else int(Heros.data().miroir)
	_miroir.form = {"race": me.race, "classe": me.classe, "look": me.allure.duplicate(true), "eq": me.equipement}
	_miroir.cancelled.connect(func():
		_miroir.queue_free()
		_miroir = null)
	_miroir.done.connect(func(form):
		Net.send(Net.OP_MIROIR, {"look": form.look})
		_miroir.queue_free()
		_miroir = null
		_show_toast("🪞 Le miroir de Solenn scintille…"))
	hud.get_child(0).add_child(_miroir)


func _on_voie(d: Dictionary) -> void:
	var n := int(d.get("n", 0))
	var mx := int(d.get("max", 5))
	match str(d.k):
		"quete":
			_show_toast("🌟 Les voies s'ouvrent ! Épreuve : vaincs %d Effacés." % mx)
			_add_chat_line("", "🌟 L'épreuve de la voie commence : vaincs %d Effacés (%d / %d)." % [mx, n, mx], true)
			if braise != null:
				braise.say("Braise : « Niveau 10 ! Les Effacés t'attendent, et ta voie aussi. »")
		"progres":
			_add_chat_line("", "🗡️ Effacé vaincu : %d / %d." % [n, mx], true)
		"pret":
			_show_toast("✨ Épreuve réussie ! Appuie sur C pour choisir ta voie.")
			_add_chat_line("", "✨ L'épreuve est réussie ! Ouvre ta fiche (📜 Héros, touche C) pour choisir ta voie.", true)
		"choisie":
			var v := Heros.competence(str(_me.get("cl", "")), 8, n)
			for sp in Heros.classe(str(_me.get("cl", ""))).voies:
				if sp.ultime.id == v.get("id", ""):
					_show_toast("🌟 Tu suis la voie : %s ! Ultime : %s (touche 8)." % [sp.nom, sp.ultime.nom])
			if braise != null:
				braise.say("Braise fait trois loopings : « Quelle voie ! »")
