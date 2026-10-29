extends Node2D
## Le monde : la carte, les héros (le tien et ceux des autres), les
## commandes (clic, ZQSD, joystick) et le chat.
##
## Le client n'envoie que des intentions. Le serveur calcule les positions
## et les renvoie 10 fois par seconde ; on les adoucit à l'écran.

signal quit_world

const INTERP_DELAY := 0.11  # secondes de retard volontaire pour un mouvement fluide

var hero := {}
var colors: Array = []

var iso_map: IsoMap
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
var toast: Label

var _last_dir := Vector2.ZERO
var _last_dir_sent := 0.0
var _clock := 0.0
var _marker_t := 0.0


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
	_build_hud()

	var list := await Net.hero_list()
	colors = list.get("colors", [])
	if not iso_map.load_map(hero.get("map", "clairiere")):
		_show_toast("Cette carte manque dans ton jeu. Mets-le à jour !")
		return
	iso_map.spawn_props(entities)
	camera.position = Iso.to_screen(Vector2(hero.x, hero.y))

	Net.match_state.connect(_on_state)
	_show_toast("Entrée dans la Clairière…")
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
	var chat_btn := UI.button("💬 Parler")
	chat_btn.pressed.connect(_open_chat)
	buttons.add_child(chat_btn)
	var quit := UI.button("Quitter")
	quit.pressed.connect(func(): quit_world.emit())
	buttons.add_child(quit)
	root.add_child(buttons)

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

	toast = UI.label("", 20, UI.LUMEN)
	toast.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	toast.set_anchors_and_offsets_preset(Control.PRESET_CENTER_TOP, Control.PRESET_MODE_MINSIZE, 70)
	toast.grow_horizontal = Control.GROW_DIRECTION_BOTH
	toast.add_theme_color_override("font_outline_color", UI.NIGHT)
	toast.add_theme_constant_override("outline_size", 6)
	root.add_child(toast)
	_update_info()


func _update_info() -> void:
	info.text = "%s · La Clairière · %d %s" % [hero.get("name", ""), pawns.size(), "aventurier en ligne" if pawns.size() <= 1 else "aventuriers en ligne"]


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
			for p in d.players:
				_add_pawn(p)
			_add_chat_line("", "Bienvenue dans la Clairière ! Braise te souhaite une belle balade.", true)
			_show_toast("La Clairière")
		Net.OP_JOIN:
			_add_pawn(d)
			_add_chat_line("", "%s arrive dans la Clairière." % d.n, true)
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
					buffers[sid].append([_clock, Vector2(e[1], e[2]) / 100.0, int(e[3])])
		Net.OP_CHAT_OUT:
			var sid := int(d.s)
			_add_chat_line(d.n, d.m)
			if pawns.has(sid):
				pawns[sid].say(d.m)
		Net.OP_NOTICE:
			_add_chat_line("", d.m, true)
			_show_toast(d.m)


func _add_pawn(p: Dictionary) -> void:
	var sid := int(p.s)
	if pawns.has(sid):
		return
	var pawn := Pawn.new()
	var ci := int(p.c)
	pawn.color = Color(colors[ci]) if ci < colors.size() else Color.WHITE
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
	if w != _last_dir and (_clock - _last_dir_sent > 0.05 or w == Vector2.ZERO):
		_last_dir = w
		_last_dir_sent = _clock
		Net.send(Net.OP_MOVE_DIR, {"d": [w.x, w.y]})


func _unhandled_input(event: InputEvent) -> void:
	if event is InputEventKey and event.pressed and not event.echo:
		if event.keycode == KEY_ENTER or event.keycode == KEY_KP_ENTER:
			if not chat_edit.visible:
				_open_chat()
				get_viewport().set_input_as_handled()
		elif event.keycode == KEY_ESCAPE and chat_edit.visible:
			chat_edit.visible = false
			chat_edit.release_focus()
	var target = null
	if event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_LEFT:
		target = get_global_mouse_position()
	if target != null:
		var wp := Iso.to_world(target)
		if iso_map.walkable(wp):
			Net.send(Net.OP_MOVE_TO, {"t": [snappedf(wp.x, 0.01), snappedf(wp.y, 0.01)]})
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
