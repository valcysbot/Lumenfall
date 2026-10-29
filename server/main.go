// Module serveur de Lumenfall, chargé par Nakama comme greffon Go.
//
// Il ajoute à Nakama : la connexion Discord, la gestion des héros et la
// simulation du monde (déplacements, chat de proximité), toujours décidée
// par le serveur pour empêcher la triche.
package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

// Config est lue dans les variables runtime.env de la configuration Nakama.
type Config struct {
	Env                 string // "dev", "test" ou "prod"
	AllowTestLogin      bool   // connexion sans Discord (serveur de test uniquement)
	PublicURL           string // adresse publique du serveur, ex. https://jeu.lumenfall.fr
	DiscordClientID     string
	DiscordClientSecret string
	DiscordAPI          string // remplaçable dans les tests
}

func loadConfig(ctx context.Context) Config {
	env, _ := ctx.Value(runtime.RUNTIME_CTX_ENV).(map[string]string)
	get := func(k, def string) string {
		if v, ok := env[k]; ok && v != "" {
			return v
		}
		return def
	}
	c := Config{
		Env:                 get("LF_ENV", "dev"),
		PublicURL:           get("LF_PUBLIC_URL", "http://127.0.0.1:7350"),
		DiscordClientID:     get("LF_DISCORD_CLIENT_ID", ""),
		DiscordClientSecret: get("LF_DISCORD_CLIENT_SECRET", ""),
		DiscordAPI:          get("LF_DISCORD_API", "https://discord.com/api/v10"),
	}
	c.AllowTestLogin = get("LF_ALLOW_TEST_LOGIN", "false") == "true" && c.Env != "prod"
	return c
}

func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
	start := time.Now()
	cfg := loadConfig(ctx)

	if err := migrate(ctx, db); err != nil {
		return err
	}

	auth := newAuth(cfg, logger)
	steps := []error{
		initializer.RegisterBeforeAuthenticateDevice(auth.beforeDevice),
		initializer.RegisterBeforeAuthenticateCustom(auth.beforeCustom),
		initializer.RegisterAfterAuthenticateCustom(auth.afterCustom),
		initializer.RegisterHttp("/lumenfall/discord/start", auth.httpStart, "GET"),
		initializer.RegisterHttp("/lumenfall/discord/callback", auth.httpCallback, "GET"),
		initializer.RegisterRpc("hero_list", rpcHeroList),
		initializer.RegisterRpc("hero_create", rpcHeroCreate),
		initializer.RegisterRpc("world_join", rpcWorldJoin),
		initializer.RegisterMatch(worldModule, newWorldMatch),
	}
	for _, err := range steps {
		if err != nil {
			return err
		}
	}

	// Une instance de monde par carte, créée au démarrage du serveur.
	for _, id := range worldMaps {
		matchID, err := nk.MatchCreate(ctx, worldModule, map[string]interface{}{"map": id})
		if err != nil {
			return err
		}
		worlds.set(id, matchID)
		logger.Info("Monde %q ouvert (partie %s)", id, matchID)
	}

	logger.Info("Lumenfall chargé en %v (env=%s, connexion de test=%v, Discord configuré=%v)",
		time.Since(start), cfg.Env, cfg.AllowTestLogin, cfg.DiscordClientID != "")
	return nil
}
