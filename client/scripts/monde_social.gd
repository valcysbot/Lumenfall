class_name MondeSocial
extends Node
## L'Acte VI dans le monde : le groupe (cadres à gauche), les invitations, les
## canaux de discussion et leur filtre, les émotes et les phrases rapides, le
## drapeau de guerre, le Cercle des Braves (inscription, compte à rebours,
## résultat), l'intendance de bravoure, le tableau d'honneur, la carte du val,
## les hauts faits. La fenêtre sociale (amis, guilde) est dans social_window.gd.

const CANAUX := {
	"region": ["Région", Color("efe6d6")], "groupe": ["Groupe", Color("7fc8f8")], "guilde": ["Guilde", Color("86c493")],
	"chuchot": ["Chuchot.", Color("f080b0")], "commerce": ["Commerce", Color("e8c66a")], "faction": ["Faction", Color("f08a3c")],
}
## [identifiant, étiquette courte, nom complet (bulle d'aide)]
const ONGLETS := [["tout", "Tout", "Tous les canaux"], ["region", "📍", "Région"], ["groupe", "👥", "Groupe"],
	["guilde", "🏰", "Guilde"], ["chuchot", "🤫", "Chuchotements"], ["commerce", "💰", "Commerce"], ["faction", "🛡", "Faction"]]
## Les commandes du chat : /g groupe, /gu guilde…
const COMMANDES := {"/r": "region", "/region": "region", "/g": "groupe", "/groupe": "groupe", "/gu": "guilde",
	"/guilde": "guilde", "/c": "commerce", "/commerce": "commerce", "/f": "faction", "/faction": "faction"}
const PHRASES := ["Bonjour tout le monde !", "Merci beaucoup !", "On fait un groupe ?", "J'ai besoin d'aide !",
	"Bien joué !", "J'arrive !", "Attends-moi deux minutes.", "Qui veut un duel ?", "On va au Cercle des Braves ?",
	"À plus tard, et bonne route !"]
const EMOTES := [["salut", "👋", "Saluer"], ["rire", "😂", "Rire"], ["pleurer", "😢", "Pleurer"], ["applaudir", "👏", "Applaudir"],
	["assis", "🪑", "S'asseoir"], ["dormir", "💤", "Dormir"], ["montrer", "👉", "Montrer"], ["incliner", "🙇", "S'incliner"],
	["danser", "💃", "Danser"], ["acclamer", "🎉", "Acclamer"], ["hausser", "🤷", "Hausser les épaules"],
	["colere", "💢", "Colère"], ["coeur", "❤️", "Cœur"]]
## Les gros mots remplacés par ✦✦✦ (chacun peut couper le filtre dans ⚙ Réglages).
const GROS_MOTS := ["merde", "merdique", "putain", "putains", "pute", "putes", "connard", "connards", "connasse", "connasses",
	"salope", "salopes", "salaud", "salauds", "enculé", "enculés", "encule", "enculer", "bâtard", "batard", "bâtards",
	"batards", "fdp", "ntm", "nique", "niquer", "bordel", "chier", "foutre", "con", "cons", "conne", "connes", "couille",
	"couilles", "bite", "bites", "pd", "abruti", "abrutis", "débile", "débiles", "crétin", "crétins", "ta gueule", "tg",
	"fuck", "shit", "bitch"]
const CERCLE_REACH := 10.0
const HONNEUR_REACH := 5.5

var world: Node   # l'écran du monde (world_screen.gd)
var root: Control
var touch := false

var groupe := {}        # le dernier état du groupe (k = "groupe")
var res := {}           # sid → [ressource, max]
var amis: Array = []
var guilde := {}
var gloire := {}
var hauts: Array = []    # la liste des hauts faits (demandée une fois)
var saison := {}
var bravoure: Array = []
var cercle := {}        # anneau, honneur, salles (l'arrivée)
var file := 0           # le format attendu au Cercle (0 : aucun)
var file_n := 0
var bout := {}          # le combat du Cercle en cours
var canal := "region"
var filtre := true
var drapeau := false
var win: Control         # la fenêtre sociale
var ennemis := {}        # sid → true : les adversaires (duel de groupe, Cercle)

var _party: VBoxContainer
var _prompts: VBoxContainer
var _flag_btn: Button
var _menu: PanelContainer      # émotes et phrases
var _contexte: HBoxContainer   # « Cercle des Braves », « Tableau d'honneur » quand on est près
var _cercle_btn: Button
var _honneur_btn: Button
var _cercle_win: PanelContainer
var _honneur_win: PanelContainer
var _carte_win: Control
var _big: Label
var _lieux: Node2D
var _t := 0.0
var _re: RegEx


func setup(w: Node, r: Control, t: bool) -> void:
	world = w
	root = r
	touch = t
	var cfg := ConfigFile.new()
	cfg.load(WorldSky.SETTINGS)
	filtre = bool(cfg.get_value("social", "filtre", true))
	_re = RegEx.new()
	var mots := []
	for m in GROS_MOTS:
		mots.append(m.replace(" ", "\\s+"))
	_re.compile("(*UCP)(?i)\\b(" + "|".join(mots) + ")\\b")

	_party = VBoxContainer.new()
	_party.add_theme_constant_override("separation", 4)
	_party.position = Vector2(14, 64)
	_party.mouse_filter = Control.MOUSE_FILTER_PASS
	root.add_child(_party)

	_prompts = VBoxContainer.new()
	_prompts.add_theme_constant_override("separation", 8)
	_prompts.set_anchors_and_offsets_preset(Control.PRESET_CENTER_TOP, Control.PRESET_MODE_MINSIZE, 300)
	_prompts.grow_horizontal = Control.GROW_DIRECTION_BOTH
	root.add_child(_prompts)

	_contexte = HBoxContainer.new()
	_contexte.add_theme_constant_override("separation", 8)
	_contexte.set_anchors_and_offsets_preset(Control.PRESET_CENTER_BOTTOM, Control.PRESET_MODE_MINSIZE, 70)
	_contexte.grow_horizontal = Control.GROW_DIRECTION_BOTH
	_contexte.grow_vertical = Control.GROW_DIRECTION_BEGIN
	_cercle_btn = UI.button("🏟️ Le Cercle des Braves", true)
	_cercle_btn.focus_mode = Control.FOCUS_NONE
	_cercle_btn.pressed.connect(open_cercle)
	_contexte.add_child(_cercle_btn)
	_honneur_btn = UI.button("🏆 Le tableau d'honneur", true)
	_honneur_btn.focus_mode = Control.FOCUS_NONE
	_honneur_btn.pressed.connect(func(): Net.send(Net.OP_SOCIAL, {"a": "honneur"}))
	_contexte.add_child(_honneur_btn)
	_cercle_btn.visible = false
	_honneur_btn.visible = false
	if touch:
		_contexte.position.y -= 150
	root.add_child(_contexte)

	_big = UI.label("", 64, UI.LUMEN)
	_big.autowrap_mode = TextServer.AUTOWRAP_OFF
	_big.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_big.set_anchors_and_offsets_preset(Control.PRESET_CENTER, Control.PRESET_MODE_MINSIZE)
	_big.grow_horizontal = Control.GROW_DIRECTION_BOTH
	_big.grow_vertical = Control.GROW_DIRECTION_BOTH
	_big.position.y -= 120
	_big.add_theme_color_override("font_outline_color", UI.NIGHT)
	_big.add_theme_constant_override("outline_size", 10)
	_big.mouse_filter = Control.MOUSE_FILTER_IGNORE
	root.add_child(_big)


## Les boutons de la barre du haut : 👥 Social, 😀 Émotes, 🚩 Drapeau.
func add_buttons(bar: HBoxContainer, before: int) -> void:
	var b := UI.button("👥 Social")
	b.tooltip_text = "Ton groupe, tes amis, ta guilde (touche %s)" % Touches.label("social")
	b.pressed.connect(func(): toggle_window(""))
	var e := UI.button("😀")
	e.tooltip_text = "Émotes et phrases rapides (touche %s)" % Touches.label("emotes")
	e.pressed.connect(toggle_menu)
	_flag_btn = UI.button("🏳️")
	_flag_btn.tooltip_text = "Lever ou baisser ton drapeau de guerre"
	_flag_btn.pressed.connect(func(): Net.send(Net.OP_SOCIAL, {"a": "drapeau"}))
	var m := UI.button("🗺️")
	m.tooltip_text = "La carte du val, avec ton groupe, ta guilde et tes amis (touche %s)" % Touches.label("carte")
	m.pressed.connect(toggle_carte)
	for x in [b, e, _flag_btn, m]:
		x.focus_mode = Control.FOCUS_NONE
		bar.add_child(x)
		bar.move_child(x, before)
		before += 1


# --- Les messages du serveur -------------------------------------------------------

func on_state(op: int, d: Dictionary) -> bool:
	match op:
		Net.OP_SNAPSHOT:
			var c = d.get("cercle")
			cercle = c if c is Dictionary else {}
			_poser_lieux.call_deferred()
			return false
		Net.OP_SOCIAL_OUT:
			_on_social(d)
			return true
	return false


func _on_social(d: Dictionary) -> void:
	match str(d.get("k", "")):
		"groupe":
			groupe = d
			_maj_allies()
			_render_party()
			_refresh_win()
		"groupe_res":
			for m in d.get("m", []):
				res[int(m[0])] = [int(m[1]), int(m[2])]
		"invite_groupe":
			_prompt("👥 %s t'invite dans son groupe." % d.get("n", "?"), float(d.get("sec", 60)),
				func(ok): Net.send(Net.OP_SOCIAL, {"a": "groupe_rep", "ok": ok}))
		"amis":
			amis = d.get("l") if d.get("l") is Array else []
			_refresh_win()
		"demande_ami":
			var u := str(d.get("u", ""))
			_prompt("🤝 %s voudrait être ton ami." % d.get("n", "?"), 120.0,
				func(ok): Net.send(Net.OP_SOCIAL, {"a": "ami_rep", "u": u, "ok": ok}))
		"connexion":
			if bool(d.get("g", false)):
				world._add_chat_line("", "🏰 %s se connecte." % d.get("n", "?"), true, "guilde")
			else:
				world._add_chat_line("", "🟢 Ton ami %s vient d'arriver dans Erydor." % d.get("n", "?"), true)
				world._show_toast("🟢 %s est en ligne" % d.get("n", "?"))
		"motd":
			world._add_chat_line("", "📜 %s : %s" % [d.get("g", ""), d.get("m", "")], true, "guilde")
		"guilde":
			guilde = d
			_refresh_win()
		"invite_guilde":
			_prompt("🏰 %s t'invite dans la guilde « %s »." % [d.get("n", "?"), d.get("g", "?")], float(d.get("sec", 60)),
				func(ok): Net.send(Net.OP_SOCIAL, {"a": "guilde_rep", "ok": ok}), d.get("bl", {}))
		"gloire":
			gloire = d
			_refresh_win()
			if _cercle_win != null:
				_render_cercle()
			if world.sheet != null and is_instance_valid(world.sheet) and world.sheet.public.is_empty():
				world.sheet.set_gloire(gloire, hauts)
		"hauts_faits":
			hauts = d.get("liste", [])
			if world.sheet != null and is_instance_valid(world.sheet) and world.sheet.public.is_empty():
				world.sheet.set_gloire(gloire, hauts)
		"haut_fait":
			var t := "🏅 Haut fait : %s %s (+%d)" % [d.get("ic", ""), d.get("nom", ""), int(d.get("pts", 0))]
			world._add_chat_line("", t, true)
			world._show_toast(t)
			if d.has("titre"):
				world._add_chat_line("", "🎖️ Nouveau titre : « %s ». Choisis-le dans ta fiche (C), onglet Gloire." % d.titre, true)
			if world.braise != null:
				world.braise.say("Braise : « %s ! Je le note dans mon carnet. »" % d.get("nom", ""))
		"saison":
			saison = d
		"honneur":
			_open_honneur(d)
		"bravoure":
			bravoure = d.get("liste", [])
			if _cercle_win != null:
				_render_cercle()
		"file":
			file = int(d.get("f", 0))
			file_n = int(d.get("n", 0))
			if file > 0:
				world._show_toast("🏟️ Inscrit au Cercle (%d c. %d). On cherche tes adversaires…" % [file, file])
			if _cercle_win != null:
				_render_cercle()
		"cercle":
			_on_cercle(d)


## Un choix à faire (invitation) : deux boutons, et un délai.
func _prompt(text: String, sec: float, answer: Callable, bl: Dictionary = {}) -> void:
	var card := PanelContainer.new()
	card.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.94), UI.MOSS, 14, 2, 14))
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 8)
	card.add_child(col)
	var head := HBoxContainer.new()
	head.add_theme_constant_override("separation", 8)
	if not bl.is_empty():
		head.add_child(Blason.Vue.new(bl, 36))
	var l := UI.label(text, 18, UI.PARCH)
	l.custom_minimum_size = Vector2(340, 0)
	head.add_child(l)
	col.add_child(head)
	var row := HBoxContainer.new()
	row.alignment = BoxContainer.ALIGNMENT_CENTER
	row.add_theme_constant_override("separation", 10)
	for b in [["✓ Accepter", true], ["Non merci", false]]:
		var btn := UI.button(b[0], b[1])
		btn.focus_mode = Control.FOCUS_NONE
		btn.pressed.connect(func():
			answer.call(b[1])
			card.queue_free())
		row.add_child(btn)
	col.add_child(row)
	_prompts.add_child(card)
	get_tree().create_timer(sec).timeout.connect(func():
		if is_instance_valid(card):
			card.queue_free())


# --- Le groupe ------------------------------------------------------------------------

func membres() -> Array:
	return groupe.get("m", []) if groupe.get("m") is Array else []


func in_group(sid: int) -> bool:
	for m in membres():
		if int(m.s) == sid:
			return true
	return false


func chef() -> bool:
	return int(groupe.get("chef", 0)) == world.my_sid


func _maj_allies() -> void:
	for sid in world.pawns:
		var p: Pawn = world.pawns[sid]
		var was := p.allie
		p.allie = in_group(sid) and sid != world.my_sid
		if was != p.allie:
			p._refresh_name()


func _render_party() -> void:
	for c in _party.get_children():
		c.queue_free()
	var ms := membres()
	if ms.size() < 2:
		return
	for m in ms:
		var sid := int(m.s)
		var card := PanelContainer.new()
		card.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.72), UI.MOSS if sid == world.my_sid else Color(UI.LINE, 0.9), 10, 1, 6))
		card.custom_minimum_size = Vector2(200, 0)
		card.mouse_filter = Control.MOUSE_FILTER_STOP
		card.gui_input.connect(func(e):
			if e is InputEventMouseButton and e.pressed and e.button_index == MOUSE_BUTTON_LEFT and world.pawns.has(sid):
				world._set_target(sid))
		var col := VBoxContainer.new()
		col.add_theme_constant_override("separation", 2)
		col.mouse_filter = Control.MOUSE_FILTER_IGNORE
		card.add_child(col)
		var name := ("👑 " if int(groupe.get("chef", 0)) == sid else "") + "%s · %d" % [m.get("n", "?"), int(m.get("lv", 1))]
		var l := UI.label(name, 14, UI.LUMEN if sid == world.my_sid else UI.PARCH)
		l.autowrap_mode = TextServer.AUTOWRAP_OFF
		l.mouse_filter = Control.MOUSE_FILTER_IGNORE
		col.add_child(l)
		for k in ["hp", "res"]:
			var bar := ProgressBar.new()
			bar.name = k
			bar.show_percentage = false
			bar.custom_minimum_size = Vector2(186, 7 if k == "hp" else 5)
			bar.mouse_filter = Control.MOUSE_FILTER_IGNORE
			var fill := StyleBoxFlat.new()
			fill.bg_color = UI.MOSS if k == "hp" else Color("7fa8e8")
			var bg := StyleBoxFlat.new()
			bg.bg_color = Color(0, 0, 0, 0.6)
			bar.add_theme_stylebox_override("fill", fill)
			bar.add_theme_stylebox_override("background", bg)
			col.add_child(bar)
		card.set_meta("sid", sid)
		card.set_meta("m", m)
		_party.add_child(card)
	_update_party()


func _update_party() -> void:
	for card in _party.get_children():
		if not card.has_meta("sid"):
			continue
		var sid: int = card.get_meta("sid")
		var m: Dictionary = card.get_meta("m")
		var hp := float(m.get("hp", 1))
		var hpm := float(m.get("hpm", 1))
		var near: bool = world.pawns.has(sid)
		if near:
			hp = world.pawns[sid].hp
			hpm = world.pawns[sid].hp_max
		var bars := card.get_child(0)
		var hb: ProgressBar = bars.get_node("hp")
		hb.max_value = maxf(1.0, hpm)
		hb.value = hp
		var r: Array = res.get(sid, [0, int(m.get("rm", 100))])
		var rb: ProgressBar = bars.get_node("res")
		rb.max_value = maxf(1.0, float(r[1]))
		rb.value = float(r[0])
		card.modulate.a = 1.0 if near and not world.pawns[sid].dead else 0.55


# --- Le chat --------------------------------------------------------------------------

## Ce qu'on tape : une commande (/g, /w Nom…), ou un message sur le canal choisi.
func send_chat(text: String) -> void:
	var c := canal
	var a := ""
	if text.begins_with("/"):
		var parts := text.split(" ", false, 2)
		var cmd := parts[0].to_lower()
		var rest := text.substr(parts[0].length()).strip_edges()
		if COMMANDES.has(cmd):
			c = COMMANDES[cmd]
			text = rest
		elif cmd in ["/w", "/m", "/chuchoter", "/mp"]:
			if parts.size() < 3:
				world._show_toast("Pour chuchoter : /w Nom ton message")
				return
			c = "chuchot"
			a = parts[1]
			text = parts[2]
		elif cmd in ["/inviter", "/invite"]:
			Net.send(Net.OP_SOCIAL, {"a": "inviter", "n": rest})
			return
		elif cmd == "/ami":
			Net.send(Net.OP_SOCIAL, {"a": "ami", "n": rest})
			return
		elif cmd == "/ignorer":
			Net.send(Net.OP_SOCIAL, {"a": "ignorer", "n": rest})
			return
		elif cmd in ["/danse", "/danser"]:
			Net.send(Net.OP_EMOTE, {"e": "danser"})
			return
		elif cmd in ["/e", "/emote"]:
			Net.send(Net.OP_EMOTE, {"e": rest})
			return
		elif cmd == "/quitter":
			Net.send(Net.OP_SOCIAL, {"a": "quitter"})
			return
		else:
			world._show_toast("Braise ne connaît pas « %s ». Essaie /g, /gu, /w Nom, /c, /f, /inviter Nom." % cmd)
			return
	if text == "":
		return
	if c == "chuchot" and a == "":
		a = _dernier_chuchot
		if a == "":
			world._show_toast("Pour chuchoter : /w Nom ton message")
			return
	Net.send(Net.OP_CHAT, {"m": text, "c": c, "a": a})


var _dernier_chuchot := ""


## Un message reçu : [ligne en BBCode, canal].
func format_chat(d: Dictionary) -> Array:
	var c := str(d.get("c", "region"))
	var info: Array = CANAUX.get(c, CANAUX.region)
	var who := str(d.get("n", "?")).replace("[", "[lb]")
	var text := filter_text(str(d.get("m", ""))).replace("[", "[lb]")
	var col: Color = info[1]
	var prefix := ""
	match c:
		"region":
			prefix = ""
		"chuchot":
			var to := str(d.get("a", ""))
			if int(d.get("s", 0)) == world.my_sid:
				prefix = "[À %s] " % to.replace("[", "[lb]")
			else:
				prefix = "[De] "
				_dernier_chuchot = str(d.get("n", ""))
		_:
			prefix = "[%s] " % info[0]
	var g := ""
	if d.has("g") and c != "guilde":
		g = " ‹%s›" % str(d.g).replace("[", "[lb]")
	var name_col: Color = Heros.FACTIONS.get(str(d.get("f", "")), {}).get("couleur", UI.EMBER)
	var line := "[color=#%s]%s[/color][color=#%s]%s[/color][color=#%s]%s[/color] : [color=#%s]%s[/color]" % [
		col.to_html(false), prefix, name_col.to_html(false), who, UI.MUTED.to_html(false), g, col.to_html(false), text]
	return [line, c]


## Les bulles : seulement pour la région, et seulement près de toi.
func bubble_ok(d: Dictionary) -> bool:
	if str(d.get("c", "region")) != "region":
		return false
	var sid := int(d.get("s", 0))
	if not world.pawns.has(sid) or not world.pawns.has(world.my_sid):
		return false
	return world.pawns[sid].position.distance_to(world.pawns[world.my_sid].position) < 900.0


func filter_text(t: String) -> String:
	if not filtre or _re == null or not _re.is_valid():
		return t
	return _re.sub(t, "✦✦✦", true)


func set_filtre(on: bool) -> void:
	filtre = on
	var cfg := ConfigFile.new()
	cfg.load(WorldSky.SETTINGS)
	cfg.set_value("social", "filtre", on)
	cfg.save(WorldSky.SETTINGS)


# --- Les émotes et les phrases rapides -------------------------------------------------

func toggle_menu() -> void:
	if _menu != null:
		_menu.queue_free()
		_menu = null
		return
	_menu = PanelContainer.new()
	_menu.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.95), UI.LUMEN, 14, 1, 12))
	_menu.set_anchors_and_offsets_preset(Control.PRESET_CENTER_BOTTOM, Control.PRESET_MODE_MINSIZE, 80)
	_menu.grow_horizontal = Control.GROW_DIRECTION_BOTH
	_menu.grow_vertical = Control.GROW_DIRECTION_BEGIN
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 8)
	_menu.add_child(col)
	col.add_child(UI.label("😀 Émotes", 17, UI.LUMEN))
	var grid := GridContainer.new()
	grid.columns = 7 if not touch else 5
	grid.add_theme_constant_override("h_separation", 6)
	grid.add_theme_constant_override("v_separation", 6)
	for e in EMOTES:
		var b := UI.button(e[1])
		b.custom_minimum_size = Vector2(58, 50)
		b.tooltip_text = e[2] if e[0] != "danser" else "Danser : %s" % Pawn.DANSES.get(_ma_race(), Pawn.DANSES.humain)[0]
		b.focus_mode = Control.FOCUS_NONE
		var id: String = e[0]
		b.pressed.connect(func():
			Net.send(Net.OP_EMOTE, {"e": id})
			toggle_menu())
		grid.add_child(b)
	col.add_child(grid)
	col.add_child(UI.label("💬 Phrases rapides (canal : %s)" % CANAUX.get(canal, CANAUX.region)[0], 17, UI.LUMEN))
	var pg := GridContainer.new()
	pg.columns = 2
	pg.add_theme_constant_override("h_separation", 6)
	pg.add_theme_constant_override("v_separation", 6)
	for p in PHRASES:
		var b := UI.button(p)
		b.custom_minimum_size = Vector2(250, 40)
		b.add_theme_font_size_override("font_size", 15)
		b.focus_mode = Control.FOCUS_NONE
		b.pressed.connect(func():
			send_chat(p)
			toggle_menu())
		pg.add_child(b)
	col.add_child(pg)
	root.add_child(_menu)


func _ma_race() -> String:
	if world.pawns.has(world.my_sid):
		return world.pawns[world.my_sid].race
	return "humain"


# --- Le drapeau ---------------------------------------------------------------------------

func set_my_flag(on: bool) -> void:
	drapeau = on
	if _flag_btn != null:
		_flag_btn.text = "🚩" if on else "🏳️"
		_flag_btn.tooltip_text = "Drapeau de guerre levé : hors du village, les héros des factions ennemies qui l'ont levé peuvent t'attaquer. Clique pour le baisser." if on \
			else "Drapeau baissé : tu es en paix. Clique pour le lever."
		if on:
			_flag_btn.add_theme_color_override("font_color", UI.DANGER)
		else:
			_flag_btn.remove_theme_color_override("font_color")


## La guerre : ce héros est-il un ennemi qu'on peut viser (le serveur décide) ?
func at_war(sid: int) -> bool:
	if ennemis.has(sid):
		return true
	if not drapeau or not world.pawns.has(sid) or not world.pawns.has(world.my_sid):
		return false
	var p: Pawn = world.pawns[sid]
	var me: Pawn = world.pawns[world.my_sid]
	if p.race == "" or not p.drapeau or p.allie:
		return false
	return str(Heros.peuple(p.race).get("faction", "")) != str(Heros.peuple(me.race).get("faction", ""))


# --- Le Cercle des Braves --------------------------------------------------------------------

func _on_cercle(d: Dictionary) -> void:
	match str(d.get("e", "")):
		"debut":
			bout = d
			file = 0
			ennemis.clear()
			var moi := int(d.get("moi", 0))
			var camps: Array = d.get("camps", [[], []])
			for x in camps[1 - moi]:
				ennemis[int(x)] = true
				if world.pawns.has(int(x)):
					world.pawns[int(x)].foe = true
			if _cercle_win != null:
				_cercle_win.queue_free()
				_cercle_win = null
			world._add_chat_line("", "🏟️ Le Cercle des Braves : %d c. %d, dans %s !" % [int(d.f), int(d.f), d.get("salle", "l'arène")], true)
			_countdown(int(d.get("sec", 5)))
			if world.braise != null:
				world.braise.say("Braise : « Le sable du Cercle ! Je reste sur la barrière, promis. »")
		"compte":
			pass
		"sort":
			var s := int(d.get("s", 0))
			var n: String = str(d.get("noms", {}).get(str(s), "Un combattant"))
			world._add_chat_line("", "🏟️ %s est hors du combat." % n, true)
			if s == world.my_sid:
				world._show_toast("Tu es hors du combat. Encourage les tiens !")
		"fin":
			for sid in ennemis:
				if world.pawns.has(sid):
					world.pawns[sid].foe = false
			ennemis.clear()
			bout = {}
			var r := str(d.get("res", "nul"))
			var delta := int(d.get("delta", 0))
			var t: String = {"victoire": "🏆 Victoire au Cercle !", "defaite": "Défaite au Cercle…", "nul": "Match nul au Cercle."}.get(r, "")
			t += "  Cote %d (%s%d) · +%d Insignes" % [int(d.get("cote", 0)), "+" if delta >= 0 else "", delta, int(d.get("ins", 0))]
			world._add_chat_line("", "🏟️ " + t, true)
			_big.text = {"victoire": "🏆 Victoire !", "defaite": "Défaite", "nul": "Match nul"}.get(r, "")
			_big.modulate.a = 1.0
			var tw := _big.create_tween()
			tw.tween_interval(2.0)
			tw.tween_property(_big, "modulate:a", 0.0, 0.8)
			world._show_toast(t)
			if world.braise != null:
				world.braise.say("Braise fait un looping : « Quel combat ! »" if r == "victoire" else "Braise : « On les aura la prochaine fois. »")


func _countdown(n: int) -> void:
	for i in range(n, -1, -1):
		_big.text = "⚔️ Combattez !" if i == 0 else str(i)
		_big.modulate.a = 1.0
		_big.scale = Vector2.ONE
		await get_tree().create_timer(1.0).timeout
	var tw := _big.create_tween()
	tw.tween_property(_big, "modulate:a", 0.0, 0.5)


func open_cercle() -> void:
	if _cercle_win != null:
		_cercle_win.queue_free()
		_cercle_win = null
		return
	_cercle_win = PanelContainer.new()
	_cercle_win.theme = UI.make_theme()
	_cercle_win.add_theme_stylebox_override("panel", UI.box(Color(UI.STONE, 0.97), UI.DANGER, 14, 1, 14))
	_cercle_win.set_anchors_and_offsets_preset(Control.PRESET_CENTER_RIGHT, Control.PRESET_MODE_MINSIZE, 14)
	_cercle_win.grow_horizontal = Control.GROW_DIRECTION_BEGIN
	_cercle_win.grow_vertical = Control.GROW_DIRECTION_BOTH
	_cercle_win.custom_minimum_size = Vector2(430, 0)
	root.add_child(_cercle_win)
	Net.send(Net.OP_SOCIAL, {"a": "bravoure"})
	_render_cercle()


func _render_cercle() -> void:
	if _cercle_win == null:
		return
	for c in _cercle_win.get_children():
		c.queue_free()
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 8)
	var scroll := ScrollContainer.new()
	scroll.horizontal_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
	scroll.custom_minimum_size = Vector2(410, clampf(root.get_viewport_rect().size.y - 200, 300, 600))
	scroll.add_child(col)
	col.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_cercle_win.add_child(scroll)
	var head := HBoxContainer.new()
	var t := UI.label("🏟️ Le Cercle des Braves", 21, UI.LUMEN)
	t.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	head.add_child(t)
	var x := UI.button("✕")
	x.custom_minimum_size = Vector2(44, 40)
	x.focus_mode = Control.FOCUS_NONE
	x.pressed.connect(open_cercle)
	head.add_child(x)
	col.add_child(head)
	col.add_child(UI.label("Des combats entre héros, à 1, 2 ou 3 par camp, dans les salles du Cercle. On n'y perd rien : ni usure, ni Faiblesse, même en hardcore. Le chef d'un groupe inscrit tout le groupe.", 14, UI.MUTED))
	var cotes: Dictionary = gloire.get("cote", {}) if gloire.get("cote") is Dictionary else {}
	var best: Dictionary = gloire.get("best", {}) if gloire.get("best") is Dictionary else {}
	var fin := int(saison.get("fin", 0))
	if fin > 0:
		var jours := maxi(0, int((fin / 1000 - Time.get_unix_time_from_system()) / 86400.0))
		col.add_child(UI.label("Saison %d : encore %d jour%s. Les deux meilleurs de chaque format gagnent un titre." % [int(saison.get("n", 1)), jours, "s" if jours > 1 else ""], 15, UI.PARCH))
	for f in [1, 2, 3]:
		var row := HBoxContainer.new()
		row.add_theme_constant_override("separation", 8)
		var info := UI.label("%d c. %d · cote %d (record %d)" % [f, f, int(cotes.get(str(f), 1500)), int(best.get(str(f), 0))], 16, UI.PARCH)
		info.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		row.add_child(info)
		var b := UI.button("En file (%d)" % file_n if file == f else "S'inscrire", file != f)
		b.custom_minimum_size = Vector2(150, 42)
		b.focus_mode = Control.FOCUS_NONE
		b.disabled = file == f
		b.pressed.connect(func(): Net.send(Net.OP_SOCIAL, {"a": "cercle", "f": f}))
		row.add_child(b)
		col.add_child(row)
	if file > 0:
		var q := UI.button("Quitter la file")
		q.focus_mode = Control.FOCUS_NONE
		q.pressed.connect(func(): Net.send(Net.OP_SOCIAL, {"a": "cercle_quitter"}))
		col.add_child(q)
	col.add_child(UI.label("Victoires %d · défaites %d · duels gagnés %d · héros vaincus à la guerre %d" % [
		int(gloire.get("cv", 0)), int(gloire.get("cd", 0)), int(gloire.get("dv", 0)), int(gloire.get("gv", 0))], 14, UI.MUTED))
	col.add_child(HSeparator.new())
	col.add_child(UI.label("🎖️ L'intendance de bravoure · tu as %d Insignes" % int(gloire.get("ins", 0)), 18, UI.LUMEN))
	if bravoure.is_empty():
		col.add_child(UI.label("Approche-toi du Cercle pour voir ce que propose l'intendance.", 14, UI.MUTED))
	for art in bravoure:
		var row := HBoxContainer.new()
		row.add_theme_constant_override("separation", 8)
		var txt := VBoxContainer.new()
		txt.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		txt.add_child(UI.label("%s %s" % [art.get("ic", ""), art.get("nom", "")], 15, UI.PARCH))
		txt.add_child(UI.label(str(art.get("ds", "")), 13, UI.MUTED))
		row.add_child(txt)
		var id := str(art.get("id", ""))
		var b := UI.button("%d 🎖️" % int(art.get("px", 0)))
		b.custom_minimum_size = Vector2(100, 40)
		b.focus_mode = Control.FOCUS_NONE
		b.disabled = int(gloire.get("ins", 0)) < int(art.get("px", 0))
		b.pressed.connect(func(): Net.send(Net.OP_SOCIAL, {"a": "bravoure_achat", "n": id}))
		row.add_child(b)
		col.add_child(row)


# --- Le tableau d'honneur -------------------------------------------------------------------

func _open_honneur(d: Dictionary) -> void:
	if _honneur_win != null:
		_honneur_win.queue_free()
	_honneur_win = PanelContainer.new()
	_honneur_win.theme = UI.make_theme()
	_honneur_win.add_theme_stylebox_override("panel", UI.box(Color(UI.STONE, 0.97), UI.LUMEN, 14, 1, 14))
	_honneur_win.set_anchors_and_offsets_preset(Control.PRESET_CENTER, Control.PRESET_MODE_MINSIZE)
	_honneur_win.grow_horizontal = Control.GROW_DIRECTION_BOTH
	_honneur_win.grow_vertical = Control.GROW_DIRECTION_BOTH
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 6)
	_honneur_win.add_child(col)
	var head := HBoxContainer.new()
	var t := UI.label("🏆 Le tableau d'honneur de Bourg-Lanterne", 22, UI.LUMEN)
	t.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	head.add_child(t)
	var x := UI.button("✕")
	x.custom_minimum_size = Vector2(44, 40)
	x.focus_mode = Control.FOCUS_NONE
	x.pressed.connect(func():
		_honneur_win.queue_free()
		_honneur_win = null)
	head.add_child(x)
	col.add_child(head)
	var cols := HBoxContainer.new()
	cols.add_theme_constant_override("separation", 22)
	col.add_child(cols)
	var lv := VBoxContainer.new()
	lv.add_child(UI.label("Les plus aguerris", 17, UI.EMBER))
	var i := 1
	for h in (d.get("niveaux") if d.get("niveaux") is Array else []):
		var l := UI.label("%d. %s · niv. %d" % [i, h.get("n", ""), int(h.get("lv", 1))], 15, UI.PARCH)
		l.autowrap_mode = TextServer.AUTOWRAP_OFF
		lv.add_child(l)
		i += 1
	cols.add_child(lv)
	var ct := VBoxContainer.new()
	ct.add_child(UI.label("Le Cercle (saison %d)" % int(d.get("saison", 1)), 17, UI.EMBER))
	var cotes: Dictionary = d.get("cotes") if d.get("cotes") is Dictionary else {}
	for f in ["1", "2", "3"]:
		ct.add_child(UI.label("%s c. %s" % [f, f], 15, UI.LUMEN))
		var lst: Array = cotes.get(f, []) if cotes.get(f) is Array else []
		if lst.is_empty():
			ct.add_child(UI.label("  personne encore", 14, UI.MUTED))
		var j := 1
		for h in lst:
			var l := UI.label("  %d. %s · %d" % [j, h.get("n", ""), int(h.get("c", 0))], 14, UI.PARCH)
			l.autowrap_mode = TextServer.AUTOWRAP_OFF
			ct.add_child(l)
			j += 1
	cols.add_child(ct)
	var gu := VBoxContainer.new()
	gu.add_child(UI.label("Les guildes", 17, UI.EMBER))
	for g in (d.get("guildes") if d.get("guildes") is Array else []):
		var row := HBoxContainer.new()
		row.add_theme_constant_override("separation", 6)
		row.add_child(Blason.Vue.new(g.get("bl", {}), 26))
		var l := UI.label("%s · niv. %d · %d membres" % [g.get("n", ""), int(g.get("lv", 1)), int(g.get("m", 0))], 15, UI.PARCH)
		l.autowrap_mode = TextServer.AUTOWRAP_OFF
		row.add_child(l)
		gu.add_child(row)
	cols.add_child(gu)
	root.add_child(_honneur_win)


# --- Les lieux : l'anneau du Cercle et le tableau d'honneur ----------------------------------

func _poser_lieux() -> void:
	if _lieux != null:
		_lieux.queue_free()
		_lieux = null
	if not cercle.has("anneau"):
		return
	_lieux = Node2D.new()
	_lieux.draw.connect(_dessiner_lieux)
	world.tele_layer.add_child(_lieux)
	var h: Array = cercle.honneur
	if ResourceLoader.exists("res://cartes/decors/panneau.tscn"):
		var p: Node2D = load("res://cartes/decors/panneau.tscn").instantiate()
		p.position = Iso.to_screen(Vector2(float(h[0]), float(h[1]) - 0.6)).round()
		p.scale = Vector2(1.25, 1.25)
		world.entities.add_child(p)


func _dessiner_lieux() -> void:
	var a: Array = cercle.anneau
	var c := Iso.to_screen(Vector2(float(a[0]), float(a[1])))
	# Un cercle de pierres levées, au ras du sol.
	_lieux.draw_set_transform(c, 0, Vector2(1, 0.5))
	_lieux.draw_circle(Vector2.ZERO, 74, Color(0.85, 0.75, 0.55, 0.18))
	_lieux.draw_arc(Vector2.ZERO, 74, 0, TAU, 64, Color(0.55, 0.5, 0.45, 0.9), 3.0, true)
	_lieux.draw_set_transform(Vector2.ZERO, 0, Vector2.ONE)
	for i in 12:
		var ang := TAU * i / 12.0
		var p := c + Vector2(cos(ang) * 74, sin(ang) * 37)
		var hgt := 14.0 + (i % 3) * 4.0
		_lieux.draw_rect(Rect2(p + Vector2(-4, -hgt), Vector2(8, hgt)), Color("8a8296"))
		_lieux.draw_rect(Rect2(p + Vector2(-4, -hgt), Vector2(3, hgt)), Color("a9a1b5"))
	# Deux bannières : rouge et bleue, les deux camps.
	for side in [-1, 1]:
		var p: Vector2 = c + Vector2(side * 92, -6)
		_lieux.draw_line(p, p + Vector2(0, -46), Color("6d4c41"), 3.0)
		var col := Color("d03a3a") if side < 0 else Color("2980b9")
		var w := sin(_t * 3.0 + side) * 3.0
		_lieux.draw_colored_polygon(PackedVector2Array([p + Vector2(0, -46), p + Vector2(22, -40 + w), p + Vector2(0, -30)]), col)


## Un clic sur l'anneau ou le tableau.
func click(at: Vector2) -> bool:
	if not cercle.has("anneau"):
		return false
	var a: Array = cercle.anneau
	if at.distance_to(Iso.to_screen(Vector2(float(a[0]), float(a[1])))) < 60.0:
		open_cercle()
		return true
	var h: Array = cercle.honneur
	if at.distance_to(Iso.to_screen(Vector2(float(h[0]), float(h[1]))) + Vector2(0, -30)) < 40.0:
		Net.send(Net.OP_SOCIAL, {"a": "honneur"})
		return true
	return false


func _process(delta: float) -> void:
	_t += delta
	if _lieux != null:
		_lieux.queue_redraw()
	if int(_t * 5.0) != int((_t - delta) * 5.0):
		_update_party()
		_near_places()


func _near_places() -> void:
	if not cercle.has("anneau") or not world.pawns.has(world.my_sid):
		_cercle_btn.visible = false
		_honneur_btn.visible = false
		return
	var me := Iso.to_world(world.pawns[world.my_sid].position)
	var a: Array = cercle.anneau
	var h: Array = cercle.honneur
	_cercle_btn.visible = me.distance_to(Vector2(float(a[0]), float(a[1]))) < CERCLE_REACH and bout.is_empty()
	_honneur_btn.visible = me.distance_to(Vector2(float(h[0]), float(h[1]))) < HONNEUR_REACH


# --- La fenêtre sociale ---------------------------------------------------------------------

func toggle_window(tab: String) -> void:
	if win != null and is_instance_valid(win):
		if tab == "" or win._tab == tab:
			win.queue_free()
			win = null
			return
		win._tab = tab
		win._render()
		return
	win = preload("res://scripts/social_window.gd").new()
	win.social = self
	if tab != "":
		win._tab = tab
	win.tree_exited.connect(func(): win = null)
	root.add_child(win)
	Net.send(Net.OP_SOCIAL, {"a": "amis"})
	Net.send(Net.OP_SOCIAL, {"a": "guilde"})


func _refresh_win() -> void:
	if win != null and is_instance_valid(win):
		win._render()


# --- La carte du val ----------------------------------------------------------------------------

func toggle_carte() -> void:
	if _carte_win != null:
		_carte_win.queue_free()
		_carte_win = null
		return
	_carte_win = CarteVal.new()
	_carte_win.social = self
	root.add_child(_carte_win)


## La carte : le val vu d'en haut, avec toi, ton groupe, ta guilde et tes amis.
class CarteVal:
	extends PanelContainer
	var social: MondeSocial
	var _img: ImageTexture
	var _canvas: Control

	func _ready() -> void:
		theme = UI.make_theme()
		add_theme_stylebox_override("panel", UI.box(Color(UI.STONE, 0.97), UI.LUMEN, 14, 1, 12))
		set_anchors_and_offsets_preset(Control.PRESET_CENTER, Control.PRESET_MODE_MINSIZE)
		grow_horizontal = Control.GROW_DIRECTION_BOTH
		grow_vertical = Control.GROW_DIRECTION_BOTH
		var col := VBoxContainer.new()
		add_child(col)
		var head := HBoxContainer.new()
		var t := UI.label("🗺️ " + str(social.world.iso_map.data.get("name", "Le val")), 20, UI.LUMEN)
		t.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		head.add_child(t)
		var x := UI.button("✕")
		x.custom_minimum_size = Vector2(44, 40)
		x.focus_mode = Control.FOCUS_NONE
		x.pressed.connect(social.toggle_carte)
		head.add_child(x)
		col.add_child(head)
		var data: Dictionary = social.world.iso_map.data
		var w := int(data.get("width", 1))
		var h := mini(int(data.get("height", 1)), int(data.get("indoors_from", data.get("height", 1))))
		var img := Image.create(w, h, false, Image.FORMAT_RGBA8)
		var rows: Array = data.get("rows", [])
		for y in h:
			var r: String = rows[y]
			for xx in w:
				img.set_pixel(xx, y, Color("2b2a3a") if r[xx] == "#" else Color("5d7a4a"))
		_img = ImageTexture.create_from_image(img)
		_canvas = Control.new()
		var k := clampf((get_viewport_rect().size.y - 200) / h, 1.5, 3.0)
		_canvas.custom_minimum_size = Vector2(w, h) * k
		_canvas.draw.connect(_dessine.bind(k))
		col.add_child(_canvas)
		col.add_child(UI.label("🟡 toi · 🟢 ton groupe · 🔵 ta guilde · 🩷 tes amis · 🏟️ le Cercle · 🏆 le tableau d'honneur", 14, UI.MUTED))

	func _process(_d: float) -> void:
		_canvas.queue_redraw()

	func _dessine(k: float) -> void:
		_canvas.draw_texture_rect(_img, Rect2(Vector2.ZERO, _canvas.custom_minimum_size), false)
		var font := ThemeDB.fallback_font
		if social.cercle.has("anneau"):
			for p in [[social.cercle.anneau, "🏟️"], [social.cercle.honneur, "🏆"]]:
				_canvas.draw_string(font, Vector2(float(p[0][0]), float(p[0][1])) * k + Vector2(-9, 7), p[1], HORIZONTAL_ALIGNMENT_LEFT, -1, 18)
		var amis := {}
		for a in social.amis:
			if bool(a.get("ami", false)):
				amis[str(a.get("n", ""))] = true
		var membres := {}
		for m in social.guilde.get("membres", []) if social.guilde.get("membres") is Array else []:
			membres[str(m.get("n", ""))] = true
		for sid in social.world.pawns:
			var p: Pawn = social.world.pawns[sid]
			if p.race == "" or p.npc != "":
				continue
			var at := Iso.to_world(p.position) * k
			var col := Color(0, 0, 0, 0)
			if sid == social.world.my_sid:
				col = UI.LUMEN
			elif social.in_group(sid):
				col = UI.MOSS
			elif membres.has(p.hero_name):
				col = Color("7fa8e8")
			elif amis.has(p.hero_name):
				col = Color("f080b0")
			if col.a == 0:
				continue
			_canvas.draw_circle(at, 6, Color(0, 0, 0, 0.7))
			_canvas.draw_circle(at, 4.5, col)
			_canvas.draw_string_outline(font, at + Vector2(8, 5), p.hero_name, HORIZONTAL_ALIGNMENT_LEFT, -1, 13, 4, Color(0, 0, 0, 0.8))
			_canvas.draw_string(font, at + Vector2(8, 5), p.hero_name, HORIZONTAL_ALIGNMENT_LEFT, -1, 13, UI.PARCH)

	func _unhandled_key_input(event: InputEvent) -> void:
		if event.pressed and not event.echo and (event.physical_keycode == KEY_ESCAPE or event.physical_keycode == Touches.key("carte")):
			get_viewport().set_input_as_handled()
			social.toggle_carte()
