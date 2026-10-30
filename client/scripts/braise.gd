class_name Braise
extends Node2D
## Braise, le petit dragon qui suit ton héros (tools/sprites/forge_braise.py).
## Il vole un peu derrière toi et commente ce qu'il voit de temps en temps :
## la nuit, la pluie, un habitant tout proche… On peut le renvoyer dans les
## réglages.

const DIR := "res://assets/braise/"
const BEHIND := 1.1        # cases derrière le héros
const SPEED := 6.0         # rattrapage (plus grand = plus collé)
const TALK_MIN := 45.0     # secondes entre deux remarques
const TALK_MAX := 90.0

var world_pos := Vector2.ZERO   # position au sol, en cases
var dir := 1
var _meta := {}
var _sprite: Sprite2D
var _t := 0.0
var _talk_in := 12.0
var _bubble: PanelContainer
var _bubble_label: Label
var _bubble_time := 0.0


func _ready() -> void:
	_meta = JSON.parse_string(FileAccess.get_file_as_string(DIR + "braise.json"))
	_sprite = Sprite2D.new()
	_sprite.texture = load(DIR + "braise.png")
	_sprite.texture_filter = CanvasItem.TEXTURE_FILTER_NEAREST
	_sprite.centered = false
	_sprite.region_enabled = true
	_sprite.offset = -Vector2(_meta.ancre[0], _meta.ancre[1])
	add_child(_sprite)
	_bubble = PanelContainer.new()
	_bubble.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.9), UI.EMBER, 10, 1, 6))
	_bubble_label = Label.new()
	_bubble_label.add_theme_font_size_override("font_size", 14)
	_bubble_label.add_theme_color_override("font_color", UI.LUMEN)
	_bubble.add_child(_bubble_label)
	_bubble.visible = false
	add_child(_bubble)


## Pose Braise à côté du héros d'un coup (arrivée, porte).
func snap(hero: Vector2) -> void:
	world_pos = hero + Vector2(-0.8, -0.8)
	position = Iso.to_screen(world_pos)


## Suit le héros (position en cases, direction 0 à 7 de son regard).
func follow(hero: Vector2, hero_dir: int, delta: float) -> void:
	if world_pos.distance_to(hero) > 6.0:
		snap(hero)
	var a := hero_dir * PI / 4
	var target := hero - Vector2(cos(a), sin(a)) * BEHIND + Vector2(0.35, -0.35)
	var step := (target - world_pos) * minf(1.0, SPEED * delta * 0.5)
	if step.length() > 0.004:
		dir = posmod(int(round(step.angle() / (PI / 4))), 8)
	elif world_pos.distance_to(hero) < 2.0:
		dir = hero_dir
	world_pos += step
	position = Iso.to_screen(world_pos)


func say(text: String) -> void:
	_bubble_label.text = Pawn._wrap(text, 30)
	_bubble.visible = true
	_bubble.modulate.a = 1.0
	_bubble.size = Vector2.ZERO
	_bubble.reset_size()
	_bubble_time = 6.0


## Renvoie vrai quand Braise a envie de parler (le monde choisit quoi dire).
func wants_to_talk(delta: float) -> bool:
	_talk_in -= delta
	if _talk_in > 0:
		return false
	_talk_in = randf_range(TALK_MIN, TALK_MAX)
	return true


func _process(delta: float) -> void:
	_t += delta
	var f := int(_t * float(_meta.fps)) % int(_meta.images)
	var w: float = _meta.image[0]
	var h: float = _meta.image[1]
	_sprite.region_rect = Rect2(f * w, dir * h, w, h)
	if _bubble.visible:
		_bubble_time -= delta
		_bubble.position = Vector2(-_bubble.size.x / 2, -52 - _bubble.size.y)
		if _bubble_time < 1.0:
			_bubble.modulate.a = maxf(0.0, _bubble_time)
		if _bubble_time <= 0:
			_bubble.visible = false
	queue_redraw()


func _draw() -> void:
	# Petite ombre au sol, sous le dragon qui vole.
	draw_set_transform(Vector2.ZERO, 0, Vector2(1, 0.5))
	draw_circle(Vector2.ZERO, 7, Color(0, 0, 0, 0.22))
	draw_set_transform(Vector2.ZERO, 0, Vector2.ONE)
