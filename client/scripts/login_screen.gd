extends Control
## Écran titre : connexion avec Discord (ou connexion de test sur le serveur de test).

signal logged_in

var status: Label
var discord_btn: Button
var test_btn: Button
var cancel_btn: Button
var braise: HBoxContainer


func _init() -> void:
	set_anchors_preset(Control.PRESET_FULL_RECT)
	theme = UI.make_theme()
	add_child(UI.backdrop())

	var center := CenterContainer.new()
	center.set_anchors_preset(Control.PRESET_FULL_RECT)
	add_child(center)
	var col := VBoxContainer.new()
	col.custom_minimum_size = Vector2(420, 0)
	col.add_theme_constant_override("separation", 14)
	center.add_child(col)

	var title := UI.label("Lumenfall", 72, UI.LUMEN)
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	col.add_child(title)
	var sub := UI.label("les Chroniques d'Erydor", 24, UI.MUTED)
	sub.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	col.add_child(sub)
	col.add_child(_spacer(18))

	discord_btn = UI.button("Se connecter avec Discord", true)
	discord_btn.pressed.connect(_on_discord)
	col.add_child(discord_btn)

	cancel_btn = UI.button("Annuler")
	cancel_btn.visible = false
	cancel_btn.pressed.connect(func(): Net.cancel_discord())
	col.add_child(cancel_btn)

	test_btn = UI.button("Connexion de test (serveur de test)")
	test_btn.visible = bool(Net.config.get("test_login", false))
	test_btn.pressed.connect(_on_test)
	col.add_child(test_btn)

	status = UI.label("", 17, UI.MUTED)
	status.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	col.add_child(status)
	col.add_child(_spacer(8))

	braise = UI.braise(UI.TIPS[randi() % UI.TIPS.size()])
	col.add_child(braise)

	var version := UI.label("V%s · Acte I, les fondations" % ProjectSettings.get_setting("application/config/version"), 14, UI.MUTED)
	version.autowrap_mode = TextServer.AUTOWRAP_OFF
	version.set_anchors_and_offsets_preset(Control.PRESET_BOTTOM_RIGHT, Control.PRESET_MODE_MINSIZE, 12)
	version.grow_horizontal = Control.GROW_DIRECTION_BEGIN
	version.grow_vertical = Control.GROW_DIRECTION_BEGIN
	add_child(version)

	Net.discord_waiting.connect(func(_url): set_status("Ton navigateur s'est ouvert sur Discord. Accepte, puis reviens ici…"))


func _spacer(h: int) -> Control:
	var c := Control.new()
	c.custom_minimum_size = Vector2(0, h)
	return c


func set_status(text: String, is_error: bool = false) -> void:
	status.text = text
	status.add_theme_color_override("font_color", UI.DANGER if is_error else UI.MUTED)


func _busy(on: bool, discord: bool = false) -> void:
	discord_btn.disabled = on
	test_btn.disabled = on
	cancel_btn.visible = on and discord


func _on_discord() -> void:
	_busy(true, true)
	set_status("Ouverture du portail de Discord…")
	UI.set_braise(braise, "Braise dit : je garde la porte. Seuls ton identifiant, ton pseudo et ton avatar Discord entrent à Erydor.")
	var err := await Net.login_discord()
	_busy(false)
	if err != "":
		set_status(err, true)
		return
	logged_in.emit()


func _on_test() -> void:
	_busy(true)
	set_status("Connexion de test…")
	var id := "test-" + OS.get_unique_id().sha256_text().substr(0, 24)
	var err := await Net.login_test(id)
	_busy(false)
	if err != "":
		set_status(err, true)
		return
	logged_in.emit()
