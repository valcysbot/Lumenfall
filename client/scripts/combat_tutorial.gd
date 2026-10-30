class_name CombatTutorial
extends PanelContainer
## Braise apprend à se battre, une étape à la fois, au premier monstre croisé :
## viser, frapper, se protéger, rouler, boire une potion, puis les autres sorts.
## Chaque étape attend que le joueur fasse le geste ; le bouton concerné brille.

signal glow(id)        # le bouton à faire briller (null : aucun)
signal finished

## id : le geste attendu (world_screen.gd appelle did(id)) ; btn : le bouton qui brille.
const STEPS := [
	{"id": "cible", "btn": "cible",
		"pc": "Un monstre ! [color=#ffd77a]Clique dessus[/color] pour le viser (ou [color=#ffd77a]Tab[/color]). Un cercle doré apparaît sous lui.",
		"tel": "Un monstre ! [color=#ffd77a]Touche-le du doigt[/color] pour le viser (ou le bouton 🎯). Un cercle doré apparaît sous lui."},
	{"id": "frappe", "btn": 1,
		"pc": "Lance ta [color=#ffd77a]Frappe sacrée[/color] : touche [color=#ffd77a]1[/color] ⚔️. Tu cours jusqu'à lui, puis tu frappes. Entre deux sorts, ton héros tape aussi tout seul.",
		"tel": "Lance ta [color=#ffd77a]Frappe sacrée[/color] : le bouton ⚔️. Tu cours jusqu'à lui, puis tu frappes. Entre deux sorts, ton héros tape aussi tout seul."},
	{"id": "garde", "btn": "garde",
		"pc": "Il riposte ! [color=#ffd77a]Maintiens le clic droit[/color] pour lever ton bouclier 🛡️ : les coups de face font beaucoup moins mal.",
		"tel": "Il riposte ! [color=#ffd77a]Garde le doigt sur 🛡️[/color] pour lever ton bouclier : les coups de face font beaucoup moins mal."},
	{"id": "roulade", "btn": "roulade",
		"pc": "Une [color=#ff7a5a]zone rouge[/color] au sol annonce un gros coup : [color=#ffd77a]roule[/color] pour en sortir avec [color=#ffd77a]Espace[/color] 💨 (vers où tu marches). Deux roulades, puis on souffle. Essaie !",
		"tel": "Une [color=#ff7a5a]zone rouge[/color] au sol annonce un gros coup : [color=#ffd77a]roule[/color] pour en sortir avec le gros bouton 💨 (vers où pousse le joystick). Deux roulades, puis on souffle. Essaie !"},
	{"id": "potion", "btn": "potion",
		"pc": "Ta vie, c'est la [color=#7fe08a]barre verte[/color] en haut à gauche. Elle baisse ? Bois une [color=#ffd77a]potion[/color] 🧪 : touche [color=#ffd77a]R[/color]. Elles se remplissent au village.",
		"tel": "Ta vie, c'est la [color=#7fe08a]barre verte[/color] en haut à gauche. Elle baisse ? Bois une [color=#ffd77a]potion[/color] 🧪. Elles se remplissent au village."},
	{"id": "fin", "btn": null,
		"pc": "Bravo ! Tes autres sorts : [color=#ffd77a]2[/color] 🛡️ étourdit, [color=#ffd77a]3[/color] 💛 te soigne, [color=#ffd77a]4[/color] 🌟 te renforce, [color=#ffd77a]5[/color] ☀️ frappe de loin. Ils coûtent de la [color=#ffd77a]Ferveur[/color] (la barre jaune), qui monte quand tu frappes. Survole un bouton pour relire ce qu'il fait.",
		"tel": "Bravo ! Les autres boutons de l'éventail : 🛡️ étourdit, 💛 te soigne, 🌟 te renforce, ☀️ frappe de loin. Ils coûtent de la [color=#ffd77a]Ferveur[/color] (la barre jaune), qui monte quand tu frappes."},
]

const CHEERS := ["Bien joué !", "Parfait !", "Exactement !", "Tu apprends vite !", "Comme un vrai Paladin !"]

var touch := false
var step := -1
var _title: Label
var _text: RichTextLabel
var _ok: Button
var _cheer: Label
var _cheer_t := 0.0


func _ready() -> void:
	add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.9), UI.EMBER, 14, 2, 14))
	position = Vector2(14, 172)   # sous ta vie : le héros reste visible au centre
	custom_minimum_size = Vector2(400, 0)
	mouse_filter = Control.MOUSE_FILTER_STOP
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 8)
	add_child(col)
	var head := HBoxContainer.new()
	col.add_child(head)
	_title = UI.label("", 16, UI.EMBER)
	_title.autowrap_mode = TextServer.AUTOWRAP_OFF
	_title.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	head.add_child(_title)
	_cheer = UI.label("", 16, UI.MOSS)
	_cheer.autowrap_mode = TextServer.AUTOWRAP_OFF
	head.add_child(_cheer)
	_text = RichTextLabel.new()
	_text.bbcode_enabled = true
	_text.fit_content = true
	_text.scroll_active = false
	_text.custom_minimum_size = Vector2(372, 0)
	_text.add_theme_font_size_override("normal_font_size", 16)
	_text.add_theme_color_override("default_color", UI.PARCH)
	_text.mouse_filter = Control.MOUSE_FILTER_IGNORE
	col.add_child(_text)
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 8)
	row.alignment = BoxContainer.ALIGNMENT_END
	col.add_child(row)
	var skip := UI.button("Passer")
	skip.tooltip_text = "Tu pourras le revoir dans les réglages ⚙"
	skip.pressed.connect(stop)
	row.add_child(skip)
	_ok = UI.button("C'est parti !", true)
	_ok.pressed.connect(stop)
	row.add_child(_ok)
	for b in [skip, _ok]:
		b.focus_mode = Control.FOCUS_NONE   # Espace sert à rouler
	visible = false


func running() -> bool:
	return step >= 0


func start(skip_first := false) -> void:
	step = 1 if skip_first else 0
	visible = true
	_show()


func stop() -> void:
	if step < 0:
		return
	step = -1
	visible = false
	glow.emit(null)
	finished.emit()


## Le joueur vient de faire un geste : si c'est celui qu'on attend, étape suivante.
func did(id: String) -> bool:
	if step < 0 or STEPS[step].id != id:
		return false
	step += 1
	_cheer.text = "✓ " + CHEERS.pick_random()
	_cheer_t = 2.0
	_show()
	return true


func _show() -> void:
	var s: Dictionary = STEPS[step]
	_title.text = "🐉 Braise t'apprend · %d / %d" % [step + 1, STEPS.size()]
	_text.text = s.tel if touch else s.pc
	_ok.visible = s.id == "fin"
	var btn = s.btn
	if str(btn) == "cible" and not touch:
		btn = null   # sur PC, on vise à la souris : pas de bouton
	glow.emit(btn)


func _process(delta: float) -> void:
	if _cheer_t > 0:
		_cheer_t -= delta
		_cheer.modulate.a = clampf(_cheer_t, 0, 1)
