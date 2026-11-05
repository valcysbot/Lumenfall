extends Control
## Choix du héros : 3 emplacements, et le miroir de création (Miroir) pour en faire un nouveau.

signal play(hero: Dictionary)
signal logout

var slots_row: HBoxContainer
var status: Label
var braise: HBoxContainer
var heroes: Array = []
var slot_count := 3



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
	card.custom_minimum_size = Vector2(240, 320)
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
		var hs := _hero_in(pic, hero, 2.4, 72)
		hs.dir = 1
		var l := UI.label(("💀 " if hero.get("hardcore", false) else "") + hero.name, 26, UI.LUMEN)
		l.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		col.add_child(l)
		var lk: Dictionary = hero.get("look", {})
		var a := int(lk.get("a", 0))
		var c := Heros.classe(str(hero.get("classe", "")))
		var who := UI.label("%s %s · %s · niv. %d" % [Heros.faction_of(hero.race).blason, Heros.accorde(Heros.peuple(hero.race).nom, a),
			Heros.accorde(c.noms, a), int(hero.get("level", 1))], 15, UI.PARCH)
		who.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		col.add_child(who)
		var where := UI.label("Le Val de Glennmor", 16, UI.MUTED)
		where.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		col.add_child(where)
		var b := UI.button("Jouer", true)
		b.pressed.connect(func(): play.emit(hero))
		col.add_child(b)
		var del := UI.button("🗑️ Effacer")
		del.add_theme_font_size_override("font_size", 15)
		del.add_theme_color_override("font_color", UI.MUTED)
		del.pressed.connect(_ask_delete.bind(hero))
		col.add_child(del)
	return card


# --- Effacer un héros -----------------------------------------------------------

## Demande confirmation : on tape le nom du héros, comme on grave une stèle.
func _ask_delete(hero: Dictionary) -> void:
	var root := Control.new()
	root.name = "Effacer"
	root.set_anchors_preset(Control.PRESET_FULL_RECT)
	add_child(root)
	var shade := ColorRect.new()
	shade.color = Color(0, 0, 0, 0.6)
	shade.set_anchors_preset(Control.PRESET_FULL_RECT)
	root.add_child(shade)
	var center := CenterContainer.new()
	center.set_anchors_preset(Control.PRESET_FULL_RECT)
	root.add_child(center)
	var panel := PanelContainer.new()
	panel.add_theme_stylebox_override("panel", UI.box(UI.STONE, UI.DANGER, 16, 1, 18))
	panel.custom_minimum_size = Vector2(minf(480, get_viewport_rect().size.x - 32), 0)
	center.add_child(panel)
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 12)
	panel.add_child(col)
	col.add_child(UI.label("Effacer %s ?" % hero.name, 26, UI.DANGER))
	var warn := UI.label("Son niveau, son or et ses objets disparaîtront pour toujours, et son nom redeviendra libre. Pour confirmer, tape son nom :", 16)
	warn.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	col.add_child(warn)
	var field := LineEdit.new()
	field.placeholder_text = hero.name
	field.custom_minimum_size = Vector2(0, 44)
	col.add_child(field)
	var err := UI.label("", 15, UI.DANGER)
	err.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	col.add_child(err)
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 10)
	col.add_child(row)
	var no := UI.button("Garder mon héros", true)
	no.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	no.pressed.connect(root.queue_free)
	row.add_child(no)
	var yes := UI.button("Effacer pour toujours")
	yes.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	yes.add_theme_color_override("font_color", UI.DANGER)
	yes.disabled = true
	row.add_child(yes)
	field.text_changed.connect(func(t: String):
		yes.disabled = t.strip_edges().to_lower() != str(hero.name).to_lower())
	yes.pressed.connect(func():
		yes.disabled = true
		var r := await Net.hero_delete(str(hero.id), field.text.strip_edges())
		if r.has("error"):
			err.text = r.error
			yes.disabled = false
			return
		root.queue_free()
		UI.set_braise(braise, "Braise dit : %s s'en est %s en paix. Un emplacement attend un nouveau destin." % [hero.name,
			Heros.accorde(["allé", "allée", "allé·e"], int((hero.get("look", {}) as Dictionary).get("a", 0)))])
		refresh())
	field.grab_focus.call_deferred()


# --- Le miroir de création ------------------------------------------------------

func _open_form(slot: int) -> void:
	var m := Miroir.new()
	m.form.race = Heros.ORDRE_PEUPLES.pick_random()
	m.form.classe = Heros.data().classes.pick_random().id
	m.form.dieu = Heros.DIEUX.keys().pick_random()
	add_child(m)
	m.cancelled.connect(m.queue_free)
	m.done.connect(func(form: Dictionary):
		var f := form.duplicate(true)
		f.slot = slot
		var r := await Net.hero_create(f)
		if r.has("error"):
			m.error(r.error)
			return
		m.queue_free()
		UI.set_braise(braise, "Braise dit : bienvenue à %s ! Je sens qu'on va bien s'entendre." % r.get("name", ""))
		refresh())


## Place un héros au centre d'une zone, pieds à `foot` pixels sous le centre.
func _hero_in(box: Control, hero: Dictionary, zoom: float, foot: float) -> PeupleSprite:
	var hs := PeupleSprite.new()
	var lk: Dictionary = hero.get("look", {})
	var race := str(hero.get("race", "humain"))
	var z := zoom * Heros.echelle(race, lk)
	hs.scale = Vector2(z, z)
	box.add_child(hs)
	var lift := 10.0 if Heros.peuple(race).get("flotte", false) else 0.0
	var place := func(): hs.position = box.size / 2 + Vector2(0, foot - lift)
	box.resized.connect(place)
	place.call()
	hs.dress(Heros.couches(race, str(hero.get("classe", "")), lk, hero.get("eq")))
	return hs
