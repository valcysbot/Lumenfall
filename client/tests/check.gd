extends Node
## Vérifie que tous les scripts du jeu se chargent sans erreur.
## godot --headless --path client res://tests/check.tscn

func _ready() -> void:
	var failed := 0
	var dir := DirAccess.open("res://scripts")
	for f in dir.get_files():
		if f.ends_with(".gd"):
			var s = load("res://scripts/" + f)
			if s == null or not (s as GDScript).can_instantiate():
				printerr("Script en erreur : ", f)
				failed += 1
	var map = JSON.parse_string(FileAccess.get_file_as_string("res://data/maps/glennmor.json"))
	if not (map is Dictionary) or map.rows.size() != int(map.height):
		printerr("Carte du Val de Glennmor illisible")
		failed += 1
	if not ResourceLoader.exists("res://cartes/glennmor.tscn"):
		printerr("Scène du Val de Glennmor manquante")
		failed += 1
	# Aller-retour monde <-> écran.
	var p := Vector2(12.25, 30.5)
	if Iso.to_world(Iso.to_screen(p)).distance_to(p) > 0.001:
		printerr("Conversion isométrique incohérente")
		failed += 1
	if Pawn._wrap("On fait un tour près de l'étang ce soir ?", 26).split("\n").size() != 2:
		printerr("Découpage des bulles incorrect")
		failed += 1
	# Toutes les planches du héros en couches existent, pour chaque tenue possible.
	var m := HeroSprite.meta()
	for anim in ["repos", "marche", "course", "salut", "attaque", "bouclier", "garde", "incante", "sort", "roulade", "mort"]:
		if not m.anims.has(anim):
			printerr("Animation manquante : ", anim)
			failed += 1
	for c in 12:
		for who in ["Elowyn", "D'Arvel", "Mael-Tor", "Braise"]:
			var look := HeroSprite.look_for(who, c)
			for piece in look:
				var v: String = look[piece]
				var file: String = ("cheveux_" + v) if piece == "cheveux" else (piece + ("_" + v if v != "" else ""))
				if not ResourceLoader.exists(HeroSprite.DIR + file + ".png"):
					printerr("Planche manquante : ", file)
					failed += 1
	failed += _check_heros()
	failed += _check_talents()
	failed += _check_npcs()
	failed += _check_monsters()
	failed += _check_objets()
	if not ResourceLoader.exists(Braise.DIR + "braise.png"):
		printerr("Braise n'a pas de planche")
		failed += 1
	print("Vérification du client : %s" % ("OK" if failed == 0 else "%d erreur(s)" % failed))
	get_tree().quit(1 if failed > 0 else 0)


## Acte IV : les 15 peuples, les 17 classes et chaque pièce que le miroir
## peut demander existent ; la répartition conseillée place tous les points.
func _check_talents() -> int:
	var failed := 0
	for c in Heros.data().classes:
		for v in 3:
			var tree := Heros.arbre(c.id, v)
			if tree.size() != 10:
				printerr("Arbre de talents incomplet : ", c.id, " ", v)
				failed += 1
			var t := Heros.talents_conseilles(c.id, v, 50)
			if Heros.points_places(t) != Heros.points_de_talent(50):
				printerr("Talents conseillés incomplets : ", c.id, " ", v)
				failed += 1
	# La fiche s'affiche avec l'onglet des talents.
	var sheet := HeroSheet.new()
	sheet.me = {"cl": "mage", "v": 0, "lv": 23, "ta": Heros.talents_conseilles("mage", 0, 20), "at": [], "pt": 110}
	sheet._tab = "talents"
	add_child(sheet)
	if sheet._body.get_child_count() < 10:
		printerr("Onglet des talents vide")
		failed += 1
	sheet.queue_free()
	return failed


func _check_heros() -> int:
	var failed := 0
	var D := Heros.peuples()
	if D.peuples.size() != 15 or Heros.ORDRE_PEUPLES.size() != 15:
		printerr("Il faut 15 peuples")
		failed += 1
	var data := Heros.data()
	if data.classes.size() != 17:
		printerr("Il faut 17 classes")
		failed += 1
	var pm := PeupleSprite.meta()
	for anim in ["repos", "marche", "course", "salut", "attaque", "bouclier", "garde", "incante", "sort", "roulade", "mort", "tir"]:
		if not pm.anims.has(anim):
			printerr("Animation des peuples manquante : ", anim)
			failed += 1
	for c in data.classes:
		if c.competences.size() != 7 or c.voies.size() != 3:
			printerr("Classe incomplète : ", c.id)
			failed += 1
		if HeroSheet._sum(HeroSheet.conseil(c.conseil, 45)) != 45:
			printerr("Répartition conseillée incomplète : ", c.id)
			failed += 1
		if not pm.anims.has(str(c.attaque)):
			printerr("Animation d'attaque inconnue : ", c.attaque)
			failed += 1
		for sk in c.competences:
			if str(sk.get("icone", "")) == "":
				printerr("Compétence sans icône : ", sk.id)
				failed += 1
	var seen := {}
	for r in Heros.ORDRE_PEUPLES:
		if D.peuples[r].peaux.size() != 24:
			printerr("24 teintes de peau attendues : ", r)
			failed += 1
		for c in data.classes:
			for n in 3:
				var lk := {"g": n % 3, "a": n % 3, "t": n, "k": n % 3, "v": n * 5, "p": n * 7, "y": n, "h": n * 6,
					"hc": n, "b": n - 1, "m": [[n, n], [n + 1, 2]], "tu": n}
				for couche in Heros.couches(r, c.id, lk):
					seen[couche[0]] = true
	for piece in seen:
		if not pm.pieces.has(piece) or not ResourceLoader.exists(PeupleSprite.DIR + piece + ".png"):
			printerr("Pièce du miroir manquante : ", piece)
			failed += 1
	return failed


## Chaque habitant du serveur a sa tenue et ses dialogues côté client, et
## chaque choix mène à un nœud qui existe.
func _check_npcs() -> int:
	var failed := 0
	var server_path := ProjectSettings.globalize_path("res://").path_join("../server/world/maps/glennmor.pnj.json")
	var server: Dictionary = JSON.parse_string(FileAccess.get_file_as_string(server_path))
	var data := DialogueBox.data("glennmor")
	for n in server.pnj:
		var who: Dictionary = data.get("pnj", {}).get(n.id, {})
		if who.is_empty():
			printerr("Habitant sans dialogues : ", n.id)
			failed += 1
			continue
		if not Heros.peuples().peuples.has(str(who.get("peuple", {}).get("race", ""))):
			printerr("Habitant sans peuple : ", n.id)
			failed += 1
			continue
		for c in Heros.habitant(who.peuple):
			if not PeupleSprite.has_piece(c[0]):
				printerr("Allure de %s : planche manquante %s" % [n.id, c[0]])
				failed += 1
		var tree: Dictionary = who.dialogue
		if not tree.has("debut") or who.get("bulles", []).is_empty():
			printerr("%s n'a pas de début de dialogue ou de bulles" % n.id)
			failed += 1
		for key in tree:
			for ch in tree[key].get("choix", []):
				if ch.has("action"):
					if not ch.action in ["boutique", "maitre", "coffre"]:
						printerr("Dialogue de %s : action inconnue %s" % [n.id, ch.action])
						failed += 1
				elif ch.vers != "" and not tree.has(ch.vers):
					printerr("Dialogue de %s : « %s » mène nulle part (%s)" % [n.id, ch.texte, ch.vers])
					failed += 1
	return failed


## Chaque sorte de monstre du serveur a sa planche, avec toutes ses animations.
func _check_monsters() -> int:
	var failed := 0
	var m := MonsterSprite.meta()
	for anim in ["repos", "marche", "course", "attaque", "mort"]:
		if not m.anims.has(anim):
			printerr("Animation de monstre manquante : ", anim)
			failed += 1
	var src := FileAccess.get_file_as_string(ProjectSettings.globalize_path("res://").path_join("../server/world/monster.go"))
	var re := RegEx.create_from_string('\n\t"([a-z_]+)": \\{ID:')
	var n := 0
	for hit in re.search_all(src):
		n += 1
		if not MonsterSprite.has_kind(hit.get_string(1)):
			printerr("Monstre sans planche : ", hit.get_string(1))
			failed += 1
	if n < 5:
		printerr("Bestiaire du serveur illisible (%d sortes trouvées)" % n)
		failed += 1
	return failed


## Acte V : les points de récolte et les ateliers ont leur image, les pièces
## d'armure existent, la monnaie s'écrit bien, un héros équipé s'habille.
func _check_objets() -> int:
	var failed := 0
	var meta = JSON.parse_string(FileAccess.get_file_as_string(MondeObjets.ASSETS + "metiers.json"))
	for n in ["filon_cuivre", "filon_fer", "sauge", "lin", "lysimaque", "chardon_gris", "fleur_de_lune", "chene", "arbre_gris",
			"forge", "joaillerie", "alchimie", "couture", "cuir", "cuisine"]:
		if not (meta is Dictionary and meta.has(n)) or not ResourceLoader.exists(MondeObjets.ASSETS + n + ".png"):
			printerr("Image de métier manquante : ", n)
			failed += 1
	for c in ["mince", "normale", "costaud"]:
		for p in ["plastron_", "epaulieres_"]:
			if not PeupleSprite.has_piece(p + c):
				printerr("Pièce d'armure manquante : ", p + c)
				failed += 1
	if not PeupleSprite.has_piece("capuche"):
		printerr("Pièce d'armure manquante : capuche")
		failed += 1
	if Objets.monnaie(12345) != "1 or 23 ar 45 cu" or Objets.monnaie(5, true) != "5 cuivre":
		printerr("Monnaie mal écrite : ", Objets.monnaie(12345))
		failed += 1
	var eq := {"tete": "tete_plaques:3", "epaules": "epaules_mailles:1", "torse": "torse_cuir:2", "mains": "mains_tissu:0",
		"jambes": "jambes_plaques:0", "pieds": "pieds_cuir:0", "dos": "dos_cape:4", "arme": "arme_epee:2", "main_gauche": "mg_bouclier:1"}
	var pieces := Heros.couches("humain", "guerrier", {}, eq).map(func(e): return e[0])
	for p in ["casque", "epaulieres_normale", "plastron_normale", "gants_normale", "cape_normale", "arme_epee", "arme_bouclier"]:
		if not pieces.has(p):
			printerr("Héros équipé : il manque ", p)
			failed += 1
	# Les fenêtres se dessinent avec des données comme celles du serveur (listes vides en null comprises).
	var it := {"id": "1", "b": "tete_plaques", "nom": "Heaume", "ic": "🪖", "k": "equipement", "sl": "tete", "ar": "plaques", "lv": 3, "r": 2,
		"a": [["force", 2, "+2 Force"]], "arm": 1.5, "d": 80, "px": 12, "ok": true, "st": 2, "sc": 3.0}
	var sac := {"sac": [it, null, {"id": "2", "b": "potion", "nom": "Potion", "ic": "🧪", "k": "provision", "n": 3, "u": true, "r": 0, "px": 30}],
		"eq": {"torse": it}, "me": {"minage": {"lv": 3, "xp": 4, "xpn": 22}, "couture": {"lv": 0}}, "art": null, "rap": ["", "potion"],
		"po": "potion", "bu": {"repas": {"b": "ragout", "nom": "Ragoût", "ic": "🍲", "t": 600}}, "or": 12345, "rep": 40, "arm": 9.1, "pw": 9.2}
	var win := SacWindow.new()
	win.data = sac
	add_child(win)
	for tab in ["sac", "metiers"]:
		win._tab = tab
		win._sel = {"sac": 0}
		win._render()
	win.set_ctx("boutique")
	win.queue_free()
	var rec := {"id": "forge:tete_plaques:1", "nom": "Heaume", "ic": "🪖", "r": 1, "lv": 1, "req": 1, "n": 1, "need": [["Lingot", "🟧", 2, 4]], "ok": false, "eq": true, "sl": "tete", "ar": "plaques", "pour_moi": true, "px": 20}
	for d in [{"p": "boutique", "articles": [{"b": "potion", "nom": "Potion", "ic": "🧪", "px": 120, "it": it}], "rachat": null},
			{"p": "atelier", "m": "forge", "nom": "Forge", "ic": "⚒️", "lv": 1, "connu": true, "recettes": [rec], "choix": [["force", "Force"]]},
			{"p": "maitre", "npc": "doran", "metiers": [{"id": "forge", "nom": "Forge", "ic": "⚒️", "recolte": false, "lv": 1, "connu": true, "recettes": [rec]},
				{"id": "minage", "nom": "Minage", "ic": "⛏️", "recolte": true, "lv": 1, "connu": true, "recettes": null}], "art": ["forge"], "max": 2, "rep": 0, "forge": true},
			{"p": "coffre", "it": [null, it], "or": 500},
			{"p": "miroir", "apps": [{"lk": "tete_plaques:3", "nom": "Heaume", "ic": "🪖", "r": 3, "sl": "tete"}], "px": 25}]:
		var pn := Panneau.new()
		pn.sac = sac
		add_child(pn)
		pn.set_data(d)
		pn.queue_free()
	var ech := EchangeWindow.new()
	add_child(ech)
	ech.set_view({"k": "maj", "moi": {"s": 1, "n": "A", "it": null, "or": 0, "ok": false}, "lui": {"s": 2, "n": "B", "it": [it], "or": 500, "ok": true}})
	ech.queue_free()
	if Heros.couches("humain", "guerrier", {}, {}).map(func(e): return e[0]).has("arme_epee"):
		printerr("Héros sans arme : il ne devrait pas en porter")
		failed += 1
	return failed
