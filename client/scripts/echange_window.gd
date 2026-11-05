class_name EchangeWindow
extends PanelContainer
## L'échange face à face (Acte V) : chacun pose ses objets et son argent, les
## deux valident, et l'échange se fait d'un coup. Changer son offre retire les
## validations. Un objet porté est lié : il ne s'échange plus.

signal closed

var view := {}     # {moi: {n, it, or, ok}, lui: {…}}
var _body: VBoxContainer
var _amount: SpinBox


func _ready() -> void:
	theme = UI.make_theme()
	add_theme_stylebox_override("panel", UI.box(Color(UI.STONE, 0.97), UI.MOSS, 14, 1, 12))
	set_anchors_and_offsets_preset(Control.PRESET_CENTER_RIGHT, Control.PRESET_MODE_MINSIZE, 14)
	grow_horizontal = Control.GROW_DIRECTION_BEGIN
	grow_vertical = Control.GROW_DIRECTION_BOTH
	custom_minimum_size = Vector2(420, 0)
	_body = VBoxContainer.new()
	_body.add_theme_constant_override("separation", 8)
	add_child(_body)
	_render()


func set_view(v: Dictionary) -> void:
	view = v
	if is_inside_tree():
		_render()


## Ajoute un objet du sac à son offre.
func offer_item(id: String) -> void:
	var ids := _mine()
	if not ids.has(id):
		ids.append(id)
	Net.send(Net.OP_OBJET, {"a": "echange_offre", "it": ids, "n": int(view.get("moi", {}).get("or", 0))})


func _mine() -> Array:
	var out := []
	for it in Objets.liste(view.get("moi", {}).get("it")):
		if it != null:
			out.append(str(it.id))
	return out


func cancel() -> void:
	Net.send(Net.OP_OBJET, {"a": "echange_annuler"})


func _render() -> void:
	for c in _body.get_children():
		c.queue_free()
	var head := HBoxContainer.new()
	var t := UI.label("🤝 Échange avec %s" % view.get("lui", {}).get("n", "…"), 21, UI.LUMEN)
	t.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	head.add_child(t)
	var x := UI.button("✕")
	x.custom_minimum_size = Vector2(44, 40)
	x.focus_mode = Control.FOCUS_NONE
	x.pressed.connect(cancel)
	head.add_child(x)
	_body.add_child(head)
	_body.add_child(UI.label("Ajoute tes objets depuis le sac (clic droit, ou « Proposer »). Touche un de tes objets ici pour le reprendre.", 14, UI.MUTED))
	for side in ["lui", "moi"]:
		var s: Dictionary = view.get(side, {})
		var ok := bool(s.get("ok", false))
		_body.add_child(UI.label(("%s propose" % s.get("n", "")) if side == "lui" else "Tu proposes", 17, UI.MOSS if ok else UI.PARCH))
		var grid := GridContainer.new()
		grid.columns = 8
		grid.add_theme_constant_override("h_separation", 4)
		var items: Array = Objets.liste(s.get("it"))
		for it in items:
			var c := Objets.Case.new(it, "", 46)
			if side == "moi":
				var id := str(it.id)
				c.gui_input.connect(func(e):
					if e is InputEventMouseButton and e.pressed:
						var ids := _mine()
						ids.erase(id)
						Net.send(Net.OP_OBJET, {"a": "echange_offre", "it": ids, "n": int(view.get("moi", {}).get("or", 0))}))
			grid.add_child(c)
		if items.is_empty():
			var rien := UI.label("rien pour l'instant", 14, UI.MUTED)
			rien.autowrap_mode = TextServer.AUTOWRAP_OFF
			grid.add_child(rien)
		_body.add_child(grid)
		_body.add_child(UI.label("🪙 %s%s" % [Objets.monnaie(int(s.get("or", 0)), true), "   ✓ a validé" if ok else ""], 15, UI.LUMEN))
	var money := HBoxContainer.new()
	money.add_theme_constant_override("separation", 6)
	_amount = SpinBox.new()
	_amount.min_value = 0
	_amount.max_value = 1000000
	_amount.value = int(view.get("moi", {}).get("or", 0)) / 100
	_amount.suffix = "argent"
	_amount.custom_minimum_size = Vector2(150, 40)
	money.add_child(_amount)
	var put := UI.button("Proposer cet argent")
	put.custom_minimum_size = Vector2(0, 40)
	put.focus_mode = Control.FOCUS_NONE
	put.pressed.connect(func(): Net.send(Net.OP_OBJET, {"a": "echange_offre", "it": _mine(), "n": int(_amount.value) * 100}))
	money.add_child(put)
	_body.add_child(money)
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 8)
	var ok_b := UI.button("✓ Valider l'échange", true)
	ok_b.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	ok_b.focus_mode = Control.FOCUS_NONE
	ok_b.disabled = bool(view.get("moi", {}).get("ok", false))
	ok_b.pressed.connect(func(): Net.send(Net.OP_OBJET, {"a": "echange_valider"}))
	row.add_child(ok_b)
	var no := UI.button("Annuler")
	no.focus_mode = Control.FOCUS_NONE
	no.pressed.connect(cancel)
	row.add_child(no)
	_body.add_child(row)


func _unhandled_key_input(event: InputEvent) -> void:
	if event.pressed and not event.echo and event.physical_keycode == KEY_ESCAPE:
		get_viewport().set_input_as_handled()
		cancel()
