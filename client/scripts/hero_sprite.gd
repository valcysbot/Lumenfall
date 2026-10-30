class_name HeroSprite
extends Node2D
## Le héros en couches : une pile de planches dessinées par
## tools/sprites/forge_heros.py (corps, pantalon, bottes, tunique, gants, cape,
## cheveux, casque, épée, et pour les habitants robe, tablier, barbe, chapeau). Toutes les planches ont la même grille : une ligne
## par direction, une colonne par image d'animation.

const DIR := "res://assets/heros/humain/"
const ORDRE := ["corps", "pantalon", "bottes", "robe", "tunique", "tablier", "gants", "cape", "barbe",
		"cheveux", "chapeau", "casque", "epee", "bouclier"]
const STYLES_CHEVEUX := ["court", "long"]
const COULEURS_CHEVEUX := ["noir", "brun", "chatain", "blond", "roux", "blanc"]

static var _meta: Dictionary = {}
static var _textures: Dictionary = {}

var dir := 1
var anim := "repos"
## Posture tenue tant qu'on ne bouge pas : "garde" (bouclier levé),
## "incante" (sort en cours), "mort" (à terre, même si l'on glisse).
var stance := ""
var _t := 0.0
var _once := ""
var _t_once := 0.0
var _layers: Array[Sprite2D] = []


## Tenue par défaut d'un héros, en attendant l'équipement (Acte V) et la
## personnalisation (Acte IV) : la tunique prend la couleur choisie à la
## création, les cheveux sont tirés du nom.
static func look_for(hero_name: String, color: int) -> Dictionary:
	var h := absi(hero_name.to_lower().hash())
	return {
		"tunique": str(clampi(color, 0, 11)),
		"cheveux": "%s_%s" % [STYLES_CHEVEUX[h % 2], COULEURS_CHEVEUX[(h / 2) % COULEURS_CHEVEUX.size()]],
		"corps": "", "pantalon": "", "bottes": "", "gants": "", "cape": "", "epee": "", "bouclier": "",
	}


static func meta() -> Dictionary:
	if _meta.is_empty():
		var f := FileAccess.open(DIR + "humain.json", FileAccess.READ)
		_meta = JSON.parse_string(f.get_as_text())
	return _meta


static func _texture(file: String) -> Texture2D:
	if not _textures.has(file):
		_textures[file] = load(DIR + file + ".png")
	return _textures[file]


## Habille le héros. `look` associe une pièce à sa variante ("" = sans variante).
## Une pièce absente du dictionnaire n'est pas portée.
func dress(look: Dictionary) -> void:
	for l in _layers:
		l.queue_free()
	_layers.clear()
	var m := meta()
	for piece in ORDRE:
		if not look.has(piece):
			continue
		var v: String = look[piece]
		var file: String = piece
		if piece == "cheveux":
			file = "cheveux_" + v
		elif v != "":
			file = piece + "_" + v
		var s := Sprite2D.new()
		s.texture = _texture(file)
		s.texture_filter = CanvasItem.TEXTURE_FILTER_NEAREST
		s.centered = false
		s.region_enabled = true
		s.offset = -Vector2(m.ancre[0], m.ancre[1])
		add_child(s)
		_layers.append(s)
	_apply()


## Joue une animation une seule fois (ex. « salut », « attaque »), puis
## revient à la normale.
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
	# Se mettre en marche interrompt le salut, pas un coup d'épée ni une roulade.
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
	var m := meta()
	var which := _once if _once != "" else anim
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
	var r := Rect2(col * w, clampi(dir, 0, 7) * h, w, h)
	for l in _layers:
		l.region_rect = r
