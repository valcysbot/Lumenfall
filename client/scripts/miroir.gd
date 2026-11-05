class_name Miroir
extends Control
## Le miroir de création (Acte IV) : peuple, genre et accords, allure (taille,
## carrure, visage, peau, yeux, cheveux, barbe, marques, tunique), classe,
## dieu, hardcore et nom. Au temple, le même miroir (sans peuple ni classe)
## sert au miroir de Solenn.

signal done(form: Dictionary)
signal cancelled

## "creation" : un nouveau héros. "solenn" : changer d'allure au temple.
var mode := "creation"
var price := 0          # or demandé par le miroir de Solenn
var form := {"race": "humain", "classe": "guerrier", "dieu": "solenn", "hardcore": false, "name": "", "look": {}}

var _preview: Control
var _hero: PeupleSprite
var _turn := 1
var _auto_turn := true
var _tabs: HBoxContainer
var _page: VBoxContainer
var _tab := ""
var _summary: Label
var _err: Label
var _ok_btn: Button
var _name_edit: LineEdit
var _t := 0.0


func _init() -> void:
	set_anchors_preset(Control.PRESET_FULL_RECT)
	theme = UI.make_theme()


func _ready() -> void:
	if form.look.is_empty():
		form.look = Heros.allure_au_hasard(form.race)
	var shade := ColorRect.new()
	shade.color = Color(0, 0, 0, 0.7)
	shade.set_anchors_preset(Control.PRESET_FULL_RECT)
	add_child(shade)
	var margin := MarginContainer.new()
	margin.set_anchors_preset(Control.PRESET_FULL_RECT)
	for side in ["left", "right", "top", "bottom"]:
		margin.add_theme_constant_override("margin_" + side, 18)
	add_child(margin)
	var panel := PanelContainer.new()
	panel.add_theme_stylebox_override("panel", UI.box(UI.STONE, UI.LUMEN, 16, 1, 14))
	margin.add_child(panel)
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 18)
	panel.add_child(row)

	# À gauche : le reflet, qui tourne.
	var left := VBoxContainer.new()
	left.custom_minimum_size = Vector2(300, 0)
	left.add_theme_constant_override("separation", 8)
	row.add_child(left)
	var title := UI.label("🪞 Le miroir de création" if mode == "creation" else "🪞 Le miroir de Solenn", 26, UI.LUMEN)
	left.add_child(title)
	_preview = Control.new()
	_preview.size_flags_vertical = Control.SIZE_EXPAND_FILL
	_preview.custom_minimum_size = Vector2(280, 300)
	_preview.draw.connect(_draw_preview)
	left.add_child(_preview)
	_hero = PeupleSprite.new()
	_preview.add_child(_hero)
	var turn := HBoxContainer.new()
	turn.alignment = BoxContainer.ALIGNMENT_CENTER
	for d in [["⟲", 1], ["🎲 Au hasard", 0], ["⟳", -1]]:
		var b := UI.button(d[0])
		b.custom_minimum_size = Vector2(64, 44)
		var step: int = d[1]
		b.pressed.connect(func():
			if step == 0:
				_random()
			else:
				_auto_turn = false
				_turn = posmod(_turn + step, 8))
		turn.add_child(b)
	left.add_child(turn)
	_summary = UI.label("", 16, UI.PARCH)
	_summary.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	left.add_child(_summary)

	# À droite : les onglets.
	var right := VBoxContainer.new()
	right.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	right.add_theme_constant_override("separation", 10)
	row.add_child(right)
	_tabs = HBoxContainer.new()
	_tabs.add_theme_constant_override("separation", 6)
	right.add_child(_tabs)
	var scroll := ScrollContainer.new()
	scroll.size_flags_vertical = Control.SIZE_EXPAND_FILL
	scroll.horizontal_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
	right.add_child(scroll)
	_page = VBoxContainer.new()
	_page.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_page.add_theme_constant_override("separation", 10)
	scroll.add_child(_page)
	_err = UI.label("", 16, UI.DANGER)
	right.add_child(_err)
	var actions := HBoxContainer.new()
	actions.add_theme_constant_override("separation", 10)
	var cancel := UI.button("Annuler")
	cancel.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	cancel.pressed.connect(func(): cancelled.emit())
	actions.add_child(cancel)
	_ok_btn = UI.button("", true)
	_ok_btn.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_ok_btn.pressed.connect(_validate)
	actions.add_child(_ok_btn)
	right.add_child(actions)

	var tabs := [["peuple", "🌍 Peuple"], ["allure", "🪞 Allure"], ["classe", "⚔️ Classe"], ["serment", "✨ Serment"]]
	if mode != "creation":
		tabs = [["allure", "🪞 Allure"]]
	for t in tabs:
		var b := UI.button(t[1])
		b.toggle_mode = true
		b.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		b.custom_minimum_size = Vector2(0, 46)
		var id: String = t[0]
		b.pressed.connect(func(): _show(id))
		b.set_meta("tab", id)
		_tabs.add_child(b)
	_show(tabs[0][0])
	_refresh()


func error(text: String) -> void:
	_err.text = text
	_ok_btn.disabled = false


func _validate() -> void:
	if mode == "creation":
		var n := str(form.name).strip_edges()
		if n.length() < 3:
			_show("serment")
			error("Un nom d'au moins 3 lettres, s'il te plaît. Même Braise en a 6 !")
			return
		form.name = n
	_ok_btn.disabled = true
	_err.text = ""
	done.emit(form)


func _random() -> void:
	var keep: Array = form.look.get("m", [])
	form.look = Heros.allure_au_hasard(form.race)
	form.look.m = keep
	if form.race in ["nain"] and randf() < 0.7:
		form.look.b = randi() % 6
	_show(_tab)
	_refresh()


# --- Le reflet ----------------------------------------------------------------

func _refresh() -> void:
	var lk: Dictionary = form.look
	_hero.dress(Heros.couches(form.race, form.classe, lk))
	var P := Heros.peuple(form.race)
	var c := Heros.classe(form.classe)
	var a := int(lk.get("a", 0))
	var bits := [Heros.accorde(P.nom, a)]
	if mode == "creation":
		bits.append(Heros.accorde(c.noms, a))
	_summary.text = " · ".join(bits)
	if mode == "creation":
		_summary.text += "\n" + Heros.faction_of(form.race).blason + " " + Heros.faction_of(form.race).nom
		_ok_btn.text = "Créer ce héros"
	else:
		_ok_btn.text = "Garder ce reflet" + (" (%d or)" % price if price > 0 else " (offert)")
	_place()


func _place() -> void:
	var z := 3.4 * Heros.echelle(form.race, form.look)
	_hero.scale = Vector2(z, z)
	var lift := 0.0
	if Heros.peuple(form.race).get("flotte", false):
		lift = 14.0 + sin(_t * 2.2) * 4.0
	_hero.position = Vector2(_preview.size.x / 2, _preview.size.y * 0.78 - lift)


func _draw_preview() -> void:
	var c := Vector2(_preview.size.x / 2, _preview.size.y * 0.78)
	_preview.draw_set_transform(c, 0, Vector2(1, 0.4))
	_preview.draw_circle(Vector2.ZERO, 70, Color(UI.LUMEN, 0.08))
	_preview.draw_arc(Vector2.ZERO, 70, 0, TAU, 64, Color(UI.LUMEN, 0.4), 2, true)
	_preview.draw_circle(Vector2.ZERO, 30, Color(0, 0, 0, 0.35))
	_preview.draw_set_transform(Vector2.ZERO, 0, Vector2.ONE)


func _process(delta: float) -> void:
	_t += delta
	if _auto_turn:
		_turn = int(_t / 1.1) % 8
	_hero.dir = _turn
	_place()
	_preview.queue_redraw()


# --- Les onglets ----------------------------------------------------------------

func _show(id: String) -> void:
	_tab = id
	for b in _tabs.get_children():
		(b as Button).button_pressed = b.get_meta("tab") == id
	for c in _page.get_children():
		c.queue_free()
	match id:
		"peuple":
			_page_peuple()
		"allure":
			_page_allure()
		"classe":
			_page_classe()
		"serment":
			_page_serment()


func _h(text: String) -> void:
	_page.add_child(UI.label(text, 18, UI.LUMEN))


func _note(text: String) -> Label:
	var l := UI.label(text, 15, UI.MUTED)
	_page.add_child(l)
	return l


## Une rangée de choix ; `on` est appelé avec l'indice choisi.
func _choices(labels: Array, picked: int, on: Callable, cols: int = 0) -> void:
	var box: Container
	if cols > 0:
		var g := GridContainer.new()
		g.columns = cols
		g.add_theme_constant_override("h_separation", 6)
		g.add_theme_constant_override("v_separation", 6)
		box = g
	else:
		var h := HFlowContainer.new()
		h.add_theme_constant_override("h_separation", 6)
		h.add_theme_constant_override("v_separation", 6)
		box = h
	for i in labels.size():
		var b := UI.button(str(labels[i]))
		b.custom_minimum_size = Vector2(0, 44)
		if cols > 0:
			b.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		if i == picked:
			b.add_theme_stylebox_override("normal", UI.box(UI.STONE_2, UI.LUMEN, 10, 2))
			b.add_theme_color_override("font_color", UI.LUMEN)
		var k := i
		b.pressed.connect(func(): on.call(k))
		box.add_child(b)
	_page.add_child(box)


## Des pastilles de couleur.
func _swatches(colors: Array, picked: int, on: Callable) -> void:
	var g := HFlowContainer.new()
	g.add_theme_constant_override("h_separation", 6)
	g.add_theme_constant_override("v_separation", 6)
	for i in colors.size():
		var b := Button.new()
		b.custom_minimum_size = Vector2(38, 38)
		var col := Color(colors[i])
		var sel := i == picked
		b.add_theme_stylebox_override("normal", UI.box(col, UI.LUMEN if sel else UI.LINE, 19, 3 if sel else 1, 0))
		b.add_theme_stylebox_override("hover", UI.box(col, UI.PARCH, 19, 2, 0))
		b.add_theme_stylebox_override("pressed", UI.box(col, UI.LUMEN, 19, 3, 0))
		var k := i
		b.pressed.connect(func(): on.call(k))
		g.add_child(b)
	_page.add_child(g)


## ◀ valeur ▶ pour parcourir une longue liste.
func _stepper(title: String, value: int, n: int, text: Callable, on: Callable) -> void:
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 8)
	var l := UI.label(title, 17)
	l.custom_minimum_size = Vector2(120, 0)
	row.add_child(l)
	var minus := UI.button("◀")
	minus.custom_minimum_size = Vector2(52, 44)
	minus.pressed.connect(func(): on.call(posmod(value - 1, n)))
	row.add_child(minus)
	var v := UI.label(text.call(value), 17, UI.LUMEN)
	v.custom_minimum_size = Vector2(150, 0)
	v.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	row.add_child(v)
	var plus := UI.button("▶")
	plus.custom_minimum_size = Vector2(52, 44)
	plus.pressed.connect(func(): on.call(posmod(value + 1, n)))
	row.add_child(plus)
	_page.add_child(row)


func _set_look(key: String, v) -> void:
	form.look[key] = v
	_show(_tab)
	_refresh()


func _page_peuple() -> void:
	var data := Heros.data()
	var a := int(form.look.get("a", 0))
	for f in ["concorde", "pacte", "ligue"]:
		var fac: Dictionary = Heros.FACTIONS[f]
		_h("%s %s" % [fac.blason, fac.nom.capitalize() if false else fac.nom])
		var ids := Heros.ORDRE_PEUPLES.filter(func(r): return Heros.peuple(r).faction == f)
		var labels := ids.map(func(r): return Heros.accorde(Heros.peuple(r).nom, a))
		_choices(labels, ids.find(form.race), func(i):
			form.race = ids[i]
			if form.race == "nain" and int(form.look.get("b", -1)) < 0 and int(form.look.get("g", 0)) == 0:
				form.look.b = 2
			_show("peuple")
			_refresh(), 5)
	var P := Heros.peuple(form.race)
	var r: Dictionary = data.races.get(form.race, {})
	var bits := ["Capitale : %s" % P.capitale]
	if str(r.get("bonus", "")) != "":
		bits.append("+1 en %s" % Heros.ATTRIBUTS[r.bonus][0])
	if str(r.get("resiste", "")) != "":
		bits.append("résiste un peu à l'élément %s" % str(r.resiste).replace("lumiere", "lumière"))
	bits.append("taille %d %%" % int(round(float(P.taille) * 100)))
	if P.get("flotte", false):
		bits.append("flotte au-dessus du sol")
	_note(" · ".join(bits))
	_h("Genre")
	_choices(["Homme", "Femme", "Autre"], int(form.look.get("g", 0)), func(i):
		form.look.g = i
		form.look.a = mini(i, 2)
		_show("peuple")
		_refresh())
	_h("Accords")
	_note("Comment le jeu parle de ton héros : « il est tombé », « elle est tombée » ou « iel est tombé·e ».")
	_choices(["Il (masculin)", "Elle (féminin)", "Iel (neutre)"], int(form.look.get("a", 0)), func(i): _set_look("a", i))


func _page_allure() -> void:
	var lk: Dictionary = form.look
	var D := Heros.peuples()
	var P := Heros.peuple(form.race)
	_h("Silhouette")
	_choices(["Très petit", "Petit", "Moyen", "Grand", "Très grand"], int(lk.get("t", 2)), func(i): _set_look("t", i))
	_choices(["Mince", "Normale", "Costaud"], int(lk.get("k", 1)), func(i): _set_look("k", i))
	_h("Visage")
	_stepper("Visage", int(lk.get("v", 0)), 16, func(v): return "n° %d / 16" % (v + 1), func(v): _set_look("v", v))
	_h("Peau")
	_swatches(P.peaux, int(lk.get("p", 0)), func(i): _set_look("p", i))
	_h("Yeux")
	_swatches(D.yeux, int(lk.get("y", 0)), func(i): _set_look("y", i))
	if P.forme == "standard":
		_h("Cheveux")
		_stepper("Coupe", int(lk.get("h", 0)), 20, func(v): return str(D.coiffures[v]).replace("_", " "), func(v): _set_look("h", v))
		_swatches(D.cheveux, int(lk.get("hc", 0)), func(i): _set_look("hc", i))
		_h("Barbe et moustache")
		var barbes: Array = ["Aucune"]
		for b in D.barbes:
			barbes.append(str(b).replace("_", " ").capitalize())
		_choices(barbes, int(lk.get("b", -1)) + 1, func(i): _set_look("b", i - 1))
	else:
		_note("Les %s n'ont pas de cheveux : %s." % [Heros.accorde(P.nom, 2).to_lower() + "s", "de la mousse et des fleurs" if P.forme == "pierre" else "des éclats de cristal"])
	_h("Marques (jusqu'à 3, empilées)")
	var marks: Array = lk.get("m", [])
	for i in marks.size():
		var m: Array = marks[i]
		var j := i
		_stepper("Marque %d" % (i + 1), int(m[0]), D.marques.size(), func(v): return str(D.marques[v]).capitalize(), func(v):
			form.look.m[j][0] = v
			_show(_tab)
			_refresh())
		_swatches(D.cheveux, int(m[1]), func(c):
			form.look.m[j][1] = c
			_show(_tab)
			_refresh())
	var mrow := HBoxContainer.new()
	mrow.add_theme_constant_override("separation", 8)
	if marks.size() < 3:
		var add := UI.button("+ Ajouter une marque")
		add.pressed.connect(func():
			form.look.m.append([randi() % D.marques.size(), randi() % 24])
			_show(_tab)
			_refresh())
		mrow.add_child(add)
	if marks.size() > 0:
		var rm := UI.button("− Retirer la dernière")
		rm.pressed.connect(func():
			form.look.m.pop_back()
			_show(_tab)
			_refresh())
		mrow.add_child(rm)
	_page.add_child(mrow)
	_h("Couleur de la tunique")
	_swatches(D.tuniques, int(lk.get("tu", 0)), func(i): _set_look("tu", i))


func _page_classe() -> void:
	var data := Heros.data()
	var a := int(form.look.get("a", 0))
	var ids: Array = data.classes.map(func(c): return c.id)
	_choices(data.classes.map(func(c): return Heros.accorde(c.noms, a)), ids.find(form.classe), func(i):
		form.classe = ids[i]
		_hero.play_once(str(Heros.classe(form.classe).attaque))
		_show("classe")
		_refresh(), 3)
	var c := Heros.classe(form.classe)
	_h(Heros.accorde(c.noms, a))
	_note(c.aide)
	_note("Ressource : %s%s" % [c.ressource.nom, " · bouclier (%s)" % Touches.label("garde").to_lower() if c.bouclier else ""])
	_h("Compétences")
	for sk in c.competences:
		_note("%s  niv. %d · %s : %s" % [sk.icone, int(sk.niveau), sk.nom, sk.aide])
	_h("Les trois voies (au niveau %d)" % int(data.voie))
	for v in c.voies:
		_note("%s · %s : %s\n      Ultime : %s %s" % [Heros.ROLES.get(v.role, ""), v.nom, v.aide, v.ultime.icone, v.ultime.nom])


func _page_serment() -> void:
	_h("Ton nom")
	_note("3 à 16 lettres, unique dans tout Erydor.")
	_name_edit = LineEdit.new()
	_name_edit.placeholder_text = "Ex. : Elowyn, D'Arvel, Mael-Tor"
	_name_edit.max_length = 16
	_name_edit.text = form.name
	_name_edit.custom_minimum_size = Vector2(0, 48)
	_name_edit.text_changed.connect(func(t): form.name = t)
	_name_edit.text_submitted.connect(func(_t): _validate())
	_page.add_child(_name_edit)
	_h("Ton dieu")
	_note("Pour l'allure et les mots du jeu : chaque dieu veille sur ses fidèles à sa façon.")
	var ids := Heros.DIEUX.keys()
	_choices(ids.map(func(d): return "%s %s" % [Heros.DIEUX[d][2], Heros.DIEUX[d][0]]), ids.find(form.dieu), func(i):
		form.dieu = ids[i]
		_show("serment"), 3)
	_note("%s, %s." % [Heros.DIEUX[form.dieu][0], Heros.DIEUX[form.dieu][1]])
	_h("Mode hardcore")
	var hc := CheckButton.new()
	hc.text = "💀 Une seule vie"
	hc.button_pressed = bool(form.hardcore)
	hc.toggled.connect(func(on):
		form.hardcore = on
		_show("serment"))
	_page.add_child(hc)
	var warn := _note("Si ton héros tombe, il est effacé pour toujours : pas de retour au temple, aucun moyen de le récupérer.")
	if form.hardcore:
		warn.add_theme_color_override("font_color", UI.DANGER)
