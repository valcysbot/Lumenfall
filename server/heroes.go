package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/valcysbot/lumenfall/server/world"
)

// Codes d'erreur gRPC renvoyés au client.
const (
	codeInvalidArgument = 3
	codeNotFound        = 5
	codeAlreadyExists   = 6
	codePermission      = 7
	codeFailedPrecond   = 9
	codeUnauthenticated = 16
	codeInternal        = 13
)

func migrate(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS lf_heroes (
	id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	slot       SMALLINT NOT NULL CHECK (slot BETWEEN 0 AND 2),
	name       TEXT NOT NULL,
	name_key   TEXT NOT NULL UNIQUE,
	color      SMALLINT NOT NULL,
	map        TEXT NOT NULL,
	x          REAL NOT NULL,
	y          REAL NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (user_id, slot)
)`)
	if err != nil {
		return err
	}
	// Acte III : niveau, expérience, or, potions et trophées.
	_, err = db.ExecContext(ctx, `
ALTER TABLE lf_heroes ADD COLUMN IF NOT EXISTS level    SMALLINT NOT NULL DEFAULT 1;
ALTER TABLE lf_heroes ADD COLUMN IF NOT EXISTS xp       INTEGER  NOT NULL DEFAULT 0;
ALTER TABLE lf_heroes ADD COLUMN IF NOT EXISTS gold     INTEGER  NOT NULL DEFAULT 0;
ALTER TABLE lf_heroes ADD COLUMN IF NOT EXISTS potions  SMALLINT NOT NULL DEFAULT 5;
ALTER TABLE lf_heroes ADD COLUMN IF NOT EXISTS trophies JSONB    NOT NULL DEFAULT '{}'::jsonb`)
	if err != nil {
		return err
	}
	// Acte IV : peuple, classe, voie, dieu, attributs, allure… et le nouveau
	// départ voulu au Conseil des Héros : les héros d'avant l'Acte IV
	// (marqués 3 en ajoutant la colonne) sont effacés une fois pour toutes.
	_, err = db.ExecContext(ctx, `
ALTER TABLE lf_heroes ADD COLUMN IF NOT EXISTS acte     SMALLINT NOT NULL DEFAULT 3;
DELETE FROM lf_heroes WHERE acte < 4;
ALTER TABLE lf_heroes ALTER COLUMN acte SET DEFAULT 4;
ALTER TABLE lf_heroes ADD COLUMN IF NOT EXISTS race     TEXT     NOT NULL DEFAULT 'humain';
ALTER TABLE lf_heroes ADD COLUMN IF NOT EXISTS classe   TEXT     NOT NULL DEFAULT 'paladin';
ALTER TABLE lf_heroes ADD COLUMN IF NOT EXISTS voie     SMALLINT NOT NULL DEFAULT -1;
ALTER TABLE lf_heroes ADD COLUMN IF NOT EXISTS dieu     TEXT     NOT NULL DEFAULT 'solenn';
ALTER TABLE lf_heroes ADD COLUMN IF NOT EXISTS hardcore BOOLEAN  NOT NULL DEFAULT false;
ALTER TABLE lf_heroes ADD COLUMN IF NOT EXISTS attrs    JSONB    NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE lf_heroes ADD COLUMN IF NOT EXISTS quest    SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE lf_heroes ADD COLUMN IF NOT EXISTS look     JSONB    NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE lf_heroes ADD COLUMN IF NOT EXISTS miroir   SMALLINT NOT NULL DEFAULT 0`)
	return err
}

// Hero est un héros tel que le client le voit.
type Hero struct {
	ID    string  `json:"id"`
	Slot  int     `json:"slot"`
	Name  string  `json:"name"`
	Color int     `json:"color"`
	Map   string  `json:"map"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Level int     `json:"level"`

	Race     string      `json:"race"`
	Class    string      `json:"classe"`
	Spec     int         `json:"voie"`
	God      string      `json:"dieu"`
	Hardcore bool        `json:"hardcore"`
	Look     world.Look  `json:"look"`
	attrs    world.Attrs // lus en base, privés
	quest    int
	miroir   int

	xp, gold, potions int
	trophies          map[string]int
}

// heroCols : les colonnes lues pour un héros, dans l'ordre de scanHero.
const heroCols = `id, slot, name, color, map, x, y, level, xp, gold, potions, trophies,
	race, classe, voie, dieu, hardcore, attrs, quest, look, miroir`

func scanHero(row interface{ Scan(...any) error }, h *Hero) error {
	var tro, at, look []byte
	if err := row.Scan(&h.ID, &h.Slot, &h.Name, &h.Color, &h.Map, &h.X, &h.Y, &h.Level, &h.xp, &h.gold, &h.potions, &tro,
		&h.Race, &h.Class, &h.Spec, &h.God, &h.Hardcore, &at, &h.quest, &look, &h.miroir); err != nil {
		return err
	}
	h.trophies = map[string]int{}
	_ = json.Unmarshal(tro, &h.trophies)
	_ = json.Unmarshal(at, &h.attrs)
	h.Look = world.DefaultLook()
	_ = json.Unmarshal(look, &h.Look)
	return nil
}

func userID(ctx context.Context) (string, error) {
	id, _ := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
	if id == "" {
		return "", runtime.NewError("Il faut être connecté.", codeUnauthenticated)
	}
	return id, nil
}

func listHeroes(ctx context.Context, db *sql.DB, uid string) ([]Hero, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+heroCols+` FROM lf_heroes WHERE user_id = $1 ORDER BY slot`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	heroes := []Hero{}
	for rows.Next() {
		var h Hero
		if err := scanHero(rows, &h); err != nil {
			return nil, err
		}
		heroes = append(heroes, h)
	}
	return heroes, rows.Err()
}

func rpcHeroList(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	uid, err := userID(ctx)
	if err != nil {
		return "", err
	}
	heroes, err := listHeroes(ctx, db, uid)
	if err != nil {
		logger.Error("hero_list: %v", err)
		return "", runtime.NewError("Impossible de lire tes héros.", codeInternal)
	}
	out, _ := json.Marshal(map[string]interface{}{"heroes": heroes, "slots": world.MaxHeroes})
	return string(out), nil
}

func rpcHeroCreate(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	uid, err := userID(ctx)
	if err != nil {
		return "", err
	}
	var in heroForm
	if err := json.Unmarshal([]byte(payload), &in); err != nil {
		return "", runtime.NewError("Demande illisible.", codeInvalidArgument)
	}
	if err := in.validate(); err != nil {
		return "", runtime.NewError(err.Error(), codeInvalidArgument)
	}
	name := world.NormalizeName(in.Name)
	if err := world.ValidateName(name); err != nil {
		return "", runtime.NewError(err.Error(), codeInvalidArgument)
	}
	m := worldMapsByID[defaultMap]
	var h Hero
	row := db.QueryRowContext(ctx, `
INSERT INTO lf_heroes (user_id, slot, name, name_key, color, map, x, y, acte, race, classe, dieu, hardcore, look)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 4, $9, $10, $11, $12, $13)
RETURNING `+heroCols,
		uid, in.Slot, name, world.NameKey(name), in.Look.Tunique, m.ID, m.Spawn[0], m.Spawn[1],
		in.Race, in.Class, in.God, in.Hardcore, string(mustJSON(in.Look)),
	)
	err = scanHero(row, &h)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "name_key"):
			return "", runtime.NewError("Ce nom est déjà porté par un autre héros d'Erydor.", codeAlreadyExists)
		case strings.Contains(msg, "user_id_slot") || strings.Contains(msg, "lf_heroes_user_id_slot_key"):
			return "", runtime.NewError("Cet emplacement est déjà occupé.", codeAlreadyExists)
		}
		logger.Error("hero_create: %v", err)
		return "", runtime.NewError("La création du héros a échoué.", codeInternal)
	}
	logger.Info("Nouveau héros %q (utilisateur %s)", h.Name, uid)
	out, _ := json.Marshal(h)
	return string(out), nil
}

// rpcHeroDelete efface un héros à la demande de son joueur. Pour éviter un
// clic malheureux, le client renvoie le nom du héros, tapé à la main.
func rpcHeroDelete(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	uid, err := userID(ctx)
	if err != nil {
		return "", err
	}
	var in struct {
		HeroID string `json:"hero_id"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal([]byte(payload), &in); err != nil || in.HeroID == "" {
		return "", runtime.NewError("Demande illisible.", codeInvalidArgument)
	}
	h, err := loadHero(ctx, db, uid, in.HeroID)
	if err != nil {
		return "", runtime.NewError("Ce héros n'existe pas sur ton compte.", codeNotFound)
	}
	if world.NameKey(world.NormalizeName(in.Name)) != world.NameKey(h.Name) {
		return "", runtime.NewError("Le nom tapé ne correspond pas : le héros est toujours là.", codeInvalidArgument)
	}
	if err := deleteHero(ctx, db, h.ID); err != nil {
		logger.Error("hero_delete: %v", err)
		return "", runtime.NewError("Impossible d'effacer ce héros.", codeInternal)
	}
	logger.Info("Héros %q effacé par son joueur (utilisateur %s)", h.Name, uid)
	return `{"ok":true}`, nil
}

// heroForm : ce que le miroir de création envoie.
type heroForm struct {
	Slot     int        `json:"slot"`
	Name     string     `json:"name"`
	Race     string     `json:"race"`
	Class    string     `json:"classe"`
	God      string     `json:"dieu"`
	Hardcore bool       `json:"hardcore"`
	Look     world.Look `json:"look"`
}

func (in *heroForm) validate() error {
	switch {
	case in.Slot < 0 || in.Slot >= world.MaxHeroes:
		return errors.New("Emplacement de héros inconnu.")
	case world.Races[in.Race].ID == "":
		return errors.New("Ce peuple n'existe pas en Erydor.")
	case world.ClassByID(in.Class) == nil:
		return errors.New("Cette classe n'existe pas.")
	case !world.ValidGod(in.God):
		return errors.New("Ce dieu n'existe pas.")
	}
	if in.Look.Marques == nil {
		in.Look.Marques = [][2]int{}
	}
	return in.Look.Validate()
}

// deleteHero efface un héros (la mort hardcore, ou à la demande du joueur).
func deleteHero(ctx context.Context, db *sql.DB, heroID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM lf_heroes WHERE id = $1`, heroID)
	return err
}

var errHeroNotFound = errors.New("héros introuvable")

// loadHero lit un héros en vérifiant qu'il appartient bien à l'utilisateur.
func loadHero(ctx context.Context, db *sql.DB, uid, heroID string) (Hero, error) {
	var h Hero
	err := scanHero(db.QueryRowContext(ctx, `SELECT `+heroCols+` FROM lf_heroes WHERE id = $1 AND user_id = $2`, heroID, uid), &h)
	if errors.Is(err, sql.ErrNoRows) {
		return h, errHeroNotFound
	}
	// Un héros resté dans une région fermée (la Clairière de l'Acte I)
	// revient au point d'apparition de la région principale.
	if err == nil && worldMapsByID[h.Map] == nil {
		m := worldMapsByID[defaultMap]
		h.Map, h.X, h.Y = m.ID, m.Spawn[0], m.Spawn[1]
	}
	return h, err
}

func rpcWorldJoin(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	uid, err := userID(ctx)
	if err != nil {
		return "", err
	}
	var in struct {
		HeroID  string `json:"hero_id"`
		Version string `json:"v"`
	}
	if err := json.Unmarshal([]byte(payload), &in); err != nil || in.HeroID == "" {
		return "", runtime.NewError("Demande illisible.", codeInvalidArgument)
	}
	// Un jeu trop ancien ne connaît pas les régions actuelles : mieux vaut le
	// dire que d'afficher un monde vide.
	if !versionAtLeast(in.Version, minClientVersion) {
		return "", runtime.NewError("Ton jeu est trop ancien : télécharge la dernière version pour continuer l'aventure !", codeFailedPrecond)
	}
	h, err := loadHero(ctx, db, uid, in.HeroID)
	if err != nil {
		return "", runtime.NewError("Ce héros n'existe pas sur ton compte.", codeNotFound)
	}
	matchID, ok := worlds.get(h.Map)
	if !ok {
		return "", runtime.NewError("Cette région d'Erydor est fermée pour l'instant.", codeFailedPrecond)
	}
	out, _ := json.Marshal(map[string]string{"match_id": matchID, "map": h.Map})
	return string(out), nil
}

// saveHeroes enregistre la position et la progression de plusieurs héros
// en une requête.
func saveHeroes(ctx context.Context, db *sql.DB, mapID string, ps []heroPos) error {
	if len(ps) == 0 {
		return nil
	}
	rows, _ := json.Marshal(ps)
	_, err := db.ExecContext(ctx, `
UPDATE lf_heroes h SET map = $1, x = v.x, y = v.y, level = v.level, xp = v.xp, gold = v.gold,
	potions = v.potions, trophies = v.trophies, voie = v.voie, attrs = v.attrs, quest = v.quest,
	look = v.look, miroir = v.miroir, updated_at = now()
FROM jsonb_to_recordset($2::jsonb) AS v(id uuid, x real, y real, level smallint, xp integer, gold integer,
	potions smallint, trophies jsonb, voie smallint, attrs jsonb, quest smallint, look jsonb, miroir smallint)
WHERE h.id = v.id`, mapID, string(rows))
	return err
}

// heroPos : ce qu'on sauvegarde d'un héros en jeu.
type heroPos struct {
	ID       string         `json:"id"`
	X        float64        `json:"x"`
	Y        float64        `json:"y"`
	Level    int            `json:"level"`
	XP       int            `json:"xp"`
	Gold     int            `json:"gold"`
	Potions  int            `json:"potions"`
	Trophies map[string]int `json:"trophies"`
	Voie     int            `json:"voie"`
	Attrs    world.Attrs    `json:"attrs"`
	Quest    int            `json:"quest"`
	Look     world.Look     `json:"look"`
	Miroir   int            `json:"miroir"`
}

// minClientVersion est la plus ancienne version du jeu acceptée dans le monde.
// À monter quand le serveur envoie quelque chose qu'un ancien jeu ne sait pas
// afficher (une nouvelle région, par exemple).
const minClientVersion = "0.4.0"

// versionAtLeast compare deux versions « 0.2.0 ». Une version absente ou
// illisible compte comme trop ancienne.
func versionAtLeast(v, min string) bool {
	a, b := strings.Split(v, "."), strings.Split(min, ".")
	for i := range b {
		var x, y int
		if i < len(a) {
			n, err := strconv.Atoi(a[i])
			if err != nil {
				return false
			}
			x = n
		}
		y, _ = strconv.Atoi(b[i])
		if x != y {
			return x > y
		}
	}
	return v != ""
}
