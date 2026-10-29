class_name UI
## Couleurs, thème et petits éléments d'interface partagés.

const NIGHT := Color("15131d")
const STONE := Color("201c2b")
const STONE_2 := Color("2a2538")
const LINE := Color("3b3450")
const PARCH := Color("efe6d6")
const MUTED := Color("a99fba")
const EMBER := Color("f08a3c")
const LUMEN := Color("e8c66a")
const MOSS := Color("86c493")
const DANGER := Color("e86a5a")

## Astuces de Braise pour les écrans d'attente.
const TIPS := [
	"Braise dit : sur PC, clique pour marcher, ou utilise ZQSD.",
	"Braise dit : appuie sur Entrée pour parler aux autres aventuriers.",
	"Braise dit : un nom d'Erydor, ça sonne bien avec une apostrophe. D'Arvel, par exemple.",
	"Braise dit : l'étang de la Clairière est très profond. Enfin, je crois. Je n'ai jamais osé y aller.",
	"Braise dit : je ne crache du feu que sur les bugs. Promis.",
	"Braise dit : sur téléphone, le joystick est en bas à gauche, et un toucher sur la carte t'y emmène.",
]


static func make_theme() -> Theme:
	var t := Theme.new()
	t.default_font_size = 20
	t.set_color("font_color", "Label", PARCH)
	t.set_color("font_color", "Button", PARCH)
	t.set_color("font_hover_color", "Button", LUMEN)
	t.set_color("font_pressed_color", "Button", NIGHT)
	t.set_color("font_disabled_color", "Button", MUTED)
	t.set_color("font_color", "LineEdit", PARCH)
	t.set_color("font_placeholder_color", "LineEdit", MUTED)
	t.set_color("caret_color", "LineEdit", EMBER)
	t.set_stylebox("normal", "Button", box(STONE_2, LINE, 10))
	t.set_stylebox("hover", "Button", box(STONE_2, LUMEN, 10))
	t.set_stylebox("pressed", "Button", box(LUMEN, LUMEN, 10))
	t.set_stylebox("disabled", "Button", box(STONE, LINE, 10))
	t.set_stylebox("focus", "Button", box(Color(0, 0, 0, 0), EMBER, 10, 2))
	t.set_stylebox("normal", "LineEdit", box(NIGHT, LINE, 10))
	t.set_stylebox("focus", "LineEdit", box(NIGHT, EMBER, 10, 2))
	t.set_stylebox("panel", "PanelContainer", box(STONE, LINE, 14))
	return t


static func box(bg: Color, border: Color, radius: int, border_w: int = 1, pad: int = 12) -> StyleBoxFlat:
	var s := StyleBoxFlat.new()
	s.bg_color = bg
	s.border_color = border
	s.set_border_width_all(border_w)
	s.set_corner_radius_all(radius)
	s.content_margin_left = pad + 4
	s.content_margin_right = pad + 4
	s.content_margin_top = pad
	s.content_margin_bottom = pad
	s.anti_aliasing = true
	return s


static func label(text: String, size: int = 20, color: Color = PARCH) -> Label:
	var l := Label.new()
	l.text = text
	l.add_theme_font_size_override("font_size", size)
	l.add_theme_color_override("font_color", color)
	l.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	return l


static func button(text: String, primary: bool = false) -> Button:
	var b := Button.new()
	b.text = text
	b.custom_minimum_size = Vector2(0, 52)
	if primary:
		b.add_theme_stylebox_override("normal", box(EMBER, EMBER, 10))
		b.add_theme_stylebox_override("hover", box(LUMEN, LUMEN, 10))
		b.add_theme_color_override("font_color", NIGHT)
		b.add_theme_color_override("font_hover_color", NIGHT)
	return b


## Petite bulle de Braise, le dragonneau qui commente tout.
static func braise(text: String) -> HBoxContainer:
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 10)
	var d := label("🐉", 34)
	d.autowrap_mode = TextServer.AUTOWRAP_OFF
	row.add_child(d)
	var p := PanelContainer.new()
	p.add_theme_stylebox_override("panel", box(STONE_2, EMBER, 12, 1, 8))
	p.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	var l := label(text, 17)
	l.name = "Text"
	p.add_child(l)
	row.add_child(p)
	return row


static func set_braise(row: HBoxContainer, text: String) -> void:
	(row.get_child(1).get_node("Text") as Label).text = text


## Fond étoilé commun aux écrans de menu.
static func backdrop() -> ColorRect:
	var bg := ColorRect.new()
	bg.color = NIGHT
	bg.set_anchors_preset(Control.PRESET_FULL_RECT)
	bg.mouse_filter = Control.MOUSE_FILTER_IGNORE
	var glow := Control.new()
	glow.set_anchors_preset(Control.PRESET_FULL_RECT)
	glow.mouse_filter = Control.MOUSE_FILTER_IGNORE
	glow.draw.connect(func():
		var sz := glow.size
		var rng := RandomNumberGenerator.new()
		rng.seed = 1717
		for i in 90:
			var p := Vector2(rng.randf() * sz.x, rng.randf() * sz.y)
			glow.draw_circle(p, rng.randf_range(0.6, 1.8), Color(LUMEN, rng.randf_range(0.15, 0.6)))
		for i in 40:
			glow.draw_circle(Vector2(sz.x * 0.5, -sz.y * 0.1), sz.x * (0.55 - i * 0.012), Color(EMBER, 0.006))
	)
	glow.resized.connect(glow.queue_redraw)
	bg.add_child(glow)
	return bg
