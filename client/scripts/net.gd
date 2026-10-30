extends Node
## Lien avec le serveur Lumenfall (Nakama) : connexion, héros, monde.

signal match_state(op_code: int, data: Dictionary)
signal disconnected(reason: String)
signal discord_waiting(url: String)

const OP_MOVE_DIR := 1
const OP_MOVE_TO := 2
const OP_CHAT := 3
const OP_EMOTE := 4
const OP_SNAPSHOT := 10
const OP_DELTA := 11
const OP_JOIN := 12
const OP_LEAVE := 13
const OP_CHAT_OUT := 14
const OP_NOTICE := 15
const OP_EMOTE_OUT := 16
const OP_WEATHER := 17

const SESSION_PATH := "user://session.cfg"

var config := {}
var client: NakamaClient
var session: NakamaSession
var socket: NakamaSocket
var match_id := ""
var display_name := ""
var _discord_cancel := false


func _ready() -> void:
	config = _load_config()
	client = Nakama.create_client(config.server_key, config.host, int(config.port), config.scheme, 10, NakamaLogger.LOG_LEVEL.ERROR)


func _load_config() -> Dictionary:
	var cfg := {"scheme": "http", "host": "127.0.0.1", "port": 7350, "server_key": "lumenfall-dev-key", "test_login": false}
	for path in ["res://config.json", "user://config.json"]:
		if FileAccess.file_exists(path):
			var parsed = JSON.parse_string(FileAccess.get_file_as_string(path))
			if parsed is Dictionary:
				cfg.merge(parsed, true)
	return cfg


func server_url() -> String:
	var default_port: int = 443 if config.scheme == "https" else 80
	var port := "" if int(config.port) == default_port else ":%d" % int(config.port)
	return "%s://%s%s" % [config.scheme, config.host, port]


# --- Connexion -------------------------------------------------------------

## Reprend la session enregistrée si elle est encore valable.
func resume() -> bool:
	var cfg := ConfigFile.new()
	if cfg.load(SESSION_PATH) != OK:
		return false
	var s: NakamaSession = NakamaClient.restore_session(cfg.get_value("session", "token", ""))
	if s == null or s.is_exception() or s.would_expire_in(600):
		return false
	session = s
	display_name = cfg.get_value("session", "name", "")
	return true


func _remember(name: String) -> void:
	display_name = name
	var cfg := ConfigFile.new()
	cfg.set_value("session", "token", session.token)
	cfg.set_value("session", "name", name)
	cfg.save(SESSION_PATH)


func logout() -> void:
	leave_world()
	session = null
	DirAccess.remove_absolute(ProjectSettings.globalize_path(SESSION_PATH))


## Connexion de test (refusée par le vrai serveur).
func login_test(device_id: String) -> String:
	var s := await client.authenticate_device_async(device_id, null, true)
	if s.is_exception():
		return _error_text(s.get_exception())
	session = s
	_remember("Testeur")
	return ""


## Connexion Discord : ouvre le navigateur puis attend que Discord réponde.
func login_discord() -> String:
	_discord_cancel = false
	var state := _random_state()
	var url := server_url() + "/lumenfall/discord/start?state=" + state
	OS.shell_open(url)
	discord_waiting.emit(url)
	var deadline := Time.get_ticks_msec() + 5 * 60 * 1000
	while Time.get_ticks_msec() < deadline and not _discord_cancel:
		await get_tree().create_timer(2.0).timeout
		if _discord_cancel:
			break
		var s := await client.authenticate_custom_async("discord-state:" + state, null, true)
		if not s.is_exception():
			session = s
			var name := await _account_name()
			_remember(name)
			return ""
		var e := s.get_exception()
		if e.grpc_status_code != 9:  # 9 = Discord n'a pas encore répondu
			return _error_text(e)
	return "La connexion Discord a été annulée." if _discord_cancel else "Discord n'a pas répondu à temps. On réessaie ?"


func cancel_discord() -> void:
	_discord_cancel = true


func _account_name() -> String:
	var acc = await client.get_account_async(session)
	if acc.is_exception():
		return ""
	return acc.user.display_name if acc.user.display_name != "" else acc.user.username


func _random_state() -> String:
	var chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"
	var crypto := Crypto.new()
	var bytes := crypto.generate_random_bytes(40)
	var out := ""
	for b in bytes:
		out += chars[b % chars.length()]
	return out


# --- Héros -----------------------------------------------------------------

func _rpc(id: String, payload: Dictionary) -> Dictionary:
	var res = await client.rpc_async(session, id, JSON.stringify(payload))
	if res.is_exception():
		return {"error": _error_text(res.get_exception())}
	var data = JSON.parse_string(res.payload)
	return data if data is Dictionary else {"error": "Réponse illisible du serveur."}


func hero_list() -> Dictionary:
	return await _rpc("hero_list", {})


func hero_create(slot: int, hero_name: String, color: int) -> Dictionary:
	return await _rpc("hero_create", {"slot": slot, "name": hero_name, "color": color})


# --- Monde -----------------------------------------------------------------

func join_world(hero_id: String) -> String:
	var r := await _rpc("world_join", {"hero_id": hero_id, "v": ProjectSettings.get_setting("application/config/version")})
	if r.has("error"):
		return r.error
	if socket == null or not socket.is_connected_to_host():
		socket = Nakama.create_socket_from(client)
		socket.received_match_state.connect(_on_match_state)
		socket.closed.connect(func(): disconnected.emit("La connexion au serveur s'est fermée."))
		var c = await socket.connect_async(session)
		if c.is_exception():
			return _error_text(c.get_exception())
	var m = await socket.join_match_async(r.match_id, {"hero_id": hero_id})
	if m.is_exception():
		return _error_text(m.get_exception())
	match_id = r.match_id
	return ""


func leave_world() -> void:
	if socket != null and match_id != "":
		socket.leave_match_async(match_id)
	match_id = ""


func send(op: int, data: Dictionary) -> void:
	if socket != null and match_id != "":
		socket.send_match_state_async(match_id, op, JSON.stringify(data))


func _on_match_state(state) -> void:
	var data = JSON.parse_string(state.data)
	if data is Dictionary:
		match_state.emit(state.op_code, data)


# --- Erreurs ----------------------------------------------------------------

## Transforme une erreur technique en message lisible (et un peu drôle).
func _error_text(e: NakamaException) -> String:
	if e.status_code == -1 or e.status_code == 0 or e.status_code >= 500 and e.message == "":
		return "Le serveur ne répond pas. Braise souffle sur les braises du serveur… Réessaie dans un instant."
	if e.message != "":
		return e.message
	return "Une erreur mystérieuse est survenue (code %d)." % e.status_code
