class_name PeupleSprite
extends Node2D
## Un héros de l'Acte IV : une pile de planches dessinées par
## tools/sprites/forge_peuples.py. Les planches ne portent pas de couleurs,
## seulement un numéro de matériau et un ton par pixel ; un petit shader les
## remplit avec les couleurs choisies au miroir (peau, yeux, cheveux, tunique…).
## Ainsi 116 planches suffisent pour toutes les allures de tous les peuples.
##
## Grille : 32 images par ligne ; chaque bloc de 8 lignes (une par direction)
## contient 32 images, dans l'ordre des animations.

const DIR := "res://assets/heros/peuples/"
const SHADER := """
shader_type canvas_item;
uniform sampler2D palette : filter_nearest;
varying vec4 teinte;
void vertex() { teinte = COLOR; }
void fragment() {
	vec4 t = texture(TEXTURE, UV);
	if (t.a < 0.5) { discard; }
	int idx = int(round(t.r * 255.0 / 6.0));
	int slot = idx / 5;
	vec4 c = texelFetch(palette, ivec2(idx - slot * 5, slot), 0);
	COLOR = c * teinte;
}
"""

static var _meta: Dictionary = {}
static var _textures: Dictionary = {}
static var _palettes: Dictionary = {}
static var _shader: Shader

var dir := 1
var anim := "repos"
var stance := ""
var _t := 0.0
var _once := ""
var _t_once := 0.0
var _layers: Array[Sprite2D] = []
var _dos: Array[Sprite2D] = []   # cape, queue, ailes, nageoires : dans le dos
var _devant := 0                 # où les glisser quand le héros nous tourne le dos
var _de_dos := false


## Les directions où l'on voit le héros de dos (il marche vers le haut de l'écran).
const DE_DOS := [4, 5, 6]
const PIECES_DU_DOS := ["cape_", "trait_queue", "trait_ailes", "trait_nageoires"]


static func meta() -> Dictionary:
	if _meta.is_empty():
		_meta = JSON.parse_string(FileAccess.get_file_as_string(DIR + "planches.json"))
	return _meta


static func has_piece(piece: String) -> bool:
	return meta().pieces.has(piece)


static func _texture(piece: String) -> Texture2D:
	if not _textures.has(piece):
		_textures[piece] = load(DIR + piece + ".png")
	return _textures[piece]


## La palette d'une pièce : une ligne de 5 tons par matériau.
static func _palette(piece: String, colors: Dictionary) -> ImageTexture:
	var slots: Array = meta().pieces[piece]
	var D := Heros.peuples()
	var key := piece
	var hexas := []
	for s in slots:
		var hexa: String = colors.get(s, D.defauts.get(s, "#ff00ff"))
		hexas.append(hexa)
		key += hexa
	if _palettes.has(key):
		return _palettes[key]
	var img := Image.create(5, maxi(1, slots.size()), false, Image.FORMAT_RGBA8)
	for i in slots.size():
		var r := Heros.rampe(hexas[i], float(D.alpha.get(slots[i], 1.0)))
		for t in 5:
			img.set_pixel(t, i, r[t])
	var tex := ImageTexture.create_from_image(img)
	_palettes[key] = tex
	return tex


## Habille le héros : couches = [[pièce, {matériau: couleur}], …] (Heros.couches).
func dress(couches: Array) -> void:
	for l in _layers:
		l.queue_free()
	_layers.clear()
	_dos.clear()
	_devant = 0
	_de_dos = false
	if _shader == null:
		_shader = Shader.new()
		_shader.code = SHADER
	var m := meta()
	var visage := false
	for c in couches:
		var piece: String = c[0]
		if not m.pieces.has(piece):
			continue
		var s := Sprite2D.new()
		s.texture = _texture(piece)
		s.texture_filter = CanvasItem.TEXTURE_FILTER_NEAREST
		s.centered = false
		s.region_enabled = true
		s.offset = -Vector2(m.ancre[0], m.ancre[1])
		var mat := ShaderMaterial.new()
		mat.shader = _shader
		mat.set_shader_parameter("palette", _palette(piece, c[1]))
		s.material = mat
		add_child(s)
		_layers.append(s)
		if PIECES_DU_DOS.any(func(p): return piece.begins_with(p)):
			_dos.append(s)
		elif piece.begins_with("visage_"):
			visage = true
		elif not visage:
			# Juste après le corps et les habits, sous le visage et les cheveux.
			_devant = _layers.size()
	_apply()


## Vu de dos, la cape et la queue passent devant le corps ; sinon derrière.
func _range_le_dos() -> void:
	var de_dos := DE_DOS.has(dir)
	if de_dos == _de_dos or _dos.is_empty():
		return
	_de_dos = de_dos
	if de_dos:
		for l in _dos:
			move_child(l, _devant - 1)
	else:
		for i in range(_dos.size() - 1, -1, -1):
			move_child(_dos[i], 0)


func play_once(which: String) -> void:
	if stance == "mort" or not meta().anims.has(which):
		return
	_once = which
	_t_once = 0.0


func set_motion(moving: bool, running: bool) -> void:
	var want := "repos"
	if moving:
		want = "course" if running else "marche"
	if stance == "mort" or (stance != "" and not moving):
		want = stance
	if (moving and _once == "salut") or want == "mort":
		_once = ""
	if want != anim:
		anim = want
		_t = 0.0


func _process(delta: float) -> void:
	_t += delta
	_t_once += delta
	_apply()


func _apply() -> void:
	if _layers.is_empty():
		return
	_range_le_dos()
	var m := meta()
	var which := _once if _once != "" else anim
	if not m.anims.has(which):
		which = "repos"
	var a: Dictionary = m.anims[which]
	var n := int(a.images)
	var f := int((_t_once if _once != "" else _t) * float(a.fps))
	if f >= n:
		if _once != "":
			_once = ""
			_apply()
			return
		f = f % n if a.boucle else n - 1
	var col := int(a.debut) + f
	var w: float = m.image[0]
	var h: float = m.image[1]
	var cols := int(m.colonnes)
	var r := Rect2((col % cols) * w, ((col / cols) * 8 + clampi(dir, 0, 7)) * h, w, h)
	for l in _layers:
		l.region_rect = r
