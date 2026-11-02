class_name HeroSheet
extends Control
## La fiche du héros (touche C) : ses 9 attributs à placer, sa voie à choisir
## au niveau 10. Et la fiche publique des autres héros (clic sur leur nom).

signal attrs(points: Array)
signal voie(i: int)
signal closed

var me := {}        # le dernier état privé (opMe) : cl, v, q, qm, at, pt, lv, or
var hero := {}      # race, dieu, look, name : ce que le monde montre du héros
var public := {}    # la fiche d'un autre héros (opFiche) ; vide pour la sienne

var _body: VBoxContainer
var _tab := "attributs"
var _draft: Array = []
var _left: Label


func _init() -> void:
	set_anchors_preset(Control.PRESET_FULL_RECT)
	theme = UI.make_theme()


func _ready() -> void:
	var shade := ColorRect.new()
	shade.color = Color(0, 0, 0, 0.55)
	shade.set_anchors_preset(Control.PRESET_FULL_RECT)
	shade.gui_input.connect(func(e):
		if e is InputEventMouseButton and e.pressed:
			close())
	add_child(shade)
	var center := CenterContainer.new()
	center.set_anchors_preset(Control.PRESET_FULL_RECT)
	center.mouse_filter = Control.MOUSE_FILTER_IGNORE
	add_child(center)
	var panel := PanelContainer.new()
	panel.add_theme_stylebox_override("panel", UI.box(UI.STONE, UI.LUMEN, 16, 1, 18))
	panel.custom_minimum_size = Vector2(560, 0)
	center.add_child(panel)
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 10)
	panel.add_child(col)
	var head := HBoxContainer.new()
	col.add_child(head)
	var title := UI.label("", 26, UI.LUMEN)
	title.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	head.add_child(title)
	var x := UI.button("✕")
	x.custom_minimum_size = Vector2(48, 44)
	x.pressed.connect(close)
	head.add_child(x)
	if public.is_empty():
		title.text = "📜 Ton héros"
		var tabs := HBoxContainer.new()
		tabs.add_theme_constant_override("separation", 6)
		for t in [["attributs", "💪 Attributs"], ["voie", "🌟 Voie"]]:
			var b := UI.button(t[1])
			b.size_flags_horizontal = Control.SIZE_EXPAND_FILL
			var id: String = t[0]
			b.pressed.connect(func():
				_tab = id
				_render())
			tabs.add_child(b)
		col.add_child(tabs)
	else:
		title.text = "📜 " + str(public.get("n", ""))
	var scroll := ScrollContainer.new()
	scroll.horizontal_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
	scroll.custom_minimum_size = Vector2(0, clampf(get_viewport_rect().size.y - 260, 260, 640))
	col.add_child(scroll)
	_body = VBoxContainer.new()
	_body.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_body.add_theme_constant_override("separation", 8)
	scroll.add_child(_body)
	_draft = _ints(me.get("at", []))
	_render()


func close() -> void:
	closed.emit()
	queue_free()


func _unhandled_key_input(event: InputEvent) -> void:
	if event.pressed and (event.physical_keycode == KEY_ESCAPE or event.physical_keycode == KEY_C):
		get_viewport().set_input_as_handled()
		close()


## L'état privé a changé (niveau, points, épreuve…).
func update_me(d: Dictionary) -> void:
	var before: Array = me.get("at", [])
	me = d
	if (d.get("at", []) as Array) != before:
		_draft = _ints(d.get("at", []))
	if is_inside_tree():
		_render()


func _line(text: String, size: int = 16, color: Color = UI.PARCH) -> Label:
	var l := UI.label(text, size, color)
	_body.add_child(l)
	return l


func _render() -> void:
	for c in _body.get_children():
		c.queue_free()
	if not public.is_empty():
		_render_public()
	elif _tab == "attributs":
		_render_attrs()
	else:
		_render_voie()


func _who(race: String, cl: String, a: int, lv: int, d: String) -> String:
	var dieu: Array = Heros.DIEUX.get(d, ["", "", ""])
	return "%s · %s · niv. %d · %s %s" % [Heros.accorde(Heros.peuple(race).nom, a), Heros.accorde(Heros.classe(cl).noms, a),
		lv, dieu[2], dieu[0]]


func _render_public() -> void:
	var p := public
	var a := int((p.get("lk", {}) as Dictionary).get("a", 0))
	var fac: Dictionary = Heros.FACTIONS.get(str(p.get("f", "concorde")), Heros.FACTIONS.concorde)
	_line(_who(p.r, p.cl, a, int(p.lv), str(p.get("dieu", ""))), 17)
	_line("%s %s" % [fac.blason, fac.nom], 16, fac.couleur)
	var v := str(p.get("v", ""))
	if v != "":
		for sp in Heros.classe(p.cl).voies:
			if sp.id == v:
				_line("🌟 Voie : %s (%s)" % [sp.nom, Heros.ROLES.get(sp.role, "")], 16, UI.LUMEN)
	if p.get("hc", false):
		_line("💀 Hardcore : une seule vie.", 16, UI.DANGER)
	_line("❤️ %d points de vie · 🏆 %d trophées" % [int(p.get("hpm", 0)), int(p.get("tr", 0))], 16)
	var at: Array = p.get("at", [])
	var keys: Array = Heros.data().attributs
	var parts := []
	for i in mini(at.size(), keys.size()):
		if int(at[i]) > 0:
			parts.append("%s %d" % [Heros.ATTRIBUTS[keys[i]][1], int(at[i])])
	if not parts.is_empty():
		_line("  ".join(parts), 16, UI.MUTED)


## La « répartition conseillée » : la même méthode que le serveur (stats.go, Recommend).
static func conseil(weights: Array, n: int) -> Array:
	var out := []
	out.resize(weights.size())
	out.fill(0)
	var total := 0
	for w in weights:
		total += int(w)
	if total == 0 or n <= 0:
		return out
	for p in n:
		var best := 0
		var gap_best := -1.0
		for i in weights.size():
			var gap := float(weights[i]) * float(p + 1) / float(total) - float(out[i])
			if gap > gap_best:
				best = i
				gap_best = gap
		out[best] += 1
	return out


func _render_attrs() -> void:
	var lv := int(me.get("lv", 1))
	var total := int(me.get("pt", 0))
	var server: Array = _ints(me.get("at", []))
	var keys: Array = Heros.data().attributs
	var free := lv <= int(Heros.data().reset)
	var bonus := str((Heros.data().races.get(str(hero.get("race", "")), {}) as Dictionary).get("bonus", ""))
	_left = _line("", 18, UI.LUMEN)
	_line("5 points à chaque niveau. " + ("Jusqu'au niveau 10, tu peux tout redistribuer gratuitement." if free
		else "Les points placés le sont pour de bon."), 15, UI.MUTED)
	for i in keys.size():
		var info: Array = Heros.ATTRIBUTS[keys[i]]
		var row := HBoxContainer.new()
		row.add_theme_constant_override("separation", 8)
		var l := UI.label("%s %s" % [info[1], info[0]], 17)
		l.custom_minimum_size = Vector2(170, 0)
		l.tooltip_text = info[2]
		l.mouse_filter = Control.MOUSE_FILTER_PASS
		row.add_child(l)
		var minus := UI.button("−")
		minus.custom_minimum_size = Vector2(48, 40)
		var floor_v := 0 if free else (int(server[i]) if i < server.size() else 0)
		minus.disabled = int(_draft[i]) <= floor_v
		var k := i
		minus.pressed.connect(func():
			_draft[k] = int(_draft[k]) - 1
			_render())
		row.add_child(minus)
		var v := UI.label(str(int(_draft[i])) + (" +1" if keys[i] == bonus else ""), 17, UI.LUMEN)
		v.custom_minimum_size = Vector2(56, 0)
		v.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		row.add_child(v)
		var plus := UI.button("+")
		plus.custom_minimum_size = Vector2(48, 40)
		plus.disabled = _sum(_draft) >= total
		plus.pressed.connect(func():
			_draft[k] = int(_draft[k]) + 1
			_render())
		row.add_child(plus)
		var help := UI.label(info[2], 13, UI.MUTED)
		help.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		row.add_child(help)
		_body.add_child(row)
	_left.text = "Points à placer : %d" % (total - _sum(_draft))
	var actions := HBoxContainer.new()
	actions.add_theme_constant_override("separation", 8)
	var rec := UI.button("⭐ Répartition conseillée")
	rec.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	rec.pressed.connect(func():
		var c := Heros.classe(str(me.get("cl", "")))
		if free:
			_draft = conseil(c.conseil, total)
		else:
			# Après le niveau 10, le conseil ne place que les points restants.
			var extra := conseil(c.conseil, total - _sum(server))
			for i in _draft.size():
				_draft[i] = int(server[i]) + int(extra[i])
		_render())
	actions.add_child(rec)
	var ok := UI.button("Valider", true)
	ok.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	ok.disabled = _draft == server
	ok.pressed.connect(func(): attrs.emit(_draft.duplicate()))
	actions.add_child(ok)
	_body.add_child(actions)


static func _ints(a: Array) -> Array:
	var out := []
	for v in a:
		out.append(int(v))
	while out.size() < 9:
		out.append(0)
	return out


static func _sum(a: Array) -> int:
	var n := 0
	for v in a:
		n += int(v)
	return n


func _render_voie() -> void:
	var c := Heros.classe(str(me.get("cl", "")))
	var lv := int(me.get("lv", 1))
	var v := int(me.get("v", -1))
	var q := int(me.get("q", 0))
	var qm := int(me.get("qm", 5))
	var need := int(Heros.data().voie)
	if v < 0 and lv < need:
		_line("Les voies s'ouvrent au niveau %d. Tu es niveau %d : encore un peu de route !" % [need, lv], 17)
	elif v < 0 and q < qm:
		_line("🗡️ L'épreuve de la voie : vaincs %d Effacés pour prouver ta valeur (%d / %d)." % [qm, q, qm], 17, UI.LUMEN)
		_line("On les trouve à la Lisière Grise, là où les couleurs s'en vont.", 15, UI.MUTED)
	elif v < 0:
		_line("✨ L'épreuve est réussie ! Choisis ta voie : tu pourras en changer plus tard au temple, contre de l'or.", 17, UI.LUMEN)
	for i in 3:
		var sp: Dictionary = c.voies[i]
		var card := PanelContainer.new()
		var here := i == v
		card.add_theme_stylebox_override("panel", UI.box(UI.STONE_2, UI.LUMEN if here else UI.LINE, 12, 2 if here else 1, 10))
		var box := VBoxContainer.new()
		card.add_child(box)
		box.add_child(UI.label("%s · %s" % [sp.nom, Heros.ROLES.get(sp.role, "")], 18, UI.LUMEN if here else UI.PARCH))
		box.add_child(UI.label(sp.aide, 15, UI.MUTED))
		box.add_child(UI.label("Ultime (touche 8) : %s %s. %s" % [sp.ultime.icone, sp.ultime.nom, sp.ultime.aide], 15))
		if here:
			box.add_child(UI.label("C'est ta voie.", 15, UI.MOSS))
		elif lv >= need and (v >= 0 or q >= qm):
			var b := UI.button("Choisir cette voie" if v < 0 else "En changer au temple (%d or)" % (int(Heros.data().changement) * lv), v < 0)
			var k := i
			b.pressed.connect(func(): voie.emit(k))
			box.add_child(b)
		_body.add_child(card)
