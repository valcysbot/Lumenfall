package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/valcysbot/lumenfall/server/world"
)

// L'Acte V dans le monde : le sac, le butin, les métiers, Aelis, Doran, le
// coffre et l'échange. Le client envoie une seule sorte de message (opObjet,
// avec une action) ; le serveur répond par l'état du sac (opSac) et des
// panneaux (opPanneau).

const (
	opObjet    = 37 // client → serveur : {"a":"equiper","i":3}, {"a":"fabriquer","r":"forge:…","c":"force","n":1}…
	opSac      = 38 // serveur → toi : le sac, l'équipement, les métiers, les repas
	opSol      = 39 // serveur → client : un objet au sol {"k":"pose"|"gagne"|"prend"|"fin","g":…}
	opDes      = 40 // serveur → toi : les dés {"g","it","sec","besoin"} ; puis {"g","fin":true,"jets":[…]}
	opNoeuds   = 41 // serveur → client : points de récolte et ateliers {"n":[…],"at":[…]} ; puis {"i","up"}
	opEchange  = 42 // serveur → toi : l'échange {"k":"demande"|"ouvre"|"maj"|"fin", …}
	opPanneau  = 43 // serveur → toi : un panneau ouvert (boutique, atelier, maître, coffre, miroir)
	opRecu     = 44 // serveur → toi : un objet reçu {"nom","r","n","fab","chance"}
	opMetierUp = 45 // serveur → toi : {"m":"forge","lv":12} (lv 1 : appris, -1 : recette apprise)

	npcReach = 4.5 // cases : assez près d'un habitant pour commercer
)

// Les maîtres des métiers et ce qu'ils enseignent.
var masters = map[string][]string{
	"doran": {"forge", "joaillerie", "minage"},
	"nessa": {"alchimie", "herboristerie"},
	"gwenn": {"couture", "cuir", "depecage"},
	"mabon": {"cuisine"},
	"aubin": {"bucheronnage"},
}

// openInventaire : le sac d'un héros qui arrive. Un héros de l'Acte IV reçoit
// son équipement commun de départ, ses trophées deviennent des matériaux et
// son or devient de l'argent (1 ancien or = 1 argent = 100 cuivres).
func openInventaire(h *Hero, rng *rand.Rand) *world.Inventaire {
	if h.inv != nil {
		return h.inv
	}
	inv := world.NewInventaire()
	inv.Starter(rng, h.Class, h.Level)
	names := make([]string, 0, len(h.trophies))
	for name := range h.trophies {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if id := world.TrophyMat[name]; id != "" && h.trophies[name] > 0 {
			inv.Add(world.MakeStack(rng, id, h.trophies[name]))
		}
	}
	h.trophies = map[string]int{}
	h.gold *= world.Argent
	h.potions = min(h.potions, world.FreePotions)
	h.inv = inv
	return inv
}

// loadCoffre lit le coffre d'un compte (vide s'il n'existe pas encore).
func loadCoffre(ctx context.Context, db *sql.DB, uid string) *world.Coffre {
	c := world.NewCoffre()
	var raw []byte
	if err := db.QueryRowContext(ctx, `SELECT coffre FROM lf_coffres WHERE user_id = $1`, uid).Scan(&raw); err == nil {
		_ = json.Unmarshal(raw, c)
		c.Fix()
	}
	return c
}

func saveCoffre(ctx context.Context, db *sql.DB, uid string, c *world.Coffre) error {
	_, err := db.ExecContext(ctx, `
INSERT INTO lf_coffres (user_id, coffre) VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE SET coffre = EXCLUDED.coffre, updated_at = now()`, uid, string(mustJSON(c)))
	return err
}

// saveLedger ajoute au suivi de l'or ce qui est entré et sorti depuis la dernière fois.
func saveLedger(ctx context.Context, db *sql.DB, l *world.Ledger) error {
	day := time.Now().UTC().Format("2006-01-02")
	for sens, m := range map[string]map[string]int{"entree": l.In, "sortie": l.Out} {
		for why, n := range m {
			if n == 0 {
				continue
			}
			if _, err := db.ExecContext(ctx, `
INSERT INTO lf_economie (jour, raison, sens, montant) VALUES ($1, $2, $3, $4)
ON CONFLICT (jour, raison, sens) DO UPDATE SET montant = lf_economie.montant + EXCLUDED.montant`, day, why, sens, n); err != nil {
				return err
			}
			m[why] = 0
		}
	}
	return nil
}

// --- Ce que voit le joueur -----------------------------------------------------------

// sendSac envoie au joueur son sac, son équipement et ses métiers.
func sendSac(d runtime.MatchDispatcher, pl *player) {
	f := pl.fighter
	inv := f.Hero.Inv
	if inv == nil {
		return
	}
	cl := f.Hero.Class.ID
	eq := map[string]world.ItemView{}
	for s, it := range inv.Eq {
		if it != nil {
			eq[world.SlotIDs[s]] = it.View(cl)
		}
	}
	metiers := map[string]map[string]int{}
	for id, m := range inv.Metiers {
		metiers[id] = map[string]int{"lv": m.Lv, "xp": m.XP, "xpn": world.MetierXP(m.Lv)}
	}
	buffs := map[string]map[string]interface{}{}
	for k, b := range inv.Buffs {
		if base := world.Bases[b.Base]; base != nil {
			buffs[k] = map[string]interface{}{"b": b.Base, "nom": base.Name, "ic": base.Icon, "t": int(b.Left), "ds": base.Desc}
		}
	}
	art := inv.Artisanats
	if art == nil {
		art = []string{}
	}
	_ = d.BroadcastMessage(opSac, mustJSON(map[string]interface{}{
		"sac": world.ItemsView(inv.Sac, cl), "eq": eq, "me": metiers, "art": art,
		"rap": inv.Rapides, "po": inv.Potion, "bu": buffs, "or": f.Hero.Gold, "rep": inv.RepairCost(),
		"cl": cl, "arm": math.Round(f.Hero.Stats.Armor*10) / 10, "pw": math.Round(f.Weapon()*10) / 10,
	}), []runtime.Presence{pl.presence}, nil, true)
}

func groundMsg(g *world.GroundItem, k string) map[string]interface{} {
	return map[string]interface{}{"k": k, "g": g.ID, "x": cent(g.X), "y": cent(g.Y), "r": g.It.R, "nom": g.It.Name(),
		"ic": g.It.B().Icon, "o": g.Owner, "d": g.Roll != nil}
}

// sendWorldItems : à l'arrivée, les objets au sol, les points de récolte et les ateliers.
func sendWorldItems(d runtime.MatchDispatcher, s *worldState, pl *player) {
	to := []runtime.Presence{pl.presence}
	nodes := make([][5]interface{}, 0, len(s.arena.Nodes))
	for _, n := range s.arena.Nodes {
		nodes = append(nodes, [5]interface{}{n.ID, n.Kind.ID, cent(n.X), cent(n.Y), n.Up()})
	}
	ats := make([][4]interface{}, 0, len(s.arena.Ateliers))
	for _, at := range s.arena.Ateliers {
		ats = append(ats, [4]interface{}{at.ID, at.Metier, cent(at.X), cent(at.Y)})
	}
	kinds := map[string]map[string]interface{}{}
	for id, k := range world.NodeKinds {
		kinds[id] = map[string]interface{}{"nom": k.Name, "m": k.Metier, "req": k.Req}
	}
	_ = d.BroadcastMessage(opNoeuds, mustJSON(map[string]interface{}{"n": nodes, "at": ats, "sortes": kinds}), to, nil, true)
	for _, g := range s.arena.Ground {
		_ = d.BroadcastMessage(opSol, mustJSON(groundMsg(g, "pose")), to, nil, true)
	}
}

// sendActeV diffuse ce que l'arène a produit pour l'Acte V pendant ce tick.
func sendActeV(d runtime.MatchDispatcher, s *worldState, e world.Event, only func(int) []runtime.Presence) {
	a := s.arena
	switch e.Kind {
	case world.EvGround:
		g := a.Ground[e.Value]
		switch e.Text {
		case "pose":
			if g != nil {
				_ = d.BroadcastMessage(opSol, mustJSON(groundMsg(g, "pose")), nil, nil, true)
			}
		case "des":
			if g == nil {
				return
			}
			for sid := range g.Roll.Votes {
				if to := only(sid); to != nil {
					h := a.Heroes[sid]
					_ = d.BroadcastMessage(opDes, mustJSON(map[string]interface{}{"g": g.ID, "it": g.It.View(h.Hero.Class.ID),
						"sec": int(world.DiceTime), "besoin": world.UsableBy(g.It.B(), h.Hero.Class.ID)}), to, nil, true)
				}
			}
		case "gagne":
			if g == nil {
				return
			}
			_ = d.BroadcastMessage(opSol, mustJSON(groundMsg(g, "gagne")), nil, nil, true)
			name := ""
			if w := a.Heroes[e.Src]; w != nil {
				name = w.Name
			}
			for sid := range g.Elig {
				if to := only(sid); to != nil {
					_ = d.BroadcastMessage(opDes, mustJSON(map[string]interface{}{"g": g.ID, "fin": true, "gagnant": name,
						"toi": e.Src == sid, "nom": g.It.Name(), "r": g.It.R}), to, nil, true)
				}
			}
		case "prend", "fin":
			_ = d.BroadcastMessage(opSol, mustJSON(map[string]interface{}{"k": "fin", "g": e.Value}), nil, nil, true)
		}
	case world.EvItem:
		if to := only(e.Tgt); to != nil {
			_ = d.BroadcastMessage(opRecu, mustJSON(map[string]interface{}{"nom": e.Text, "r": e.Value, "n": e.Flags & 0xffff,
				"fab": e.Skill == 1, "chance": e.Flags>>16 != 0}), to, nil, true)
		}
	case world.EvNode:
		_ = d.BroadcastMessage(opNoeuds, mustJSON(map[string]interface{}{"i": e.Value, "up": e.Flags == 1}), nil, nil, true)
	case world.EvMetier:
		if to := only(e.Tgt); to != nil {
			_ = d.BroadcastMessage(opMetierUp, mustJSON(map[string]interface{}{"m": e.Text, "lv": e.Value}), to, nil, true)
		}
	case world.EvLook:
		for _, pl := range s.players {
			if pl.sid == e.Src {
				_ = d.BroadcastMessage(opJoin, mustJSON(describe(pl)), nil, nil, true)
			}
		}
	case world.EvTrade:
		sendTrade(d, s, e, only)
	}
}

func sendTrade(d runtime.MatchDispatcher, s *worldState, e world.Event, only func(int) []runtime.Presence) {
	a := s.arena
	switch {
	case e.Text == "demande":
		if to := only(e.Tgt); to != nil {
			name := ""
			if h := a.Heroes[e.Src]; h != nil {
				name = h.Name
			}
			_ = d.BroadcastMessage(opEchange, mustJSON(map[string]interface{}{"k": "demande", "n": name, "s": e.Src}), to, nil, true)
		}
		if to := only(e.Src); to != nil {
			_ = d.BroadcastMessage(opEchange, mustJSON(map[string]interface{}{"k": "attente"}), to, nil, true)
		}
	case e.Text == "ouvre" || e.Text == "maj":
		t := a.Trades[e.Src]
		if t == nil {
			return
		}
		for _, sid := range []int{t.A, t.B} {
			if to := only(sid); to != nil {
				msg := a.TradeView(t, sid)
				msg["k"] = e.Text
				_ = d.BroadcastMessage(opEchange, mustJSON(msg), to, nil, true)
			}
		}
	case strings.HasPrefix(e.Text, "fin:"):
		for _, sid := range []int{e.Src, e.Tgt} {
			if to := only(sid); to != nil {
				_ = d.BroadcastMessage(opEchange, mustJSON(map[string]interface{}{"k": "fin", "pq": strings.TrimPrefix(e.Text, "fin:")}), to, nil, true)
			}
		}
	}
}

// --- Les panneaux ---------------------------------------------------------------

func panneau(d runtime.MatchDispatcher, pl *player, p string, v map[string]interface{}) {
	v["p"] = p
	_ = d.BroadcastMessage(opPanneau, mustJSON(v), []runtime.Presence{pl.presence}, nil, true)
}

func needsView(inv *world.Inventaire, r *world.Recipe) [][4]interface{} {
	out := make([][4]interface{}, 0, len(r.Needs))
	for _, n := range r.Needs {
		b := world.Bases[n.Base]
		out = append(out, [4]interface{}{b.Name, b.Icon, inv.Count(n.Base), n.N})
	}
	return out
}

func recipeView(inv *world.Inventaire, cl string, r *world.Recipe) map[string]interface{} {
	b := world.Bases[r.Out]
	v := map[string]interface{}{"id": r.ID, "nom": r.Name(), "ic": b.Icon, "r": r.R, "lv": r.Lv, "req": r.Req, "n": r.N,
		"need": needsView(inv, r), "connue": inv.Recettes[r.ID], "px": r.Price, "plan": r.Plan,
		"ok": inv.CanCraft(r) == nil, "eq": b.Kind == world.KEquipement}
	if b.Kind == world.KEquipement {
		v["pour_moi"] = world.UsableBy(b, cl)
		if b.Armor >= 0 {
			v["ar"] = world.ArmureIDs[b.Armor]
		}
		v["sl"] = world.SlotIDs[b.Slot]
	} else if b.Desc != "" {
		v["ds"] = b.Desc
	}
	return v
}

func (s *worldState) openPanel(d runtime.MatchDispatcher, pl *player, what, arg string) {
	f := pl.fighter
	inv := f.Hero.Inv
	cl := f.Hero.Class.ID
	if inv == nil {
		return
	}
	pl.panel = [2]string{what, arg}
	switch what {
	case "boutique":
		var arts []map[string]interface{}
		for _, id := range world.Shop {
			b := world.Bases[id]
			v := world.MakeStack(rand.New(rand.NewSource(1)), id, 1).View(cl)
			arts = append(arts, map[string]interface{}{"b": id, "nom": b.Name, "ic": b.Icon, "px": b.Price, "ds": b.Desc, "it": v})
		}
		var rachat []map[string]interface{}
		for _, it := range inv.Rachat {
			rachat = append(rachat, map[string]interface{}{"it": it.View(cl), "px": world.RachatPrice(it)})
		}
		panneau(d, pl, "boutique", map[string]interface{}{"articles": arts, "rachat": rachat})
	case "atelier":
		if world.Metiers[arg] == nil {
			return
		}
		var recs []map[string]interface{}
		for _, r := range world.RecipesOf(arg) {
			if inv.Recettes[r.ID] {
				recs = append(recs, recipeView(inv, cl, r))
			}
		}
		choix := make([][2]string, 0, len(world.AffixStats))
		for _, st := range world.AffixStats {
			choix = append(choix, [2]string{st, world.AffixNoms[st]})
		}
		me := inv.Metiers[arg]
		panneau(d, pl, "atelier", map[string]interface{}{"m": arg, "nom": world.Metiers[arg].Name, "ic": world.Metiers[arg].Icon,
			"lv": me.Lv, "recettes": recs, "choix": choix, "connu": inv.Knows(arg)})
	case "maitre":
		list := masters[arg]
		var ms []map[string]interface{}
		for _, id := range list {
			def := world.Metiers[id]
			m := inv.Metiers[id]
			var recs []map[string]interface{}
			if !def.Recolte && inv.Knows(id) {
				for _, r := range world.RecipesOf(id) {
					if r.Plan == "" && !inv.Recettes[r.ID] {
						recs = append(recs, recipeView(inv, cl, r))
					}
				}
			}
			ms = append(ms, map[string]interface{}{"id": id, "nom": def.Name, "ic": def.Icon, "recolte": def.Recolte,
				"lv": m.Lv, "connu": inv.Knows(id), "recettes": recs})
		}
		panneau(d, pl, "maitre", map[string]interface{}{"npc": arg, "metiers": ms, "art": inv.Artisanats,
			"max": world.ArtisanatsMax, "rep": inv.RepairCost(), "forge": arg == "doran"})
	case "coffre":
		if pl.coffre == nil {
			return
		}
		panneau(d, pl, "coffre", map[string]interface{}{"it": world.ItemsView(pl.coffre.Items, cl), "or": pl.coffre.Money})
	case "miroir":
		var apps []map[string]interface{}
		for k := range inv.Apparences {
			base, r := splitLook(k)
			if b := world.Bases[base]; b != nil {
				apps = append(apps, map[string]interface{}{"lk": k, "nom": b.Name, "ic": b.Icon, "r": r, "sl": world.SlotIDs[b.Slot]})
			}
		}
		sort.Slice(apps, func(i, j int) bool { return apps[i]["lk"].(string) < apps[j]["lk"].(string) })
		panneau(d, pl, "miroir", map[string]interface{}{"apps": apps, "px": world.MiroirPerLv})
	}
}

func splitLook(k string) (string, int) {
	i := strings.LastIndex(k, ":")
	if i < 0 {
		return k, 0
	}
	r := 0
	for _, c := range k[i+1:] {
		r = r*10 + int(c-'0')
	}
	return k[:i], r
}

// nearNPC : le héros est-il près de cet habitant (debout, visible) ?
func (s *worldState) nearNPC(pl *player, id string) bool {
	for _, n := range s.npcs {
		if n.ID == id && n.Visible && math.Hypot(n.Body.X-pl.body.X, n.Body.Y-pl.body.Y) <= npcReach {
			return true
		}
	}
	return false
}

func (s *worldState) nearAtelier(pl *player, metier string) bool {
	for _, at := range s.arena.Ateliers {
		if at.Metier == metier && math.Hypot(at.X-pl.body.X, at.Y-pl.body.Y) <= world.AtelierReach {
			return true
		}
	}
	return false
}

var errLoin = errors.New("Approche-toi d'abord.")

// objet : une action du sac, du butin, des métiers ou du commerce.
func (w *worldMatch) objet(d runtime.MatchDispatcher, s *worldState, pl *player, data []byte) {
	var in struct {
		A  string   `json:"a"`
		I  int      `json:"i"`
		J  int      `json:"j"`
		S  string   `json:"s"`
		B  string   `json:"b"`
		R  string   `json:"r"`
		C  string   `json:"c"`
		N  int      `json:"n"`
		T  int      `json:"t"`
		G  int      `json:"g"`
		V  int      `json:"v"`
		OK bool     `json:"ok"`
		It []string `json:"it"`
		Lk string   `json:"lk"`
	}
	if json.Unmarshal(data, &in) != nil || pl.fighter.Hero.Inv == nil {
		return
	}
	a := s.arena
	sid := pl.sid
	slot := world.Slot(-1)
	for i, id := range world.SlotIDs {
		if id == in.S {
			slot = world.Slot(i)
		}
	}
	fail := func(err error) {
		if err != nil && err.Error() != "" {
			refuse(d, pl, err.Error())
		}
	}
	switch in.A {
	case "equiper":
		a.Equip(sid, in.I, slot)
	case "retirer":
		a.Unequip(sid, slot)
	case "jeter":
		a.Destroy(sid, in.I)
	case "ranger":
		a.MoveItem(sid, in.I, in.J)
	case "trier":
		a.SortBag(sid)
	case "utiliser":
		a.UseItem(sid, in.I)
	case "utiliser_base":
		a.UseBase(sid, in.B)
	case "rapide":
		a.SetQuick(sid, in.N, in.B)
	case "ramasser":
		a.Pick(sid, in.G)
	case "des":
		a.Vote(sid, in.G, in.V)
	case "recolter":
		a.Gather(sid, in.N)
	case "depecer":
		a.Skin(sid, in.T)
	case "fabriquer":
		a.Craft(sid, in.R, in.C, in.N)
	case "panneau":
		switch in.B {
		case "boutique":
			if !s.nearNPC(pl, "aelis") {
				fail(errLoin)
				return
			}
		case "atelier":
			if !s.nearAtelier(pl, in.R) {
				fail(errLoin)
				return
			}
		case "maitre":
			if masters[in.R] == nil || !s.nearNPC(pl, in.R) {
				fail(errLoin)
				return
			}
		case "coffre":
			if !s.nearNPC(pl, "mabon") {
				fail(errors.New("Le coffre est à l'auberge, près de Mabon."))
				return
			}
		case "miroir":
			if math.Hypot(pl.body.X-a.Temple[0], pl.body.Y-a.Temple[1]) > world.VillageRadius {
				fail(errors.New("Le miroir de Solenn est au temple."))
				return
			}
		}
		s.openPanel(d, pl, in.B, in.R)
		return
	case "acheter", "vendre", "racheter":
		if !s.nearNPC(pl, "aelis") {
			fail(errors.New("Aelis tient son étal au marché."))
			return
		}
		switch in.A {
		case "acheter":
			a.Buy(sid, in.B, max(1, in.N))
		case "vendre":
			a.Sell(sid, in.I)
		case "racheter":
			a.Buyback(sid, in.I)
		}
		s.openPanel(d, pl, "boutique", "")
	case "reparer":
		if !s.nearNPC(pl, "doran") {
			fail(errors.New("Doran répare à sa forge."))
			return
		}
		a.Repair(sid)
		s.openPanel(d, pl, "maitre", "doran")
	case "apprendre", "oublier", "recette":
		id := in.R
		if in.A == "recette" {
			if r := world.Recipes[in.R]; r != nil {
				id = r.Metier
			}
		}
		master := ""
		for npc, ms := range masters {
			for _, m := range ms {
				if m == id {
					master = npc
				}
			}
		}
		if master == "" || !s.nearNPC(pl, master) {
			fail(errors.New("Il faut aller voir le maître de ce métier."))
			return
		}
		switch in.A {
		case "apprendre":
			fail(a.LearnMetier(sid, id))
		case "oublier":
			fail(a.ForgetMetier(sid, id))
		case "recette":
			fail(a.LearnRecipe(sid, in.R))
		}
		s.openPanel(d, pl, "maitre", master)
	case "miroir":
		if math.Hypot(pl.body.X-a.Temple[0], pl.body.Y-a.Temple[1]) > world.VillageRadius {
			fail(errors.New("Le miroir de Solenn est au temple."))
			return
		}
		fail(a.Transmog(sid, slot, in.Lk))
	case "coffre_depot", "coffre_retrait", "coffre_or":
		if !s.nearNPC(pl, "mabon") {
			fail(errors.New("Le coffre est à l'auberge, près de Mabon."))
			return
		}
		switch in.A {
		case "coffre_depot":
			a.Deposit(sid, in.I, pl.coffre)
		case "coffre_retrait":
			a.Withdraw(sid, in.I, pl.coffre)
		case "coffre_or":
			a.CoffreMoney(sid, in.N, pl.coffre)
		}
		s.openPanel(d, pl, "coffre", "")
	case "echange_demande":
		a.TradeAsk(sid, in.T)
	case "echange_reponse":
		a.TradeAnswer(sid, in.OK)
	case "echange_offre":
		a.TradeOffer(sid, in.It, in.N)
	case "echange_valider":
		a.TradeOK(sid)
	case "echange_annuler":
		a.TradeCancel(sid)
	case "fermer":
		pl.panel = [2]string{}
	}
}

// ficheObjets : ce que les autres voient de l'équipement d'un héros (fiche publique).
func ficheObjets(f *world.Fighter) map[string]world.ItemView {
	inv := f.Hero.Inv
	if inv == nil {
		return nil
	}
	out := map[string]world.ItemView{}
	for s, it := range inv.Eq {
		if it != nil {
			out[world.SlotIDs[s]] = it.View(f.Hero.Class.ID)
		}
	}
	return out
}
