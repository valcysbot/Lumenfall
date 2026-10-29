class_name TouchStick
extends Control
## Joystick virtuel pour téléphone. `value` va de (-1, -1) à (1, 1).

var value := Vector2.ZERO
var _touch := -1
var _knob := Vector2.ZERO
const RADIUS := 70.0


func _init() -> void:
	custom_minimum_size = Vector2(RADIUS * 2 + 20, RADIUS * 2 + 20)
	mouse_filter = Control.MOUSE_FILTER_STOP


func _center() -> Vector2:
	return size / 2


func _gui_input(event: InputEvent) -> void:
	if event is InputEventScreenTouch:
		if event.pressed and _touch == -1:
			_touch = event.index
			_update(event.position)
		elif not event.pressed and event.index == _touch:
			_touch = -1
			_update(_center())
		accept_event()
	elif event is InputEventScreenDrag and event.index == _touch:
		_update(event.position)
		accept_event()


func _update(p: Vector2) -> void:
	var d := (p - _center()).limit_length(RADIUS)
	_knob = d
	value = d / RADIUS
	queue_redraw()


func _draw() -> void:
	var c := _center()
	draw_circle(c, RADIUS, Color(UI.NIGHT, 0.45))
	draw_arc(c, RADIUS, 0, TAU, 48, Color(UI.PARCH, 0.35), 2, true)
	draw_circle(c + _knob, 30, Color(UI.EMBER, 0.85))
	draw_arc(c + _knob, 30, 0, TAU, 32, Color(UI.LUMEN, 0.9), 2, true)
