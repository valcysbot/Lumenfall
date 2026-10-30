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
	failed += _check_npcs()
	if not ResourceLoader.exists(Braise.DIR + "braise.png"):
		printerr("Braise n'a pas de planche")
		failed += 1
	print("Vérification du client : %s" % ("OK" if failed == 0 else "%d erreur(s)" % failed))
	get_tree().quit(1 if failed > 0 else 0)


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
		for piece in who.look:
			var v: String = who.look[piece]
			var file: String = ("cheveux_" + v) if piece == "cheveux" else (piece + ("_" + v if v != "" else ""))
			if not piece in HeroSprite.ORDRE or not ResourceLoader.exists(HeroSprite.DIR + file + ".png"):
				printerr("Tenue de %s : planche manquante %s" % [n.id, file])
				failed += 1
		var tree: Dictionary = who.dialogue
		if not tree.has("debut") or who.get("bulles", []).is_empty():
			printerr("%s n'a pas de début de dialogue ou de bulles" % n.id)
			failed += 1
		for key in tree:
			for ch in tree[key].get("choix", []):
				if ch.vers != "" and not tree.has(ch.vers):
					printerr("Dialogue de %s : « %s » mène nulle part (%s)" % [n.id, ch.texte, ch.vers])
					failed += 1
	return failed
