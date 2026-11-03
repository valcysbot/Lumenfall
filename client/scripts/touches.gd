class_name Touches
## Les touches du clavier, à changer dans ⚙ Réglages → ⌨ Touches (retour de la
## soirée de jeu de la V0.5). On retient la touche physique : sur un clavier
## AZERTY, « haut » est Z par défaut, sur un QWERTY c'est W.
## Les flèches, le pavé numérique et la souris marchent toujours en plus.

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
	["saluer", "Saluer", KEY_H],
	["courir", "Courir (maintenir)", KEY_SHIFT],
]

static var _keys: Dictionary = {}


static func _load() -> void:
	if not _keys.is_empty():
		return
	var cfg := ConfigFile.new()
	cfg.load(WorldSky.SETTINGS)
	for a in ACTIONS:
		_keys[a[0]] = int(cfg.get_value("touches", a[0], a[2]))


## La touche physique de l'action.
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
static func label(action: String) -> String:
	var k := key(action)
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
