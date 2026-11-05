class_name Heros
## Les données des héros (Acte IV), lues une fois : les 15 peuples et leurs
## couleurs (data/heros/peuples.json, écrit par la forge des peuples), les 17
## classes, leurs compétences et leurs 51 voies (data/heros/classes.json, écrit
## par le serveur). Et quelques mots pour les afficher.

const PEUPLES_PATH := "res://data/heros/peuples.json"
const CLASSES_PATH := "res://data/heros/classes.json"

## Les peuples dans l'ordre du miroir, par faction.
const ORDRE_PEUPLES := ["humain", "nain", "halfelin", "gnome", "golem",
		"elfe", "fee", "felin", "homme_bete", "sirene",
		"orque", "drakeide", "revenant", "elfe_noir", "lumin"]
const FACTIONS := {
	"concorde": {"nom": "la Concorde d'Argent", "blason": "🛡️", "couleur": Color("8fb4e8")},
	"pacte": {"nom": "le Pacte Sylvestre", "blason": "🌿", "couleur": Color("86c493")},
	"ligue": {"nom": "la Ligue des Cendres", "blason": "🔥", "couleur": Color("e86a5a")},
}
## Nom, ce que le dieu protège, symbole (docs/bible-du-monde.md).
const DIEUX := {
	"solenn": ["Solenn", "la Lumière", "☀️"],
	"maelis": ["Maelis", "l'Eau", "🌊"],
	"brannoc": ["Brannoc", "la Terre", "⛰️"],
	"caelia": ["Caelia", "l'Air", "🌬️"],
	"fergal": ["Fergal", "le Feu", "🔥"],
	"seren": ["Seren", "l'Ombre", "🌘"],
	"aurel": ["Aurel", "le Premier Éclat, un Silencieux", "🕯️"],
	"selis": ["Sélis", "la Blanche, une Silencieuse", "🌕"],
	"cernos": ["Cernos", "le Cornu, un Silencieux", "🦌"],
}
const ATTRIBUTS := {
	"force": ["Force", "💪", "Dégâts des armes au corps à corps."],
	"intelligence": ["Intelligence", "📘", "Dégâts des sorts et soins."],
	"agilite": ["Agilité", "🏹", "Dégâts à distance, un peu de critique."],
	"vitesse": ["Vitesse", "👟", "Tu cours plus vite (jusqu'à +25 %)."],
	"esquive": ["Esquive", "🌀", "Une chance d'éviter les coups."],
	"vitalite": ["Vitalité", "❤️", "+8 points de vie par point."],
	"chance": ["Chance", "🍀", "Plus de coups critiques."],
	"endurance": ["Endurance", "🪨", "Tu encaisses mieux."],
	"precision": ["Précision", "🎯", "Critiques plus fréquents et plus forts."],
}
const ROLES := {"tank": "🛡️ Tank", "soin": "💚 Soin", "degats": "⚔️ Dégâts", "soutien": "✨ Soutien"}
const RESSOURCE_COULEURS := {
	"rage": "d84a3a", "furie": "b8302a", "elan": "e8a040", "souffle_draconique": "f07a3c", "ferveur": "e8c66a",
	"souffle": "8ad0e0", "mana": "4a7bd0", "ames": "8a6ad8", "lien": "c06ad8", "reactifs": "5fbf6a",
	"vapeur": "c8c8d0", "grace": "f2e6a0", "seve": "6abf4a", "esprits": "40c0c8", "harmonie": "e080b0",
	"energie": "f2d94e", "concentration": "86c493",
}
const EFFETS := {
	"brulure": "🔥 Brûlure", "poison": "🧪 Poison", "etourdi": "💫 Étourdi", "ralenti": "🐌 Ralenti",
	"faiblesse": "💔 Faiblesse", "aura": "🌟 Aura", "gel": "❄️ Gel", "saignement": "🩸 Saignement",
	"silence": "🤐 Silence", "aveugle": "🙈 Aveuglé", "peur": "😱 Peur", "charme": "💗 Charmé",
	"bouclier": "🛡️ Bouclier", "regen": "💚 Régénération", "furtif": "🌫️ Furtif", "rage": "😤 Rage",
	"protection": "🧱 Protection", "provoque": "📣 Provoqué", "hate": "👟 Hâte", "marque": "🎯 Marqué",
	"inspire": "🎵 Inspiré", "compagnon": "🐾 Compagnon",
}
const CARRURES := ["mince", "normale", "costaud"]
const TAILLES := [0.9, 0.95, 1.0, 1.05, 1.1]
## Les classes qui portent une robe plutôt qu'une tunique, et celles qui ont des gants ou une cape.
const ROBES := ["mage", "necromancien", "invocateur", "pretre", "druide", "chaman"]
const GANTS := ["guerrier", "berserker", "lancier", "chevalier_dragon", "paladin", "ingenieur", "archer"]
const CAPES := ["paladin", "chevalier_dragon", "necromancien", "barde", "voleur"]

static var _peuples: Dictionary = {}
static var _classes: Dictionary = {}
static var _by_id: Dictionary = {}


static func peuples() -> Dictionary:
	if _peuples.is_empty():
		_peuples = JSON.parse_string(FileAccess.get_file_as_string(PEUPLES_PATH))
	return _peuples


static func data() -> Dictionary:
	if _classes.is_empty():
		_classes = JSON.parse_string(FileAccess.get_file_as_string(CLASSES_PATH))
		for c in _classes.classes:
			_by_id[c.id] = c
	return _classes


static func classe(id: String) -> Dictionary:
	data()
	return _by_id.get(id, _by_id.get("paladin", {}))


static func peuple(id: String) -> Dictionary:
	return peuples().peuples.get(id, peuples().peuples.humain)


## Le mot accordé : noms = [masculin, féminin, neutre].
static func accorde(noms: Array, accord: int) -> String:
	return str(noms[clampi(accord, 0, noms.size() - 1)])


## La compétence k (1 à 7) d'une classe, ou l'ultime de sa voie (8). {} si rien.
static func competence(cl: String, k: int, voie: int) -> Dictionary:
	var c := classe(cl)
	if k >= 1 and k <= 7:
		return c.competences[k - 1]
	if k == 8 and voie >= 0 and voie < 3:
		return c.voies[voie].ultime
	return {}


## L'arbre de talents : le palier commun de la classe, puis ceux de la voie (s'il y en a une).
static func arbre(cl: String, voie: int) -> Array:
	var c := classe(cl)
	var out: Array = [c.commun]
	if voie >= 0 and voie < 3:
		out.append_array(c.voies[voie].paliers)
	return out


## Les points de talent gagnés à ce niveau (1 par niveau à partir du 5).
static func points_de_talent(lv: int) -> int:
	return maxi(0, lv - int(data().talents.premier) + 1)


## Le style ⭐ d'une voie (ou « seul » avant la voie).
static func etoile(cl: String, voie: int) -> String:
	if voie < 0 or voie > 2:
		return "seul"
	return str(classe(cl).voies[voie].etoile)


## Les talents conseillés : comme le serveur (talents.go, RecommendTalents).
static func talents_conseilles(cl: String, voie: int, lv: int) -> Array:
	var tree := arbre(cl, voie)
	var out := vide_talents()
	var styles := ["seul", "groupe", "utile"]
	var first := styles.find(etoile(cl, voie))
	var left := points_de_talent(lv)
	for s in 3:
		var j := (first + s) % 3
		for i in tree.size():
			if lv < int(tree[i].niveau):
				break
			var n := mini(left, int(tree[i].talents[j].max) - int(out[i][j]))
			out[i][j] += n
			left -= n
	return out


static func vide_talents() -> Array:
	var out := []
	for i in 10:
		out.append([0, 0, 0])
	return out


static func talents_ints(a) -> Array:
	var out := vide_talents()
	if a is Array:
		for i in mini(10, a.size()):
			for j in mini(3, (a[i] as Array).size()):
				out[i][j] = int(a[i][j])
	return out


static func points_places(t: Array) -> int:
	var n := 0
	for row in t:
		for v in row:
			n += int(v)
	return n


## Un palier s'ouvre-t-il à ce niveau ?
static func palier_au_niveau(lv: int) -> bool:
	var first := int(data().talents.premier)
	return lv >= first and (lv - first) % int(data().talents.pas) == 0 and lv <= first + 9 * int(data().talents.pas)


static func faction_of(race: String) -> Dictionary:
	return FACTIONS.get(str(peuple(race).get("faction", "concorde")), FACTIONS.concorde)


## Une allure tirée au hasard pour un peuple (le miroir s'ouvre dessus).
static func allure_au_hasard(race: String) -> Dictionary:
	var lk := {"g": randi() % 2, "t": 2, "k": randi() % 3, "v": randi() % 16, "p": randi() % 24, "y": randi() % 12,
		"h": randi() % 20, "hc": randi() % 24, "b": -1, "m": [], "tu": randi() % 12}
	lk.a = lk.g
	return lk


## Les couches à empiler pour un héros : [[pièce, {matériau: couleur}], …],
## de l'arrière vers l'avant.
## `eq` (Acte V) : ce que le héros porte, {emplacement: "base:rareté"} ; null
## pour un héros sans sac (création, anciens héros) : il garde la tenue de sa classe.
static func couches(race: String, cl: String, lk: Dictionary, eq: Variant = null) -> Array:
	var P := peuple(race)
	var D := peuples()
	var carr: String = CARRURES[clampi(int(lk.get("k", 1)), 0, 2)]
	var forme: String = P.forme
	var hc: String = D.cheveux[clampi(int(lk.get("hc", 0)), 0, 23)]
	var cs := {
		"peau": P.peaux[clampi(int(lk.get("p", 0)), 0, 23)],
		"yeux": D.yeux[clampi(int(lk.get("y", 0)), 0, 11)],
		"sourcils": hc, "cheveux": hc,
		"tissu": D.tuniques[clampi(int(lk.get("tu", 0)), 0, 11)],
	}
	if P.get("ames", false):
		cs.yeux = "#bff4ff"   # les Revenants ont les yeux qui luisent doucement
	if eq is Dictionary:
		return _couches_equipees(P, cl, carr, forme, lk, cs, eq)
	var out := []
	var dos := ["queue", "ailes", "nageoires"]
	for t in P.traits:
		if dos.any(func(x): return t.begins_with(x)):
			out.append(["trait_" + t, cs])
	if cl != "" and CAPES.has(cl):
		out.append(["cape_" + carr, cs])
	out.append(["corps_%s_%s" % [forme, carr], cs])
	out.append(["pantalon_" + carr, cs])
	out.append(["bottes_" + carr, cs])
	out.append([("robe_" if ROBES.has(cl) else "tunique_") + carr, cs])
	if cl == "alchimiste" or cl == "ingenieur":
		out.append(["tablier_" + carr, cs])
	if GANTS.has(cl):
		out.append(["gants_" + carr, cs])
	out.append(["visage_%02d" % clampi(int(lk.get("v", 0)), 0, 15), cs])
	for m in lk.get("m", []):
		var nom: String = D.marques[clampi(int(m[0]), 0, D.marques.size() - 1)]
		out.append(["marque_" + nom, {nom: D.cheveux[clampi(int(m[1]), 0, 23)]}])
	if forme == "standard":
		out.append(["cheveux_%02d" % clampi(int(lk.get("h", 0)), 0, 19), cs])
		var b := int(lk.get("b", -1))
		if b >= 0:
			out.append(["barbe_%02d" % clampi(b, 0, 5), cs])
	for t in P.traits:
		if not dos.any(func(x): return t.begins_with(x)):
			out.append(["trait_" + t, cs])
	if cl != "":
		for w in classe(cl).armes:
			out.append(["arme_" + w, cs])
	return out


## Acte V : les couleurs des pièces selon le type d'armure, et la teinte de la rareté.
const ARMURES := {"tissu": "", "cuir": "#8a5a32", "mailles": "#a8b0bc", "plaques": "#d4d8e2"}
const JAMBES := {"tissu": "#6e4c35", "cuir": "#5e3c24", "mailles": "#8d96a8", "plaques": "#b8bece"}
const PIEDS := {"tissu": "#5a4636", "cuir": "#4a3328", "mailles": "#6a7080", "plaques": "#a4adc0"}
const RARETES := ["#9d9d9d", "#4fd16a", "#4a9ce8", "#b35de8", "#f0902c", "#ffd84a"]


## Ce que porte un emplacement : [base, type d'armure, rareté] (base vide : rien).
static func piece_portee(eq: Dictionary, slot: String) -> Array:
	var v := str(eq.get(slot, ""))
	if v == "":
		return ["", "", 0]
	var parts := v.split(":")
	var base: String = parts[0]
	var r := int(parts[1]) if parts.size() > 1 else 0
	var t := ""
	for a in ARMURES:
		if base.ends_with("_" + a):
			t = a
	return [base, t, clampi(r, 0, 5)]


static func _couches_equipees(P: Dictionary, cl: String, carr: String, forme: String, lk: Dictionary, cs: Dictionary, eq: Dictionary) -> Array:
	var D := peuples()
	var out := []
	var dos := ["queue", "ailes", "nageoires"]
	var avec := func(extra: Dictionary) -> Dictionary:
		var c := cs.duplicate()
		c.merge(extra, true)
		return c
	for t in P.traits:
		if dos.any(func(x): return t.begins_with(x)):
			out.append(["trait_" + t, cs])
	var cape := piece_portee(eq, "dos")
	if cape[0] != "":
		out.append(["cape_" + carr, avec.call({"cape": RARETES[cape[2]]})])
	out.append(["corps_%s_%s" % [forme, carr], cs])
	var jambes := piece_portee(eq, "jambes")
	out.append(["pantalon_" + carr, avec.call({"pantalon": JAMBES.get(jambes[1], JAMBES.tissu)})])
	var pieds := piece_portee(eq, "pieds")
	out.append(["bottes_" + carr, avec.call({"bottes": PIEDS.get(pieds[1], PIEDS.tissu)})])
	var torse := piece_portee(eq, "torse")
	var boucle := {"boucle": RARETES[torse[2]]} if torse[0] != "" else {}
	out.append([("robe_" if ROBES.has(cl) else "tunique_") + carr, avec.call(boucle)])
	if torse[1] != "" and torse[1] != "tissu":
		out.append(["plastron_" + carr, avec.call({"armure": ARMURES[torse[1]], "bordure": RARETES[torse[2]]})])
	var mains := piece_portee(eq, "mains")
	if mains[0] != "":
		out.append(["gants_" + carr, avec.call({"gants": ARMURES[mains[1]] if mains[1] != "tissu" else "#8a6a52"})])
	var ep := piece_portee(eq, "epaules")
	if ep[0] != "":
		var couleur: String = ARMURES[ep[1]] if ep[1] != "tissu" else str(cs.tissu)
		out.append(["epaulieres_" + carr, avec.call({"armure": couleur, "bordure": RARETES[ep[2]]})])
	out.append(["visage_%02d" % clampi(int(lk.get("v", 0)), 0, 15), cs])
	for m in lk.get("m", []):
		var nom: String = D.marques[clampi(int(m[0]), 0, D.marques.size() - 1)]
		out.append(["marque_" + nom, {nom: D.cheveux[clampi(int(m[1]), 0, 23)]}])
	var tete := piece_portee(eq, "tete")
	if forme == "standard":
		if tete[1] != "mailles" and tete[1] != "plaques" and tete[1] != "cuir":
			out.append(["cheveux_%02d" % clampi(int(lk.get("h", 0)), 0, 19), cs])
		var b := int(lk.get("b", -1))
		if b >= 0:
			out.append(["barbe_%02d" % clampi(b, 0, 5), cs])
	for t in P.traits:
		if not dos.any(func(x): return t.begins_with(x)):
			out.append(["trait_" + t, cs])
	match tete[1]:
		"tissu":
			out.append(["chapeau", avec.call({"chapeau": str(cs.tissu), "ceinture": RARETES[tete[2]]})])
		"cuir":
			out.append(["capuche", avec.call({"capuche": ARMURES.cuir})])
		"mailles", "plaques":
			out.append(["casque", avec.call({"casque": ARMURES[tete[1]], "garde": RARETES[tete[2]]})])
	for slot in ["arme", "main_gauche"]:
		var a := piece_portee(eq, slot)
		if a[0] != "":
			var w: String = a[0].trim_prefix("arme_").trim_prefix("mg_")
			out.append(["arme_" + w, avec.call({"garde": RARETES[a[2]]} if a[2] >= 2 else {})])
	return out


## Les couches d'un habitant (res://data/pnj/<carte>.json, clé « peuple ») : un
## héros sans classe, avec sa robe ou sa tunique, et ses pièces en plus
## (tablier, gants, cape, chapeau, casque, arme_…) dans leurs couleurs.
static func habitant(pe: Dictionary) -> Array:
	var race: String = pe.get("race", "humain")
	var carr: String = CARRURES[clampi(int(pe.get("k", 1)), 0, 2)]
	var plus: Array = pe.get("plus", [])
	var out := couches(race, "", pe)
	var cs: Dictionary = {}
	var at := 0
	for i in out.size():
		var e: Array = out[i]
		if e[0] == "tunique_" + carr:
			cs = e[1]
			at = i
			if pe.get("robe", false):
				e[0] = "robe_" + carr
	cs.merge(pe.get("couleurs", {}), true)
	for p in ["gants", "tablier"]:   # au-dessus de la tunique
		if plus.has(p):
			out.insert(at + 1, [p + "_" + carr, cs])
	if plus.has("cape"):   # derrière le corps
		var corps := out.map(func(e): return e[0]).find("corps_%s_%s" % [peuple(race).forme, carr])
		out.insert(maxi(corps, 0), ["cape_" + carr, cs])
	for p in plus:
		if p == "chapeau" or p == "casque" or str(p).begins_with("arme_"):
			out.append([p, cs])
	return out


## La taille à l'écran : celle du peuple, puis le cran choisi au miroir.
static func echelle(race: String, lk: Dictionary) -> float:
	return float(peuple(race).taille) * float(TAILLES[clampi(int(lk.get("t", 2)), 0, 4)])


## 4 tons (ombre profonde, ombre, base, lumière) et le contour d'une couleur :
## la même recette que la forge (tools/sprites/forge_heros.py, rampe).
static func rampe(hexa: String, alpha: float = 1.0) -> Array:
	var c := Color(hexa)
	var h := c.h
	var s := c.s
	var v := c.v
	var vers := func(cible: float, k: float) -> float:
		return (fposmod(cible - h + 0.5, 1.0) - 0.5) * k
	var ton := func(dh: float, ds: float, dv: float) -> Color:
		var col := Color.from_hsv(fposmod(h + dh, 1.0), clampf(s + ds, 0, 1), clampf(v + dv, 0, 1))
		col.a = alpha
		return col
	return [ton.call(vers.call(0.72, 0.18), 0.12, -0.42), ton.call(vers.call(0.72, 0.08), 0.06, -0.2),
		ton.call(0.0, 0.0, 0.0), ton.call(vers.call(0.14, 0.08), -0.12, 0.14),
		ton.call(vers.call(0.72, 0.22), 0.1, -0.62)]
