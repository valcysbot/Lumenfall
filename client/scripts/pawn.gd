class_name Pawn
extends Node2D
## Le héros de l'Acte I : un pion propre et provisoire, avec son nom,
## sa direction et ses bulles de chat. Le vrai héros en couches arrive à l'Acte II.

var color := Color.WHITE
var hero_name := ""
var dir := 2
var is_me := false
var walking := false

var _bob := 0.0
var _name_label: Label
var _bubble: PanelContainer
var _bubble_label: Label
var _bubble_time := 0.0


func _ready() -> void:
	_name_label = Label.new()
	_name_label.text = hero_name
	_name_label.add_theme_font_size_override("font_size", 15)
	_name_label.add_theme_color_override("font_color", UI.LUMEN if is_me else UI.PARCH)
	_name_label.add_theme_color_override("font_outline_color", Color(0, 0, 0, 0.85))
	_name_label.add_theme_constant_override("outline_size", 5)
	_name_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	add_child(_name_label)
	_name_label.reset_size()
	_name_label.position = Vector2(-_name_label.size.x / 2, -74)

	_bubble = PanelContainer.new()
	_bubble.add_theme_stylebox_override("panel", UI.box(Color(UI.PARCH, 0.95), Color(UI.NIGHT, 0.4), 10, 1, 6))
	_bubble_label = Label.new()
	_bubble_label.add_theme_font_size_override("font_size", 15)
	_bubble_label.add_theme_color_override("font_color", UI.NIGHT)
	_bubble.add_child(_bubble_label)
	_bubble.visible = false
	add_child(_bubble)


func say(text: String) -> void:
	_bubble_label.text = _wrap(text, 26)
	_bubble.visible = true
	_bubble.modulate.a = 1.0
	_bubble.size = Vector2.ZERO
	_bubble.reset_size()
	_bubble_time = 6.0


## Coupe un texte en lignes d'environ `width` caractères, entre les mots.
static func _wrap(text: String, width: int) -> String:
	var lines: Array[String] = []
	var line := ""
	for word in text.split(" ", false):
		if line != "" and line.length() + 1 + word.length() > width:
			lines.append(line)
			line = word
		else:
			line = word if line == "" else line + " " + word
	if line != "":
		lines.append(line)
	return "\n".join(lines)


func _process(delta: float) -> void:
	_bob = fmod(_bob + delta * (9.0 if walking else 2.0), TAU)
	if _bubble.visible:
		_bubble_time -= delta
		_bubble.position = Vector2(-_bubble.size.x / 2, -84 - _bubble.size.y)
		if _bubble_time < 1.0:
			_bubble.modulate.a = max(0.0, _bubble_time)
		if _bubble_time <= 0:
			_bubble.visible = false
	queue_redraw()


func _draw() -> void:
	draw_pawn(self, color, dir, _bob, walking, is_me)


## Dessine un pion. Partagé avec l'aperçu de l'écran de création.
static func draw_pawn(ci: CanvasItem, c: Color, d: int, bob: float, moving: bool, me: bool, s: float = 1.0, o: Vector2 = Vector2.ZERO) -> void:
	var lift := (absf(sin(bob)) * 4.0 if moving else sin(bob) * 0.8) * s
	# Ombre au sol.
	ci.draw_set_transform(o, 0, Vector2(1, 0.5))
	ci.draw_circle(Vector2.ZERO, 17 * s, Color(0, 0, 0, 0.32))
	if me:
		ci.draw_arc(Vector2.ZERO, 22 * s, 0, TAU, 40, Color(UI.LUMEN, 0.8), 2.0 * s, true)
	ci.draw_set_transform(o + Vector2(0, -lift), 0, Vector2.ONE)
	var dark := c.darkened(0.35)
	# Socle.
	ci.draw_set_transform(o + Vector2(0, -lift - 6 * s), 0, Vector2(1, 0.5))
	ci.draw_circle(Vector2.ZERO, 15 * s, dark)
	ci.draw_set_transform(o + Vector2(0, -lift - 9 * s), 0, Vector2(1, 0.5))
	ci.draw_circle(Vector2.ZERO, 15 * s, c)
	ci.draw_set_transform(o + Vector2(0, -lift), 0, Vector2.ONE)
	# Corps en cloche.
	var body := PackedVector2Array([
		Vector2(-11, -9), Vector2(-8, -28), Vector2(-5, -36), Vector2(5, -36), Vector2(8, -28), Vector2(11, -9),
	])
	for i in body.size():
		body[i] *= s
	ci.draw_colored_polygon(body, c)
	ci.draw_polyline(body, dark, 1.5 * s, true)
	# Tête.
	ci.draw_circle(Vector2(0, -45) * s, 11 * s, c.lightened(0.08))
	ci.draw_arc(Vector2(0, -45) * s, 11 * s, 0, TAU, 32, dark, 1.5 * s, true)
	ci.draw_circle(Vector2(-3.5, -49) * s, 3.5 * s, Color(1, 1, 1, 0.35))
	# Regard : deux yeux tournés vers la direction du héros (vue isométrique).
	var f := Iso.dir_to_screen(d)
	if f.y > -0.3:
		var side := Vector2(-f.y, f.x).normalized() * 3.6
		var eye := Vector2(0, -45) + f * Vector2(6.0, 3.0)
		for k in [-1, 1]:
			ci.draw_circle((eye + side * k) * s, 1.9 * s, UI.NIGHT)
	ci.draw_set_transform(Vector2.ZERO, 0, Vector2.ONE)
