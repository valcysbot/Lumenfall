class_name Pawn
extends Node2D
## Un héros dans le monde : son allure en couches (HeroSprite), son ombre,
## son nom et ses bulles de chat.

var look := {}
var hero_name := ""
var dir := 2
var is_me := false
var walking := false
var running := false

var sprite: HeroSprite
var _name_label: Label
var _bubble: PanelContainer
var _bubble_label: Label
var _bubble_time := 0.0


func _ready() -> void:
	sprite = HeroSprite.new()
	add_child(sprite)
	sprite.dress(look)
	sprite.dir = dir

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
	sprite.dir = dir
	sprite.set_motion(walking, running)
	if _bubble.visible:
		_bubble_time -= delta
		_bubble.position = Vector2(-_bubble.size.x / 2, -84 - _bubble.size.y)
		if _bubble_time < 1.0:
			_bubble.modulate.a = max(0.0, _bubble_time)
		if _bubble_time <= 0:
			_bubble.visible = false
	queue_redraw()


func _draw() -> void:
	# Ombre au sol, et un anneau doré sous ton propre héros.
	draw_set_transform(Vector2.ZERO, 0, Vector2(1, 0.5))
	draw_circle(Vector2.ZERO, 14, Color(0, 0, 0, 0.32))
	if is_me:
		draw_arc(Vector2.ZERO, 19, 0, TAU, 40, Color(UI.LUMEN, 0.8), 2.0, true)
	draw_set_transform(Vector2.ZERO, 0, Vector2.ONE)
