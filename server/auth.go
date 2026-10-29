package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Connexion avec Discord, sans serveur web à part :
//
//  1. Le jeu tire un code secret au hasard (state) et ouvre le navigateur sur
//     /lumenfall/discord/start?state=…, qui redirige vers Discord.
//  2. Discord renvoie le navigateur sur /lumenfall/discord/callback. Le serveur
//     échange le code contre l'identité Discord et la garde 10 minutes,
//     rangée sous le state.
//  3. Pendant ce temps, le jeu tente toutes les 2 secondes une connexion
//     Nakama « custom » avec l'identifiant "discord-state:<state>". Le crochet
//     beforeCustom remplace ce state par l'identité Discord vérifiée.
//
// Un client ne peut donc jamais se connecter en se prétendant quelqu'un
// d'autre : seul un state validé par Discord ouvre un compte.

const (
	statePrefix  = "discord-state:"
	pendingTTL   = 10 * time.Minute
	codePending  = 9 // FAILED_PRECONDITION : « Discord n'a pas encore répondu »
	discordScope = "identify"
)

var stateRe = regexp.MustCompile(`^[A-Za-z0-9_-]{24,64}$`)

type discordUser struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
	Avatar     string `json:"avatar"`
}

type pendingLogin struct {
	user    discordUser
	expires time.Time
}

type auth struct {
	cfg    Config
	logger runtime.Logger
	client *http.Client

	mu      sync.Mutex
	pending map[string]pendingLogin
}

func newAuth(cfg Config, logger runtime.Logger) *auth {
	return &auth{cfg: cfg, logger: logger, client: &http.Client{Timeout: 10 * time.Second}, pending: map[string]pendingLogin{}}
}

func (a *auth) redirectURI() string {
	return strings.TrimRight(a.cfg.PublicURL, "/") + "/lumenfall/discord/callback"
}

// beforeDevice n'autorise la connexion de test que sur le serveur de test.
func (a *auth) beforeDevice(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, in *api.AuthenticateDeviceRequest) (*api.AuthenticateDeviceRequest, error) {
	if !a.cfg.AllowTestLogin {
		return nil, runtime.NewError("La connexion de test est fermée sur ce serveur. Connecte-toi avec Discord.", codePermission)
	}
	return in, nil
}

func (a *auth) beforeCustom(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, in *api.AuthenticateCustomRequest) (*api.AuthenticateCustomRequest, error) {
	id := in.GetAccount().GetId()
	if !strings.HasPrefix(id, statePrefix) {
		return nil, runtime.NewError("Méthode de connexion inconnue.", codePermission)
	}
	u, ok := a.take(strings.TrimPrefix(id, statePrefix))
	if !ok {
		return nil, runtime.NewError("En attente de Discord…", codePending)
	}
	in.Account.Id = "discord:" + u.ID
	in.Username = "" // Nakama choisit un identifiant interne ; le nom affiché vient de Discord.
	in.Create = wrapBool(true)
	if in.Account.Vars == nil {
		in.Account.Vars = map[string]string{}
	}
	in.Account.Vars["discord_name"] = u.displayName()
	in.Account.Vars["discord_avatar"] = u.avatarURL()
	return in, nil
}

func (a *auth) afterCustom(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, out *api.Session, in *api.AuthenticateCustomRequest) error {
	uid := tokenUserID(out.GetToken())
	if uid == "" {
		return nil
	}
	vars := in.GetAccount().GetVars()
	if err := nk.AccountUpdateId(ctx, uid, "", nil, vars["discord_name"], "", "", "", vars["discord_avatar"]); err != nil {
		logger.Warn("Mise à jour du profil Discord de %s: %v", uid, err)
	}
	return nil
}

// tokenUserID lit l'identifiant d'utilisateur dans un jeton de session
// fraîchement créé par Nakama (pas besoin de vérifier sa signature ici).
func tokenUserID(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		UID string `json:"uid"`
	}
	_ = json.Unmarshal(raw, &claims)
	return claims.UID
}

func (a *auth) httpStart(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if !stateRe.MatchString(state) {
		page(w, http.StatusBadRequest, "😵 Lien abîmé", "Ce lien de connexion est incomplet. Relance la connexion depuis le jeu.")
		return
	}
	if a.cfg.DiscordClientID == "" {
		page(w, http.StatusServiceUnavailable, "🔧 Pas encore prêt", "La connexion Discord n'est pas encore configurée sur ce serveur.")
		return
	}
	q := url.Values{
		"response_type": {"code"},
		"client_id":     {a.cfg.DiscordClientID},
		"scope":         {discordScope},
		"state":         {state},
		"redirect_uri":  {a.redirectURI()},
		"prompt":        {"none"},
	}
	http.Redirect(w, r, "https://discord.com/oauth2/authorize?"+q.Encode(), http.StatusFound)
}

func (a *auth) httpCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	state, code := q.Get("state"), q.Get("code")
	if q.Get("error") != "" {
		page(w, http.StatusOK, "🚪 Connexion annulée", "Pas de souci ! Retourne dans Lumenfall pour réessayer quand tu veux.")
		return
	}
	if !stateRe.MatchString(state) || code == "" {
		page(w, http.StatusBadRequest, "😵 Lien abîmé", "Discord ne nous a pas tout renvoyé. Relance la connexion depuis le jeu.")
		return
	}
	u, err := a.exchange(r.Context(), code)
	if err != nil {
		a.logger.Warn("Échange Discord échoué: %v", err)
		page(w, http.StatusBadGateway, "🌩 Discord ne répond pas", "Le sceau a résisté. Relance la connexion depuis le jeu dans un instant.")
		return
	}
	a.put(state, u)
	page(w, http.StatusOK, "🐉 Le sceau est brisé !",
		fmt.Sprintf("Bienvenue, %s ! Tu peux fermer cette page et retourner dans Lumenfall.", html.EscapeString(u.displayName())))
}

func (a *auth) exchange(ctx context.Context, code string) (discordUser, error) {
	var u discordUser
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {a.redirectURI()}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.DiscordAPI+"/oauth2/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(a.cfg.DiscordClientID, a.cfg.DiscordClientSecret)
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := a.doJSON(req, &tok); err != nil {
		return u, fmt.Errorf("jeton: %w", err)
	}
	if tok.AccessToken == "" {
		return u, fmt.Errorf("jeton vide")
	}
	req, _ = http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.DiscordAPI+"/users/@me", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	if err := a.doJSON(req, &u); err != nil {
		return u, fmt.Errorf("profil: %w", err)
	}
	if u.ID == "" {
		return u, fmt.Errorf("profil sans identifiant")
	}
	return u, nil
}

func (a *auth) doJSON(req *http.Request, out interface{}) error {
	res, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", res.StatusCode)
	}
	return json.Unmarshal(body, out)
}

func (a *auth) put(state string, u discordUser) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	for k, v := range a.pending {
		if now.After(v.expires) {
			delete(a.pending, k)
		}
	}
	a.pending[state] = pendingLogin{user: u, expires: now.Add(pendingTTL)}
}

// take renvoie l'identité rangée sous un state, une seule fois.
func (a *auth) take(state string) (discordUser, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	p, ok := a.pending[state]
	if !ok || time.Now().After(p.expires) {
		return discordUser{}, false
	}
	delete(a.pending, state)
	return p.user, true
}

func (u discordUser) displayName() string {
	if u.GlobalName != "" {
		return u.GlobalName
	}
	return u.Username
}

func (u discordUser) avatarURL() string {
	if u.Avatar == "" {
		return ""
	}
	return "https://cdn.discordapp.com/avatars/" + u.ID + "/" + u.Avatar + ".png"
}

func page(w http.ResponseWriter, status int, title, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<!doctype html><html lang="fr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Lumenfall</title>
<style>body{margin:0;min-height:100vh;display:grid;place-items:center;background:#15131d;color:#efe6d6;font:18px/1.5 Georgia,serif;text-align:center;padding:24px;box-sizing:border-box}h1{color:#e8c66a;font-weight:normal}p{max-width:30em}</style>
</head><body><main><h1>%s</h1><p>%s</p></main></body></html>`, html.EscapeString(title), msg)
}

func wrapBool(v bool) *wrapperspb.BoolValue { return wrapperspb.Bool(v) }
