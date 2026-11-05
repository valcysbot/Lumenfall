class_name Touches
## Les touches du clavier, à changer dans ⚙ Réglages → ⌨ Touches (retour de la
## soirée de jeu de la V0.5). On retient la touche physique : sur un clavier
## AZERTY, « haut » est Z par défaut, sur un QWERTY c'est W.
## Les flèches, le pavé numérique et la souris marchent toujours en plus.
## Une action peut aussi aller sur un bouton de la souris (sauf le clic gauche,
## qui sert à marcher et à viser) ou sur un bouton de manette (V0.5.3).

## [action, nom affiché, touche physique par défaut]
const ACTIONS := [
	["haut", "Avancer (haut)", KEY_W],
	["bas", "Reculer (bas)", KEY_S],
	["gauche", "Aller à gauche", KEY_A],
	["droite", "Aller à droite", KEY_D],
	["comp_1", "Compétence 1", KEY_1],
	["comp_2", "Compétence 2", KEY_2],
	["comp_3", "Compétence 3", KEY_3],
	["comp_4", "Compétence 4", KEY_4],
	["comp_5", "Compétence 5", KEY_5],
	["comp_6", "Compétence 6", KEY_6],
	["comp_7", "Compétence 7", KEY_7],
	["comp_8", "Compétence 8 (ultime)", KEY_8],
	["potion", "Potion de soin", KEY_R],
	["roulade", "Roulade", KEY_SPACE],
	["cible", "Cible suivante", KEY_TAB],
	["fiche", "Ta fiche de héros", KEY_C],
	["sac", "Ton sac", KEY_I],
	["rapide_1", "Raccourci 1 (provision)", KEY_F],
	["rapide_2", "Raccourci 2 (provision)", KEY_G],
	["ramasser", "Ramasser, récolter (le plus proche)", KEY_E],
	["saluer", "Saluer", KEY_H],
	["courir", "Courir (maintenir)", KEY_SHIFT],
	["garde", "Lever le bouclier (maintenir)", 0x4000002],   # SOURIS + clic droit
]

## Les boutons de souris et de manette sont rangés au-dessus des touches du clavier.
const SOURIS := 0x4000000
const MANETTE := 0x8000000

const NOMS_SOURIS := {MOUSE_BUTTON_RIGHT: "Clic droit", MOUSE_BUTTON_MIDDLE: "Clic molette",
	MOUSE_BUTTON_WHEEL_UP: "Molette ↑", MOUSE_BUTTON_WHEEL_DOWN: "Molette ↓",
	MOUSE_BUTTON_WHEEL_LEFT: "Molette ←", MOUSE_BUTTON_WHEEL_RIGHT: "Molette →",
	MOUSE_BUTTON_XBUTTON1: "Souris 4", MOUSE_BUTTON_XBUTTON2: "Souris 5"}
const NOMS_MANETTE := {JOY_BUTTON_A: "A", JOY_BUTTON_B: "B", JOY_BUTTON_X: "X", JOY_BUTTON_Y: "Y",
	JOY_BUTTON_BACK: "Select", JOY_BUTTON_GUIDE: "Guide", JOY_BUTTON_START: "Start",
	JOY_BUTTON_LEFT_STICK: "L3", JOY_BUTTON_RIGHT_STICK: "R3",
	JOY_BUTTON_LEFT_SHOULDER: "LB", JOY_BUTTON_RIGHT_SHOULDER: "RB",
	JOY_BUTTON_DPAD_UP: "↑", JOY_BUTTON_DPAD_DOWN: "↓", JOY_BUTTON_DPAD_LEFT: "←", JOY_BUTTON_DPAD_RIGHT: "→"}

static var _keys: Dictionary = {}


static func _load() -> void:
	if not _keys.is_empty():
		return
	var cfg := ConfigFile.new()
	cfg.load(WorldSky.SETTINGS)
	for a in ACTIONS:
		_keys[a[0]] = int(cfg.get_value("touches", a[0], a[2]))


## Le code d'un appui (touche, bouton de souris ou de manette), ou 0 s'il ne peut pas servir.
static func code_of(event: InputEvent) -> int:
	if event is InputEventKey:
		return event.physical_keycode if event.physical_keycode != KEY_NONE else event.keycode
	if event is InputEventMouseButton and event.button_index != MOUSE_BUTTON_LEFT:
		return SOURIS + event.button_index
	if event is InputEventJoypadButton:
		return MANETTE + event.button_index
	return 0


## L'action d'un appui, ou "".
static func action_for_event(event: InputEvent) -> String:
	var c := code_of(event)
	return action_for(c) if c != 0 else ""


## L'action est-elle maintenue en ce moment ?
static func pressed(action: String) -> bool:
	var c := key(action)
	if c == KEY_NONE:
		return false
	if c >= MANETTE:
		for j in Input.get_connected_joypads():
			if Input.is_joy_button_pressed(j, c - MANETTE):
				return true
		return false
	if c >= SOURIS:
		return Input.is_mouse_button_pressed(c - SOURIS)
	return Input.is_physical_key_pressed(c)


## La touche (ou le bouton) de l'action.
static func key(action: String) -> int:
	_load()
	return int(_keys.get(action, KEY_NONE))


## L'action d'une touche physique, ou "" si aucune.
static func action_for(physical: int) -> String:
	_load()
	for a in ACTIONS:
		if int(_keys[a[0]]) == physical:
			return a[0]
	return ""


## Change une touche. Si une autre action l'avait, elles échangent leurs touches.
static func set_key(action: String, physical: int) -> void:
	_load()
	var other := action_for(physical)
	if other != "" and other != action:
		_keys[other] = _keys[action]
	_keys[action] = physical
	_save()


static func reset() -> void:
	_keys.clear()
	var cfg := ConfigFile.new()
	cfg.load(WorldSky.SETTINGS)
	if cfg.has_section("touches"):
		cfg.erase_section("touches")
	cfg.save(WorldSky.SETTINGS)
	_load()


static func _save() -> void:
	var cfg := ConfigFile.new()
	cfg.load(WorldSky.SETTINGS)
	for a in ACTIONS:
		cfg.set_value("touches", a[0], _keys[a[0]])
	cfg.save(WorldSky.SETTINGS)


## Le nom de la touche sur le clavier du joueur (« Z » sur un AZERTY).
## court : pour les petits boutons de la barre de combat (« S4 » au lieu de « Souris 4 »).
static func label(action: String, court := false) -> String:
	var k := key(action)
	if k >= MANETTE:
		return ("🎮" if court else "🎮 ") + str(NOMS_MANETTE.get(k - MANETTE, k - MANETTE))
	if k >= SOURIS:
		var nom := str(NOMS_SOURIS.get(k - SOURIS, "Souris %d" % (k - SOURIS)))
		if court:
			nom = nom.replace("Souris ", "S").replace("Clic droit", "Clic D").replace("Clic molette", "Mol.").replace("Molette ", "Mol")
		return nom
	match k:
		KEY_SPACE:
			return "Esp."
		KEY_SHIFT:
			return "Maj"
		KEY_TAB:
			return "Tab"
	if k >= KEY_0 and k <= KEY_9:
		return str(k - KEY_0)   # la rangée des chiffres (sur AZERTY, sans Maj)
	var shown := DisplayServer.keyboard_get_keycode_from_physical(k)
	var s := OS.get_keycode_string(shown if shown != KEY_NONE else k)
	return s if s != "" else "?"
