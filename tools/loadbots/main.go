// loadbots lance de faux joueurs sur un serveur Lumenfall pour tester la charge.
//
//	go run . -n 100 -duree 90s -serveur http://127.0.0.1:7350 -cle lumenfall-dev-key
//
// Chaque bot se connecte avec la connexion de test (serveur de test uniquement),
// crée un héros si besoin, entre dans la Clairière puis se promène et bavarde.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var (
	nBots   = flag.Int("n", 100, "nombre de faux joueurs")
	duree   = flag.Duration("duree", 60*time.Second, "durée du test")
	serveur = flag.String("serveur", "http://127.0.0.1:7350", "adresse du serveur Nakama")
	cle     = flag.String("cle", "lumenfall-dev-key", "clé serveur Nakama (socket.server_key)")
	prefixe = flag.String("prefixe", "bot", "préfixe des comptes de test")
	version = flag.String("version", "0.3.0", "version du jeu annoncée (celle de client/project.godot)")
)

var (
	joined, failed  int64
	msgsIn, bytesIn int64
	msgsOut         int64
	latSum, latN    int64
)

func main() {
	flag.Parse()
	log.SetFlags(log.Ltime)
	stop := time.Now().Add(*duree)
	var wg sync.WaitGroup
	for i := 0; i < *nBots; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := runBot(i, stop); err != nil {
				atomic.AddInt64(&failed, 1)
				log.Printf("bot %d : %v", i, err)
			}
		}(i)
		time.Sleep(30 * time.Millisecond) // arrivées étalées, comme de vrais joueurs
	}
	go func() {
		for time.Now().Before(stop) {
			time.Sleep(10 * time.Second)
			log.Printf("… %d bots en jeu, %d messages reçus", atomic.LoadInt64(&joined), atomic.LoadInt64(&msgsIn))
		}
	}()
	wg.Wait()
	secs := duree.Seconds()
	n := atomic.LoadInt64(&joined)
	fmt.Println("=== Bilan du test de charge ===")
	fmt.Printf("Bots entrés dans le monde : %d / %d (échecs : %d)\n", n, *nBots, failed)
	fmt.Printf("Messages envoyés : %d (%.0f/s)\n", msgsOut, float64(msgsOut)/secs)
	if n > 0 {
		fmt.Printf("Reçu par bot : %.1f messages/s, %.1f Ko/s\n", float64(msgsIn)/secs/float64(n), float64(bytesIn)/secs/float64(n)/1024)
		fmt.Printf("Trafic total sortant du serveur : %.2f Mo/s\n", float64(bytesIn)/secs/1024/1024)
	}
	if latN > 0 {
		fmt.Printf("Délai moyen entre deux mises à jour du monde : %.0f ms\n", float64(latSum)/float64(latN))
	}
}

func runBot(i int, stop time.Time) error {
	token, err := authenticate(fmt.Sprintf("%s-%04d-lumenfall", *prefixe, i))
	if err != nil {
		return fmt.Errorf("connexion : %w", err)
	}
	var list struct {
		Heroes []struct {
			ID string `json:"id"`
		} `json:"heroes"`
	}
	if err := rpc(token, "hero_list", map[string]interface{}{}, &list); err != nil {
		return fmt.Errorf("liste des héros : %w", err)
	}
	heroID := ""
	if len(list.Heroes) > 0 {
		heroID = list.Heroes[0].ID
	} else {
		var h struct {
			ID string `json:"id"`
		}
		// Les noms sont uniques sur le serveur : on en essaie plusieurs.
		var err error
		for try := 0; try < 6; try++ {
			name := botName(i + try*1000 + int(fnv(*prefixe)%997)*7)
			if err = rpc(token, "hero_create", map[string]interface{}{"slot": 0, "name": name, "color": i % 12}, &h); err == nil {
				break
			}
		}
		if err != nil {
			return fmt.Errorf("création du héros : %w", err)
		}
		heroID = h.ID
	}
	var wj struct {
		MatchID string `json:"match_id"`
	}
	if err := rpc(token, "world_join", map[string]string{"hero_id": heroID, "v": *version}, &wj); err != nil {
		return fmt.Errorf("entrée dans le monde : %w", err)
	}

	u, _ := url.Parse(*serveur)
	u.Scheme = strings.Replace(u.Scheme, "http", "ws", 1)
	u.Path = "/ws"
	u.RawQuery = url.Values{"token": {token}, "format": {"json"}}.Encode()
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("websocket : %w", err)
	}
	defer conn.Close()
	var wmu sync.Mutex
	send := func(v interface{}) error {
		wmu.Lock()
		defer wmu.Unlock()
		atomic.AddInt64(&msgsOut, 1)
		return conn.WriteJSON(v)
	}
	if err := send(map[string]interface{}{"cid": "1", "match_join": map[string]interface{}{"match_id": wj.MatchID, "metadata": map[string]string{"hero_id": heroID, "v": *version}}}); err != nil {
		return err
	}

	inWorld := make(chan struct{})
	go func() {
		var last time.Time
		once := sync.Once{}
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var env struct {
				MatchData *struct {
					OpCode json.Number `json:"op_code"`
				} `json:"match_data"`
				Error *struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			_ = json.Unmarshal(raw, &env)
			if env.Error != nil {
				log.Printf("bot %d : erreur serveur %s", i, env.Error.Message)
			}
			if env.MatchData == nil {
				continue
			}
			atomic.AddInt64(&msgsIn, 1)
			atomic.AddInt64(&bytesIn, int64(len(raw)))
			switch env.MatchData.OpCode.String() {
			case "10":
				once.Do(func() { atomic.AddInt64(&joined, 1); close(inWorld) })
			case "11":
				now := time.Now()
				if !last.IsZero() && i%10 == 0 {
					atomic.AddInt64(&latSum, now.Sub(last).Milliseconds())
					atomic.AddInt64(&latN, 1)
				}
				last = now
			}
		}
	}()
	select {
	case <-inWorld:
	case <-time.After(15 * time.Second):
		return fmt.Errorf("pas d'arrivée dans le monde après 15 s")
	}

	rng := rand.New(rand.NewSource(int64(i)))
	data := func(op int, v interface{}) map[string]interface{} {
		b, _ := json.Marshal(v)
		return map[string]interface{}{"match_data_send": map[string]interface{}{"match_id": wj.MatchID, "op_code": op, "data": base64.StdEncoding.EncodeToString(b)}}
	}
	for time.Now().Before(stop) {
		switch r := rng.Intn(10); {
		case r < 5:
			_ = send(data(2, map[string]interface{}{"t": []float64{4 + rng.Float64()*32, 4 + rng.Float64()*32}}))
		case r < 9:
			a := rng.Float64() * 6.283
			_ = send(data(1, map[string]interface{}{"d": []float64{cos(a), sin(a)}}))
		default:
			msgs := []string{"Bonjour Erydor !", "Quelqu'un a vu Braise ?", "Belle journée dans la Clairière", "On fait un tour près de l'étang ?", "Je teste la charge du serveur, ne faites pas attention à moi"}
			_ = send(data(3, map[string]string{"m": msgs[rng.Intn(len(msgs))]}))
		}
		time.Sleep(time.Duration(1500+rng.Intn(2500)) * time.Millisecond)
	}
	return nil
}

func fnv(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h = (h ^ uint32(s[i])) * 16777619
	}
	return h
}

func botName(i int) string {
	syll := []string{"ar", "bel", "cor", "dun", "el", "fin", "gal", "hal", "ith", "kor", "lun", "mor", "nor", "ol", "per", "quel", "ran", "sil", "tor", "ul", "val", "wyn"}
	n := len(syll)
	a, b, c := syll[i%n], syll[(i/n)%n], syll[(i/(n*n)+i*7+3)%n]
	s := a + b + c
	return strings.ToUpper(s[:1]) + s[1:]
}

func authenticate(deviceID string) (string, error) {
	body, _ := json.Marshal(map[string]string{"id": deviceID})
	req, _ := http.NewRequest("POST", *serveur+"/v2/account/authenticate/device?create=true", bytes.NewReader(body))
	req.SetBasicAuth(*cle, "")
	var out struct {
		Token string `json:"token"`
	}
	if err := do(req, &out); err != nil {
		return "", err
	}
	return out.Token, nil
}

func rpc(token, id string, payload, out interface{}) error {
	p, _ := json.Marshal(payload)
	body, _ := json.Marshal(string(p))
	req, _ := http.NewRequest("POST", *serveur+"/v2/rpc/"+id, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	var env struct {
		Payload string `json:"payload"`
	}
	if err := do(req, &env); err != nil {
		return err
	}
	return json.Unmarshal([]byte(env.Payload), out)
}

var client = &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{Proxy: nil, MaxIdleConnsPerHost: 200}}

func do(req *http.Request, out interface{}) error {
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		return fmt.Errorf("HTTP %d : %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	return json.Unmarshal(b, out)
}
