class_name Pawn
extends Node2D
## Quelqu'un dans le monde : un héros (son allure en couches, HeroSprite), un
## habitant (identifiant `npc`) ou un monstre (sorte `kind`, MonsterSprite).
## Avec son ombre, son nom, ses bulles de chat, sa barre de vie et ses effets.

const FX_COLORS := [Color("f08a3c"), Color("7cc45a"), Color("f2d94e"), Color("6fa8e8"), Color("a070d8"), Color("e8c66a")]
const FX_BRULURE := 1
const FX_POISON := 2
const FX_ETOURDI := 4
const FX_RALENTI := 8
const FX_FAIBLESSE := 16
const FX_AURA := 32
const FX_GARDE := 64
const FX_INCANTE := 128
const FX_PHASE2 := 256

var look := {}
var hero_name := ""
var dir := 2
var is_me := false
var walking := false
var running := false
var npc := ""        # identifiant d'habitant, vide pour un joueur
var kind := ""       # sorte de monstre, vide pour un héros ou un habitant
var elite := false
var stature := 1.0   # les enfants sont plus petits

# Le combat.
var level := 1
var hp := 1
var hp_max := 1
var fx := 0
var dead := false
var targeted := false

var sprite: HeroSprite
var mon: MonsterSprite
var torch: PointLight2D   # allumée par le ciel quand il fait nuit
var _name_label: Label
var _bubble: PanelContainer
var _bubble_label: Label
var _bubble_time := 0.0
var _hp_shown := 1.0
var _flash := 0.0
var _t := 0.0
var _jitter := 0.0
var _show_bar := 0.0   # secondes pendant lesquelles la barre reste visible


func _ready() -> void:
	if kind != "":
		mon = MonsterSprite.new()
		add_child(mon)
		mon.setup(kind)
		mon.dir = dir
		if kind == "grand_sanglier" and fx & FX_PHASE2:
			mon.set_kind("grand_sanglier_efface")
	else:
		sprite = HeroSprite.new()
		add_child(sprite)
		sprite.dress(look)
		sprite.dir = dir
		sprite.scale = Vector2(stature, stature)
	_hp_shown = _pct()

	_name_label = Label.new()
	_name_label.add_theme_font_size_override("font_size", 15)
	var color := UI.LUMEN if is_me else (UI.MOSS if npc != "" else UI.PARCH)
	if kind != "":
		color = Color("f2c14e") if elite else (Color("d8d6e0") if kind.ends_with("_efface") else Color("f0a08a"))
	_name_label.add_theme_color_override("font_color", color)
	_name_label.add_theme_color_override("font_outline_color", Color(0, 0, 0, 0.85))
	_name_label.add_theme_constant_override("outline_size", 5)
	_name_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	add_child(_name_label)
	_refresh_name()

	_bubble = PanelContainer.new()
	_bubble.add_theme_stylebox_override("panel", UI.box(Color(UI.PARCH, 0.95), Color(UI.NIGHT, 0.4), 10, 1, 6))
	_bubble_label = Label.new()
	_bubble_label.add_theme_font_size_override("font_size", 15)
	_bubble_label.add_theme_color_override("font_color", UI.NIGHT)
	_bubble.add_child(_bubble_label)
	_bubble.visible = false
	add_child(_bubble)
	if dead:
		die()


func _head() -> float:
	match kind:
		"":
			return 74.0 * stature
		"corbeau_efface":
			return 56.0
		"grand_sanglier":
			return 112.0
	return 44.0


func _refresh_name() -> void:
	var t := hero_name
	if kind != "" or (npc == "" and level > 1):
		t = "%s · niv. %d" % [hero_name, level]
	if elite and kind != "":
		t = "✦ " + t
	_name_label.text = t
	_name_label.reset_size()
	_name_label.position = Vector2(-_name_label.size.x / 2, -_head())


func _pct() -> float:
	return clampf(float(hp) / maxf(1.0, float(hp_max)), 0.0, 1.0)


## Le serveur annonce la vie, les effets et le niveau.
func set_vitals(new_hp: int, new_max: int, new_fx: int, new_level: int) -> void:
	if new_hp < hp:
		_show_bar = 6.0
	hp = new_hp
	hp_max = new_max
	fx = new_fx
	if mon != null and kind == "grand_sanglier":
		mon.set_kind("grand_sanglier_efface" if fx & FX_PHASE2 else "grand_sanglier")
	if new_level != level:
		level = new_level
		_refresh_name()
	if sprite != null and not dead:
		if fx & FX_GARDE:
			sprite.stance = "garde"
		elif fx & FX_INCANTE:
			sprite.stance = "incante"
		else:
			sprite.stance = ""


## Un coup reçu : un éclair blanc sur le corps.
func flash() -> void:
	_flash = 0.12
	_show_bar = 6.0


func play(anim: String) -> void:
	if mon != null:
		mon.play_once(anim)
	elif sprite != null:
		sprite.play_once(anim)


func die() -> void:
	dead = true
	hp = 0
	fx = 0
	if mon != null:
		mon.die()
		var tw := create_tween()
		tw.tween_interval(2.2)
		tw.tween_property(self, "modulate:a", 0.0, 0.7)
	elif sprite != null:
		sprite.stance = "mort"


func revive() -> void:
	dead = false
	modulate.a = 1.0
	if sprite != null:
		sprite.stance = ""
	if mon != null:
		mon.revive()


func say(text: String) -> void:
	_bubble_label.text = _wrap(text, 26)
	_bubble.visible = true
	_bubble.modulate.a = 1.0
	_bubble.size = Vector2.ZERO
	_bubble.reset_size()
	_bubble_time = 6.0


## Coupe un texte en lignes d'environ `width` caractères, entre les mots.
static func _wrap(text: String, width: int) -> String:
	var lines: Array[String] = []
	var line := ""
	for word in text.split(" ", false):
		if line != "" and line.length() + 1 + word.length() > width:
			lines.append(line)
			line = word
		else:
			line = word if line == "" else line + " " + word
	if line != "":
		lines.append(line)
	return "\n".join(lines)


func _process(delta: float) -> void:
	_t += delta
	var body: Node2D = mon if mon != null else sprite
	if mon != null:
		mon.dir = dir
		mon.set_motion(walking, running)
		# Les Effacés tremblent, comme si le monde hésitait à les dessiner.
		if kind.ends_with("_efface") or fx & FX_PHASE2:
			_jitter -= delta
			if _jitter <= 0:
				_jitter = randf_range(0.05, 0.35)
				mon.position = Vector2(randi_range(-1, 1), 0) if randf() < 0.5 else Vector2.ZERO
				mon.modulate.a = randf_range(0.75, 1.0)
	else:
		sprite.dir = dir
		sprite.set_motion(walking and not dead, running)
	if _flash > 0:
		_flash -= delta
		body.self_modulate = Color(2.2, 2.2, 2.2) if _flash > 0 else Color.WHITE
	_show_bar = maxf(0.0, _show_bar - delta)
	_hp_shown = move_toward(_hp_shown, _pct(), delta * 1.5)
	if _bubble.visible:
		_bubble_time -= delta
		_bubble.position = Vector2(-_bubble.size.x / 2, -_head() - 14 - _bubble.size.y)
		if _bubble_time < 1.0:
			_bubble.modulate.a = max(0.0, _bubble_time)
		if _bubble_time <= 0:
			_bubble.visible = false
	queue_redraw()


func _draw() -> void:
	# Ombre au sol, et un anneau sous ton propre héros ou ta cible.
	draw_set_transform(Vector2.ZERO, 0, Vector2(1, 0.5))
	var shadow := 18.0 if kind != "" else 14.0
	if kind == "corbeau_efface":
		shadow = 9.0
	elif kind == "grand_sanglier":
		shadow = 40.0
	draw_circle(Vector2.ZERO, shadow, Color(0, 0, 0, 0.32 if not dead else 0.15))
	if is_me:
		draw_arc(Vector2.ZERO, 19, 0, TAU, 40, Color(UI.LUMEN, 0.8), 2.0, true)
	if targeted and not dead:
		var c := UI.DANGER if kind != "" else UI.MOSS
		var r := shadow + 8 + sin(_t * 6) * 1.5
		draw_arc(Vector2.ZERO, r, 0, TAU, 48, Color(c, 0.95), 2.5, true)
		draw_arc(Vector2.ZERO, r + 4, 0, TAU, 48, Color(c, 0.35), 1.5, true)
	if fx & FX_AURA:
		var k := 0.6 + 0.4 * sin(_t * 3)
		draw_arc(Vector2.ZERO, 26, 0, TAU, 48, Color(UI.LUMEN, 0.55 * k), 3.0, true)
	draw_set_transform(Vector2.ZERO, 0, Vector2.ONE)
	if elite and kind != "" and not dead:
		# Le Doré scintille.
		for i in 5:
			var a := _t * 1.7 + i * TAU / 5
			var p := Vector2(cos(a) * 20, -18 + sin(a * 1.3) * 12)
			draw_circle(p, 1.5 + sin(_t * 8 + i) * 0.8, Color(1, 0.9, 0.5, 0.9))
	if dead:
		return
	# Barre de vie : toujours sur les monstres touchés ou visés, sur les héros blessés.
	var hurt := hp < hp_max
	if npc == "" and (targeted or _show_bar > 0 or (hurt and kind == "") or (kind != "" and hurt)):
		var w := 44.0 if kind != "" else 36.0
		if kind == "grand_sanglier":
			w = 90.0
		var y := -_head() - 8
		var r := Rect2(-w / 2, y, w, 5)
		draw_rect(r.grow(1), Color(0, 0, 0, 0.75))
		draw_rect(Rect2(r.position, Vector2(w * _hp_shown, 5)), Color(1, 0.9, 0.6, 0.9))
		var col := UI.DANGER if kind != "" else UI.MOSS
		draw_rect(Rect2(r.position, Vector2(w * _pct(), 5)), col)
	# Les effets : une rangée de petites pastilles.
	var dots: Array[int] = []
	for e in 5:
		if fx & (1 << e):
			dots.append(e)
	var x0 := -float(dots.size() - 1) * 5.0
	for i in dots.size():
		var p := Vector2(x0 + i * 10, -_head() - 16)
		draw_circle(p, 4, Color(0, 0, 0, 0.7))
		draw_circle(p, 3, FX_COLORS[dots[i]])
	if fx & FX_ETOURDI:
		for i in 3:
			var a := _t * 5 + i * TAU / 3
			draw_circle(Vector2(cos(a) * 12, -_head() - 24 + sin(a) * 3), 2.2, Color("f2d94e"))
