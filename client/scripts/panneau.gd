class_name Panneau
extends PanelContainer
## Les panneaux des habitants et des ateliers (Acte V), à droite de l'écran,
## à côté du sac : l'étal d'Aelis, un atelier, un maître de métier, le coffre
## de l'auberge, le miroir de Solenn (apparences). Le serveur envoie leur
## contenu (opPanneau) et le renvoie quand le sac change.

signal closed

var kind := ""          # boutique, atelier, maitre, coffre, miroir
var data := {}
var sac := {}           # le dernier opSac (pour le miroir et l'argent)
var touch := false

var _body: VBoxContainer
var _title: Label
var _choix := ""         # atelier : le bonus choisi pour la prochaine fabrication
var _repete := 1
var _scroll_y := 0


func _ready() -> void:
	theme = UI.make_theme()
	add_theme_stylebox_override("panel", UI.box(Color(UI.STONE, 0.97), UI.EMBER, 14, 1, 12))
	set_anchors_and_offsets_preset(Control.PRESET_CENTER_RIGHT, Control.PRESET_MODE_MINSIZE, 14)
	grow_horizontal = Control.GROW_DIRECTION_BEGIN
	grow_vertical = Control.GROW_DIRECTION_BOTH
	custom_minimum_size = Vector2(400, 0)
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 8)
	add_child(col)
	var head := HBoxContainer.new()
	col.add_child(head)
	_title = UI.label("", 21, UI.LUMEN)
	_title.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	head.add_child(_title)
	var x := UI.button("✕")
	x.custom_minimum_size = Vector2(44, 40)
	x.focus_mode = Control.FOCUS_NONE
	x.pressed.connect(close)
	head.add_child(x)
	var scroll := ScrollContainer.new()
	scroll.horizontal_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
	scroll.custom_minimum_size = Vector2(380, clampf(get_viewport_rect().size.y - 200, 300, 560))
	col.add_child(scroll)
	_body = VBoxContainer.new()
	_body.add_theme_constant_override("separation", 8)
	_body.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	scroll.add_child(_body)
	# Lâcher un objet du sac sur le panneau : le vendre, le déposer au coffre.
	set_drag_forwarding(Callable(), func(_p, d): return d is Dictionary and d.has("sac") and (kind == "boutique" or kind == "coffre"),
		func(_p, d): _send({"a": "vendre" if kind == "boutique" else "coffre_depot", "i": int(d.sac)}))
	_render()


func close() -> void:
	Net.send(Net.OP_OBJET, {"a": "fermer"})
	closed.emit()
	queue_free()


func set_data(d: Dictionary) -> void:
	if str(d.get("p", "")) != kind:
		_choix = ""
	kind = str(d.get("p", ""))
	data = d
	if is_inside_tree():
		_render()


func set_sac(d: Dictionary) -> void:
	sac = d
	if kind == "miroir" and is_inside_tree():
		_render()


func _send(d: Dictionary) -> void:
	Net.send(Net.OP_OBJET, d)


func _render() -> void:
	if _body == null:
		return
	var sc := _body.get_parent() as ScrollContainer
	_scroll_y = sc.scroll_vertical
	for c in _body.get_children():
		c.queue_free()
	match kind:
		"boutique":
			_title.text = "🛒 L'étal d'Aelis"
			_boutique()
		"atelier":
			_title.text = "%s Atelier : %s (niv. %d)" % [data.get("ic", ""), data.get("nom", ""), int(data.get("lv", 0))]
			_atelier()
		"maitre":
			_title.text = "🎓 " + {"doran": "Doran", "nessa": "Nessa", "gwenn": "Gwenn", "mabon": "Maître Mabon", "aubin": "Aubin"}.get(str(data.get("npc", "")), "Maître")
			_maitre()
		"coffre":
			_title.text = "🧰 Le coffre de ton compte"
			_coffre()
		"miroir":
			_title.text = "🪞 Le miroir de Solenn"
			_miroir()
	(func(): sc.scroll_vertical = _scroll_y).call_deferred()


func _line(t: String, size := 15, c := UI.MUTED) -> Label:
	var l := UI.label(t, size, c)
	_body.add_child(l)
	return l


func _court(t: String) -> Label:
	var l := UI.label(t, 15, UI.PARCH)
	l.autowrap_mode = TextServer.AUTOWRAP_OFF
	return l


func _btn(t: String, prim := false) -> Button:
	var b := UI.button(t, prim)
	b.custom_minimum_size = Vector2(0, 40)
	b.focus_mode = Control.FOCUS_NONE
	return b


func _rang(it: Dictionary, droite: Control) -> HBoxContainer:
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 8)
	var c := Objets.Case.new(it, "", 46)
	row.add_child(c)
	var col := VBoxContainer.new()
	col.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	var n := UI.label(str(it.get("nom", "")), 16, Objets.couleur(int(it.get("r", 0))))
	col.add_child(n)
	if str(it.get("ds", "")) != "":
		col.add_child(UI.label(str(it.ds), 13, UI.MUTED))
	row.add_child(col)
	row.add_child(droite)
	return row


# --- L'étal d'Aelis ---------------------------------------------------------------

func _boutique() -> void:
	_line("« Je vends l'essentiel, et je rachète tout. Pas cher, mais je rachète ! » Pour vendre : clic droit (ou « Vendre ») sur un objet du sac." + ("" if touch else " Tu peux aussi le glisser ici."), 14)
	var argent := int(sac.get("or", 0))
	for a in Objets.liste(data.get("articles")):
		var px := int(a.px)
		var box := HBoxContainer.new()
		var b := _btn(Objets.monnaie(px))
		b.disabled = argent < px
		b.custom_minimum_size = Vector2(120, 40)
		var base := str(a.b)
		b.pressed.connect(func(): _send({"a": "acheter", "b": base, "n": 1}))
		box.add_child(b)
		var it: Dictionary = a.get("it", {})
		it["ds"] = str(a.get("ds", ""))
		_body.add_child(_rang(it, box))
	var rachat: Array = Objets.liste(data.get("rachat"))
	if not rachat.is_empty():
		_line("Racheter ce que tu as vendu :", 16, UI.PARCH)
		for i in rachat.size():
			var r: Dictionary = rachat[i]
			var b := _btn(Objets.monnaie(int(r.px)))
			b.custom_minimum_size = Vector2(120, 40)
			b.disabled = argent < int(r.px)
			var k := i
			b.pressed.connect(func(): _send({"a": "racheter", "i": k}))
			_body.add_child(_rang(r.it, b))


# --- Un atelier -------------------------------------------------------------------

func _atelier() -> void:
	if not bool(data.get("connu", false)):
		_line("Tu ne connais pas encore ce métier. Son maître te l'apprendra.", 16, UI.DANGER)
		return
	var recs: Array = Objets.liste(data.get("recettes"))
	_line("Choisis une recette. Pour une pièce d'équipement, choisis aussi le bonus que tu y mets : c'est l'avantage de l'artisan. Avec un peu de chance, elle sort d'une rareté au-dessus !", 14)
	# Le bonus choisi.
	var choix: Array = Objets.liste(data.get("choix"))
	if not choix.is_empty():
		var opt := OptionButton.new()
		opt.focus_mode = Control.FOCUS_NONE
		opt.add_item("Bonus au hasard")
		var sel := 0
		for i in choix.size():
			opt.add_item(str(choix[i][1]))
			if str(choix[i][0]) == _choix:
				sel = i + 1
		opt.select(sel)
		opt.item_selected.connect(func(i): _choix = "" if i == 0 else str(choix[i - 1][0]))
		var row := HBoxContainer.new()
		row.add_theme_constant_override("separation", 8)
		row.add_child(_court("Bonus choisi :"))
		opt.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		opt.custom_minimum_size = Vector2(0, 40)
		row.add_child(opt)
		_body.add_child(row)
	var rep := HBoxContainer.new()
	rep.add_theme_constant_override("separation", 6)
	rep.add_child(_court("Combien :"))
	for n in [1, 5, 20]:
		var b := _btn("× %d" % n)
		b.toggle_mode = true
		b.button_pressed = _repete == n
		var nn: int = n
		b.pressed.connect(func():
			_repete = nn
			_render())
		rep.add_child(b)
	_body.add_child(rep)
	for r in _tri(recs):
		_body.add_child(_recette(r, true))
	if recs.is_empty():
		_line("Aucune recette pour l'instant : va voir le maître de ce métier.", 15)


## Les recettes : d'abord celles pour sa classe, puis par niveau.
static func _tri(recs: Array) -> Array:
	var out := recs.duplicate()
	out.sort_custom(func(a, b):
		var pa := bool(a.get("pour_moi", true))
		var pb := bool(b.get("pour_moi", true))
		if pa != pb:
			return pa
		if int(a.get("req", 1)) != int(b.get("req", 1)):
			return int(a.get("req", 1)) < int(b.get("req", 1))
		return str(a.get("nom", "")) < str(b.get("nom", "")))
	return out


func _recette(r: Dictionary, fabriquer: bool) -> PanelContainer:
	var card := PanelContainer.new()
	card.add_theme_stylebox_override("panel", UI.box(UI.STONE_2, Objets.couleur(int(r.get("r", 0))) if bool(r.get("ok", false)) or not fabriquer else UI.LINE, 10, 1, 8))
	var col := VBoxContainer.new()
	card.add_child(col)
	var head := HBoxContainer.new()
	col.add_child(head)
	var t := UI.label("%s %s%s" % [r.get("ic", ""), r.get("nom", ""), " ×%d" % int(r.n) if int(r.get("n", 1)) > 1 else ""], 16, Objets.couleur(int(r.get("r", 0))))
	t.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	head.add_child(t)
	var req := UI.label("métier %d" % int(r.get("req", 1)), 13, UI.MUTED)
	req.autowrap_mode = TextServer.AUTOWRAP_OFF
	head.add_child(req)
	var bits := []
	if bool(r.get("eq", false)):
		bits.append(Objets.nom_emplacement(str(r.get("sl", ""))))
		if r.has("ar"):
			bits.append(Objets.ARMURES.get(str(r.ar), ""))
		if not bool(r.get("pour_moi", true)):
			bits.append("pas pour ta classe")
	elif str(r.get("ds", "")) != "":
		bits.append(str(r.ds))
	if not bits.is_empty():
		col.add_child(UI.label(" · ".join(bits), 13, UI.MUTED))
	var need := []
	for n in Objets.liste(r.get("need")):
		need.append("%s %s %d/%d" % [n[1], n[0], int(n[2]), int(n[3])])
	var nl := UI.label(" · ".join(need), 13, UI.MOSS if bool(r.get("ok", false)) else UI.PARCH)
	col.add_child(nl)
	if fabriquer:
		var b := _btn("Fabriquer" + (" × %d" % _repete if _repete > 1 else ""), bool(r.get("ok", false)))
		b.disabled = not bool(r.get("ok", false))
		var id := str(r.id)
		b.pressed.connect(func(): _send({"a": "fabriquer", "r": id, "c": _choix if bool(r.get("eq", false)) else "", "n": _repete}))
		col.add_child(b)
	return card


# --- Un maître de métier ----------------------------------------------------------

func _maitre() -> void:
	var art: Array = Objets.liste(data.get("art"))
	if bool(data.get("forge", false)):
		var rep := int(data.get("rep", 0))
		var b := _btn("🔧 Réparer tout (%s)" % Objets.monnaie(rep) if rep > 0 else "🔧 Rien à réparer", rep > 0)
		b.disabled = rep <= 0
		b.pressed.connect(func(): _send({"a": "reparer"}))
		_body.add_child(b)
	for m in Objets.liste(data.get("metiers")):
		var id := str(m.id)
		var card := PanelContainer.new()
		card.add_theme_stylebox_override("panel", UI.box(UI.STONE_2, UI.LUMEN if bool(m.connu) else UI.LINE, 10, 1, 8))
		var col := VBoxContainer.new()
		col.add_theme_constant_override("separation", 6)
		card.add_child(col)
		col.add_child(UI.label("%s %s%s" % [m.ic, m.nom, " — niveau %d" % int(m.lv) if bool(m.connu) else ""], 18, UI.PARCH))
		if bool(m.recolte):
			col.add_child(UI.label("Une récolte : tout le monde la connaît déjà. Monte de niveau en récoltant.", 13, UI.MUTED))
		elif not bool(m.connu):
			var b := _btn("Apprendre ce métier", true)
			var plein := art.size() >= int(data.get("max", 2)) and id != "cuisine"
			b.disabled = plein
			if plein:
				col.add_child(UI.label("Tu as déjà deux artisanats. Oublie-en un pour apprendre celui-ci.", 13, UI.DANGER))
			b.pressed.connect(func(): _send({"a": "apprendre", "r": id}))
			col.add_child(b)
		else:
			var recs: Array = Objets.liste(m.get("recettes"))
			if recs.is_empty():
				col.add_child(UI.label("Tu connais toutes mes recettes pour l'instant. Les plans rares se trouvent sur les monstres…", 13, UI.MUTED))
			for r in _tri(recs):
				var rc := _recette(r, false)
				var b := _btn("Apprendre (%s)" % Objets.monnaie(int(r.get("px", 0))), true)
				var rid := str(r.id)
				b.disabled = int(sac.get("or", 0)) < int(r.get("px", 0)) or int(m.lv) < int(r.get("req", 1))
				if int(m.lv) < int(r.get("req", 1)):
					b.text = "Il faut le niveau %d" % int(r.req)
				b.pressed.connect(func(): _send({"a": "recette", "r": rid}))
				rc.get_child(0).add_child(b)
				col.add_child(rc)
			if id != "cuisine":
				var f := _btn("Oublier ce métier")
				var armed := [false]
				f.pressed.connect(func():
					if armed[0]:
						_send({"a": "oublier", "r": id})
					else:
						armed[0] = true
						f.text = "Vraiment ? Niveau et recettes perdus (re-clic)"
						f.add_theme_color_override("font_color", UI.DANGER))
				col.add_child(f)
		_body.add_child(card)


# --- Le coffre ---------------------------------------------------------------------

func _coffre() -> void:
	_line("Partagé entre les héros de ton compte. Pour déposer : clic droit (ou « Déposer ») sur un objet du sac. Toucher un objet du coffre le reprend.", 14)
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 6)
	row.add_child(UI.label("🪙 " + Objets.monnaie(int(data.get("or", 0))), 17, UI.LUMEN))
	_body.add_child(row)
	var money := HBoxContainer.new()
	money.add_theme_constant_override("separation", 6)
	var amount := SpinBox.new()
	amount.min_value = 0
	amount.max_value = 100000
	amount.value = 1
	amount.suffix = "argent"
	amount.custom_minimum_size = Vector2(140, 40)
	money.add_child(amount)
	var dep := _btn("Déposer")
	dep.pressed.connect(func(): _send({"a": "coffre_or", "n": int(amount.value) * 100}))
	money.add_child(dep)
	var ret := _btn("Reprendre")
	ret.pressed.connect(func(): _send({"a": "coffre_or", "n": -int(amount.value) * 100}))
	money.add_child(ret)
	_body.add_child(money)
	var grid := GridContainer.new()
	grid.columns = 6
	grid.add_theme_constant_override("h_separation", 4)
	grid.add_theme_constant_override("v_separation", 4)
	_body.add_child(grid)
	var items: Array = Objets.liste(data.get("it"))
	for i in items.size():
		var it = items[i]
		var c := Objets.Case.new(it if it is Dictionary else {}, "", 54)
		var k := i
		c.gui_input.connect(func(e):
			if e is InputEventMouseButton and e.pressed and (e.button_index == MOUSE_BUTTON_LEFT or e.button_index == MOUSE_BUTTON_RIGHT) and it is Dictionary:
				_send({"a": "coffre_retrait", "i": k}))
		grid.add_child(c)


# --- Le miroir de Solenn : les apparences --------------------------------------------

func _miroir() -> void:
	_line("Une pièce que tu portes peut prendre l'allure d'une autre pièce du même genre que tu as déjà portée. Le miroir demande une petite offrande (%s par niveau de la pièce)." % Objets.monnaie(int(data.get("px", 25)), true), 14)
	var apps: Array = Objets.liste(data.get("apps"))
	var eq: Dictionary = sac.get("eq", {})
	for e in Objets.EMPLACEMENTS:
		var slot: String = e[0]
		if not eq.has(slot) or slot in ["cou", "anneau1", "anneau2"]:
			continue
		var it: Dictionary = eq[slot]
		var col := VBoxContainer.new()
		var t := UI.label("%s %s : %s" % [e[2], e[1], it.get("nom", "")], 16, Objets.couleur(int(it.get("r", 0))))
		col.add_child(t)
		var flow := HFlowContainer.new()
		flow.add_theme_constant_override("h_separation", 4)
		flow.add_theme_constant_override("v_separation", 4)
		var genre := str(it.get("b", ""))
		var n := 0
		for a in apps:
			var base := str(a.lk).split(":")[0]
			if base != genre or str(a.lk) == "%s:%d" % [genre, int(it.r)]:
				continue
			n += 1
			var b := _btn("%s %s" % [a.ic, Objets.RARETES[clampi(int(a.r), 0, 5)]])
			b.add_theme_color_override("font_color", Objets.couleur(int(a.r)))
			if str(it.get("lk", "")) == str(a.lk):
				b.text = "✓ " + b.text
			var lk := str(a.lk)
			b.pressed.connect(func(): _send({"a": "miroir", "s": slot, "lk": lk}))
			flow.add_child(b)
		if str(it.get("lk", "")) != "":
			var b := _btn("Son allure à elle")
			b.pressed.connect(func(): _send({"a": "miroir", "s": slot, "lk": ""}))
			flow.add_child(b)
		if n == 0 and str(it.get("lk", "")) == "":
			flow.add_child(UI.label("Aucune autre allure connue pour cette pièce.", 13, UI.MUTED))
		col.add_child(flow)
		_body.add_child(col)


func _unhandled_key_input(event: InputEvent) -> void:
	if event.pressed and not event.echo and event.physical_keycode == KEY_ESCAPE:
		get_viewport().set_input_as_handled()
		close()
