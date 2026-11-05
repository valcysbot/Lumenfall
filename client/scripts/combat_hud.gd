class_name CombatHud
extends Control
## L'interface de combat : ta vie, la ressource de ta classe, tes roulades, ta
## barre de compétences (touches 1 à 8, la 8 étant l'ultime de ta voie ; R pour
## la potion), le cadre de ta cible, la barre d'incantation et l'écran « Tu es
## tombé ». Sur téléphone, les compétences forment un éventail sous le pouce droit.

signal skill(k: int)
signal potion
signal roll
signal next_target
signal block(on: bool)
signal challenge
signal fiche
signal trade
signal social(a: String)   # Acte VI : inviter dans le groupe, demander en ami, inviter dans la guilde

const NSKILLS := 8

var touch := false
var buttons := {}     # 1..8, "potion", "roulade", "cible", "garde"
var classe := ""
var voie := -1
var accord := 0
var hardcore := false
var res := 0
var res_max := 100
var overheat := 0.0
var skills := []      # 0 : rien, puis les 8 compétences de la classe (Heros.competence)
var endurance := 2.0
var potions := 5
var level := 1
var xp := 0
var xp_next := 100
var gold := 0
var hp := 1
var hp_max := 1
var fx := 0
var target: Pawn
var in_duel := false

var _frame: PanelContainer
var _hp_bar: ProgressBar
var _hp_text: Label
var _res_bar: ProgressBar
var _res_text: Label
var _xp_bar: ProgressBar
var _lvl_text: Label
var _fx_text: Label
var _target_box: PanelContainer
var _target_name: Label
var _target_bar: ProgressBar
var _target_fx: Label
var _duel_btn: Button
var _fiche_btn: Button
var _trade_btn: Button
var _social_row: HBoxContainer
var _cast_box: PanelContainer
var _cast_bar: ProgressBar
var _cast_label: Label
var _cast_end := 0.0
var _cast_total := 1.0
var _death: PanelContainer
var _death_label: Label
var _death_t := 0.0
var _tip: Label


func _ready() -> void:
	set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	mouse_filter = Control.MOUSE_FILTER_IGNORE
	_build_frame()
	_build_target()
	if touch:
		_build_fan()
	else:
		_build_bar()
	_build_cast()
	_build_death()
	_refresh()


func _bar(color: Color, h: float, w: float) -> ProgressBar:
	var b := ProgressBar.new()
	b.show_percentage = false
	b.custom_minimum_size = Vector2(w, h)
	var bg := StyleBoxFlat.new()
	bg.bg_color = Color(0, 0, 0, 0.6)
	bg.set_corner_radius_all(4)
	var fill := StyleBoxFlat.new()
	fill.bg_color = color
	fill.set_corner_radius_all(4)
	b.add_theme_stylebox_override("background", bg)
	b.add_theme_stylebox_override("fill", fill)
	b.mouse_filter = Control.MOUSE_FILTER_IGNORE
	return b


func _build_frame() -> void:
	_frame = PanelContainer.new()
	_frame.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.7), Color(UI.LINE, 0.8), 12, 1, 8))
	_frame.position = Vector2(14, 66)
	_frame.mouse_filter = Control.MOUSE_FILTER_IGNORE
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 4)
	_frame.add_child(col)
	_lvl_text = UI.label("", 15, UI.LUMEN)
	_lvl_text.autowrap_mode = TextServer.AUTOWRAP_OFF
	col.add_child(_lvl_text)
	var hp_row := Control.new()
	hp_row.custom_minimum_size = Vector2(250, 20)
	_hp_bar = _bar(Color("5fbf6a"), 20, 250)
	hp_row.add_child(_hp_bar)
	_hp_text = UI.label("", 14)
	_hp_text.autowrap_mode = TextServer.AUTOWRAP_OFF
	_hp_text.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_hp_text.size = Vector2(250, 20)
	_hp_text.add_theme_color_override("font_outline_color", Color(0, 0, 0, 0.9))
	_hp_text.add_theme_constant_override("outline_size", 4)
	hp_row.add_child(_hp_text)
	col.add_child(hp_row)
	var res_row := Control.new()
	res_row.custom_minimum_size = Vector2(250, 14)
	_res_bar = _bar(UI.LUMEN, 14, 250)
	_res_bar.max_value = 100
	res_row.add_child(_res_bar)
	_res_text = UI.label("", 11)
	_res_text.autowrap_mode = TextServer.AUTOWRAP_OFF
	_res_text.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_res_text.size = Vector2(250, 14)
	_res_text.add_theme_color_override("font_outline_color", Color(0, 0, 0, 0.9))
	_res_text.add_theme_constant_override("outline_size", 3)
	res_row.add_child(_res_text)
	col.add_child(res_row)
	_xp_bar = _bar(Color("8a6ad8"), 5, 250)
	col.add_child(_xp_bar)
	_fx_text = UI.label("", 14, UI.MUTED)
	_fx_text.autowrap_mode = TextServer.AUTOWRAP_OFF
	col.add_child(_fx_text)
	add_child(_frame)


func _build_target() -> void:
	_target_box = PanelContainer.new()
	_target_box.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.75), Color(UI.DANGER, 0.7), 12, 1, 8))
	_target_box.set_anchors_and_offsets_preset(Control.PRESET_CENTER_TOP, Control.PRESET_MODE_MINSIZE, 70)
	_target_box.grow_horizontal = Control.GROW_DIRECTION_BOTH
	_target_box.mouse_filter = Control.MOUSE_FILTER_IGNORE
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 4)
	_target_box.add_child(col)
	_target_name = UI.label("", 16)
	_target_name.autowrap_mode = TextServer.AUTOWRAP_OFF
	_target_name.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	col.add_child(_target_name)
	_target_bar = _bar(UI.DANGER, 14, 280)
	col.add_child(_target_bar)
	_target_fx = UI.label("", 13, UI.MUTED)
	_target_fx.autowrap_mode = TextServer.AUTOWRAP_OFF
	_target_fx.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	col.add_child(_target_fx)
	_duel_btn = UI.button("⚔️ Défier en duel")
	_duel_btn.focus_mode = Control.FOCUS_NONE
	_duel_btn.tooltip_text = "Un combat pour l'honneur : personne ne meurt."
	_duel_btn.pressed.connect(func(): challenge.emit())
	col.add_child(_duel_btn)
	_fiche_btn = UI.button("📜 Sa fiche")
	_fiche_btn.focus_mode = Control.FOCUS_NONE
	_fiche_btn.pressed.connect(func(): fiche.emit())
	col.add_child(_fiche_btn)
	_trade_btn = UI.button("🤝 Échanger")
	_trade_btn.focus_mode = Control.FOCUS_NONE
	_trade_btn.tooltip_text = "Proposer un échange face à face (assez près l'un de l'autre)."
	_trade_btn.pressed.connect(func(): trade.emit())
	col.add_child(_trade_btn)
	_social_row = HBoxContainer.new()
	_social_row.add_theme_constant_override("separation", 4)
	for b in [["👥 Groupe", "inviter", "L'inviter dans ton groupe"], ["🫂 Ami", "ami", "Lui demander d'être ton ami"],
			["🏰 Guilde", "guilde_inviter", "L'inviter dans ta guilde"]]:
		var btn := UI.button(b[0])
		btn.focus_mode = Control.FOCUS_NONE
		btn.custom_minimum_size = Vector2(0, 40)
		btn.add_theme_font_size_override("font_size", 14)
		btn.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		btn.tooltip_text = b[2]
		var a: String = b[1]
		btn.pressed.connect(func(): social.emit(a))
		_social_row.add_child(btn)
	col.add_child(_social_row)
	_target_box.visible = false
	add_child(_target_box)


func _make(id, icon: String, key: String, sz: float, round_shape: bool, tip: String) -> SkillButton:
	var b := SkillButton.new(icon, key, sz, round_shape)
	b.tooltip_text = tip
	buttons[id] = b
	if id is int:
		b.activated.connect(func(): skill.emit(id))
	elif id == "potion":
		b.activated.connect(func(): potion.emit())
		b.tint = UI.MOSS
	elif id == "roulade":
		b.activated.connect(func(): roll.emit())
		b.tint = UI.MOSS
	elif id == "cible":
		b.activated.connect(func(): next_target.emit())
		b.tint = UI.DANGER
	elif id == "garde":
		b.held.connect(func(on): block.emit(on))
		b.tint = Color("6fa8e8")
	return b


func _build_bar() -> void:
	var bar := HBoxContainer.new()
	bar.add_theme_constant_override("separation", 6)
	bar.mouse_filter = Control.MOUSE_FILTER_IGNORE
	for k in range(1, NSKILLS + 1):
		bar.add_child(_make(k, "", Touches.label("comp_%d" % k, true), 56, false, ""))
	bar.add_child(_make("potion", "🧪", Touches.label("potion", true), 56, false, "Potion de soin\nRend 40 % de ta vie. Recharge 30 s."))
	bar.add_child(_make("roulade", "💨", Touches.label("roulade", true), 56, false, "Roulade\nTu esquives tout pendant la roulade. Deux d'affilée, puis tu souffles."))
	bar.set_anchors_and_offsets_preset(Control.PRESET_CENTER_BOTTOM, Control.PRESET_MODE_MINSIZE, 14)
	bar.grow_horizontal = Control.GROW_DIRECTION_BOTH
	bar.grow_vertical = Control.GROW_DIRECTION_BEGIN
	add_child(bar)
	_tip = UI.label("", 13, UI.MUTED)
	_tip.autowrap_mode = TextServer.AUTOWRAP_OFF
	_tip.set_anchors_and_offsets_preset(Control.PRESET_CENTER_BOTTOM, Control.PRESET_MODE_MINSIZE, 82)
	_tip.grow_horizontal = Control.GROW_DIRECTION_BOTH
	_tip.grow_vertical = Control.GROW_DIRECTION_BEGIN
	_tip.add_theme_color_override("font_outline_color", Color(0, 0, 0, 0.8))
	_tip.add_theme_constant_override("outline_size", 4)
	add_child(_tip)
	refresh_keys()


## Après un changement dans ⌨ Touches : les lettres sur les boutons suivent.
func refresh_keys() -> void:
	if touch:
		return
	for id in buttons:
		var b: SkillButton = buttons[id]
		if id is int:
			b.key = Touches.label("comp_%d" % id, true)
		elif id == "potion" or id == "roulade":
			b.key = Touches.label(id, true)
		b.queue_redraw()
	if _tip != null:
		_tip.text = "Clic : viser · %s : cible suivante · %s : ton héros" % [Touches.label("cible"), Touches.label("fiche")]


## Sur téléphone : un éventail sous le pouce droit, la roulade au centre.
func _build_fan() -> void:
	var fan := Control.new()
	fan.mouse_filter = Control.MOUSE_FILTER_IGNORE
	fan.custom_minimum_size = Vector2(330, 360)
	fan.set_anchors_and_offsets_preset(Control.PRESET_BOTTOM_RIGHT, Control.PRESET_MODE_MINSIZE, 16)
	fan.grow_horizontal = Control.GROW_DIRECTION_BEGIN
	fan.grow_vertical = Control.GROW_DIRECTION_BEGIN
	add_child(fan)
	var center := Vector2(250, 230)
	var big := _make("roulade", "💨", "", 96, true, "Roulade")
	big.position = center - Vector2(48, 48)
	fan.add_child(big)
	# Deux arcs : les compétences 1 à 5 tout près du pouce, 6 à 8 (et l'ultime) plus haut.
	for k in range(1, NSKILLS + 1):
		var inner := k <= 5
		var a := deg_to_rad(180.0 + (k - 1) * 22.5) if inner else deg_to_rad(192.0 + (k - 6) * 33.0)
		var r := 150.0 if inner else 228.0
		var b := _make(k, "", "", 58, true, "")
		b.position = center + Vector2(cos(a), sin(a)) * r - Vector2(29, 29)
		fan.add_child(b)
	var extras := [["cible", "🎯", Vector2(-232, 70)], ["garde", "🛡️", Vector2(-160, 78)], ["potion", "🧪", Vector2(-88, 76)]]
	for e in extras:
		var b := _make(e[0], e[1], "", 58, true, e[0])
		b.position = center + e[2] - Vector2(29, 29)
		fan.add_child(b)


func _build_cast() -> void:
	_cast_box = PanelContainer.new()
	_cast_box.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.8), UI.LUMEN, 10, 1, 6))
	_cast_box.set_anchors_and_offsets_preset(Control.PRESET_CENTER_BOTTOM, Control.PRESET_MODE_MINSIZE, 120 if not touch else 330)
	_cast_box.grow_horizontal = Control.GROW_DIRECTION_BOTH
	_cast_box.grow_vertical = Control.GROW_DIRECTION_BEGIN
	_cast_box.mouse_filter = Control.MOUSE_FILTER_IGNORE
	var col := VBoxContainer.new()
	_cast_box.add_child(col)
	_cast_label = UI.label("", 15, UI.LUMEN)
	_cast_label.autowrap_mode = TextServer.AUTOWRAP_OFF
	_cast_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	col.add_child(_cast_label)
	_cast_bar = _bar(UI.LUMEN, 10, 240)
	col.add_child(_cast_bar)
	_cast_box.visible = false
	add_child(_cast_box)


func _build_death() -> void:
	_death = PanelContainer.new()
	_death.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.88), UI.DANGER, 16, 2, 20))
	_death.set_anchors_and_offsets_preset(Control.PRESET_CENTER, Control.PRESET_MODE_MINSIZE)
	_death.grow_horizontal = Control.GROW_DIRECTION_BOTH
	_death.grow_vertical = Control.GROW_DIRECTION_BOTH
	_death.mouse_filter = Control.MOUSE_FILTER_IGNORE
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 8)
	_death.add_child(col)
	var title := UI.label("💀 Tu es tombé…", 30, UI.DANGER)
	title.name = "Title"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	col.add_child(title)
	_death_label = UI.label("", 17)
	_death_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_death_label.custom_minimum_size = Vector2(420, 0)
	col.add_child(_death_label)
	_death.visible = false
	add_child(_death)


# --- Ce que le serveur raconte -----------------------------------------------

func set_me(d: Dictionary) -> void:
	var cl := str(d.get("cl", classe))
	var v := int(d.get("v", voie))
	if cl != classe or v != voie:
		classe = cl
		voie = v
		_apply_class()
	res = int(d.get("r", res))
	res_max = maxi(1, int(d.get("rm", res_max)))
	overheat = float(d.get("oh", 0)) / 1000.0
	endurance = float(d.get("en", endurance * 10)) / 10.0
	potions = int(d.get("po", potions))
	level = int(d.get("lv", level))
	xp = int(d.get("xp", xp))
	xp_next = int(d.get("xpn", xp_next))
	gold = int(d.get("or", gold))
	if d.has("hp"):
		hp = int(d.hp)
		hp_max = int(d.hpm)
	var gcd := float(d.get("gcd", 0)) / 1000.0
	var cds: Array = d.get("cd", [])
	var cdt: Array = d.get("cdt", [])  # les recharges avec les talents
	for k in range(1, NSKILLS + 1):
		var left := float(cds[k - 1]) / 1000.0 if k - 1 < cds.size() else 0.0
		var total: float = float(skills[k].get("cd", 1.0)) if k < skills.size() else 1.0
		if k - 1 < cdt.size() and int(cdt[k - 1]) > 0:
			total = float(cdt[k - 1]) / 1000.0
		if gcd > left:
			left = gcd
			total = 1.0
		buttons[k].start_cd(left, total)
	buttons.potion.start_cd(float(d.get("pocd", 0)) / 1000.0, 30.0)
	_refresh()


func set_vitals(new_hp: int, new_max: int, new_fx: int, new_level: int) -> void:
	hp = new_hp
	hp_max = new_max
	fx = new_fx
	level = new_level
	_refresh()


func start_cast(k: int, ms: int, text: String = "") -> void:
	if k == 0 or ms <= 0:
		_cast_box.visible = false
		return
	_cast_total = ms / 1000.0
	_cast_end = SkillButton.now() + _cast_total
	_cast_label.text = text if text != "" else (str(skills[k].get("nom", "")) if k < skills.size() else "")
	_cast_box.visible = true


func show_death(on: bool) -> void:
	_death.visible = on
	(_death.get_child(0).get_node("Title") as Label).text = ("💀 Tu es %s pour toujours…" if hardcore else "💀 Tu es %s…") % \
			["tombé", "tombée", "tombé·e"][clampi(accord, 0, 2)]
	_death_t = 5.0
	if on:
		_cast_box.visible = false


static func fx_text(bits: int) -> String:
	var parts: Array[String] = []
	var names: Array = Heros.data().effets
	for e in names.size():
		if bits & (1 << e) and names[e] != "compagnon":
			parts.append(Heros.EFFETS.get(names[e], names[e]))
	return "  ".join(parts)


## La classe ou la voie a changé : les boutons prennent leurs compétences.
func _apply_class() -> void:
	skills = [{}]
	for k in range(1, NSKILLS + 1):
		skills.append(Heros.competence(classe, k, voie))
	var c := Heros.classe(classe)
	var rname: String = c.ressource.nom
	var col := Color(Heros.RESSOURCE_COULEURS.get(str(c.ressource.id), "e8c66a"))
	(_res_bar.get_theme_stylebox("fill") as StyleBoxFlat).bg_color = col
	_res_bar.tooltip_text = rname
	for k in range(1, NSKILLS + 1):
		var sk: Dictionary = skills[k]
		var b: SkillButton = buttons[k]
		if sk.is_empty():
			b.set_icon("🌟")
			b.tooltip_text = "Ultime (8)\nChoisis ta voie au niveau 10 pour l'apprendre."
			continue
		b.set_icon(sk.icone)
		var cost := ""
		if float(sk.cout) > 0:
			cost = "\n%s %d %s" % ["+" if c.ressource.chaleur else "−", int(sk.cout), rname]
		var cast := "\nIncantation %.1f s : ne bouge pas !" % float(sk.incante) if float(sk.incante) > 0 else ""
		b.tooltip_text = "%s (%d) · niv. %d\n%s%s%s\nRecharge %d s" % [sk.nom, k, int(sk.niveau), sk.aide, cost, cast, int(sk.cd)]
	if buttons.has("garde"):
		buttons.garde.visible = bool(c.bouclier)


func _refresh() -> void:
	_hp_bar.max_value = maxi(1, hp_max)
	_hp_bar.value = hp
	_hp_text.text = "%d / %d" % [hp, hp_max]
	_res_bar.max_value = res_max
	_res_bar.value = res
	if not skills.is_empty():
		var c := Heros.classe(classe)
		_res_text.text = "%s %d / %d%s" % [c.ressource.nom, res, res_max, "  · SURCHAUFFE" if overheat > 0 else ""]
	_xp_bar.max_value = maxi(1, xp_next)
	_xp_bar.value = xp
	_lvl_text.text = "Niv. %d · ✨ %d / %d XP · 🪙 %s" % [level, xp, xp_next, Objets.monnaie(gold)]
	_fx_text.text = fx_text(fx)
	_fx_text.visible = _fx_text.text != ""
	for k in range(1, NSKILLS + 1):
		var b: SkillButton = buttons[k]
		var sk: Dictionary = skills[k] if k < skills.size() else {}
		b.locked = sk.is_empty() or level < int(sk.niveau)
		if sk.is_empty():
			continue
		if Heros.classe(classe).ressource.chaleur:
			b.dim = overheat > 0
		elif sk.get("final", false):
			b.dim = res < 1
		else:
			b.dim = res < int(sk.cout)
	buttons.potion.count = potions
	buttons.potion.dim = potions <= 0
	buttons.roulade.pips = endurance
	buttons.roulade.dim = endurance < 1.0


func _process(delta: float) -> void:
	if target != null and is_instance_valid(target) and not target.dead:
		_target_box.visible = true
		var crest: String = (Heros.faction_of(target.race).blason + " ") if target.race != "" else ""
		_target_name.text = "%s%s%s · niv. %d" % ["✦ " if target.elite else "", crest, target.hero_name, target.level]
		_target_bar.max_value = maxi(1, target.hp_max)
		_target_bar.value = target.hp
		var foe := target.kind != "" or target.foe
		_duel_btn.visible = target.kind == "" and target.npc == "" and not target.is_me and not in_duel
		_fiche_btn.visible = target.race != "" and not target.is_me
		_trade_btn.visible = target.race != "" and not target.is_me and not in_duel
		_social_row.visible = target.race != "" and not target.is_me and not in_duel
		(_target_bar.get_theme_stylebox("fill") as StyleBoxFlat).bg_color = UI.DANGER if foe else Color("5fbf6a")
		_target_fx.text = fx_text(target.fx)
		_target_fx.visible = _target_fx.text != ""
	else:
		_target_box.visible = false
	if _cast_box.visible:
		var left := _cast_end - SkillButton.now()
		_cast_bar.max_value = _cast_total
		_cast_bar.value = _cast_total - left
		if left <= -0.3:
			_cast_box.visible = false
	if _death.visible:
		_death_t = maxf(0, _death_t - delta)
		_death_label.text = "Sœur Clémence te relève au temple de Solenn dans %d s.\nBraise te garde une place au chaud." % int(ceil(_death_t))
