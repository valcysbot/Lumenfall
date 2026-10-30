class_name WorldSky
extends Node2D
## Le ciel : la lumière du jour et de la nuit, les torches, et la météo
## (pluie, brouillard, orage). L'heure vient du serveur, la même pour tous ;
## la météo aussi, région par région.

signal changed(text: String)

const SETTINGS := "user://reglages.cfg"
const MAX_TORCHES := 10   # les téléphones n'aiment pas les lumières trop nombreuses

# Couleur de la lumière au fil de la journée (0 = lever du soleil, 1 = lever suivant).
const KEYS := [
	[0.00, Color(1.00, 0.80, 0.70)],
	[0.06, Color(1, 1, 1)],
	[0.58, Color(1, 1, 1)],
	[0.635, Color(1.00, 0.74, 0.62)],
	[0.675, Color(0.52, 0.46, 0.72)],
	[0.72, Color(0, 0, 0, 0)],   # nuit : remplacée par la couleur réglable
	[0.955, Color(0, 0, 0, 0)],
	[0.985, Color(0.55, 0.48, 0.66)],
	[1.00, Color(1.00, 0.80, 0.70)],
]

var day_ms := 7200000.0
var daylight_ms := 4800000.0
var weather := "clair"
var night_brightness := 0.35     # réglage du joueur, de 0 (noir d'encre) à 1

var _offset_ms := 0.0            # heure du serveur - heure locale
var _forced_phase := -1.0
var _modulate: CanvasModulate
var _weather_layer: CanvasLayer
var _rain: CPUParticles2D
var _fog: ColorRect
var _flash: ColorRect
var _flash_wait := 8.0
var _torch_texture: Texture2D
var _night := 0.0                # 0 en plein jour, 1 en pleine nuit
var lamps: Array = []            # lumières des décors (lanternes, forge…)
## À l'intérieur d'une maison : lumière chaude, ni pluie ni brouillard.
var indoors := false:
	set(v):
		if v != indoors:
			indoors = v
			set_weather(weather)


func _ready() -> void:
	_load_settings()
	_modulate = CanvasModulate.new()
	add_child(_modulate)
	_torch_texture = _make_torch_texture()

	_weather_layer = CanvasLayer.new()
	_weather_layer.layer = 1
	add_child(_weather_layer)
	_fog = ColorRect.new()
	_fog.set_anchors_preset(Control.PRESET_FULL_RECT)
	_fog.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_fog.material = _fog_material()
	_fog.visible = false
	_weather_layer.add_child(_fog)
	_rain = _make_rain()
	_weather_layer.add_child(_rain)
	_flash = ColorRect.new()
	_flash.set_anchors_preset(Control.PRESET_FULL_RECT)
	_flash.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_flash.color = Color(0.85, 0.9, 1.0, 0.0)
	_weather_layer.add_child(_flash)

	# Mode capture : on peut forcer l'heure et la météo pour les captures d'écran.
	var args := OS.get_cmdline_user_args()
	if "--capture=world" in args:
		for a in args:
			if a.begins_with("--heure="):
				_forced_phase = fposmod((float(a.split("=")[1]) - 6.0) / 24.0, 1.0)
			elif a.begins_with("--meteo="):
				set_weather(a.split("=")[1])
	get_viewport().size_changed.connect(_place_rain)
	_place_rain()


## Reçu à l'arrivée dans le monde : {"now", "day", "jour", "w"}.
func set_sky(d: Dictionary) -> void:
	_offset_ms = float(d.get("now", 0)) - Time.get_unix_time_from_system() * 1000.0
	day_ms = float(d.get("day", day_ms))
	daylight_ms = float(d.get("jour", daylight_ms))
	if _forced_phase < 0 or not ("--capture=world" in OS.get_cmdline_user_args()):
		set_weather(d.get("w", "clair"))


func set_weather(kind: String) -> void:
	weather = kind
	if indoors:
		kind = "clair"
	_rain.emitting = kind == "pluie" or kind == "orage"
	_rain.amount = 520 if kind == "orage" else 300
	_fog.visible = kind == "brouillard"
	_flash_wait = randf_range(3.0, 7.0)
	changed.emit(describe())


## Avancée de la journée, de 0 (lever) à 1.
func phase() -> float:
	if _forced_phase >= 0:
		return _forced_phase
	var ms := Time.get_unix_time_from_system() * 1000.0 + _offset_ms
	return fposmod(ms, day_ms) / day_ms


## Heure d'Erydor (0 à 24) : le soleil se lève à 6 h et se couche à 22 h.
func hour() -> float:
	return fposmod(6.0 + phase() * 24.0, 24.0)


func is_night() -> bool:
	return phase() >= daylight_ms / day_ms


func describe() -> String:
	var h := hour()
	var icon := "🌙" if is_night() else ("🌅" if h < 8 or h >= 20 else "☀️")
	var w: String = {"pluie": " · 🌧️ pluie", "brouillard": " · 🌫️ brouillard", "orage": " · ⛈️ orage"}.get(weather, "")
	return "%s %d h%s" % [icon, int(h), w]


func night_color() -> Color:
	return Color(0.05, 0.06, 0.14).lerp(Color(0.34, 0.36, 0.55), night_brightness)


func set_night_brightness(v: float) -> void:
	night_brightness = clampf(v, 0.0, 1.0)
	var cfg := ConfigFile.new()
	cfg.load(SETTINGS)
	cfg.set_value("affichage", "luminosite_nuit", night_brightness)
	cfg.save(SETTINGS)


func _load_settings() -> void:
	var cfg := ConfigFile.new()
	if cfg.load(SETTINGS) == OK:
		night_brightness = float(cfg.get_value("affichage", "luminosite_nuit", night_brightness))


func light_color() -> Color:
	if indoors:
		return Color(1.0, 0.92, 0.8)
	var p := phase()
	var c := Color.WHITE
	for i in KEYS.size() - 1:
		var a = KEYS[i]
		var b = KEYS[i + 1]
		if p >= a[0] and p <= b[0]:
			var ca: Color = night_color() if a[1].a == 0 else a[1]
			var cb: Color = night_color() if b[1].a == 0 else b[1]
			c = ca.lerp(cb, (p - a[0]) / maxf(0.0001, b[0] - a[0]))
			break
	match weather:
		"pluie": c = c * Color(0.78, 0.8, 0.88)
		"orage": c = c * Color(0.6, 0.62, 0.74)
		"brouillard": c = c.lerp(Color(0.75, 0.78, 0.82) * c.v, 0.35)
	c.a = 1.0
	return c


func _process(delta: float) -> void:
	var c := light_color()
	_modulate.color = c
	# Les torches s'allument d'elles-mêmes quand il fait sombre.
	_night = clampf((0.8 - c.v) / 0.45, 0.0, 1.0)
	update_lamps()
	if weather == "orage" and not indoors:
		_flash_wait -= delta
		if _flash_wait <= 0:
			_flash_wait = randf_range(6.0, 15.0)
			var tw := create_tween()
			tw.tween_property(_flash, "color:a", 0.55, 0.05)
			tw.tween_property(_flash, "color:a", 0.0, 0.12)
			tw.tween_property(_flash, "color:a", 0.35, 0.05)
			tw.tween_property(_flash, "color:a", 0.0, 0.4)


## Les lanternes du village s'allument à la tombée de la nuit.
func update_lamps() -> void:
	var on := _night > 0.01
	for l in lamps:
		if not is_instance_valid(l):
			continue
		if l.texture == null:
			l.texture = _torch_texture
			l.color = Color(1.0, 0.7, 0.38)
			l.texture_scale = 1.3
		l.enabled = on
		l.energy = 1.1 * _night


## Allume ou éteint les torches : seulement les héros les plus proches du tien.
func update_torches(pawns: Dictionary, me: int) -> void:
	var center: Vector2 = pawns[me].position if pawns.has(me) else Vector2.ZERO
	var list := pawns.values()
	list.sort_custom(func(a, b): return a.position.distance_squared_to(center) < b.position.distance_squared_to(center))
	for i in list.size():
		var pawn: Pawn = list[i]
		var lit := _night > 0.01 and i < MAX_TORCHES
		if lit and pawn.torch == null:
			pawn.torch = _make_torch()
			pawn.add_child(pawn.torch)
		if pawn.torch != null:
			pawn.torch.enabled = lit
			pawn.torch.energy = 0.9 * _night * (0.94 + 0.06 * sin(Time.get_ticks_msec() / 90.0 + i))


func _make_torch() -> PointLight2D:
	var l := PointLight2D.new()
	l.texture = _torch_texture
	l.color = Color(1.0, 0.6, 0.32)
	l.texture_scale = 1.0
	l.position = Vector2(0, -20)
	return l


func _make_torch_texture() -> Texture2D:
	var g := Gradient.new()
	g.set_color(0, Color(1, 1, 1, 1))
	g.set_color(1, Color(1, 1, 1, 0))
	g.add_point(0.45, Color(1, 1, 1, 0.55))
	var t := GradientTexture2D.new()
	t.gradient = g
	t.fill = GradientTexture2D.FILL_RADIAL
	t.fill_from = Vector2(0.5, 0.5)
	t.fill_to = Vector2(1.0, 0.5)
	t.width = 360
	t.height = 220   # ellipse : le sol est vu de biais
	return t


func _make_rain() -> CPUParticles2D:
	var r := CPUParticles2D.new()
	r.emitting = false
	r.amount = 300
	r.lifetime = 0.7
	r.preprocess = 0.7
	r.emission_shape = CPUParticles2D.EMISSION_SHAPE_RECTANGLE
	r.direction = Vector2(-0.25, 1)
	r.spread = 2.0
	r.gravity = Vector2.ZERO
	r.initial_velocity_min = 900
	r.initial_velocity_max = 1100
	r.particle_flag_align_y = true
	var g := Gradient.new()
	g.set_color(0, Color(0.75, 0.85, 1.0, 0.0))
	g.set_color(1, Color(0.8, 0.88, 1.0, 0.55))
	var t := GradientTexture2D.new()
	t.gradient = g
	t.width = 2
	t.height = 18
	t.fill_from = Vector2(0, 0)
	t.fill_to = Vector2(0, 1)
	r.texture = t
	return r


func _place_rain() -> void:
	var s := get_viewport().get_visible_rect().size
	_rain.position = Vector2(s.x / 2 + 120, -40)
	_rain.emission_rect_extents = Vector2(s.x * 0.7, 10)


func _fog_material() -> ShaderMaterial:
	var sh := Shader.new()
	sh.code = """
shader_type canvas_item;
// Brouillard : deux couches de bruit qui glissent lentement.
float hash(vec2 p) { return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453); }
float noise(vec2 p) {
	vec2 i = floor(p), f = fract(p);
	f = f * f * (3.0 - 2.0 * f);
	return mix(mix(hash(i), hash(i + vec2(1, 0)), f.x), mix(hash(i + vec2(0, 1)), hash(i + vec2(1, 1)), f.x), f.y);
}
void fragment() {
	vec2 uv = SCREEN_UV * vec2(4.0, 2.4);
	float n = noise(uv + vec2(TIME * 0.05, TIME * 0.01)) * 0.6 + noise(uv * 2.3 - vec2(TIME * 0.08, 0.0)) * 0.4;
	COLOR = vec4(0.82, 0.85, 0.9, 0.18 + 0.32 * n);
}
"""
	var m := ShaderMaterial.new()
	m.shader = sh
	return m
