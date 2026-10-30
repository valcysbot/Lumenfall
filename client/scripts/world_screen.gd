extends Node2D
## Le monde : la carte, les héros (le tien et ceux des autres), les
## commandes (clic, ZQSD, joystick) et le chat.
##
## Le client n'envoie que des intentions. Le serveur calcule les positions
## et les renvoie 10 fois par seconde ; on les adoucit à l'écran.

signal quit_world

const INTERP_DELAY := 0.11  # secondes de retard volontaire pour un mouvement fluide
const RUN_SEEN := 5.2       # au-dessus de cette vitesse (cases/s), on montre la course
const ZOOM_MIN := 0.8
const ZOOM_MAX := 2.2

var hero := {}

var iso_map: IsoMap
var sky: WorldSky
var entities: Node2D
var camera: Camera2D
var marker: Node2D
var pawns := {}       # sid -> Pawn
var buffers := {}     # sid -> Array de [temps, position monde]
var my_sid := -1

var hud: CanvasLayer
var info: Label
var chat_log: VBoxContainer
var chat_edit: LineEdit
var joystick: TouchStick
var run_btn: Button
var settings: PanelContainer
var toast: Label

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
	marker = Node2D.new()
	marker.draw.connect(_draw_marker)
	marker.visible = false
	add_child(marker)
	entities = Node2D.new()
	entities.y_sort_enabled = true
	add_child(entities)
	camera = Camera2D.new()
	camera.position_smoothing_enabled = true
	camera.position_smoothing_speed = 8.0
	add_child(camera)
	camera.make_current()
	sky = WorldSky.new()
	add_child(sky)
	_build_hud()

	camera.position = Iso.to_screen(Vector2(hero.x, hero.y))
	Net.match_state.connect(_on_state)
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

	var buttons := HBoxContainer.new()
	buttons.add_theme_constant_override("separation", 8)
	buttons.set_anchors_and_offsets_preset(Control.PRESET_TOP_RIGHT, Control.PRESET_MODE_MINSIZE, 14)
	buttons.grow_horizontal = Control.GROW_DIRECTION_BEGIN
	var wave_btn := UI.button("👋 Saluer")
	wave_btn.tooltip_text = "Touche H"
	wave_btn.pressed.connect(_wave)
	buttons.add_child(wave_btn)
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
	_build_settings(root)

	var touch := DisplayServer.is_touchscreen_available()
	chat_log = VBoxContainer.new()
	chat_log.mouse_filter = Control.MOUSE_FILTER_IGNORE
	chat_log.set_anchors_and_offsets_preset(Control.PRESET_BOTTOM_LEFT, Control.PRESET_MODE_MINSIZE, 14)
	chat_log.grow_vertical = Control.GROW_DIRECTION_BEGIN
	chat_log.custom_minimum_size = Vector2(420, 0)
	chat_log.position.y -= 190 if touch else 60
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
		side.set_anchors_and_offsets_preset(Control.PRESET_BOTTOM_RIGHT, Control.PRESET_MODE_MINSIZE, 24)
		side.grow_horizontal = Control.GROW_DIRECTION_BEGIN
		side.grow_vertical = Control.GROW_DIRECTION_BEGIN
		for z in [["＋", 1.25], ["－", 0.8]]:
			var zb := UI.button(z[0])
			zb.custom_minimum_size = Vector2(64, 56)
			zb.pressed.connect(_zoom_by.bind(z[1]))
			side.add_child(zb)
		run_btn = UI.button("🏃 Courir")
		run_btn.toggle_mode = true
		run_btn.custom_minimum_size = Vector2(120, 72)
		run_btn.toggled.connect(func(on): _run_toggle = on)
		side.add_child(run_btn)
		root.add_child(side)

	toast = UI.label("", 20, UI.LUMEN)
	toast.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	toast.autowrap_mode = TextServer.AUTOWRAP_OFF
	toast.set_anchors_and_offsets_preset(Control.PRESET_CENTER_TOP, Control.PRESET_MODE_MINSIZE, 70)
	toast.grow_horizontal = Control.GROW_DIRECTION_BOTH
	toast.add_theme_color_override("font_outline_color", UI.NIGHT)
	toast.add_theme_constant_override("outline_size", 6)
	root.add_child(toast)
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
	root.add_child(settings)


## Charge la carte annoncée par le serveur : la scène Godot (sol, décors) si
## elle existe, et toujours les cases praticables pour les clics.
func _load_map(id: String) -> bool:
	if not iso_map.load_map(id):
		return false
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
	info.text = "%s · %s · %s · %d %s" % [hero.get("name", ""), _map_name(), sky.describe(), pawns.size(), "aventurier en ligne" if pawns.size() <= 1 else "aventuriers en ligne"]


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

func _on_state(op: int, d: Dictionary) -> void:
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
			var where := _map_name()
			for art in ["Le ", "La ", "Les "]:
				if where.begins_with(art):
					where = art.to_lower() + where.substr(art.length())
			_add_chat_line("", "Bienvenue dans %s ! Braise te souhaite une belle balade." % where, true)
			_show_toast(_map_name())
		Net.OP_JOIN:
			_add_pawn(d)
			_add_chat_line("", "%s arrive dans la région." % d.n, true)
		Net.OP_LEAVE:
			var sid := int(d.s)
			if pawns.has(sid):
				_add_chat_line("", "%s est parti." % pawns[sid].hero_name, true)
				pawns[sid].queue_free()
				pawns.erase(sid)
				buffers.erase(sid)
				_update_info()
		Net.OP_DELTA:
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
					else:
						buf.append([_clock, p, int(e[3])])
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
				pawns[sid].sprite.play_once(d.e)


func _add_pawn(p: Dictionary) -> void:
	var sid := int(p.s)
	if pawns.has(sid):
		return
	var pawn := Pawn.new()
	pawn.look = HeroSprite.look_for(p.n, int(p.c))
	pawn.hero_name = p.n
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
	_update_info()


# --- Boucle -----------------------------------------------------------------

func _process(delta: float) -> void:
	_clock += delta
	_interpolate()
	_read_direction()
	if pawns.has(my_sid):
		sky.indoors = Iso.to_world(pawns[my_sid].position).y >= float(iso_map.data.get("indoors_from", 1e9))
	sky.update_torches(pawns, my_sid)
	_info_t += delta
	if _info_t > 1.0:
		_info_t = 0.0
		_update_info()
	if my_sid != -1 and pawns.has(my_sid):
		camera.position = pawns[my_sid].position
	if marker.visible:
		_marker_t += delta
		marker.modulate.a = clampf(1.2 - _marker_t, 0, 1)
		marker.queue_redraw()
		if _marker_t > 1.2:
			marker.visible = false


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
			pos = a[1].lerp(b[1], k)
			moving = a[1].distance_to(b[1]) > 0.001
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
		if Input.is_physical_key_pressed(KEY_W) or Input.is_key_pressed(KEY_UP): d.y -= 1
		if Input.is_physical_key_pressed(KEY_S) or Input.is_key_pressed(KEY_DOWN): d.y += 1
		if Input.is_physical_key_pressed(KEY_A) or Input.is_key_pressed(KEY_LEFT): d.x -= 1
		if Input.is_physical_key_pressed(KEY_D) or Input.is_key_pressed(KEY_RIGHT): d.x += 1
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
	return _run_toggle or (not chat_edit.has_focus() and Input.is_key_pressed(KEY_SHIFT))


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
		elif event.physical_keycode == KEY_H and not chat_edit.has_focus():
			_wave()
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
	if target != null:
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
