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
	var map = JSON.parse_string(FileAccess.get_file_as_string("res://data/maps/clairiere.json"))
	if not (map is Dictionary) or map.rows.size() != int(map.height):
		printerr("Carte de la Clairière illisible")
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
	for anim in ["repos", "marche", "course", "salut"]:
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
	print("Vérification du client : %s" % ("OK" if failed == 0 else "%d erreur(s)" % failed))
	get_tree().quit(1 if failed > 0 else 0)
