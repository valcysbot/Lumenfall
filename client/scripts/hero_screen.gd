extends Control
## Choix du héros : 3 emplacements, création avec un nom unique et une couleur de tunique.

signal play(hero: Dictionary)
signal logout

var slots_row: HBoxContainer
var status: Label
var braise: HBoxContainer
var colors: Array = []
var heroes: Array = []
var slot_count := 3

# Formulaire de création.
var form: PanelContainer
var name_edit: LineEdit
var preview: Control
var preview_hero: HeroSprite
var create_btn: Button
var err_label: Label
var color_grid: GridContainer
var form_shade: ColorRect
var form_slot := 0
var form_color := 6


func _init() -> void:
	set_anchors_preset(Control.PRESET_FULL_RECT)
	theme = UI.make_theme()
	add_child(UI.backdrop())

	var margin := MarginContainer.new()
	margin.set_anchors_preset(Control.PRESET_FULL_RECT)
	for side in ["left", "right", "top", "bottom"]:
		margin.add_theme_constant_override("margin_" + side, 28)
	add_child(margin)
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 16)
	margin.add_child(col)

	var head := HBoxContainer.new()
	var title := UI.label("Tes héros", 40, UI.LUMEN)
	title.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	head.add_child(title)
	var out := UI.button("Se déconnecter")
	out.pressed.connect(func(): logout.emit())
	head.add_child(out)
	col.add_child(head)

	var who := UI.label(("Connecté en tant que %s." % Net.display_name) if Net.display_name != "" else "", 17, UI.MUTED)
	col.add_child(who)

	var scroll := ScrollContainer.new()
	scroll.size_flags_vertical = Control.SIZE_EXPAND_FILL
	scroll.horizontal_scroll_mode = ScrollContainer.SCROLL_MODE_AUTO
	col.add_child(scroll)
	var center := CenterContainer.new()
	center.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	center.size_flags_vertical = Control.SIZE_EXPAND_FILL
	scroll.add_child(center)
	slots_row = HBoxContainer.new()
	slots_row.add_theme_constant_override("separation", 18)
	center.add_child(slots_row)

	status = UI.label("Braise compte tes héros…", 17, UI.MUTED)
	status.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	col.add_child(status)
	braise = UI.braise("Braise dit : trois emplacements, trois destins. Choisis bien tes noms, ils sont uniques dans tout Erydor !")
	col.add_child(braise)

	_build_form()


func _ready() -> void:
	refresh()


func refresh() -> void:
	var r := await Net.hero_list()
	if r.has("error"):
		status.text = r.error
		status.add_theme_color_override("font_color", UI.DANGER)
		if "connect" in r.error.to_lower() or "token" in r.error.to_lower() or "auth" in r.error.to_lower():
			logout.emit()
		return
	heroes = r.get("heroes", [])
	colors = r.get("colors", [])
	slot_count = int(r.get("slots", 3))
	status.text = ""
	_render()


func _render() -> void:
	for c in slots_row.get_children():
		c.queue_free()
	for slot in slot_count:
		var hero: Dictionary = {}
		for h in heroes:
			if int(h.slot) == slot:
				hero = h
		slots_row.add_child(_slot_card(slot, hero))


func _slot_card(slot: int, hero: Dictionary) -> PanelContainer:
	var card := PanelContainer.new()
	card.custom_minimum_size = Vector2(230, 300)
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 10)
	card.add_child(col)
	var pic := Control.new()
	pic.custom_minimum_size = Vector2(0, 170)
	col.add_child(pic)
	if hero.is_empty():
		card.add_theme_stylebox_override("panel", UI.box(Color(UI.STONE, 0.6), UI.LINE, 14))
		pic.draw.connect(func():
			var c := pic.size / 2 + Vector2(0, 30)
			pic.draw_set_transform(c, 0, Vector2(1, 0.5))
			pic.draw_arc(Vector2.ZERO, 40, 0, TAU, 48, UI.LINE, 2, true)
			pic.draw_set_transform(Vector2.ZERO, 0, Vector2.ONE)
			pic.draw_string(ThemeDB.fallback_font, c + Vector2(-12, -20), "+", HORIZONTAL_ALIGNMENT_LEFT, -1, 44, UI.MUTED))
		var l := UI.label("Emplacement libre", 20, UI.MUTED)
		l.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		col.add_child(l)
		var b := UI.button("Créer un héros", true)
		b.pressed.connect(_open_form.bind(slot))
		col.add_child(b)
	else:
		var hs := _hero_in(pic, HeroSprite.look_for(hero.name, int(hero.color)), 2.4, 72)
		hs.dir = 1
		var l := UI.label(hero.name, 26, UI.LUMEN)
		l.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		col.add_child(l)
		var where := UI.label("La Clairière", 16, UI.MUTED)
		where.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		col.add_child(where)
		var b := UI.button("Jouer", true)
		b.pressed.connect(func(): play.emit(hero))
		col.add_child(b)
	return card


func _color(i: int) -> Color:
	return Color(colors[i]) if i >= 0 and i < colors.size() else Color.WHITE


# --- Formulaire de création ---------------------------------------------------

func _build_form() -> void:
	var shade := ColorRect.new()
	shade.color = Color(0, 0, 0, 0.6)
	shade.set_anchors_preset(Control.PRESET_FULL_RECT)
	shade.visible = false
	add_child(shade)
	var center := CenterContainer.new()
	center.set_anchors_preset(Control.PRESET_FULL_RECT)
	shade.add_child(center)
	form = PanelContainer.new()
	form.add_theme_stylebox_override("panel", UI.box(UI.STONE, UI.LUMEN, 16, 1, 22))
	center.add_child(form)
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 24)
	form.add_child(row)

	preview = Control.new()
	preview.custom_minimum_size = Vector2(170, 240)
	row.add_child(preview)
	preview_hero = _hero_in(preview, {}, 3.0, 100)

	var col := VBoxContainer.new()
	col.custom_minimum_size = Vector2(360, 0)
	col.add_theme_constant_override("separation", 12)
	row.add_child(col)
	col.add_child(UI.label("Nouveau héros", 30, UI.LUMEN))
	col.add_child(UI.label("Son nom (3 à 16 lettres, unique dans Erydor)", 16, UI.MUTED))
	name_edit = LineEdit.new()
	name_edit.placeholder_text = "Ex. : Elowyn, D'Arvel, Mael-Tor"
	name_edit.max_length = 16
	name_edit.custom_minimum_size = Vector2(0, 48)
	name_edit.text_submitted.connect(func(_t): _create())
	name_edit.text_changed.connect(func(_t): _dress_preview())
	col.add_child(name_edit)
	col.add_child(UI.label("La couleur de sa tunique", 16, UI.MUTED))
	color_grid = GridContainer.new()
	color_grid.columns = 6
	color_grid.add_theme_constant_override("h_separation", 8)
	color_grid.add_theme_constant_override("v_separation", 8)
	col.add_child(color_grid)
	var actions := HBoxContainer.new()
	actions.add_theme_constant_override("separation", 10)
	var cancel := UI.button("Annuler")
	cancel.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	cancel.pressed.connect(func(): shade.visible = false)
	actions.add_child(cancel)
	create_btn = UI.button("Créer", true)
	create_btn.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	create_btn.pressed.connect(_create)
	actions.add_child(create_btn)
	col.add_child(actions)
	err_label = UI.label("", 16, UI.DANGER)
	col.add_child(err_label)
	form_shade = shade


func _open_form(slot: int) -> void:
	form_slot = slot
	form_color = randi() % max(1, colors.size())
	name_edit.text = ""
	err_label.text = ""
	for c in color_grid.get_children():
		c.queue_free()
	for i in colors.size():
		var b := Button.new()
		b.custom_minimum_size = Vector2(44, 44)
		b.add_theme_stylebox_override("normal", UI.box(Color(colors[i]), UI.LINE, 22))
		b.add_theme_stylebox_override("hover", UI.box(Color(colors[i]), UI.PARCH, 22, 2))
		b.add_theme_stylebox_override("pressed", UI.box(Color(colors[i]), UI.LUMEN, 22, 3))
		b.tooltip_text = "Couleur %d" % (i + 1)
		b.pressed.connect(func(): form_color = i; _mark_swatch(); _dress_preview())
		color_grid.add_child(b)
	_mark_swatch()
	_dress_preview()
	form_shade.visible = true
	name_edit.grab_focus()


## Entoure d'or la couleur choisie.
func _mark_swatch() -> void:
	for i in color_grid.get_child_count():
		var b := color_grid.get_child(i) as Button
		var picked := i == form_color
		b.add_theme_stylebox_override("normal", UI.box(Color(colors[i]), UI.LUMEN if picked else UI.LINE, 22, 3 if picked else 1))


## Place un héros en couches au centre d'une zone, pieds à `foot` pixels sous le centre.
func _hero_in(box: Control, look: Dictionary, zoom: float, foot: float) -> HeroSprite:
	var hs := HeroSprite.new()
	hs.scale = Vector2(zoom, zoom)
	box.add_child(hs)
	var place := func(): hs.position = box.size / 2 + Vector2(0, foot)
	box.resized.connect(place)
	place.call()
	if not look.is_empty():
		hs.dress(look)
	return hs


## L'aperçu suit le nom (pour les cheveux) et la couleur choisie, et tourne sur lui-même.
func _dress_preview() -> void:
	preview_hero.dress(HeroSprite.look_for(name_edit.text.strip_edges(), form_color))


func _process(_d: float) -> void:
	if preview_hero != null and preview.is_visible_in_tree():
		preview_hero.dir = int(Time.get_ticks_msec() / 900) % 8


func _create() -> void:
	if name_edit.text.strip_edges().length() < 3:
		err_label.text = "Un nom d'au moins 3 lettres, s'il te plaît. Même Braise en a 6 !"
		return
	create_btn.disabled = true
	err_label.text = ""
	var r := await Net.hero_create(form_slot, name_edit.text, form_color)
	create_btn.disabled = false
	if r.has("error"):
		err_label.text = r.error
		return
	form_shade.visible = false
	UI.set_braise(braise, "Braise dit : bienvenue à %s ! Je sens qu'on va bien s'entendre." % r.get("name", ""))
	refresh()
