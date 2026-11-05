extends PanelContainer
## La fenêtre sociale (touche O) : le groupe, les amis, la guilde, les comptes
## ignorés. Les données viennent de MondeSocial (monde_social.gd), qui la
## redessine quand le serveur envoie du neuf.

const DROITS := [[1, "Inviter"], [2, "Exclure"], [4, "Reprendre au coffre"], [8, "Message du jour"],
	[16, "Calendrier"], [32, "Changer les rangs"]]
const JOURS := ["dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"]

var social: MondeSocial
var _tab := "groupe"
var _body: VBoxContainer
var _tabs: HBoxContainer
var _bl := {"f": 0, "m": 0, "c1": 5, "c2": 2}   # le blason en cours de composition
var _nom := ""
var _edit_rangs := false
var _rangs_draft: Array = []
var _scroll_y := 0


func _ready() -> void:
	theme = UI.make_theme()
	add_theme_stylebox_override("panel", UI.box(Color(UI.STONE, 0.97), UI.MOSS, 14, 1, 12))
	set_anchors_and_offsets_preset(Control.PRESET_CENTER_LEFT, Control.PRESET_MODE_MINSIZE, 14)
	grow_vertical = Control.GROW_DIRECTION_BOTH
	position.x += 220
	custom_minimum_size = Vector2(460, 0)
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 8)
	add_child(col)
	var head := HBoxContainer.new()
	var t := UI.label("👥 Social", 22, UI.LUMEN)
	t.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	head.add_child(t)
	var x := UI.button("✕")
	x.custom_minimum_size = Vector2(44, 40)
	x.focus_mode = Control.FOCUS_NONE
	x.pressed.connect(queue_free)
	head.add_child(x)
	col.add_child(head)
	_tabs = HBoxContainer.new()
	_tabs.add_theme_constant_override("separation", 6)
	for tb in [["groupe", "👥 Groupe"], ["amis", "🤝 Amis"], ["guilde", "🏰 Guilde"], ["ignores", "🙈 Ignorés"]]:
		var b := UI.button(tb[1])
		b.custom_minimum_size = Vector2(0, 42)
		b.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		b.focus_mode = Control.FOCUS_NONE
		var id: String = tb[0]
		b.pressed.connect(func():
			_tab = id
			_edit_rangs = false
			_render())
		b.set_meta("id", id)
		_tabs.add_child(b)
	col.add_child(_tabs)
	var scroll := ScrollContainer.new()
	scroll.horizontal_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
	scroll.custom_minimum_size = Vector2(440, clampf(get_viewport_rect().size.y - 220, 300, 600))
	col.add_child(scroll)
	_body = VBoxContainer.new()
	_body.add_theme_constant_override("separation", 8)
	_body.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	scroll.add_child(_body)
	_render()


func _unhandled_key_input(event: InputEvent) -> void:
	if event.pressed and not event.echo and (event.physical_keycode == KEY_ESCAPE or event.physical_keycode == Touches.key("social")):
		var f := get_viewport().gui_get_focus_owner()
		if f is LineEdit and event.physical_keycode != KEY_ESCAPE:
			return
		get_viewport().set_input_as_handled()
		queue_free()


func _send(d: Dictionary) -> void:
	Net.send(Net.OP_SOCIAL, d)


func _line(t: String, size := 15, c := UI.MUTED) -> Label:
	var l := UI.label(t, size, c)
	_body.add_child(l)
	return l


func _btn(t: String, prim := false) -> Button:
	var b := UI.button(t, prim)
	b.custom_minimum_size = Vector2(0, 38)
	b.focus_mode = Control.FOCUS_NONE
	b.add_theme_font_size_override("font_size", 15)
	return b


## Un champ et un bouton : « Inviter », « Ajouter »…
func _champ(placeholder: String, action: String, on_send: Callable) -> void:
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 6)
	var e := LineEdit.new()
	e.placeholder_text = placeholder
	e.max_length = 24
	e.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	e.custom_minimum_size = Vector2(0, 40)
	row.add_child(e)
	var b := _btn(action, true)
	var go := func():
		if e.text.strip_edges() != "":
			on_send.call(e.text.strip_edges())
			e.text = ""
		e.release_focus()
	b.pressed.connect(go)
	e.text_submitted.connect(func(_t): go.call())
	row.add_child(b)
	_body.add_child(row)


func _render() -> void:
	if _body == null:
		return
	var sc := _body.get_parent() as ScrollContainer
	_scroll_y = sc.scroll_vertical
	for c in _body.get_children():
		c.queue_free()
	for b in _tabs.get_children():
		b.add_theme_color_override("font_color", UI.LUMEN if b.get_meta("id") == _tab else UI.PARCH)
	match _tab:
		"groupe":
			_groupe()
		"amis":
			_amis(false)
		"ignores":
			_amis(true)
		"guilde":
			_guilde()
	(func(): sc.scroll_vertical = _scroll_y).call_deferred()


# --- Le groupe -------------------------------------------------------------------------

func _groupe() -> void:
	var ms := social.membres()
	if ms.size() < 2:
		_line("Tu voyages seul. Invite quelqu'un : clic sur son nom puis « Inviter », ou son nom ici, ou /inviter Nom dans le chat.", 15)
	else:
		_line("Ton groupe (%d / %d). L'expérience se partage, avec 10 %% en plus par compagnon ; les objets se jouent aux dés." % [ms.size(), int(social.groupe.get("max", 5))], 15)
	for m in ms:
		var sid := int(m.s)
		var row := HBoxContainer.new()
		row.add_theme_constant_override("separation", 6)
		var chef := int(social.groupe.get("chef", 0)) == sid
		var l := UI.label("%s%s · %s niv. %d" % ["👑 " if chef else "", m.get("n", "?"),
			Heros.classe(str(m.get("cl", ""))).get("noms", ["?"])[0], int(m.get("lv", 1))], 16, UI.LUMEN if sid == social.world.my_sid else UI.PARCH)
		l.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		row.add_child(l)
		if social.chef() and sid != social.world.my_sid:
			var lead := _btn("👑")
			lead.tooltip_text = "Lui confier la tête du groupe"
			lead.pressed.connect(func(): _send({"a": "chef", "t": sid}))
			row.add_child(lead)
			var kick := _btn("Exclure")
			kick.pressed.connect(func(): _send({"a": "exclure", "t": sid}))
			row.add_child(kick)
		_body.add_child(row)
	if ms.size() < 2 or social.chef():
		_champ("Nom du héros à inviter", "Inviter", func(n): _send({"a": "inviter", "n": n}))
	if ms.size() >= 2:
		var q := _btn("Quitter le groupe")
		q.pressed.connect(func(): _send({"a": "quitter"}))
		_body.add_child(q)
		_line("Un duel de groupe : le chef défie le chef de l'autre groupe (jusqu'à 5 contre 5). Au Cercle des Braves, le chef inscrit tout le groupe.", 13)


# --- Les amis et les comptes ignorés ---------------------------------------------------------

func _amis(ignores: bool) -> void:
	var list: Array = []
	for a in social.amis:
		if bool(a.get("ig", false)) == ignores:
			list.append(a)
	if ignores:
		_line("Tu ne vois plus leurs messages, leurs invitations ni leurs défis. « Signaler » garde leurs derniers messages pour Elowan.", 14)
	else:
		_line("Tes amis, par compte : tu les vois en ligne, avec leur héros et leur région. Une demande doit être acceptée.", 14)
	if list.is_empty():
		_line("Personne pour l'instant." if not ignores else "Tu n'ignores personne. Erydor te remercie.", 15, UI.PARCH)
	for a in list:
		var u := str(a.get("u", ""))
		var box := PanelContainer.new()
		box.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.5), UI.LINE, 10, 1, 8))
		var col := VBoxContainer.new()
		box.add_child(col)
		var on := bool(a.get("on", false))
		var etat := ""
		if bool(a.get("attente", false)):
			etat = " · demande envoyée"
		elif bool(a.get("demande", false)):
			etat = " · te demande en ami"
		col.add_child(UI.label("%s %s%s" % ["🟢" if on else "⚫", a.get("n", "?"), etat], 16, UI.PARCH if on else UI.MUTED))
		var info := "%s niv. %d" % [Heros.classe(str(a.get("cl", ""))).get("noms", ["?"])[0], int(a.get("lv", 1))]
		if on and str(a.get("r", "")) != "":
			info += " · " + str(a.get("r", ""))
		col.add_child(UI.label(info, 13, UI.MUTED))
		var row := HBoxContainer.new()
		row.add_theme_constant_override("separation", 6)
		if ignores:
			var b := _btn("Ne plus ignorer")
			b.pressed.connect(func(): _send({"a": "ignorer_fin", "u": u}))
			row.add_child(b)
		elif bool(a.get("demande", false)):
			var ok := _btn("✓ Accepter", true)
			ok.pressed.connect(func(): _send({"a": "ami_rep", "u": u, "ok": true}))
			row.add_child(ok)
			var no := _btn("Refuser")
			no.pressed.connect(func(): _send({"a": "ami_rep", "u": u, "ok": false}))
			row.add_child(no)
		else:
			if on:
				var w := _btn("💬")
				w.tooltip_text = "Lui chuchoter un message"
				var n := str(a.get("n", ""))
				w.pressed.connect(func():
					social.world._open_chat()
					social.world.chat_edit.text = "/w %s " % n
					social.world.chat_edit.caret_column = social.world.chat_edit.text.length())
				row.add_child(w)
				var inv := _btn("Inviter")
				inv.pressed.connect(func(): _send({"a": "inviter", "n": n}))
				row.add_child(inv)
			var rm := _btn("Retirer")
			rm.pressed.connect(func(): _send({"a": "ami_retrait", "u": u}))
			row.add_child(rm)
		col.add_child(row)
		_body.add_child(box)
	if ignores:
		_champ("Nom du héros à ignorer", "Ignorer", func(n): _send({"a": "ignorer", "n": n}))
		_line("Signaler un héros (ses derniers messages sont gardés pour Elowan) :", 14)
		var row := HBoxContainer.new()
		row.add_theme_constant_override("separation", 6)
		var who := LineEdit.new()
		who.placeholder_text = "Nom"
		who.custom_minimum_size = Vector2(130, 40)
		row.add_child(who)
		var why := LineEdit.new()
		why.placeholder_text = "Ce qui s'est passé"
		why.max_length = 200
		why.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		row.add_child(why)
		var b := _btn("Signaler", true)
		b.pressed.connect(func():
			if who.text.strip_edges() != "":
				_send({"a": "signaler", "n": who.text.strip_edges(), "m": why.text.strip_edges()})
				who.text = ""
				why.text = "")
		row.add_child(b)
		_body.add_child(row)
	else:
		_champ("Nom d'un héros de ton ami", "Ajouter", func(n): _send({"a": "ami", "n": n}))


# --- La guilde ---------------------------------------------------------------------------------

func _droit(d: int) -> bool:
	var g := social.guilde
	var r := int(g.get("rg", 4))
	if r == 0:
		return true
	var rangs: Array = g.get("rangs", [])
	return r < rangs.size() and int(rangs[r].get("dr", 0)) & d != 0


func _guilde() -> void:
	var g := social.guilde
	if g.is_empty():
		_line("Un instant, Fiacre cherche ton nom dans son registre…", 15)
		return
	if bool(g.get("aucune", false)):
		_fonder(g)
		return
	var head := HBoxContainer.new()
	head.add_theme_constant_override("separation", 10)
	head.add_child(Blason.Vue.new(g.get("bl", {}), 64))
	var hc := VBoxContainer.new()
	hc.add_child(UI.label(str(g.get("nom", "")), 22, UI.LUMEN))
	hc.add_child(UI.label("Niveau %d · %d / %d membres" % [int(g.get("lv", 1)), (g.get("membres", []) as Array).size(), int(g.get("max", 50))], 15, UI.PARCH))
	var xp := ProgressBar.new()
	xp.show_percentage = false
	xp.custom_minimum_size = Vector2(220, 8)
	xp.max_value = maxf(1, float(g.get("xpn", 1)))
	xp.value = float(g.get("xp", 0))
	hc.add_child(xp)
	var bonus := "Bonus : +5 % d'expérience quand un membre de ta guilde chasse avec toi." if bool(g.get("bonus", false)) else "Au niveau 2 : +5 % d'expérience entre membres."
	hc.add_child(UI.label(bonus, 13, UI.MUTED))
	head.add_child(hc)
	_body.add_child(head)
	# Le message du jour.
	_line("📜 Message du jour", 16, UI.LUMEN)
	if _droit(8):
		var m := LineEdit.new()
		m.text = str(g.get("motd", ""))
		m.max_length = 200
		m.placeholder_text = "Un mot pour toute la guilde"
		m.text_submitted.connect(func(t): _send({"a": "guilde_motd", "m": t}))
		_body.add_child(m)
	else:
		_line(str(g.get("motd", "")) if str(g.get("motd", "")) != "" else "(aucun)", 15, UI.PARCH)
	# Le calendrier.
	_line("📅 Calendrier", 16, UI.LUMEN)
	var agenda: Array = g.get("agenda", []) if g.get("agenda") is Array else []
	if agenda.is_empty():
		_line("Aucune soirée prévue.", 14)
	for i in agenda.size():
		var e: Dictionary = agenda[i]
		var row := HBoxContainer.new()
		var l := UI.label("%s · %s (%s)" % [_date(int(e.get("t", 0))), e.get("ti", ""), e.get("au", "")], 15, UI.PARCH)
		l.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		row.add_child(l)
		if _droit(16):
			var k := i
			var del := _btn("✕")
			del.pressed.connect(func(): _send({"a": "guilde_agenda_suppr", "i": k}))
			row.add_child(del)
		_body.add_child(row)
	if _droit(16):
		_ajout_soiree()
	# Les membres.
	_line("👥 Membres", 16, UI.LUMEN)
	var rangs: Array = g.get("rangs", [])
	var my := int(g.get("rg", 4))
	for m in g.get("membres", []):
		var row := HBoxContainer.new()
		row.add_theme_constant_override("separation", 6)
		var on := bool(m.get("on", false))
		var rg := int(m.get("rg", 4))
		var rn: String = str(rangs[rg].get("nom", "?")) if rg < rangs.size() else "?"
		var l := UI.label("%s %s · %s · niv. %d%s" % ["🟢" if on else "⚫", m.get("n", "?"), rn, int(m.get("lv", 1)),
			(" · " + str(m.get("r", ""))) if on else ""], 15, UI.PARCH if on else UI.MUTED)
		l.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		row.add_child(l)
		var h := str(m.get("h", ""))
		if h != str(g.get("moi", "")):
			if _droit(32) and rg > my:
				var opt := OptionButton.new()
				opt.focus_mode = Control.FOCUS_NONE
				for i in rangs.size():
					opt.add_item(str(rangs[i].get("nom", "")), i)
					opt.set_item_disabled(i, i <= my and my != 0)
				opt.selected = rg
				opt.item_selected.connect(func(i):
					if i == 0:
						social.world._show_toast("Passer chef : %s deviendra chef, et toi officier." % m.get("n", ""))
					_send({"a": "guilde_rang", "h": h, "r": i}))
				row.add_child(opt)
			if _droit(2) and rg > my:
				var k := _btn("Exclure")
				k.pressed.connect(func(): _send({"a": "guilde_exclure", "h": h}))
				row.add_child(k)
		_body.add_child(row)
	if _droit(1):
		_champ("Nom du héros à inviter", "Inviter", func(n): _send({"a": "guilde_inviter", "n": n}))
	# Les rangs (le chef les renomme et choisit leurs droits).
	if my == 0:
		_line("🎖️ Les rangs", 16, UI.LUMEN)
		if not _edit_rangs:
			var names := []
			for r in rangs:
				names.append(str(r.get("nom", "")))
			_line(" › ".join(names), 15, UI.PARCH)
			var e := _btn("Renommer les rangs, choisir leurs droits")
			e.pressed.connect(func():
				_edit_rangs = true
				_rangs_draft = rangs.duplicate(true)
				_render())
			_body.add_child(e)
		else:
			_rangs_editor()
		_line("🛡️ Le blason (débloque de nouveaux motifs en montant de niveau)", 16, UI.LUMEN)
		if _bl.is_empty() or not _bl.has("_g"):
			_bl = (g.get("bl", {}) as Dictionary).duplicate()
			_bl["_g"] = true
		_blason_editor((g.get("motifs", []) as Array).size())
		var save := _btn("Changer le blason", true)
		save.pressed.connect(func(): _send({"a": "guilde_blason", "bl": _bl_clean()}))
		_body.add_child(save)
	_line("🧰 Le coffre de la guilde est au registre de Fiacre, sur la place (%d onglet%s)." % [int(g.get("onglets", 1)), "s" if int(g.get("onglets", 1)) > 1 else ""], 14)
	var q := _btn("Quitter la guilde")
	q.pressed.connect(func(): _send({"a": "guilde_quitter"}))
	_body.add_child(q)


func _bl_clean() -> Dictionary:
	return {"f": int(_bl.get("f", 0)), "m": int(_bl.get("m", 0)), "c1": int(_bl.get("c1", 0)), "c2": int(_bl.get("c2", 0))}


func _fonder(g: Dictionary) -> void:
	_line("Tu n'es dans aucune guilde. Rejoins-en une sur invitation, ou fonde la tienne au registre de Fiacre, sur la place, pour %s." % Objets.monnaie(int(g.get("px", 1000)), true), 15, UI.PARCH)
	_line("Le nom (3 à 24 lettres) :", 14)
	var e := LineEdit.new()
	e.text = _nom
	e.max_length = 24
	e.placeholder_text = "Les Lanternes de Glennmor"
	e.text_changed.connect(func(t): _nom = t)
	_body.add_child(e)
	_line("Le blason :", 14)
	_blason_editor((g.get("motifs", []) as Array).size())
	var b := _btn("🪶 Fonder la guilde (%s)" % Objets.monnaie(int(g.get("px", 1000))), true)
	b.pressed.connect(func(): _send({"a": "guilde_fonder", "n": _nom.strip_edges(), "bl": _bl_clean()}))
	_body.add_child(b)


## Composer un blason : la forme, la figure et les deux couleurs.
func _blason_editor(motifs: int) -> void:
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 12)
	var preview := Blason.Vue.new(_bl_clean(), 96)
	row.add_child(preview)
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 6)
	col.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	row.add_child(col)
	var forme := OptionButton.new()
	for i in Blason.FORMES.size():
		forme.add_item(Blason.FORMES[i], i)
	forme.selected = int(_bl.get("f", 0))
	forme.focus_mode = Control.FOCUS_NONE
	forme.item_selected.connect(func(i):
		_bl.f = i
		preview.set_blason(_bl_clean()))
	col.add_child(forme)
	var motif := OptionButton.new()
	for i in mini(maxi(motifs, 12), Blason.MOTIFS.size()):
		motif.add_item("%s %s" % [Blason.MOTIFS[i][2], Blason.MOTIFS[i][1]], i)
	motif.selected = mini(int(_bl.get("m", 0)), motif.item_count - 1)
	motif.focus_mode = Control.FOCUS_NONE
	motif.item_selected.connect(func(i):
		_bl.m = i
		preview.set_blason(_bl_clean()))
	col.add_child(motif)
	for k in ["c1", "c2"]:
		col.add_child(UI.label("Couleur du fond" if k == "c1" else "Couleur de la figure", 13, UI.MUTED))
		var grid := GridContainer.new()
		grid.columns = 8
		grid.add_theme_constant_override("h_separation", 3)
		grid.add_theme_constant_override("v_separation", 3)
		for i in Blason.COULEURS.size():
			var sw := Button.new()
			sw.custom_minimum_size = Vector2(24, 24)
			sw.focus_mode = Control.FOCUS_NONE
			var st := UI.box(Blason.COULEURS[i], UI.LUMEN if int(_bl.get(k, 0)) == i else Color(0, 0, 0, 0.6), 6, 2, 0)
			sw.add_theme_stylebox_override("normal", st)
			sw.add_theme_stylebox_override("hover", UI.box(Blason.COULEURS[i], UI.PARCH, 6, 2, 0))
			var key: String = k
			var idx := i
			sw.pressed.connect(func():
				_bl[key] = idx
				_render())
			grid.add_child(sw)
		col.add_child(grid)
	_body.add_child(row)


func _rangs_editor() -> void:
	for i in _rangs_draft.size():
		var r: Dictionary = _rangs_draft[i]
		var box := VBoxContainer.new()
		var name := LineEdit.new()
		name.text = str(r.get("nom", ""))
		name.max_length = 16
		var k := i
		name.text_changed.connect(func(t): _rangs_draft[k]["nom"] = t)
		box.add_child(name)
		if i == 0:
			box.add_child(UI.label("Le chef a tous les droits.", 13, UI.MUTED))
		else:
			var flow := HFlowContainer.new()
			for d in DROITS:
				var cb := CheckBox.new()
				cb.text = d[1]
				cb.add_theme_font_size_override("font_size", 13)
				cb.button_pressed = int(r.get("dr", 0)) & int(d[0]) != 0
				cb.focus_mode = Control.FOCUS_NONE
				var bit: int = d[0]
				cb.toggled.connect(func(on):
					var v := int(_rangs_draft[k].get("dr", 0))
					_rangs_draft[k]["dr"] = (v | bit) if on else (v & ~bit))
				flow.add_child(cb)
			box.add_child(flow)
		_body.add_child(box)
	var row := HBoxContainer.new()
	var ok := _btn("Enregistrer les rangs", true)
	ok.pressed.connect(func():
		_edit_rangs = false
		_send({"a": "guilde_rangs", "rangs": _rangs_draft}))
	row.add_child(ok)
	var no := _btn("Annuler")
	no.pressed.connect(func():
		_edit_rangs = false
		_render())
	row.add_child(no)
	_body.add_child(row)


## Une soirée au calendrier : le jour, l'heure, un titre.
func _ajout_soiree() -> void:
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 6)
	var jour := OptionButton.new()
	var now := Time.get_datetime_dict_from_system()
	var wd := int(now.weekday)
	for i in 15:
		jour.add_item("aujourd'hui" if i == 0 else ("demain" if i == 1 else "%s (+%d j)" % [JOURS[(wd + i) % 7], i]), i)
	jour.focus_mode = Control.FOCUS_NONE
	row.add_child(jour)
	var heure := SpinBox.new()
	heure.min_value = 0
	heure.max_value = 23
	heure.value = 21
	heure.suffix = "h"
	row.add_child(heure)
	var minute := OptionButton.new()
	for m in [0, 15, 30, 45]:
		minute.add_item("%02d" % m, m)
	minute.focus_mode = Control.FOCUS_NONE
	row.add_child(minute)
	_body.add_child(row)
	var row2 := HBoxContainer.new()
	row2.add_theme_constant_override("separation", 6)
	var titre := LineEdit.new()
	titre.placeholder_text = "Soirée au Cercle, chasse au Grand Sanglier…"
	titre.max_length = 60
	titre.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	row2.add_child(titre)
	var b := _btn("Prévoir", true)
	b.pressed.connect(func():
		if titre.text.strip_edges() == "":
			return
		var d := Time.get_datetime_dict_from_system()
		d.hour = int(heure.value)
		d.minute = minute.get_selected_id()
		d.second = 0
		var bias := int(Time.get_time_zone_from_system().get("bias", 0))
		var t := Time.get_unix_time_from_datetime_dict(d) - bias * 60 + jour.get_selected_id() * 86400
		_send({"a": "guilde_agenda", "ev": {"t": int(t) * 1000, "ti": titre.text.strip_edges()}}))
	row2.add_child(b)
	_body.add_child(row2)


func _date(ms: int) -> String:
	var bias := int(Time.get_time_zone_from_system().get("bias", 0))
	var d := Time.get_datetime_dict_from_unix_time(ms / 1000 + bias * 60)
	return "%s %d à %dh%02d" % [JOURS[int(d.weekday)], int(d.day), int(d.hour), int(d.minute)]
