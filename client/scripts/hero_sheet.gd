class_name HeroSheet
extends Control
## La fiche du héros (touche C) : ses 9 attributs à placer, sa voie à choisir
## au niveau 10, ses talents (onglet 🌳). Et la fiche publique des autres héros (clic sur leur nom).

signal attrs(points: Array)
signal voie(i: int)
signal talents(t: Array)
signal closed

var me := {}        # le dernier état privé (opMe) : cl, v, q, qm, at, pt, lv, or
var hero := {}      # race, dieu, look, name : ce que le monde montre du héros
var public := {}    # la fiche d'un autre héros (opFiche) ; vide pour la sienne

var _body: VBoxContainer
var _tab := "attributs"
var _draft: Array = []
var _tdraft: Array = []    # les talents en cours de placement
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
		for t in [["attributs", "💪 Attributs"], ["voie", "🌟 Voie"], ["talents", "🌳 Talents"]]:
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
	_tdraft = Heros.talents_ints(me.get("ta", []))
	_render()


func close() -> void:
	closed.emit()
	queue_free()


func _unhandled_key_input(event: InputEvent) -> void:
	if event.pressed and (event.physical_keycode == KEY_ESCAPE or event.physical_keycode == Touches.key("fiche")):
		get_viewport().set_input_as_handled()
		close()


## L'état privé a changé (niveau, points, épreuve…).
## Le serveur l'envoie souvent (ressource, vie, recharges…) : on ne redessine la
## fiche que si ce qu'elle montre a changé. Sinon, chaque envoi effaçait les
## boutons sous le doigt, et le + ou « Valider » ne répondait plus.
const VUS := ["lv", "pt", "at", "ta", "tp", "cl", "v", "q", "qm", "or"]
func update_me(d: Dictionary) -> void:
	var avant := me
	me = d
	if str(d.get("at", [])) != str(avant.get("at", [])):
		_draft = _ints(d.get("at", []))
	if str(d.get("ta", [])) != str(avant.get("ta", [])):
		_tdraft = Heros.talents_ints(d.get("ta", []))
	if not is_inside_tree():
		return
	for k in VUS:
		if str(d.get(k)) != str(avant.get(k)):
			_render()
			return


func _line(text: String, size: int = 16, color: Color = UI.PARCH) -> Label:
	var l := UI.label(text, size, color)
	_body.add_child(l)
	return l


var _rendu_tab := ""
func _render() -> void:
	# Même onglet : on garde la fiche là où elle était (sinon chaque + la remontait).
	var sc := _body.get_parent() as ScrollContainer
	var y := sc.scroll_vertical if _rendu_tab == _tab else 0
	_rendu_tab = _tab
	for c in _body.get_children():
		c.queue_free()
	_garde_defilement.call_deferred(sc, y)
	if not public.is_empty():
		_render_public()
	elif _tab == "attributs":
		_render_attrs()
	elif _tab == "talents":
		_render_talents()
	else:
		_render_voie()


func _garde_defilement(sc: ScrollContainer, y: int) -> void:
	await get_tree().process_frame
	if is_instance_valid(sc):
		sc.scroll_vertical = y


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
	# Ses talents : ce qu'il a choisi, palier par palier.
	var vi := -1
	var c := Heros.classe(str(p.cl))
	for i in c.voies.size():
		if c.voies[i].id == v:
			vi = i
	var ta := Heros.talents_ints(p.get("ta", []))
	var tree := Heros.arbre(str(p.cl), vi)
	var picked := []
	for i in tree.size():
		for j in 3:
			if int(ta[i][j]) > 0:
				var tl: Dictionary = tree[i].talents[j]
				picked.append("%s %s %d/%d" % [tl.icone, tl.nom, int(ta[i][j]), int(tl.max)])
	if not picked.is_empty():
		_line("🌳 Talents", 17, UI.LUMEN)
		var l := _line(" · ".join(picked), 15, UI.PARCH)
		l.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART


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


# --- Les talents ------------------------------------------------------------------

func _render_talents() -> void:
	var cl := str(me.get("cl", ""))
	var v := int(me.get("v", -1))
	var lv := int(me.get("lv", 1))
	var server: Array = Heros.talents_ints(me.get("ta", []))
	if _tdraft.size() != 10:
		_tdraft = server.duplicate(true)
	var tree := Heros.arbre(cl, v)
	var total := Heros.points_de_talent(lv)
	var left := total - Heros.points_places(_tdraft)
	var price := int(Heros.data().talents.reprise) * lv
	_line("Points de talent à placer : %d" % left, 18, UI.LUMEN)
	var help := _line("1 point par niveau dès le niveau %d. Un talent prend jusqu'à %d points. Placer un point est gratuit ; les reprendre se fait au temple, contre %d or." % [int(Heros.data().talents.premier), int(Heros.data().talents.max), price], 15, UI.MUTED)
	help.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	var takes_back := false
	for i in 10:
		for j in 3:
			if int(_tdraft[i][j]) < int(server[i][j]):
				takes_back = true
	var actions := HBoxContainer.new()
	actions.add_theme_constant_override("separation", 8)
	var rec := UI.button("⭐ Talents conseillés")
	rec.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	rec.tooltip_text = "Place tes points comme je le conseille pour ta voie."
	rec.pressed.connect(func():
		var want := Heros.talents_conseilles(cl, v, lv)
		# Sans passer au temple : on garde ce qui est déjà placé, et on complète.
		_tdraft = server.duplicate(true)
		var spare := total - Heros.points_places(server)
		for i in 10:
			for j in 3:
				var n := mini(spare, maxi(0, int(want[i][j]) - int(_tdraft[i][j])))
				_tdraft[i][j] += n
				spare -= n
		_render())
	actions.add_child(rec)
	var ok := UI.button("Valider", true)
	ok.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	ok.disabled = _tdraft == server
	ok.pressed.connect(func(): talents.emit(_tdraft.duplicate(true)))
	actions.add_child(ok)
	_body.add_child(actions)
	if takes_back:
		var w := _line("Tu reprends des points : il faut être au temple, et ça coûte %d or." % price, 15, UI.DANGER)
		w.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	var star := ["seul", "groupe", "utile"].find(Heros.etoile(cl, v))
	for i in 10:
		if i >= tree.size():
			var l := _line("🔒 Niveaux %d à 50 : l'arbre de ta voie s'ouvre quand tu la choisis (onglet 🌟 Voie)." % (int(Heros.data().talents.premier) + int(Heros.data().talents.pas)), 15, UI.MUTED)
			l.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
			break
		var tier: Dictionary = tree[i]
		var open := lv >= int(tier.niveau)
		var head := "Niveau %d" % int(tier.niveau)
		if i == 0:
			head += " · commun aux trois voies"
		elif i == 1:
			head += " · ton ultime"
		if not open:
			head = "🔒 " + head
		_line(head, 16, UI.LUMEN if open else UI.MUTED)
		var row := HBoxContainer.new()
		row.add_theme_constant_override("separation", 6)
		for j in 3:
			row.add_child(_talent_card(tier.talents[j], i, j, open, j == star, left))
		_body.add_child(row)


func _talent_card(tl: Dictionary, i: int, j: int, open: bool, star: bool, left: int) -> PanelContainer:
	var n := int(_tdraft[i][j])
	var mx := int(tl.max)
	var card := PanelContainer.new()
	card.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	card.add_theme_stylebox_override("panel", UI.box(UI.STONE_2, UI.LUMEN if n > 0 else UI.LINE, 10, 2 if n > 0 else 1, 8))
	if not open:
		card.modulate = Color(1, 1, 1, 0.55)
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 4)
	card.add_child(col)
	var title := UI.label("%s %s%s" % [tl.icone, tl.nom, " ⭐" if star else ""], 15, UI.LUMEN if n > 0 else UI.PARCH)
	title.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	title.custom_minimum_size = Vector2(130, 0)
	col.add_child(title)
	var desc := UI.label(str(tl.aide), 13, UI.MUTED)
	desc.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	desc.custom_minimum_size = Vector2(130, 0)
	desc.size_flags_vertical = Control.SIZE_EXPAND_FILL
	col.add_child(desc)
	var ranks := HBoxContainer.new()
	ranks.add_theme_constant_override("separation", 4)
	var minus := UI.button("−")
	minus.custom_minimum_size = Vector2(38, 34)
	minus.disabled = n <= 0
	minus.pressed.connect(func():
		_tdraft[i][j] = n - 1
		_render())
	ranks.add_child(minus)
	var dots := UI.label("●".repeat(n) + "○".repeat(maxi(0, mx - n)), 15, UI.LUMEN)
	dots.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	dots.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	ranks.add_child(dots)
	var plus := UI.button("+")
	plus.custom_minimum_size = Vector2(38, 34)
	plus.disabled = not open or n >= mx or left <= 0
	plus.pressed.connect(func():
		_tdraft[i][j] = n + 1
		_render())
	ranks.add_child(plus)
	col.add_child(ranks)
	return card
