class_name SkillButton
extends Control
## Un bouton de compétence : une icône, sa touche, et la recharge qui se vide
## comme une horloge. Rond sur téléphone, carré sur PC. Marche au doigt même
## quand l'autre pouce tient le joystick.

signal activated
signal held(down: bool)

var icon := ""
var key := ""
var round_shape := false
var tint := UI.LUMEN
var cd_end := 0.0       # en secondes (Time.get_ticks_msec() / 1000)
var cd_total := 1.0
var dim := false        # pas assez de Ferveur, plus de potions…
var count := -1         # petit nombre en coin (potions), -1 : rien
var pips := -1.0        # roulades disponibles (0 à 2), -1 : rien
var _down := false
var _press_t := 0.0
var _icon_label: Label


func _init(p_icon: String = "", p_key: String = "", p_size: float = 60.0, p_round: bool = false) -> void:
	icon = p_icon
	key = p_key
	round_shape = p_round
	custom_minimum_size = Vector2(p_size, p_size)
	size = custom_minimum_size
	mouse_filter = Control.MOUSE_FILTER_STOP


func _ready() -> void:
	_icon_label = Label.new()
	_icon_label.text = icon
	_icon_label.add_theme_font_size_override("font_size", int(size.x * 0.45))
	_icon_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_icon_label.vertical_alignment = VERTICAL_ALIGNMENT_CENTER
	_icon_label.set_anchors_preset(Control.PRESET_FULL_RECT)
	_icon_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
	add_child(_icon_label)


static func now() -> float:
	return Time.get_ticks_msec() / 1000.0


func start_cd(seconds: float, total: float) -> void:
	if seconds <= 0:
		cd_end = 0
		return
	cd_end = now() + seconds
	cd_total = maxf(total, seconds)


func _gui_input(event: InputEvent) -> void:
	var pressed = null
	if event is InputEventScreenTouch:
		pressed = event.pressed
	elif event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_LEFT:
		pressed = event.pressed
	if pressed == null:
		return
	accept_event()
	if pressed and not _down:
		_down = true
		_press_t = 0.15
		activated.emit()
		held.emit(true)
	elif not pressed and _down:
		_down = false
		held.emit(false)


func _process(delta: float) -> void:
	_press_t = maxf(0, _press_t - delta)
	_icon_label.modulate = Color(1, 1, 1, 0.45 if dim else 1.0)
	queue_redraw()


func _draw() -> void:
	var r := Rect2(Vector2.ZERO, size)
	var c := size / 2
	var rad := size.x / 2
	var bg := Color(UI.NIGHT, 0.82)
	var border := tint if not dim else UI.LINE
	if _down or _press_t > 0:
		bg = Color(tint, 0.35)
	if round_shape:
		draw_circle(c, rad, bg)
		draw_arc(c, rad - 1, 0, TAU, 48, Color(border, 0.9), 2.0, true)
	else:
		draw_style_box(UI.box(bg, Color(border, 0.9), 10, 2, 0), r)
	var left := cd_end - now()
	if left > 0:
		# La recharge : une part d'ombre qui tourne.
		var k := clampf(left / cd_total, 0, 1)
		var pts := PackedVector2Array([c])
		var steps := 32
		for i in steps + 1:
			var a := -PI / 2 + TAU * k * float(i) / steps
			pts.append(c + Vector2(cos(a), sin(a)) * rad * (1.0 if round_shape else 1.45))
		if not round_shape:
			for i in pts.size():
				pts[i] = Vector2(clampf(pts[i].x, 2, size.x - 2), clampf(pts[i].y, 2, size.y - 2))
		draw_colored_polygon(pts, Color(0, 0, 0, 0.62))
		if left >= 1.0:
			var font := ThemeDB.fallback_font
			var txt := str(int(ceil(left)))
			var fs := int(size.x * 0.3)
			var tw := font.get_string_size(txt, HORIZONTAL_ALIGNMENT_LEFT, -1, fs)
			draw_string_outline(font, c + Vector2(-tw.x / 2, fs * 0.35), txt, HORIZONTAL_ALIGNMENT_LEFT, -1, fs, 4, Color(0, 0, 0, 0.9))
			draw_string(font, c + Vector2(-tw.x / 2, fs * 0.35), txt, HORIZONTAL_ALIGNMENT_LEFT, -1, fs, UI.PARCH)
	var f := ThemeDB.fallback_font
	if key != "":
		draw_string_outline(f, Vector2(5, 15), key, HORIZONTAL_ALIGNMENT_LEFT, -1, 13, 3, Color(0, 0, 0, 0.9))
		draw_string(f, Vector2(5, 15), key, HORIZONTAL_ALIGNMENT_LEFT, -1, 13, UI.MUTED)
	if count >= 0:
		var t := str(count)
		draw_string_outline(f, Vector2(size.x - 14, size.y - 6), t, HORIZONTAL_ALIGNMENT_LEFT, -1, 15, 3, Color(0, 0, 0, 0.9))
		draw_string(f, Vector2(size.x - 14, size.y - 6), t, HORIZONTAL_ALIGNMENT_LEFT, -1, 15, UI.PARCH)
	if pips >= 0:
		for i in 2:
			var p := c + Vector2((i - 0.5) * 14, rad - 8)
			var fill := clampf(pips - i, 0, 1)
			draw_circle(p, 4.5, Color(0, 0, 0, 0.8))
			if fill >= 1:
				draw_circle(p, 3.5, UI.MOSS)
			elif fill > 0:
				draw_arc(p, 3, -PI / 2, -PI / 2 + TAU * fill, 16, UI.MOSS, 2)
