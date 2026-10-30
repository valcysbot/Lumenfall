package main

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
)

type nopLogger struct{}

func (nopLogger) Debug(string, ...interface{})                       {}
func (nopLogger) Info(string, ...interface{})                        {}
func (nopLogger) Warn(string, ...interface{})                        {}
func (nopLogger) Error(string, ...interface{})                       {}
func (n nopLogger) WithField(string, interface{}) runtime.Logger     { return n }
func (n nopLogger) WithFields(map[string]interface{}) runtime.Logger { return n }
func (nopLogger) Fields() map[string]interface{}                     { return nil }

const goodState = "abcdefghijklmnopqrstuvwxyz012345"

func fakeDiscord(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			user, pass, _ := r.BasicAuth()
			if user != "cid" || pass != "secret" || r.FormValue("code") != "bon-code" {
				w.WriteHeader(401)
				return
			}
			w.Write([]byte(`{"access_token":"tok"}`))
		case "/users/@me":
			if r.Header.Get("Authorization") != "Bearer tok" {
				w.WriteHeader(401)
				return
			}
			w.Write([]byte(`{"id":"123456789012345678","username":"elowan","global_name":"Elowan","avatar":"abc"}`))
		}
	}))
}

func TestConnexionDiscordComplete(t *testing.T) {
	d := fakeDiscord(t)
	defer d.Close()
	a := newAuth(Config{DiscordClientID: "cid", DiscordClientSecret: "secret", DiscordAPI: d.URL, PublicURL: "http://jeu"}, nopLogger{})

	// Avant le retour de Discord : connexion « en attente ».
	in := &api.AuthenticateCustomRequest{Account: &api.AccountCustom{Id: statePrefix + goodState}}
	if _, err := a.beforeCustom(context.Background(), nil, nil, nil, in); err == nil || !strings.Contains(err.Error(), "attente") {
		t.Fatalf("attendu « en attente », obtenu %v", err)
	}

	// Démarrage : redirection vers Discord avec le bon state.
	rec := httptest.NewRecorder()
	a.httpStart(rec, httptest.NewRequest("GET", "/lumenfall/discord/start?state="+goodState, nil))
	if rec.Code != 302 || !strings.Contains(rec.Header().Get("Location"), "state="+goodState) {
		t.Fatalf("redirection : %d %s", rec.Code, rec.Header().Get("Location"))
	}

	// Retour de Discord.
	rec = httptest.NewRecorder()
	a.httpCallback(rec, httptest.NewRequest("GET", "/lumenfall/discord/callback?code=bon-code&state="+goodState, nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Elowan") {
		t.Fatalf("retour : %d %s", rec.Code, rec.Body.String())
	}

	// Le jeu se connecte : l'identité est celle de Discord.
	out, err := a.beforeCustom(context.Background(), nil, nil, nil, in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Account.Id != "discord:123456789012345678" || out.Account.Vars["discord_name"] != "Elowan" {
		t.Fatalf("identité : %+v", out.Account)
	}
	// Un state ne sert qu'une fois.
	in2 := &api.AuthenticateCustomRequest{Account: &api.AccountCustom{Id: statePrefix + goodState}}
	if _, err := a.beforeCustom(context.Background(), nil, nil, nil, in2); err == nil {
		t.Fatal("un state ne doit servir qu'une fois")
	}
}

func TestPasDUsurpation(t *testing.T) {
	a := newAuth(Config{}, nopLogger{})
	in := &api.AuthenticateCustomRequest{Account: &api.AccountCustom{Id: "discord:123456789012345678"}}
	if _, err := a.beforeCustom(context.Background(), nil, nil, nil, in); err == nil {
		t.Fatal("un identifiant Discord direct doit être refusé")
	}
}

func TestMauvaisCode(t *testing.T) {
	d := fakeDiscord(t)
	defer d.Close()
	a := newAuth(Config{DiscordClientID: "cid", DiscordClientSecret: "secret", DiscordAPI: d.URL}, nopLogger{})
	rec := httptest.NewRecorder()
	a.httpCallback(rec, httptest.NewRequest("GET", "/lumenfall/discord/callback?code=faux&state="+goodState, nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("code %d", rec.Code)
	}
	if _, ok := a.take(goodState); ok {
		t.Fatal("aucune identité ne doit être rangée après un échec")
	}
}

func TestConnexionDeTest(t *testing.T) {
	in := &api.AuthenticateDeviceRequest{Account: &api.AccountDevice{Id: "test-device-0001"}}
	if _, err := newAuth(Config{AllowTestLogin: false}, nopLogger{}).beforeDevice(context.Background(), nil, nil, nil, in); err == nil {
		t.Fatal("la connexion de test doit être fermée par défaut")
	}
	if _, err := newAuth(Config{AllowTestLogin: true}, nopLogger{}).beforeDevice(context.Background(), nil, nil, nil, in); err != nil {
		t.Fatal(err)
	}
}

func TestJetonUtilisateur(t *testing.T) {
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"uid":"u-1"}`))
	if tokenUserID("h."+payload+".s") != "u-1" {
		t.Fatal("uid non lu")
	}
}

func TestChatNettoye(t *testing.T) {
	if got := cleanChat("  Salut​\n  toi  "); got != "Salut toi" {
		t.Fatalf("%q", got)
	}
	if got := cleanChat(strings.Repeat("a", 300)); len([]rune(got)) != chatMaxLen {
		t.Fatalf("longueur %d", len([]rune(got)))
	}
}

func TestVersionDuJeu(t *testing.T) {
	cas := map[string]bool{
		"0.2.0": true, "0.2.1": true, "0.10.0": true, "1.0": true,
		"0.1.4": false, "0.1.0": false, "": false, "abc": false,
	}
	for v, ok := range cas {
		if versionAtLeast(v, "0.2.0") != ok {
			t.Errorf("version %q : attendu %v", v, ok)
		}
	}
}
