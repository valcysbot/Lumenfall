class_name SacWindow
extends PanelContainer
## La fenêtre du sac (touche I, bouton 🎒) : le héros au centre, ses 12
## emplacements autour, le sac à droite, les métiers dans un second onglet.
## Sur PC : glisser-déposer, double-clic ou clic droit pour équiper ou utiliser,
## bulle au survol. Sur téléphone : toucher un objet, puis « Équiper ».
## (Conseil des Objets, Acte V.)

signal closed
signal offer(id: String)        # « Proposer » pendant un échange
signal action_done

## [identifiant, nom, icône, maître, récolte ?] — comme le serveur (world/metiers.go).
const METIERS := [
	["minage", "Minage", "⛏️", "Doran", true], ["herboristerie", "Herboristerie", "🌿", "Nessa", true],
	["depecage", "Dépeçage", "🔪", "Gwenn", true], ["bucheronnage", "Bûcheronnage", "🪓", "Aubin", true],
	["forge", "Forge", "⚒️", "Doran", false], ["couture", "Couture", "🧵", "Gwenn", false],
	["cuir", "Travail du cuir", "🧶", "Gwenn", false], ["alchimie", "Alchimie", "⚗️", "Nessa", false],
	["joaillerie", "Joaillerie", "💎", "Doran", false], ["cuisine", "Cuisine", "🍲", "Mabon", false],
]
const GAUCHE := ["tete", "epaules", "torse", "dos", "cou", "anneau1"]
const DROITE := ["mains", "jambes", "pieds", "anneau2", "arme", "main_gauche"]

var data := {}          # le dernier opSac
var hero := {}          # race, classe, look : pour dessiner le héros au centre
var touch := false
var ctx := ""           # un panneau ouvert à côté : boutique, coffre, echange

var _tab := "sac"
var _sel := {}          # {"sac": i} ou {"eq": slot}
var _body: VBoxContainer
var _title: Label
var _money: Label
var _clic_t := 0.0
var _clic_i := ""


func _ready() -> void:
	theme = UI.make_theme()
	add_theme_stylebox_override("panel", UI.box(Color(UI.STONE, 0.97), UI.LUMEN, 14, 1, 12))
	set_anchors_and_offsets_preset(Control.PRESET_CENTER_LEFT, Control.PRESET_MODE_MINSIZE, 14)
	grow_vertical = Control.GROW_DIRECTION_BOTH
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 8)
	add_child(col)
	var head := HBoxContainer.new()
	head.add_theme_constant_override("separation", 8)
	col.add_child(head)
	_title = UI.label("🎒 Ton sac", 22, UI.LUMEN)
	_title.autowrap_mode = TextServer.AUTOWRAP_OFF
	head.add_child(_title)
	for t in [["sac", "🎒 Sac"], ["metiers", "⚒️ Métiers"]]:
		var b := UI.button(t[1])
		b.custom_minimum_size = Vector2(0, 40)
		b.focus_mode = Control.FOCUS_NONE
		var id: String = t[0]
		b.pressed.connect(func():
			_tab = id
			_render())
		head.add_child(b)
	_money = UI.label("", 17, UI.LUMEN)
	_money.autowrap_mode = TextServer.AUTOWRAP_OFF
	_money.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_money.horizontal_alignment = HORIZONTAL_ALIGNMENT_RIGHT
	head.add_child(_money)
	var x := UI.button("✕")
	x.custom_minimum_size = Vector2(44, 40)
	x.focus_mode = Control.FOCUS_NONE
	x.pressed.connect(close)
	head.add_child(x)
	_body = VBoxContainer.new()
	_body.add_theme_constant_override("separation", 8)
	col.add_child(_body)
	_render()


func close() -> void:
	closed.emit()
	queue_free()


func set_data(d: Dictionary) -> void:
	data = d
	# L'objet choisi a pu bouger (vendu, équipé…) : on garde le choix s'il est encore là.
	if _sel.has("sac") and _item_sac(int(_sel.sac)).is_empty():
		_sel = {}
	if _sel.has("eq") and not data.get("eq", {}).has(_sel.eq):
		_sel = {}
	if is_inside_tree():
		_render()


func set_ctx(c: String) -> void:
	ctx = c
	if is_inside_tree():
		_render()


func _send(d: Dictionary) -> void:
	Net.send(Net.OP_OBJET, d)
	action_done.emit()


func _item_sac(i: int) -> Dictionary:
	var sac: Array = Objets.liste(data.get("sac"))
	if i < 0 or i >= sac.size() or sac[i] == null:
		return {}
	return sac[i]


func _porte(slot: String) -> Dictionary:
	var eq: Dictionary = data.get("eq", {})
	var v = eq.get(slot)
	return v if v is Dictionary else {}


func _render() -> void:
	if _body == null:
		return
	for c in _body.get_children():
		c.queue_free()
	var places := 0
	var pris := 0
	for it in Objets.liste(data.get("sac")):
		places += 1
		if it != null:
			pris += 1
	_title.text = "🎒 Ton sac  %d / %d" % [pris, places] if _tab == "sac" else "⚒️ Tes métiers"
	_money.text = "🪙 " + Objets.monnaie(int(data.get("or", 0)))
	if _tab == "metiers":
		_render_metiers()
	else:
		_render_sac()


# --- Le sac et l'équipement -----------------------------------------------------

func _render_sac() -> void:
	var px := 50.0 if touch else 52.0
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 10)
	_body.add_child(row)

	# Le héros, ses emplacements autour.
	var gauche := VBoxContainer.new()
	var droite := VBoxContainer.new()
	for c in [gauche, droite]:
		c.add_theme_constant_override("separation", 4)
	for s in GAUCHE:
		gauche.add_child(_case_eq(s, px))
	for s in DROITE:
		droite.add_child(_case_eq(s, px))
	row.add_child(gauche)
	var mid := VBoxContainer.new()
	mid.custom_minimum_size = Vector2(150, 0)
	var stage := Control.new()
	stage.custom_minimum_size = Vector2(150, 250)
	stage.clip_contents = true
	mid.add_child(stage)
	var hs := PeupleSprite.new()
	var race := str(hero.get("race", "humain"))
	var lk: Dictionary = hero.get("look", {})
	var z := 2.6 * Heros.echelle(race, lk)
	hs.scale = Vector2(z, z)
	hs.position = Vector2(75, 230)
	hs.dir = 1
	stage.add_child(hs)
	hs.dress(Heros.couches(race, str(hero.get("classe", "")), lk, _eq_look()))
	var st := UI.label("🛡️ %s   ⚔️ %s" % [Objets._n(float(data.get("arm", 0))), Objets._n(float(data.get("pw", 0)))], 15, UI.PARCH)
	st.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	st.tooltip_text = "Armure (moins de dégâts reçus) · Puissance de ton arme"
	st.mouse_filter = Control.MOUSE_FILTER_PASS
	mid.add_child(st)
	if int(data.get("rep", 0)) > 0:
		var rp := UI.label("🔧 Réparer : %s (Doran)" % Objets.monnaie(int(data.rep)), 13, UI.DANGER)
		rp.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		mid.add_child(rp)
	row.add_child(mid)
	row.add_child(droite)

	# Le sac.
	var right := VBoxContainer.new()
	right.add_theme_constant_override("separation", 6)
	row.add_child(right)
	var scroll := ScrollContainer.new()
	scroll.horizontal_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
	scroll.custom_minimum_size = Vector2(7 * (px + 4) + 12, 6 * (px + 4))
	right.add_child(scroll)
	var grid := GridContainer.new()
	grid.columns = 7
	grid.add_theme_constant_override("h_separation", 4)
	grid.add_theme_constant_override("v_separation", 4)
	scroll.add_child(grid)
	var sac: Array = Objets.liste(data.get("sac"))
	for i in sac.size():
		grid.add_child(_case_sac(i, px))
	var tools := HBoxContainer.new()
	tools.add_theme_constant_override("separation", 6)
	right.add_child(tools)
	var trier := UI.button("↕ Trier")
	trier.custom_minimum_size = Vector2(0, 40)
	trier.focus_mode = Control.FOCUS_NONE
	trier.pressed.connect(func(): _send({"a": "trier"}))
	tools.add_child(trier)
	tools.add_child(_rapides())

	# Les repas et les élixirs en cours.
	var bu: Dictionary = data.get("bu", {})
	if not bu.is_empty():
		var l := []
		for k in bu:
			l.append("%s %s (%d min)" % [bu[k].ic, bu[k].nom, ceili(float(bu[k].t) / 60.0)])
		var b := UI.label("✨ " + "  ·  ".join(l), 14, UI.MOSS)
		_body.add_child(b)

	# L'objet choisi : sa fiche, et quoi en faire.
	_body.add_child(_detail())


func _eq_look() -> Dictionary:
	var out := {}
	var eq: Dictionary = data.get("eq", {})
	for s in eq:
		if s in ["cou", "anneau1", "anneau2"]:
			continue
		var it: Dictionary = eq[s]
		out[s] = str(it.get("lk", "")) if str(it.get("lk", "")) != "" else "%s:%d" % [it.b, int(it.r)]
	return out


func _case_eq(slot: String, px: float) -> Control:
	var icone := ""
	for e in Objets.EMPLACEMENTS:
		if e[0] == slot:
			icone = e[2]
	var c := Objets.Case.new(_porte(slot), icone, px)
	c.choisie = _sel.get("eq", "") == slot
	c.refresh()
	if c.it.is_empty():
		c.tooltip_text = Objets.nom_emplacement(slot)
	c.gui_input.connect(_clic.bind("eq", slot))
	c.set_drag_forwarding(
		func(_p): return _drag({"eq": slot}, c.it),
		func(_p, d): return d is Dictionary and d.has("sac"),
		func(_p, d): _send({"a": "equiper", "i": int(d.sac), "s": slot}))
	return c


func _case_sac(i: int, px: float) -> Control:
	var it := _item_sac(i)
	var c := Objets.Case.new(it, "", px)
	c.porte = _porte(Objets.emplacement_de(it)) if not it.is_empty() else {}
	c.choisie = _sel.has("sac") and int(_sel.sac) == i
	c.refresh()
	c.gui_input.connect(_clic.bind("sac", i))
	c.set_drag_forwarding(
		func(_p): return _drag({"sac": i}, it),
		func(_p, d): return d is Dictionary and (d.has("sac") or d.has("eq")),
		func(_p, d):
			if d.has("eq"):
				_send({"a": "retirer", "s": d.eq})
			elif int(d.sac) != i:
				_send({"a": "ranger", "i": int(d.sac), "j": i}))
	return c


func _drag(what: Dictionary, it: Dictionary) -> Variant:
	if it.is_empty() or touch:
		return null
	var l := UI.label(str(it.get("ic", "")), 30)
	l.autowrap_mode = TextServer.AUTOWRAP_OFF
	set_drag_preview(l)
	what["id"] = str(it.get("id", ""))
	return what


## Un clic : choisir. Double-clic ou clic droit : équiper, retirer ou utiliser.
func _clic(event: InputEvent, where: String, key) -> void:
	if not (event is InputEventMouseButton and event.pressed):
		return
	var vite := false
	if event.button_index == MOUSE_BUTTON_RIGHT:
		vite = true
	elif event.button_index == MOUSE_BUTTON_LEFT:
		var id := "%s:%s" % [where, key]
		vite = event.double_click or (_clic_i == id and Time.get_ticks_msec() / 1000.0 - _clic_t < 0.35 and not touch)
		_clic_i = id
		_clic_t = Time.get_ticks_msec() / 1000.0
	else:
		return
	if vite and not touch:
		if where == "eq":
			_send({"a": "retirer", "s": key})
		else:
			_principal(int(key))
		return
	_sel = {where: key}
	_render()


## L'action principale d'un objet du sac selon ce qui est ouvert à côté.
func _principal(i: int) -> void:
	var it := _item_sac(i)
	if it.is_empty():
		return
	match ctx:
		"boutique":
			_send({"a": "vendre", "i": i})
		"coffre":
			_send({"a": "coffre_depot", "i": i})
		"echange":
			offer.emit(str(it.id))
		_:
			if str(it.get("k", "")) == "equipement":
				_send({"a": "equiper", "i": i})
			elif bool(it.get("u", false)):
				_send({"a": "utiliser", "i": i})


func _detail() -> Control:
	var box := PanelContainer.new()
	box.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.6), UI.LINE, 10, 1, 8))
	box.custom_minimum_size = Vector2(0, 120)
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 12)
	box.add_child(row)
	var it := {}
	if _sel.has("sac"):
		it = _item_sac(int(_sel.sac))
	elif _sel.has("eq"):
		it = _porte(str(_sel.eq))
	if it.is_empty():
		var aide := "Touche un objet pour le voir, puis « Équiper »." if touch \
			else "Survole un objet pour le voir. Glisse-le sur ton héros, ou double-clic (ou clic droit) pour l'équiper ou l'utiliser."
		if ctx == "boutique":
			aide += " Aelis est là : clic droit pour vendre."
		elif ctx == "coffre":
			aide += " Le coffre est ouvert : clic droit pour y déposer."
		elif ctx == "echange":
			aide += " Échange en cours : clic droit pour proposer un objet."
		row.add_child(UI.label(aide, 15, UI.MUTED))
		row.get_child(0).size_flags_horizontal = Control.SIZE_EXPAND_FILL
		return box
	var porte := {}
	if _sel.has("sac"):
		porte = _porte(Objets.emplacement_de(it))
	var f := Objets.fiche(it, porte)
	f.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	row.add_child(f)
	var acts := VBoxContainer.new()
	acts.add_theme_constant_override("separation", 6)
	row.add_child(acts)
	var add := func(t: String, d: Dictionary, prim: bool = false):
		var b := UI.button(t, prim)
		b.custom_minimum_size = Vector2(170, 42)
		b.focus_mode = Control.FOCUS_NONE
		b.pressed.connect(func(): _send(d))
		acts.add_child(b)
	if _sel.has("eq"):
		add.call("Retirer", {"a": "retirer", "s": _sel.eq}, true)
		return box
	var i := int(_sel.sac)
	var k := str(it.get("k", ""))
	if ctx == "boutique":
		add.call("Vendre (%s)" % Objets.monnaie(int(it.get("px", 0)) * maxi(1, int(it.get("n", 1)))), {"a": "vendre", "i": i}, true)
	elif ctx == "coffre":
		add.call("Déposer au coffre", {"a": "coffre_depot", "i": i}, true)
	elif ctx == "echange" and not bool(it.get("l", false)):
		var b := UI.button("Proposer", true)
		b.custom_minimum_size = Vector2(170, 42)
		b.focus_mode = Control.FOCUS_NONE
		b.pressed.connect(func(): offer.emit(str(it.id)))
		acts.add_child(b)
	if k == "equipement":
		add.call("Équiper", {"a": "equiper", "i": i}, ctx == "")
		if str(it.get("sl", "")) == "anneau":
			add.call("Équiper (2ᵉ anneau)", {"a": "equiper", "i": i, "s": "anneau2"})
	elif bool(it.get("u", false)):
		add.call("Utiliser", {"a": "utiliser", "i": i}, ctx == "")
	if k == "provision":
		var rap: Array = data.get("rap", ["", ""])
		var b := str(it.b)
		var row2 := HBoxContainer.new()
		for n in 2:
			var on := rap.size() > n and str(rap[n]) == b
			var r := UI.button(("✓ " if on else "") + "Raccourci %d" % (n + 1))
			r.custom_minimum_size = Vector2(0, 38)
			r.focus_mode = Control.FOCUS_NONE
			var nn := n
			r.pressed.connect(func(): _send({"a": "rapide", "n": nn, "b": "" if on else b}))
			row2.add_child(r)
		acts.add_child(row2)
		if b.contains("potion"):
			var on := str(data.get("po", "")) == b
			var r := UI.button(("✓ " if on else "") + "Potion de la touche %s" % Touches.label("potion"))
			r.custom_minimum_size = Vector2(0, 38)
			r.focus_mode = Control.FOCUS_NONE
			r.pressed.connect(func(): _send({"a": "rapide", "n": 2, "b": "" if on else b}))
			acts.add_child(r)
	var jeter := UI.button("🗑 Jeter")
	jeter.custom_minimum_size = Vector2(170, 38)
	jeter.focus_mode = Control.FOCUS_NONE
	var armed := [false]
	jeter.pressed.connect(func():
		if armed[0]:
			_send({"a": "jeter", "i": i})
		else:
			armed[0] = true
			jeter.text = "Vraiment ? (re-clic)"
			jeter.add_theme_color_override("font_color", UI.DANGER))
	acts.add_child(jeter)
	return box


## Les deux raccourcis (touches F1 et F2 par défaut… ici « Raccourci 1/2 »).
func _rapides() -> Control:
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 6)
	var rap: Array = data.get("rap", ["", ""])
	for n in 2:
		var b := str(rap[n]) if rap.size() > n else ""
		var l := UI.label("%s %s" % [Touches.label("rapide_%d" % (n + 1)), _icone_base(b) if b != "" else "—"], 15, UI.MUTED)
		l.autowrap_mode = TextServer.AUTOWRAP_OFF
		l.tooltip_text = "Raccourci %d : choisis une provision dans le sac, puis « Raccourci %d »." % [n + 1, n + 1]
		l.mouse_filter = Control.MOUSE_FILTER_PASS
		row.add_child(l)
	return row


func _icone_base(b: String) -> String:
	for it in Objets.liste(data.get("sac")):
		if it != null and str(it.b) == b:
			return str(it.ic)
	return "·"


# --- Les métiers -----------------------------------------------------------------

func _render_metiers() -> void:
	var me: Dictionary = data.get("me", {})
	var art: Array = Objets.liste(data.get("art"))
	_body.add_child(UI.label("Les récoltes sont pour tout le monde ; deux artisanats au choix, et la cuisine en plus. Les maîtres du village les enseignent.", 15, UI.MUTED))
	var grid := GridContainer.new()
	grid.columns = 2
	grid.add_theme_constant_override("h_separation", 14)
	grid.add_theme_constant_override("v_separation", 8)
	_body.add_child(grid)
	for m in METIERS:
		var id: String = m[0]
		var known := int(me.get(id, {}).get("lv", 0)) > 0
		var card := PanelContainer.new()
		card.add_theme_stylebox_override("panel", UI.box(UI.STONE_2, UI.LUMEN if known else UI.LINE, 10, 1, 8))
		card.custom_minimum_size = Vector2(330, 0)
		var col := VBoxContainer.new()
		card.add_child(col)
		var lv := int(me.get(id, {}).get("lv", 0))
		col.add_child(UI.label("%s %s%s" % [m[2], m[1], "  — niveau %d" % lv if known else ""], 17, UI.PARCH if known else UI.MUTED))
		if known:
			var bar := ProgressBar.new()
			bar.show_percentage = false
			bar.custom_minimum_size = Vector2(0, 8)
			bar.max_value = maxf(1, float(me[id].get("xpn", 1)))
			bar.value = float(me[id].get("xp", 0))
			col.add_child(bar)
			var aide := ""
			match id:
				"minage": aide = "Clique sur un filon (cuivre au Val, fer dans la Lisière)."
				"herboristerie": aide = "Clique sur une plante : sauge, lin, lysimaque…"
				"depecage": aide = "Clique sur un sanglier ou un loup tombé."
				"bucheronnage": aide = "Clique sur un tronc à l'orée des bois."
				_: aide = "À son atelier, au village (maître : %s)." % m[3]
			col.add_child(UI.label(aide, 13, UI.MUTED))
		else:
			col.add_child(UI.label("À apprendre auprès de %s%s." % [m[3], "" if art.size() < 2 else " (tu as déjà deux artisanats)"], 13, UI.MUTED))
		grid.add_child(card)


func _unhandled_key_input(event: InputEvent) -> void:
	if not event.pressed or event.echo:
		return
	if event.physical_keycode == KEY_ESCAPE or Touches.action_for_event(event) == "sac":
		get_viewport().set_input_as_handled()
		close()
