extends PanelContainer
## ⚙ Réglages → ⌨ Touches : clique sur une action, puis appuie sur la nouvelle touche.

signal changed

var _waiting := ""
var _list: VBoxContainer
var _hint: Label


func _ready() -> void:
	add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.95), UI.LUMEN, 14, 2, 16))
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 8)
	add_child(col)
	var head := HBoxContainer.new()
	var title := UI.label("⌨ Tes touches", 20, UI.LUMEN)
	title.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	head.add_child(title)
	var x := UI.button("✕")
	x.focus_mode = Control.FOCUS_NONE
	x.pressed.connect(queue_free)
	head.add_child(x)
	col.add_child(head)
	_hint = UI.label("Clique sur une action, puis appuie sur sa nouvelle touche. La souris (sauf le clic gauche) et la manette marchent aussi.", 14, UI.MUTED)
	_hint.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	_hint.custom_minimum_size.x = 380
	col.add_child(_hint)
	var scroll := ScrollContainer.new()
	scroll.horizontal_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
	scroll.custom_minimum_size = Vector2(380, clampf(get_viewport_rect().size.y - 220, 240, 520))
	col.add_child(scroll)
	_list = VBoxContainer.new()
	_list.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_list.add_theme_constant_override("separation", 4)
	scroll.add_child(_list)
	var reset := UI.button("Remettre les touches d'origine")
	reset.focus_mode = Control.FOCUS_NONE
	reset.pressed.connect(func():
		Touches.reset()
		_waiting = ""
		_render()
		changed.emit())
	col.add_child(reset)
	_render()
	grow_horizontal = Control.GROW_DIRECTION_BOTH
	grow_vertical = Control.GROW_DIRECTION_BOTH
	set_anchors_and_offsets_preset.call_deferred(Control.PRESET_CENTER, Control.PRESET_MODE_MINSIZE)


func _render() -> void:
	for c in _list.get_children():
		c.queue_free()
	for a in Touches.ACTIONS:
		var row := HBoxContainer.new()
		var l := UI.label(a[1], 16)
		l.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		row.add_child(l)
		var b := UI.button("…" if _waiting == a[0] else Touches.label(a[0]), _waiting == a[0])
		b.custom_minimum_size = Vector2(120, 36)
		b.focus_mode = Control.FOCUS_NONE
		var id: String = a[0]
		b.pressed.connect(func():
			_waiting = id
			_hint.text = "Appuie sur une touche, un bouton de souris ou de manette (Échap pour annuler)."
			_render())
		row.add_child(b)
		_list.add_child(row)


func _input(event: InputEvent) -> void:
	if _waiting == "" or not event.is_pressed() or event.is_echo():
		return
	var k := Touches.code_of(event)
	if k == 0:
		return   # le clic gauche sert à cliquer : il reste libre
	get_viewport().set_input_as_handled()
	if k == KEY_ESCAPE or k == KEY_ENTER:
		_hint.text = "Échap et Entrée restent réservées (fermer, discuter)." if k == KEY_ENTER else "Clique sur une action, puis appuie sur sa nouvelle touche."
	else:
		Touches.set_key(_waiting, k)
		_hint.text = "C'est noté ! Si une autre action avait cette touche, elles ont échangé."
		changed.emit()
	_waiting = ""
	_render()
