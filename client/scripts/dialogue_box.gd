class_name DialogueBox
extends PanelContainer
## La boîte de dialogue des habitants : un portrait, le nom, le texte, et des
## choix à cliquer (ou les touches 1 à 4, Échap pour partir).
##
## Les dialogues viennent de res://data/pnj/<carte>.json : chaque habitant a
## des nœuds { texte, choix: [{ texte, vers }] } ; « vers » vide = au revoir.

signal closed

const DATA := "res://data/pnj/%s.json"
const MEMOIRE := "user://habitants.cfg"   # ce dont les habitants se souviennent

static var _data := {}

var hero_name := ""
var npc := ""
var _portrait: HeroSprite
var _name: Label
var _text: Label
var _choices: VBoxContainer
var _keys: Array[String] = []
var _tree := {}
var _sizes: Array = []


## Les habitants d'une carte (tenues, bulles, dialogues). Vide si la carte n'en a pas.
static func data(map_id: String) -> Dictionary:
	if not _data.has(map_id):
		var path := DATA % map_id
		_data[map_id] = JSON.parse_string(FileAccess.get_file_as_string(path)) if FileAccess.file_exists(path) else {}
	return _data[map_id]


static func info(map_id: String, id: String) -> Dictionary:
	return data(map_id).get("pnj", {}).get(id, {})


func _ready() -> void:
	add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.94), UI.LUMEN, 14, 2, 16))
	custom_minimum_size = Vector2(640, 0)
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 16)
	add_child(row)

	# Le portrait : l'habitant lui-même, en grand, dans un médaillon.
	var frame := PanelContainer.new()
	frame.add_theme_stylebox_override("panel", UI.box(UI.STONE_2, UI.LINE, 60, 2, 0))
	frame.custom_minimum_size = Vector2(120, 120)
	frame.clip_contents = true
	frame.size_flags_vertical = Control.SIZE_SHRINK_BEGIN
	var holder := Control.new()
	frame.add_child(holder)
	_portrait = HeroSprite.new()
	_portrait.dir = 1
	_portrait.scale = Vector2(3, 3)
	_portrait.position = Vector2(60, 168)
	holder.add_child(_portrait)
	row.add_child(frame)

	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 8)
	col.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	row.add_child(col)
	_name = UI.label("", 20, UI.LUMEN)
	_name.autowrap_mode = TextServer.AUTOWRAP_OFF
	col.add_child(_name)
	_text = UI.label("", 17)
	_text.custom_minimum_size = Vector2(460, 0)
	col.add_child(_text)
	_choices = VBoxContainer.new()
	_choices.add_theme_constant_override("separation", 6)
	col.add_child(_choices)


## Ouvre la conversation avec l'habitant `id` de la carte `map_id`.
func open(map_id: String, id: String, display_name: String) -> void:
	var who := info(map_id, id)
	if who.is_empty():
		return
	npc = id
	_portrait.dress(who.get("look", {}))
	_name.text = "%s, %s" % [display_name, who.get("titre", "")] if who.has("titre") else display_name
	_tree = who.get("dialogue", {})
	_sizes = data(map_id).get("sanglier", [])
	visible = true
	_show("debut")


func _show(key: String) -> void:
	if key == "" or not _tree.has(key):
		close()
		return
	var node: Dictionary = _tree[key]
	_text.text = _fill(node.get("texte", ""), key)
	for c in _choices.get_children():
		c.queue_free()
	_keys.clear()
	var i := 0
	for ch in node.get("choix", []):
		i += 1
		var b := UI.button("%d. %s" % [i, ch.texte])
		b.alignment = HORIZONTAL_ALIGNMENT_LEFT
		var to: String = ch.get("vers", "")
		b.pressed.connect(_show.bind(to))
		_choices.add_child(b)
		_keys.append(to)
	if _keys.is_empty():
		var b := UI.button("Au revoir")
		b.pressed.connect(close)
		_choices.add_child(b)
		_keys.append("")


## Remplace {nom} par le nom du héros, et {sanglier} par la taille du
## sanglier de Mabon, qui grandit à chaque fois qu'il raconte l'histoire.
func _fill(text: String, key: String) -> String:
	text = text.replace("{nom}", hero_name)
	if text.contains("{sanglier}") and not _sizes.is_empty():
		var cfg := ConfigFile.new()
		cfg.load(MEMOIRE)
		var n := int(cfg.get_value(hero_name, "sanglier", 0))
		text = text.replace("{sanglier}", _sizes[mini(n, _sizes.size() - 1)])
		if key == "sanglier":
			cfg.set_value(hero_name, "sanglier", n + 1)
			cfg.save(MEMOIRE)
	return text


func close() -> void:
	if not visible:
		return
	visible = false
	npc = ""
	closed.emit()


func _unhandled_key_input(event: InputEvent) -> void:
	if not visible or not event.pressed or event.echo:
		return
	if event.keycode == KEY_ESCAPE:
		close()
		get_viewport().set_input_as_handled()
		return
	var k: int = event.keycode - KEY_1
	if event.keycode >= KEY_KP_1 and event.keycode <= KEY_KP_9:
		k = event.keycode - KEY_KP_1
	if k >= 0 and k < _keys.size():
		_show(_keys[k])
		get_viewport().set_input_as_handled()
