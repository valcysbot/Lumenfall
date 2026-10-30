class_name CombatHud
extends Control
## L'interface de combat : ta vie, ta Ferveur, tes roulades, ta barre de
## compétences (touches 1 à 5, R pour la potion), le cadre de ta cible, la
## barre d'incantation et l'écran « Tu es tombé ». Sur téléphone, les
## compétences forment un éventail sous le pouce droit.

signal skill(k: int)
signal potion
signal roll
signal next_target
signal block(on: bool)

## Les compétences du Paladin (même ordre que le serveur, server/world/arena.go).
const SKILLS := [
	{},
	{"nom": "Frappe sacrée", "icone": "⚔️", "cd": 4.0, "cout": 0, "aide": "Un coup de lumière au corps à corps. +15 Ferveur."},
	{"nom": "Coup de bouclier", "icone": "🛡️", "cd": 10.0, "cout": 0, "aide": "Étourdit 2 s et attire la colère du monstre."},
	{"nom": "Lumière de Solenn", "icone": "💛", "cd": 6.0, "cout": 30, "aide": "Soigne ta cible alliée, ou toi. 1,5 s d'incantation : ne bouge pas !"},
	{"nom": "Aura de Solenn", "icone": "🌟", "cd": 30.0, "cout": 0, "aide": "12 s : ton groupe encaisse mieux et frappe plus fort."},
	{"nom": "Jugement de lumière", "icone": "☀️", "cd": 8.0, "cout": 40, "aide": "Frappe à distance et embrase la cible."},
]
const FX_NAMES := ["🔥 Brûlure", "🧪 Poison", "💫 Étourdi", "🐌 Ralenti", "💔 Faiblesse", "🌟 Aura"]

var touch := false
var buttons := {}     # 1..5, "potion", "roulade", "cible", "garde"
var ferveur := 0
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

var _frame: PanelContainer
var _hp_bar: ProgressBar
var _hp_text: Label
var _fer_bar: ProgressBar
var _xp_bar: ProgressBar
var _lvl_text: Label
var _fx_text: Label
var _target_box: PanelContainer
var _target_name: Label
var _target_bar: ProgressBar
var _target_fx: Label
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
	_fer_bar = _bar(UI.LUMEN, 9, 250)
	_fer_bar.max_value = 100
	_fer_bar.tooltip_text = "Ferveur : elle monte quand tu frappes ou encaisses."
	col.add_child(_fer_bar)
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
	for k in range(1, 6):
		var s: Dictionary = SKILLS[k]
		bar.add_child(_make(k, s.icone, str(k), 60, false, "%s (%d)\n%s" % [s.nom, k, s.aide]))
	bar.add_child(_make("potion", "🧪", "R", 60, false, "Potion de soin (R)\nRend 40 % de ta vie. Recharge 30 s."))
	bar.add_child(_make("roulade", "💨", "Esp.", 60, false, "Roulade (Espace)\nTu esquives tout pendant la roulade. Deux d'affilée, puis tu souffles."))
	bar.set_anchors_and_offsets_preset(Control.PRESET_CENTER_BOTTOM, Control.PRESET_MODE_MINSIZE, 14)
	bar.grow_horizontal = Control.GROW_DIRECTION_BOTH
	bar.grow_vertical = Control.GROW_DIRECTION_BEGIN
	add_child(bar)
	_tip = UI.label("Clic : viser · Tab : cible suivante · clic droit maintenu : bouclier", 13, UI.MUTED)
	_tip.autowrap_mode = TextServer.AUTOWRAP_OFF
	_tip.set_anchors_and_offsets_preset(Control.PRESET_CENTER_BOTTOM, Control.PRESET_MODE_MINSIZE, 82)
	_tip.grow_horizontal = Control.GROW_DIRECTION_BOTH
	_tip.grow_vertical = Control.GROW_DIRECTION_BEGIN
	_tip.add_theme_color_override("font_outline_color", Color(0, 0, 0, 0.8))
	_tip.add_theme_constant_override("outline_size", 4)
	add_child(_tip)


## Sur téléphone : un éventail sous le pouce droit, la roulade au centre.
func _build_fan() -> void:
	var fan := Control.new()
	fan.mouse_filter = Control.MOUSE_FILTER_IGNORE
	fan.custom_minimum_size = Vector2(330, 300)
	fan.set_anchors_and_offsets_preset(Control.PRESET_BOTTOM_RIGHT, Control.PRESET_MODE_MINSIZE, 16)
	fan.grow_horizontal = Control.GROW_DIRECTION_BEGIN
	fan.grow_vertical = Control.GROW_DIRECTION_BEGIN
	add_child(fan)
	var center := Vector2(250, 200)
	var big := _make("roulade", "💨", "", 96, true, "Roulade")
	big.position = center - Vector2(48, 48)
	fan.add_child(big)
	for k in range(1, 6):
		var a := deg_to_rad(180.0 + (k - 1) * 27.0)
		var b := _make(k, SKILLS[k].icone, "", 70, true, SKILLS[k].nom)
		b.position = center + Vector2(cos(a), sin(a)) * 150 - Vector2(35, 35)
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
	ferveur = int(d.get("f", ferveur))
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
	for k in range(1, 6):
		var left := float(cds[k - 1]) / 1000.0 if k - 1 < cds.size() else 0.0
		var total: float = SKILLS[k].cd
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


func start_cast(k: int, ms: int) -> void:
	if k == 0 or ms <= 0:
		_cast_box.visible = false
		return
	_cast_total = ms / 1000.0
	_cast_end = SkillButton.now() + _cast_total
	_cast_label.text = SKILLS[k].nom if k < SKILLS.size() else ""
	_cast_box.visible = true


func show_death(on: bool) -> void:
	_death.visible = on
	_death_t = 5.0
	if on:
		_cast_box.visible = false


static func fx_text(bits: int) -> String:
	var parts: Array[String] = []
	for e in FX_NAMES.size():
		if bits & (1 << e):
			parts.append(FX_NAMES[e])
	return "  ".join(parts)


func _refresh() -> void:
	_hp_bar.max_value = maxi(1, hp_max)
	_hp_bar.value = hp
	_hp_text.text = "%d / %d" % [hp, hp_max]
	_fer_bar.value = ferveur
	_xp_bar.max_value = maxi(1, xp_next)
	_xp_bar.value = xp
	_lvl_text.text = "Niv. %d · ✨ %d / %d XP · 🪙 %d" % [level, xp, xp_next, gold]
	_fx_text.text = fx_text(fx)
	_fx_text.visible = _fx_text.text != ""
	for k in range(1, 6):
		buttons[k].dim = ferveur < int(SKILLS[k].cout)
	buttons.potion.count = potions
	buttons.potion.dim = potions <= 0
	buttons.roulade.pips = endurance
	buttons.roulade.dim = endurance < 1.0


func _process(delta: float) -> void:
	if target != null and is_instance_valid(target) and not target.dead:
		_target_box.visible = true
		_target_name.text = "%s%s · niv. %d" % ["✦ " if target.elite else "", target.hero_name, target.level]
		_target_bar.max_value = maxi(1, target.hp_max)
		_target_bar.value = target.hp
		var foe := target.kind != ""
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
