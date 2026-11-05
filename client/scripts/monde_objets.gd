class_name MondeObjets
extends Node
## L'Acte V dans le monde : les objets qui brillent au sol, les dés, les
## filons, plantes et troncs à récolter, les corps à dépecer, les ateliers du
## village, le sac (I), les panneaux des habitants et l'échange.
## world_screen.gd lui passe les messages du serveur et les clics.

const ASSETS := "res://assets/metiers/"
const PICK_REACH := 2.7      # cases (le serveur accepte 3)
const GATHER_REACH := 2.0    # cases (le serveur accepte 2,2)
const ATELIER_REACH := 3.0   # cases (le serveur accepte 3,5)
const HINT_REACH := 4.0
const SKINNABLE := ["sanglier", "loup", "loup_efface", "sanglier_dore", "grand_sanglier"]

var w: Node                  # l'écran du monde
var root: Control            # la couche d'interface
var touch := false
var sac_data := {}
var sac_win: SacWindow
var panneau: Panneau
var echange: EchangeWindow

var _sol := {}               # numéro -> Sol
var _noeuds := {}            # numéro -> [Sprite2D, sorte, position monde, debout]
var _ateliers := []          # [Sprite2D, métier, position monde]
var _sortes := {}            # sorte de nœud -> {nom, m, req}
var _goal := {}              # où l'on marche pour agir : {k, id, at, reach}
var _goal_t := 0.0
var _hint: Button
var _hint_act := {}
var _hint_t := 0.0
var _des: VBoxContainer
var _demande: PanelContainer
var _apparences_btn: Button
var _meta := {}


func setup(world: Node, ui_root: Control, is_touch: bool) -> void:
	w = world
	root = ui_root
	touch = is_touch
	if FileAccess.file_exists(ASSETS + "metiers.json"):
		_meta = JSON.parse_string(FileAccess.get_file_as_string(ASSETS + "metiers.json"))
	_hint = UI.button("")
	_hint.focus_mode = Control.FOCUS_NONE
	_hint.set_anchors_and_offsets_preset(Control.PRESET_CENTER_BOTTOM, Control.PRESET_MODE_MINSIZE, 14)
	_hint.grow_horizontal = Control.GROW_DIRECTION_BOTH
	_hint.grow_vertical = Control.GROW_DIRECTION_BEGIN
	_hint.position.y -= 250 if touch else 150
	_hint.visible = false
	_hint.pressed.connect(func(): _do(_hint_act))
	root.add_child(_hint)
	_des = VBoxContainer.new()
	_des.add_theme_constant_override("separation", 8)
	_des.set_anchors_and_offsets_preset(Control.PRESET_CENTER_TOP, Control.PRESET_MODE_MINSIZE, 200)
	_des.grow_horizontal = Control.GROW_DIRECTION_BOTH
	root.add_child(_des)


## Le bouton « 👗 Apparences », à côté du miroir de Solenn.
func add_apparences_button(next_to: Button) -> void:
	_apparences_btn = UI.button("👗 Apparences")
	_apparences_btn.tooltip_text = "Le miroir de Solenn : donner à une pièce l'allure d'une autre"
	_apparences_btn.focus_mode = Control.FOCUS_NONE
	_apparences_btn.visible = false
	_apparences_btn.pressed.connect(func(): Net.send(Net.OP_OBJET, {"a": "panneau", "b": "miroir"}))
	next_to.get_parent().add_child(_apparences_btn)
	next_to.get_parent().move_child(_apparences_btn, next_to.get_index() + 1)


func _me_pos() -> Vector2:
	if not w.pawns.has(w.my_sid):
		return Vector2(-1e6, -1e6)
	return Iso.to_world(w.pawns[w.my_sid].position)


func _sprite(name: String, at: Vector2) -> Sprite2D:
	var s := Sprite2D.new()
	var path := ASSETS + name + ".png"
	if ResourceLoader.exists(path):
		s.texture = load(path)
	s.centered = false
	s.texture_filter = CanvasItem.TEXTURE_FILTER_NEAREST
	var a: Array = _meta.get(name, {}).get("ancre", [16, 28])
	s.offset = -Vector2(a[0], a[1])
	s.position = Iso.to_screen(at)
	w.entities.add_child(s)
	return s


# --- Les messages du serveur -----------------------------------------------------

func on_state(op: int, d: Dictionary) -> bool:
	match op:
		Net.OP_SAC:
			sac_data = d
			if sac_win != null:
				sac_win.set_data(d)
			if panneau != null:
				panneau.set_sac(d)
		Net.OP_NOEUDS:
			if d.has("n"):
				_place_noeuds(d)
			elif _noeuds.has(int(d.i)):
				var n: Array = _noeuds[int(d.i)]
				n[3] = bool(d.up)
				(n[0] as Sprite2D).modulate = Color(1, 1, 1, 1) if n[3] else Color(0.6, 0.6, 0.65, 0.35)
		Net.OP_SOL:
			_on_sol(d)
		Net.OP_DES:
			_on_des(d)
		Net.OP_RECU:
			var n := int(d.get("n", 1))
			var r := int(d.get("r", 0))
			var line := "🎒 %s%s" % [d.nom, " ×%d" % n if n > 1 else ""]
			if bool(d.get("fab", false)):
				line = "🔨 Tu as fabriqué : %s%s" % [d.nom, " ×%d" % n if n > 1 else ""]
			w._add_chat_line("", line, true)
			if bool(d.get("chance", false)):
				w._show_toast("✨ Coup de chance ! %s sort %s." % [d.nom, Objets.RARETES[clampi(r, 0, 5)].to_lower()])
				if w.braise != null:
					w.braise.say("Braise : « Elle brille ! Tu as la main, dis donc. »")
			elif r >= 2 or bool(d.get("fab", false)):
				w._show_toast("%s %s" % ["🔨" if bool(d.get("fab", false)) else "✨", d.nom])
			if r >= 4 and w.braise != null:
				w.braise.say("Braise fait cinq loopings : « %s !!! »" % Objets.RARETES[clampi(r, 0, 5)].to_upper())
		Net.OP_METIER_UP:
			var m := str(d.m)
			var nom := m
			var ic := ""
			for e in SacWindow.METIERS:
				if e[0] == m:
					nom = e[1]
					ic = e[2]
			match int(d.lv):
				-1:
					w._show_toast("📜 Nouvelle recette apprise !")
				1:
					w._show_toast("%s Tu apprends un métier : %s !" % [ic, nom])
				0:
					w._show_toast("%s Métier oublié : %s." % [ic, nom])
				_:
					w._show_toast("%s %s : niveau %d !" % [ic, nom, int(d.lv)])
					w._add_chat_line("", "%s %s passe au niveau %d." % [ic, nom, int(d.lv)], true)
		Net.OP_PANNEAU:
			_open_panneau(d)
		Net.OP_ECHANGE:
			_on_echange(d)
		_:
			return false
	return true


func _place_noeuds(d: Dictionary) -> void:
	for k in _noeuds:
		(_noeuds[k][0] as Node).queue_free()
	for a in _ateliers:
		(a[0] as Node).queue_free()
	_noeuds.clear()
	_ateliers.clear()
	_sortes = d.get("sortes", {})
	for n in d.n:
		var at := Vector2(n[2], n[3]) / 100.0
		var s := _sprite(str(n[1]), at)
		var up := bool(n[4])
		if not up:
			s.modulate = Color(0.6, 0.6, 0.65, 0.35)
		_noeuds[int(n[0])] = [s, str(n[1]), at, up]
	for a in d.at:
		var at := Vector2(a[2], a[3]) / 100.0
		_ateliers.append([_sprite(str(a[1]), at), str(a[1]), at])


func _on_sol(d: Dictionary) -> void:
	var id := int(d.g)
	match str(d.k):
		"pose", "gagne":
			var s: Sol = _sol.get(id)
			if s == null:
				s = Sol.new()
				s.position = Iso.to_screen(Vector2(d.x, d.y) / 100.0)
				w.entities.add_child(s)
				_sol[id] = s
			s.at = Vector2(d.x, d.y) / 100.0
			s.r = int(d.r)
			s.nom = str(d.nom)
			s.ic = str(d.ic)
			s.proprio = int(d.get("o", 0))
			s.dice = bool(d.get("d", false)) and str(d.k) == "pose"
			s.mine = s.proprio == 0 or s.proprio == w.my_sid
			s.refresh()
		"fin":
			if _sol.has(id):
				_sol[id].queue_free()
				_sol.erase(id)


func _on_des(d: Dictionary) -> void:
	var id := int(d.g)
	for c in _des.get_children():
		if c.get_meta("g", -1) == id:
			c.queue_free()
	if bool(d.get("fin", false)):
		var g := str(d.get("gagnant", ""))
		var txt := "🎲 %s : personne n'en voulait." % d.nom if g == "" else ("🎲 Tu gagnes %s !" % d.nom if bool(d.get("toi", false)) else "🎲 %s gagne %s." % [g, d.nom])
		w._add_chat_line("", txt, true)
		if bool(d.get("toi", false)):
			w._show_toast(txt + " Va le ramasser.")
		return
	var it: Dictionary = d.it
	var card := PanelContainer.new()
	card.set_meta("g", id)
	card.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.95), Objets.couleur(int(it.get("r", 0))), 12, 2, 10))
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 6)
	card.add_child(col)
	col.add_child(UI.label("🎲 Les dés ! Le butin se partage.", 15, UI.MUTED))
	col.add_child(Objets.fiche(it, sac_data.get("eq", {}).get(Objets.emplacement_de(it), {})))
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 6)
	var t := UI.label("", 15, UI.MUTED)
	for v in [[1, "🎯 Besoin"], [2, "💰 Cupidité"], [3, "Passer"]]:
		var b := UI.button(v[1], v[0] == 1)
		b.focus_mode = Control.FOCUS_NONE
		b.custom_minimum_size = Vector2(0, 44)
		if v[0] == 1:
			b.disabled = not bool(d.get("besoin", false))
			b.tooltip_text = "Seulement pour ce que ta classe peut porter. Besoin passe avant cupidité."
		var vote: int = v[0]
		b.pressed.connect(func():
			Net.send(Net.OP_OBJET, {"a": "des", "g": id, "v": vote})
			card.queue_free())
		row.add_child(b)
	row.add_child(t)
	col.add_child(row)
	_des.add_child(card)
	var left := float(d.get("sec", 20))
	var tw := card.create_tween()
	tw.tween_method(func(x: float): t.text = "%d s" % ceili(x), left, 0.0, left)
	tw.tween_callback(card.queue_free)


func _open_panneau(d: Dictionary) -> void:
	if panneau == null:
		panneau = Panneau.new()
		panneau.touch = touch
		panneau.sac = sac_data
		panneau.closed.connect(func():
			panneau = null
			if sac_win != null:
				sac_win.set_ctx("echange" if echange != null else ""))
		root.add_child(panneau)
	panneau.set_data(d)
	panneau.set_sac(sac_data)
	var p := str(d.get("p", ""))
	open_sac(p if p in ["boutique", "coffre"] else "")


func _on_echange(d: Dictionary) -> void:
	match str(d.k):
		"demande":
			_ask_trade(str(d.n))
		"attente":
			w._show_toast("🤝 Demande d'échange envoyée…")
		"ouvre", "maj":
			if _demande != null:
				_demande.queue_free()
				_demande = null
			if echange == null:
				if panneau != null:
					panneau.close()
				echange = EchangeWindow.new()
				root.add_child(echange)
			echange.set_view(d)
			open_sac("echange")
		"fin":
			if _demande != null:
				_demande.queue_free()
				_demande = null
			if echange != null:
				echange.queue_free()
				echange = null
			if sac_win != null:
				sac_win.set_ctx("")
			var why := str(d.get("pq", ""))
			var txt: String = {"fait": "🤝 Échange conclu !", "annule": "Échange annulé.", "loin": "Échange annulé : vous vous êtes éloignés.",
				"refus": "Échange refusé.", "plein": "Échange annulé : un sac est trop plein."}.get(why, "Échange terminé.")
			w._show_toast(txt)
			w._add_chat_line("", txt, true)


func _ask_trade(name: String) -> void:
	if _demande != null:
		_demande.queue_free()
	_demande = PanelContainer.new()
	_demande.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.95), UI.MOSS, 12, 2, 12))
	_demande.set_anchors_and_offsets_preset(Control.PRESET_CENTER_TOP, Control.PRESET_MODE_MINSIZE, 120)
	_demande.grow_horizontal = Control.GROW_DIRECTION_BOTH
	var col := VBoxContainer.new()
	_demande.add_child(col)
	col.add_child(UI.label("🤝 %s te propose un échange." % name, 18))
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 8)
	for v in [[true, "Accepter"], [false, "Refuser"]]:
		var b := UI.button(v[1], v[0])
		b.focus_mode = Control.FOCUS_NONE
		var ok: bool = v[0]
		b.pressed.connect(func():
			Net.send(Net.OP_OBJET, {"a": "echange_reponse", "ok": ok})
			if _demande != null:
				_demande.queue_free()
				_demande = null)
		row.add_child(b)
	col.add_child(row)
	root.add_child(_demande)


# --- Le sac --------------------------------------------------------------------------

func toggle_sac() -> void:
	if sac_win != null:
		sac_win.close()
	else:
		open_sac("")


func open_sac(ctx: String) -> void:
	if sac_win == null:
		sac_win = SacWindow.new()
		sac_win.touch = touch
		var me: Pawn = w.pawns.get(w.my_sid)
		if me != null:
			sac_win.hero = {"race": me.race, "classe": me.classe, "look": me.allure}
		sac_win.data = sac_data
		sac_win.closed.connect(func(): sac_win = null)
		sac_win.offer.connect(func(id):
			if echange != null:
				echange.offer_item(id))
		root.add_child(sac_win)
	sac_win.set_ctx(ctx)


## Les deux raccourcis : utiliser la provision choisie.
func quick(n: int) -> void:
	var rap: Array = sac_data.get("rap", ["", ""])
	var b := str(rap[n]) if rap.size() > n else ""
	if b == "":
		w._show_toast("Raccourci %d vide : choisis une provision dans le sac (%s)." % [n + 1, Touches.label("sac")])
		return
	Net.send(Net.OP_OBJET, {"a": "utiliser_base", "b": b})


## Un choix de dialogue qui ouvre un panneau (Aelis, un maître, le coffre).
func npc_action(npc: String, what: String) -> void:
	match what:
		"boutique":
			Net.send(Net.OP_OBJET, {"a": "panneau", "b": "boutique"})
		"maitre":
			Net.send(Net.OP_OBJET, {"a": "panneau", "b": "maitre", "r": npc})
		"coffre":
			Net.send(Net.OP_OBJET, {"a": "panneau", "b": "coffre"})


# --- Agir dans le monde : ramasser, récolter, dépecer, aller à l'atelier ----------------

## Un clic sur la carte : un objet au sol, un point de récolte, un atelier, un corps ?
func click(at_screen: Vector2) -> bool:
	var act := _under(at_screen)
	if act.is_empty():
		return false
	_goal = act
	_goal_t = -1e9
	w.get_viewport().set_input_as_handled()
	return true


func _under(p: Vector2) -> Dictionary:
	var best := {}
	var best_d := 28.0
	for id in _sol:
		var d := p.distance_to(_sol[id].position + Vector2(0, -6))
		if d < best_d:
			best_d = d
			best = {"k": "sol", "id": id, "at": _sol[id].at, "reach": PICK_REACH}
	for id in _noeuds:
		var n: Array = _noeuds[id]
		var d := p.distance_to(Iso.to_screen(n[2]) + Vector2(0, -12))
		if d < best_d and n[3]:
			best_d = d
			best = {"k": "noeud", "id": id, "at": n[2], "reach": GATHER_REACH}
	for a in _ateliers:
		var d := p.distance_to(Iso.to_screen(a[2]) + Vector2(0, -18))
		if d < best_d + 6:
			best_d = d
			best = {"k": "atelier", "id": a[1], "at": a[2], "reach": ATELIER_REACH}
	for sid in w.pawns:
		var pw: Pawn = w.pawns[sid]
		if pw.dead and SKINNABLE.has(pw.kind):
			var d := p.distance_to(pw.position + Vector2(0, -10))
			if d < best_d:
				best_d = d
				best = {"k": "corps", "id": sid, "at": Iso.to_world(pw.position), "reach": GATHER_REACH + 0.6}
	return best


func _do(act: Dictionary) -> void:
	if act.is_empty():
		return
	if _me_pos().distance_to(act.at) > float(act.reach):
		_goal = act
		_goal_t = -1e9
		return
	Net.send(Net.OP_MOVE_DIR, {"d": [0, 0], "r": false})
	match str(act.k):
		"sol":
			Net.send(Net.OP_OBJET, {"a": "ramasser", "g": int(act.id)})
		"noeud":
			Net.send(Net.OP_OBJET, {"a": "recolter", "n": int(act.id)})
		"corps":
			Net.send(Net.OP_OBJET, {"a": "depecer", "t": int(act.id)})
		"atelier":
			Net.send(Net.OP_OBJET, {"a": "panneau", "b": "atelier", "r": str(act.id)})


## Le plus proche de ce qu'on peut faire autour de soi (touche E, bouton ✋).
func nearest() -> Dictionary:
	var me := _me_pos()
	var best := {}
	var best_d := HINT_REACH
	for id in _sol:
		var s: Sol = _sol[id]
		var d := me.distance_to(s.at)
		if d < best_d and s.mine and not s.dice:
			best_d = d
			best = {"k": "sol", "id": id, "at": s.at, "reach": PICK_REACH, "t": "Ramasser %s %s" % [s.ic, s.nom]}
	for id in _noeuds:
		var n: Array = _noeuds[id]
		var d := me.distance_to(n[2])
		if d < best_d and n[3]:
			best_d = d
			var so: Dictionary = _sortes.get(n[1], {})
			best = {"k": "noeud", "id": id, "at": n[2], "reach": GATHER_REACH, "t": "Récolter : %s" % so.get("nom", n[1])}
	for sid in w.pawns:
		var pw: Pawn = w.pawns[sid]
		if pw.dead and SKINNABLE.has(pw.kind):
			var d := me.distance_to(Iso.to_world(pw.position))
			if d < best_d:
				best_d = d
				best = {"k": "corps", "id": sid, "at": Iso.to_world(pw.position), "reach": GATHER_REACH + 0.6, "t": "Dépecer : %s" % pw.hero_name}
	for a in _ateliers:
		var d := me.distance_to(a[2])
		if d < best_d:
			best_d = d
			var nom: String = a[1]
			for e in SacWindow.METIERS:
				if e[0] == a[1]:
					nom = "%s %s" % [e[2], e[1]]
			best = {"k": "atelier", "id": a[1], "at": a[2], "reach": ATELIER_REACH, "t": "Atelier : %s" % nom}
	return best


func act_nearest() -> void:
	var a := nearest()
	if a.is_empty():
		w._show_toast("Rien à ramasser ni à récolter ici.")
	else:
		_do(a)


func process(delta: float) -> void:
	if not _goal.is_empty():
		if w._last_dir != Vector2.ZERO or not w.pawns.has(w.my_sid) or w.pawns[w.my_sid].dead:
			_goal = {}
		elif _me_pos().distance_to(_goal.at) <= float(_goal.reach):
			var g := _goal
			_goal = {}
			_do(g)
		elif w._clock - _goal_t > 1.0:
			_goal_t = w._clock
			var at: Vector2 = _goal.at
			Net.send(Net.OP_MOVE_TO, {"t": [snappedf(at.x, 0.01), snappedf(at.y, 0.01)], "r": w._running()})
	_hint_t += delta
	if _hint_t > 0.25:
		_hint_t = 0.0
		_hint_act = nearest()
		_hint.visible = not _hint_act.is_empty() and sac_win == null
		if _hint.visible:
			_hint.text = ("✋ " if touch else "%s · " % Touches.label("ramasser")) + str(_hint_act.t)
		if _apparences_btn != null:
			_apparences_btn.visible = w.miroir_btn.visible


## Un objet qui brille au sol, dans la couleur de sa rareté.
class Sol extends Node2D:
	var at := Vector2.ZERO
	var r := 0
	var nom := ""
	var ic := ""
	var proprio := 0
	var mine := true
	var dice := false
	var _t := 0.0
	var _label: Label

	func _ready() -> void:
		_label = Label.new()
		_label.add_theme_font_size_override("font_size", 18)
		_label.position = Vector2(-12, -34)
		add_child(_label)
		refresh()

	func refresh() -> void:
		if _label != null:
			_label.text = ic
			_label.modulate.a = 1.0 if mine else 0.45
		queue_redraw()

	func _process(delta: float) -> void:
		_t += delta
		queue_redraw()

	func _draw() -> void:
		var c := Objets.couleur(r)
		var k := 0.75 + 0.25 * sin(_t * 3.0)
		c.a = (0.55 if mine else 0.2) * k
		draw_set_transform(Vector2.ZERO, 0, Vector2(1, 0.5))
		draw_circle(Vector2.ZERO, 14 + 3 * k, Color(c, c.a * 0.5))
		draw_circle(Vector2.ZERO, 8, c)
		draw_set_transform(Vector2.ZERO, 0, Vector2.ONE)
		if r >= 2 and mine:
			# Un rai de lumière : on le voit de loin.
			draw_line(Vector2(0, 0), Vector2(0, -60 - 10 * k), Color(c, 0.35 * k), 4.0)
