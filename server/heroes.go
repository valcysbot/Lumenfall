package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
}

func userID(ctx context.Context) (string, error) {
	id, _ := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
	if id == "" {
		return "", runtime.NewError("Il faut être connecté.", codeUnauthenticated)
	}
	return id, nil
}

func listHeroes(ctx context.Context, db *sql.DB, uid string) ([]Hero, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, slot, name, color, map, x, y FROM lf_heroes WHERE user_id = $1 ORDER BY slot`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	heroes := []Hero{}
	for rows.Next() {
		var h Hero
		if err := rows.Scan(&h.ID, &h.Slot, &h.Name, &h.Color, &h.Map, &h.X, &h.Y); err != nil {
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
	out, _ := json.Marshal(map[string]interface{}{"heroes": heroes, "slots": world.MaxHeroes, "colors": world.HeroColors})
	return string(out), nil
}

func rpcHeroCreate(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	uid, err := userID(ctx)
	if err != nil {
		return "", err
	}
	var in struct {
		Slot  int    `json:"slot"`
		Name  string `json:"name"`
		Color int    `json:"color"`
	}
	if err := json.Unmarshal([]byte(payload), &in); err != nil {
		return "", runtime.NewError("Demande illisible.", codeInvalidArgument)
	}
	if in.Slot < 0 || in.Slot >= world.MaxHeroes {
		return "", runtime.NewError("Emplacement de héros inconnu.", codeInvalidArgument)
	}
	if in.Color < 0 || in.Color >= len(world.HeroColors) {
		return "", runtime.NewError("Couleur inconnue.", codeInvalidArgument)
	}
	name := world.NormalizeName(in.Name)
	if err := world.ValidateName(name); err != nil {
		return "", runtime.NewError(err.Error(), codeInvalidArgument)
	}
	m := worldMapsByID[defaultMap]
	var h Hero
	err = db.QueryRowContext(ctx, `
INSERT INTO lf_heroes (user_id, slot, name, name_key, color, map, x, y)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, slot, name, color, map, x, y`,
		uid, in.Slot, name, world.NameKey(name), in.Color, m.ID, m.Spawn[0], m.Spawn[1],
	).Scan(&h.ID, &h.Slot, &h.Name, &h.Color, &h.Map, &h.X, &h.Y)
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

var errHeroNotFound = errors.New("héros introuvable")

// loadHero lit un héros en vérifiant qu'il appartient bien à l'utilisateur.
func loadHero(ctx context.Context, db *sql.DB, uid, heroID string) (Hero, error) {
	var h Hero
	err := db.QueryRowContext(ctx, `SELECT id, slot, name, color, map, x, y FROM lf_heroes WHERE id = $1 AND user_id = $2`, heroID, uid).
		Scan(&h.ID, &h.Slot, &h.Name, &h.Color, &h.Map, &h.X, &h.Y)
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
		HeroID string `json:"hero_id"`
	}
	if err := json.Unmarshal([]byte(payload), &in); err != nil || in.HeroID == "" {
		return "", runtime.NewError("Demande illisible.", codeInvalidArgument)
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

// saveHeroes enregistre la position de plusieurs héros en une requête.
func saveHeroes(ctx context.Context, db *sql.DB, mapID string, ps []heroPos) error {
	if len(ps) == 0 {
		return nil
	}
	ids := make([]string, len(ps))
	xs := make([]float64, len(ps))
	ys := make([]float64, len(ps))
	for i, p := range ps {
		ids[i], xs[i], ys[i] = p.id, p.x, p.y
	}
	idsJSON, _ := json.Marshal(ids)
	xsJSON, _ := json.Marshal(xs)
	ysJSON, _ := json.Marshal(ys)
	_, err := db.ExecContext(ctx, `
UPDATE lf_heroes h SET map = $1, x = v.x, y = v.y, updated_at = now()
FROM (
	SELECT * FROM unnest(
		ARRAY(SELECT jsonb_array_elements_text($2::jsonb))::uuid[],
		ARRAY(SELECT jsonb_array_elements_text($3::jsonb))::real[],
		ARRAY(SELECT jsonb_array_elements_text($4::jsonb))::real[]
	) AS t(id, x, y)
) v
WHERE h.id = v.id`, mapID, string(idsJSON), string(xsJSON), string(ysJSON))
	return err
}

type heroPos struct {
	id   string
	x, y float64
}
