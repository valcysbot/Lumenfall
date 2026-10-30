class_name MonsterSprite
extends Node2D
## Un monstre en pixel art : une seule planche par sorte, dessinée par
## tools/sprites/forge_betes.py (une ligne par direction, une colonne par image).

const DIR := "res://assets/betes/"

static var _meta: Dictionary = {}
static var _textures: Dictionary = {}

var dir := 1
var anim := "repos"
var _t := 0.0
var _once := ""
var _dead := false
var _sprite: Sprite2D
var _size := Vector2(64, 48)
var _kind := ""


static func meta() -> Dictionary:
	if _meta.is_empty():
		_meta = JSON.parse_string(FileAccess.get_file_as_string(DIR + "betes.json"))
	return _meta


static func has_kind(kind: String) -> bool:
	return ResourceLoader.exists(DIR + kind + ".png")


func setup(kind: String) -> void:
	_sprite = Sprite2D.new()
	_sprite.texture_filter = CanvasItem.TEXTURE_FILTER_NEAREST
	_sprite.centered = false
	_sprite.region_enabled = true
	add_child(_sprite)
	set_kind(kind)


## Change de planche (le Grand Sanglier devient un Effacé en phase 2).
func set_kind(kind: String) -> void:
	if kind == _kind:
		return
	_kind = kind
	if not _textures.has(kind):
		_textures[kind] = load(DIR + kind + ".png")
	var m := meta()
	var t: Dictionary = m.get("tailles", {}).get(kind, m)
	_size = Vector2(t.image[0], t.image[1])
	_sprite.texture = _textures[kind]
	_sprite.offset = -Vector2(t.ancre[0], t.ancre[1])
	_apply()


func play_once(which: String) -> void:
	if _dead or not meta().anims.has(which):
		return
	_once = which
	_t = 0.0


func set_motion(moving: bool, running: bool) -> void:
	if _dead:
		return
	var want := "repos"
	if moving:
		want = "course" if running else "marche"
	if want != anim:
		anim = want
		if _once == "":
			_t = 0.0


## Le monstre tombe et reste à terre.
func die() -> void:
	_dead = true
	_once = ""
	anim = "mort"
	_t = 0.0


func revive() -> void:
	_dead = false
	anim = "repos"


func _process(delta: float) -> void:
	_t += delta
	_apply()


func _apply() -> void:
	if _sprite == null:
		return
	var m := meta()
	var which := _once if _once != "" else anim
	var a: Dictionary = m.anims[which]
	var n := int(a.images)
	var f := int(_t * float(a.fps))
	if f >= n:
		if _once != "":
			_once = ""
			_t = 0.0
			_apply()
			return
		f = f % n if a.boucle else n - 1
	var w := _size.x
	var h := _size.y
	_sprite.region_rect = Rect2((int(a.debut) + f) * w, clampi(dir, 0, 7) * h, w, h)
