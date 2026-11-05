class_name Objets
## Les objets côté client (Acte V, « Mille trésors ») : couleurs des raretés,
## monnaie (or, argent, cuivre), la fiche d'un objet (bulle au survol sur PC,
## détail au toucher sur téléphone) et la case d'un objet dans une grille.
## Tout ce que le client montre vient du serveur (ItemView) : nom, chiffres,
## bonus déjà écrits en phrases.

const RARETES := ["Commun", "Peu commun", "Rare", "Épique", "Légendaire", "Lumineux"]
const COULEURS := [Color("b8b4bc"), Color("4fd16a"), Color("4a9ce8"), Color("b35de8"), Color("f0902c"), Color("ffd84a")]

## [identifiant, nom, icône de l'emplacement vide]
const EMPLACEMENTS := [
	["tete", "Tête", "🪖"], ["epaules", "Épaules", "🎽"], ["torse", "Torse", "👕"], ["mains", "Mains", "🧤"],
	["jambes", "Jambes", "👖"], ["pieds", "Pieds", "🥾"], ["dos", "Dos", "🧣"], ["arme", "Arme", "⚔️"],
	["main_gauche", "Main gauche", "🛡️"], ["cou", "Cou", "📿"], ["anneau1", "Anneau", "💍"], ["anneau2", "Anneau", "💍"],
]
const ARMURES := {"tissu": "Tissu", "cuir": "Cuir", "mailles": "Mailles", "plaques": "Plaques"}


## Une liste venue du serveur (une liste vide peut arriver en « null »).
static func liste(v: Variant) -> Array:
	return v if v is Array else []


static func couleur(r: int) -> Color:
	return COULEURS[clampi(r, 0, 5)]


## 12345 cuivres → « 1 ar 23 cu » ; 1 234 567 → « 123 or 45 ar 67 cu ».
static func monnaie(n: int, longue: bool = false) -> String:
	if n <= 0:
		return "0 cuivre" if longue else "0 cu"
	var o := n / 10000
	var a := (n / 100) % 100
	var c := n % 100
	var mots := ["or", "argent", "cuivre"] if longue else ["or", "ar", "cu"]
	var out := []
	if o > 0:
		out.append("%d %s" % [o, mots[0]])
	if a > 0:
		out.append("%d %s" % [a, mots[1]])
	if c > 0:
		out.append("%d %s" % [c, mots[2]])
	return " ".join(out)


static func nom_emplacement(id: String) -> String:
	for e in EMPLACEMENTS:
		if e[0] == id:
			return e[1]
	return "Anneau" if id == "anneau" else id


## L'emplacement où l'on comparerait cet objet (le premier anneau, pour un anneau).
static func emplacement_de(it: Dictionary) -> String:
	var sl := str(it.get("sl", ""))
	return "anneau1" if sl == "anneau" else sl


## La fiche d'un objet ; `porte` : l'objet porté au même endroit, pour comparer
## (flèches vertes et rouges).
static func fiche(it: Dictionary, porte: Dictionary = {}) -> VBoxContainer:
	var col := VBoxContainer.new()
	col.add_theme_constant_override("separation", 3)
	col.custom_minimum_size = Vector2(300, 0)
	var r := int(it.get("r", 0))
	var titre := UI.label("%s %s" % [it.get("ic", ""), it.get("nom", "")], 19, couleur(r))
	col.add_child(titre)
	var sous := []
	var k := str(it.get("k", ""))
	if k == "equipement":
		sous.append(RARETES[clampi(r, 0, 5)])
		sous.append(nom_emplacement(str(it.get("sl", ""))))
		if it.has("ar"):
			sous.append(ARMURES.get(str(it.ar), str(it.ar)))
		if int(it.get("lv", 0)) > 0:
			sous.append("niveau %d" % int(it.lv))
	elif k == "materiau":
		sous.append("Matériau")
	elif k == "provision":
		sous.append("Provision")
	elif k == "recette":
		sous.append("Plan (clic droit ou « Utiliser » pour l'apprendre)")
	elif k == "sac":
		sous.append("Sac")
	col.add_child(UI.label(" · ".join(sous), 14, UI.MUTED))
	if k == "equipement":
		if not bool(it.get("ok", true)):
			col.add_child(UI.label("Ta classe ne peut pas le porter.", 14, UI.DANGER))
		var cmp := not porte.is_empty() and str(porte.get("id", "")) != str(it.get("id", ""))
		if float(it.get("arm", 0)) > 0:
			col.add_child(_chiffre("🛡️ Armure %s" % _n(float(it.arm)), float(it.arm), float(porte.get("arm", 0)), cmp))
		if float(it.get("pw", 0)) > 0:
			col.add_child(_chiffre("⚔️ Puissance %s" % _n(float(it.pw)), float(it.pw), float(porte.get("pw", 0)), cmp))
		for a in Objets.liste(it.get("a")):
			var l := UI.label(str(a[2]) if a.size() > 2 else "+%s %s" % [a[1], a[0]], 15, Color("8fe0a0"))
			col.add_child(l)
		for lg in Objets.liste(it.get("lg")):
			col.add_child(UI.label("✦ " + str(lg), 15, COULEURS[4]))
		if cmp and it.has("sc") and porte.has("sc"):
			var d := float(it.sc) - float(porte.sc)
			if absf(d) >= 0.05:
				var mieux := d > 0
				col.add_child(UI.label(("▲ Mieux" if mieux else "▼ Moins bien") + " que ce que tu portes, pour ta voie", 14, UI.MOSS if mieux else UI.DANGER))
		var st := int(it.get("st", 0))
		if st > 0:
			col.add_child(UI.label("⭐".repeat(st) + " pour ta voie", 14, UI.LUMEN))
		var dur := int(it.get("d", 100))
		if dur < 100:
			col.add_child(UI.label("Usure : %d / 100%s" % [dur, " — cassé, ne donne plus rien !" if dur <= 0 else ""], 14, UI.DANGER if dur <= 25 else UI.MUTED))
		if bool(it.get("l", false)):
			col.add_child(UI.label("Lié : ne s'échange plus.", 14, UI.MUTED))
		if str(it.get("f", "")) != "":
			col.add_child(UI.label("Fabriqué par %s" % it.f, 14, UI.EMBER))
	if int(it.get("n", 0)) > 1:
		col.add_child(UI.label("× %d" % int(it.n), 14, UI.MUTED))
	if str(it.get("ds", "")) != "":
		col.add_child(UI.label(str(it.ds), 14, UI.PARCH))
	if int(it.get("px", 0)) > 0:
		col.add_child(UI.label("Se vend %s chez Aelis" % monnaie(int(it.px), true), 13, UI.MUTED))
	return col


static func _n(v: float) -> String:
	return str(snappedf(v, 0.1)).replace(".", ",")


static func _chiffre(texte: String, v: float, avant: float, cmp: bool) -> Label:
	var c := UI.PARCH
	if cmp and absf(v - avant) >= 0.05:
		texte += "  %s %s" % ["▲" if v > avant else "▼", _n(absf(v - avant))]
		c = UI.MOSS if v > avant else UI.DANGER
	return UI.label(texte, 16, c)


## Une bulle (PanelContainer) avec la fiche, pour les survols.
static func bulle(it: Dictionary, porte: Dictionary = {}) -> PanelContainer:
	var p := PanelContainer.new()
	p.add_theme_stylebox_override("panel", UI.box(Color(UI.NIGHT, 0.97), couleur(int(it.get("r", 0))), 10, 2, 10))
	p.add_child(fiche(it, porte))
	return p


## La case d'un objet (ou d'un emplacement vide, avec `vide` : son icône pâle).
class Case extends PanelContainer:
	var it: Dictionary = {}
	var porte: Dictionary = {}
	var vide := ""
	var choisie := false
	var taille := 58.0
	var _ic: Label
	var _n: Label

	func _init(objet: Dictionary = {}, icone_vide: String = "", px: float = 58.0) -> void:
		it = objet if objet != null else {}
		vide = icone_vide
		taille = px
		custom_minimum_size = Vector2(px, px)
		mouse_filter = Control.MOUSE_FILTER_STOP
		var stack := Control.new()
		stack.mouse_filter = Control.MOUSE_FILTER_IGNORE
		add_child(stack)
		_ic = UI.label("", int(px * 0.5))
		_ic.autowrap_mode = TextServer.AUTOWRAP_OFF
		_ic.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		_ic.vertical_alignment = VERTICAL_ALIGNMENT_CENTER
		_ic.set_anchors_preset(Control.PRESET_FULL_RECT)
		_ic.mouse_filter = Control.MOUSE_FILTER_IGNORE
		stack.add_child(_ic)
		_n = UI.label("", 13, UI.PARCH)
		_n.autowrap_mode = TextServer.AUTOWRAP_OFF
		_n.set_anchors_and_offsets_preset(Control.PRESET_BOTTOM_RIGHT)
		_n.grow_horizontal = Control.GROW_DIRECTION_BEGIN
		_n.grow_vertical = Control.GROW_DIRECTION_BEGIN
		_n.add_theme_color_override("font_outline_color", UI.NIGHT)
		_n.add_theme_constant_override("outline_size", 4)
		_n.mouse_filter = Control.MOUSE_FILTER_IGNORE
		stack.add_child(_n)
		refresh()

	func refresh() -> void:
		var r := int(it.get("r", 0))
		var bord := Objets.couleur(r) if not it.is_empty() else UI.LINE
		if choisie:
			bord = UI.LUMEN
		var fond := UI.STONE_2
		if not it.is_empty() and str(it.get("k", "")) == "equipement" and not bool(it.get("ok", true)):
			fond = Color("3a2026")   # pas pour ta classe : fond rouge sombre
		if not it.is_empty() and int(it.get("d", 100)) <= 0 and str(it.get("k", "")) == "equipement":
			fond = Color("3a2a20")   # cassé
		add_theme_stylebox_override("panel", UI.box(fond, bord, 8, 3 if choisie else 2, 2))
		_ic.text = str(it.get("ic", "")) if not it.is_empty() else vide
		_ic.modulate.a = 1.0 if not it.is_empty() else 0.25
		var n := int(it.get("n", 0))
		_n.text = str(n) if n > 1 else ""
		tooltip_text = "." if not it.is_empty() else ""

	func _make_custom_tooltip(_for_text: String) -> Object:
		if it.is_empty():
			return null
		return Objets.bulle(it, porte)
