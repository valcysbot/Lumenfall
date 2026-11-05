class_name Blason
## Le blason d'une guilde (Acte VI) : un écu (5 formes), une couleur de fond,
## une figure (30 motifs, dans l'ordre du serveur : server/world/guilde.go) et
## sa couleur. Dessiné à la main, sur la cape, à côté du nom et dans les fenêtres.

const FORMES := ["Écu", "Rond", "Bannière", "Losange", "Pavois"]
const COULEURS := [Color("c0392b"), Color("e67e22"), Color("f1c40f"), Color("27ae60"), Color("16a085"), Color("2980b9"),
	Color("8e44ad"), Color("2c3e50"), Color("ecf0f1"), Color("7f8c8d"), Color("6d4c41"), Color("1b1b24"),
	Color("e8c66a"), Color("86c493"), Color("f08a3c"), Color("d35400")]
## [identifiant, nom, emoji] : l'emoji sert de figure, sur un écu dessiné.
const MOTIFS := [["etoile", "Étoile", "★"], ["lune", "Lune", "☾"], ["soleil", "Soleil", "☀"], ["epee", "Épée", "🗡"],
	["bouclier", "Bouclier", "🛡"], ["couronne", "Couronne", "👑"], ["arbre", "Arbre", "🌳"], ["flamme", "Flamme", "🔥"],
	["goutte", "Goutte", "💧"], ["feuille", "Feuille", "🍃"], ["loup", "Loup", "🐺"], ["sanglier", "Sanglier", "🐗"],
	["corbeau", "Corbeau", "🐦"], ["dragon", "Dragon", "🐉"], ["lanterne", "Lanterne", "🏮"], ["tour", "Tour", "🏰"],
	["cle", "Clé", "🗝"], ["marteau", "Marteau", "🔨"], ["fleur", "Fleur", "🌸"], ["croix", "Croix", "✚"],
	["cerf", "Cerf", "🦌"], ["ours", "Ours", "🐻"], ["serpent", "Serpent", "🐍"], ["aigle", "Aigle", "🦅"],
	["cristal", "Cristal", "💎"], ["chene", "Chêne", "🌰"], ["harpe", "Harpe", "🎵"], ["ancre", "Ancre", "⚓"],
	["plume", "Plume", "🪶"], ["lumen", "Lumen", "✦"]]


static func couleur(i: int) -> Color:
	return COULEURS[clampi(i, 0, COULEURS.size() - 1)]


## Le contour de l'écu, dans un rectangle.
static func contour(forme: int, r: Rect2) -> PackedVector2Array:
	var p := PackedVector2Array()
	var c := r.get_center()
	var w := r.size.x
	var h := r.size.y
	match forme:
		1:  # rond
			for i in 24:
				var a := TAU * i / 24.0
				p.append(c + Vector2(cos(a) * w / 2, sin(a) * h / 2))
		2:  # bannière, à deux pointes
			p.append_array([r.position, r.position + Vector2(w, 0), r.end, c + Vector2(0, h * 0.22), r.position + Vector2(0, h)])
		3:  # losange
			p.append_array([c + Vector2(0, -h / 2), c + Vector2(w / 2, 0), c + Vector2(0, h / 2), c + Vector2(-w / 2, 0)])
		4:  # pavois : un grand rectangle aux coins coupés
			var k := w * 0.18
			p.append_array([r.position + Vector2(k, 0), r.position + Vector2(w - k, 0), r.position + Vector2(w, k),
				r.end - Vector2(0, k), r.end - Vector2(k, 0), r.position + Vector2(k, h), r.position + Vector2(0, h - k), r.position + Vector2(0, k)])
		_:  # écu classique
			p.append(r.position)
			p.append(r.position + Vector2(w, 0))
			p.append(r.position + Vector2(w, h * 0.55))
			for i in range(1, 8):
				var t := i / 8.0
				p.append(Vector2(r.end.x - w / 2 * t, r.position.y + h * 0.55 + h * 0.45 * sin(t * PI / 2)))
			p.append(c + Vector2(0, h / 2))
			for i in range(1, 8):
				var t := 1.0 - i / 8.0
				p.append(Vector2(r.position.x + w / 2 * t, r.position.y + h * 0.55 + h * 0.45 * sin(t * PI / 2)))
			p.append(r.position + Vector2(0, h * 0.55))
	return p


## Dessine le blason dans un rectangle (sur un Control ou un Node2D).
static func dessiner(ci: CanvasItem, r: Rect2, bl: Dictionary) -> void:
	var pts := contour(int(bl.get("f", 0)), r)
	ci.draw_colored_polygon(pts, couleur(int(bl.get("c1", 0))))
	var bord := pts.duplicate()
	bord.append(pts[0])
	ci.draw_polyline(bord, Color(0.08, 0.06, 0.1, 0.9), maxf(1.0, r.size.x / 18.0), true)
	var m: Array = MOTIFS[clampi(int(bl.get("m", 0)), 0, MOTIFS.size() - 1)]
	var font := ThemeDB.fallback_font
	var fs := int(r.size.y * 0.5)
	var txt: String = m[2]
	var sz := font.get_string_size(txt, HORIZONTAL_ALIGNMENT_LEFT, -1, fs)
	var at := r.get_center() + Vector2(-sz.x / 2, fs * 0.35)
	var fig := couleur(int(bl.get("c2", 8)))
	ci.draw_string_outline(font, at, txt, HORIZONTAL_ALIGNMENT_LEFT, -1, fs, maxi(1, fs / 6), Color(0, 0, 0, 0.6))
	ci.draw_string(font, at, txt, HORIZONTAL_ALIGNMENT_LEFT, -1, fs, fig)


static func nom_motif(i: int) -> String:
	return MOTIFS[clampi(i, 0, MOTIFS.size() - 1)][1]


## Un petit Control qui montre un blason.
class Vue:
	extends Control
	var bl := {}

	func _init(b: Dictionary = {}, taille := 40.0) -> void:
		bl = b
		custom_minimum_size = Vector2(taille * 0.84, taille)
		mouse_filter = Control.MOUSE_FILTER_IGNORE
		size_flags_horizontal = Control.SIZE_SHRINK_CENTER
		size_flags_vertical = Control.SIZE_SHRINK_CENTER

	func set_blason(b: Dictionary) -> void:
		bl = b
		queue_redraw()

	func _draw() -> void:
		Blason.dessiner(self, Rect2(Vector2.ZERO, size), bl)
