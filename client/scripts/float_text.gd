class_name FloatText
extends Node2D
## Un chiffre de dégâts (ou de soin) qui saute au-dessus d'une tête et s'efface.
## Les coups critiques sont gros et dorés.

var text := ""
var color := Color.WHITE
var big := false
var _label: Label
var _v := Vector2.ZERO
var _t := 0.0
const LIFE := 1.1


func _ready() -> void:
	z_index = 100
	_label = Label.new()
	_label.text = text
	_label.add_theme_font_size_override("font_size", 30 if big else 19)
	_label.add_theme_color_override("font_color", color)
	_label.add_theme_color_override("font_outline_color", Color(0.08, 0.06, 0.1, 0.95))
	_label.add_theme_constant_override("outline_size", 7 if big else 5)
	add_child(_label)
	_label.reset_size()
	_label.position = -_label.size / 2
	_v = Vector2(randf_range(-30, 30), -120 if big else -85)
	if big:
		scale = Vector2(0.4, 0.4)
		create_tween().tween_property(self, "scale", Vector2.ONE, 0.18).set_trans(Tween.TRANS_BACK).set_ease(Tween.EASE_OUT)


func _process(delta: float) -> void:
	_t += delta
	position += _v * delta
	_v.y += 140 * delta
	modulate.a = clampf((LIFE - _t) / 0.35, 0, 1)
	if _t > LIFE:
		queue_free()
