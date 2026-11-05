package world

import (
	"math"
	"sort"
	"strconv"
	"time"
)

// La gloire (Acte VI) : des compteurs, les hauts faits qu'ils débloquent, les
// titres qu'on affiche sous son nom, les cotes du Cercle et les Insignes de
// bravoure. Tout est gardé avec le héros.

// Gloire : ce qu'un héros a accompli.
type Gloire struct {
	C        map[string]int   `json:"c"`              // compteurs (tue, recolte, duel_v…)
	HF       map[string]int64 `json:"hf"`             // hauts faits réussis → date (secondes Unix)
	Titres   []string         `json:"ti"`             // titres gagnés (identifiants)
	Titre    string           `json:"t"`              // le titre affiché
	Cote     map[string]int   `json:"cote"`           // cote du Cercle par format, pour la saison
	Saison   int              `json:"saison"`         // la saison de ces cotes
	Meilleur map[string]int   `json:"best,omitempty"` // meilleure cote jamais atteinte, par format
	Insignes int              `json:"ins"`            // Insignes de bravoure
	Vus      map[string]bool  `json:"vus,omitempty"`  // habitants rencontrés
	dirty    bool
}

// NewGloire : une gloire vide.
func NewGloire() *Gloire { return (&Gloire{}).Fix() }

// Fix complète une gloire lue en base.
func (g *Gloire) Fix() *Gloire {
	if g.C == nil {
		g.C = map[string]int{}
	}
	if g.HF == nil {
		g.HF = map[string]int64{}
	}
	if g.Cote == nil {
		g.Cote = map[string]int{}
	}
	if g.Meilleur == nil {
		g.Meilleur = map[string]int{}
	}
	if g.Vus == nil {
		g.Vus = map[string]bool{}
	}
	return g
}

// TakeDirty dit si la gloire a changé depuis la dernière sauvegarde.
func (g *Gloire) TakeDirty() bool {
	d := g.dirty
	g.dirty = false
	return d
}

// CoteOf : la cote d'un format pour la saison (une nouvelle saison repart de 1500).
func (g *Gloire) CoteOf(format string, season int) int {
	if g.Saison != season {
		g.Saison = season
		g.Cote = map[string]int{}
		g.dirty = true
	}
	if v, ok := g.Cote[format]; ok {
		return v
	}
	return CoteStart
}

// Points : les points de hauts faits.
func (g *Gloire) Points() int {
	n := 0
	for id := range g.HF {
		if hf := HautsFaitsByID[id]; hf != nil {
			n += hf.Pts
		}
	}
	return n
}

// HasTitre dit si le héros a gagné ce titre.
func (g *Gloire) HasTitre(id string) bool {
	for _, t := range g.Titres {
		if t == id {
			return true
		}
	}
	return false
}

// AddTitre donne un titre (une seule fois).
func (g *Gloire) AddTitre(id string) bool {
	if g.HasTitre(id) {
		return false
	}
	g.Titres = append(g.Titres, id)
	g.dirty = true
	return true
}

// HautFait : un exploit à réussir.
type HautFait struct {
	ID    string `json:"id"`
	Nom   string `json:"nom"`
	Desc  string `json:"ds"`
	Ic    string `json:"ic"`
	Cat   string `json:"cat"` // combat, exploration, metiers, social, pvp
	Pts   int    `json:"pts"`
	Key   string `json:"-"` // le compteur qui le débloque
	N     int    `json:"n"` // à partir de combien
	Titre string `json:"ti,omitempty"`
}

// Titres : les titres et leur texte.
var Titres = map[string]string{}

// HautsFaits : la liste, dans l'ordre d'affichage.
var HautsFaits []*HautFait

// HautsFaitsByID : par identifiant.
var HautsFaitsByID = map[string]*HautFait{}

var hfByKey = map[string][]*HautFait{}

func hf(cat, id, ic, nom, desc, key string, n, pts int, titre string) {
	h := &HautFait{ID: id, Nom: nom, Desc: desc, Ic: ic, Cat: cat, Pts: pts, Key: key, N: n}
	if titre != "" {
		h.Titre = id
		Titres[id] = titre
	}
	HautsFaits = append(HautsFaits, h)
	HautsFaitsByID[id] = h
	hfByKey[key] = append(hfByKey[key], h)
}

func init() {
	// Combat.
	hf("combat", "premier_sang", "🗡️", "Premier sang", "Vaincre un monstre.", "tue", 1, 5, "")
	hf("combat", "chasseur", "🏹", "Chasseur du val", "Vaincre 100 monstres.", "tue", 100, 10, "Chasseur du val")
	hf("combat", "fleau", "⚔️", "Fléau des bêtes", "Vaincre 1 000 monstres.", "tue", 1000, 25, "Fléau des bêtes")
	hf("combat", "groin", "🐗", "Groin pour groin", "Vaincre 50 sangliers.", "tue:sanglier", 50, 10, "Tueur de sangliers")
	hf("combat", "meute", "🐺", "Plus fort que la meute", "Vaincre 50 loups.", "tue:loup", 50, 10, "Chef de meute")
	hf("combat", "efface", "🌑", "Ce qui fut effacé", "Vaincre un Effacé de la Lisière Grise.", "tue:efface", 1, 5, "")
	hf("combat", "lisiere", "🌫️", "Gardien de la Lisière", "Vaincre 100 Effacés.", "tue:efface", 100, 15, "Gardien de la Lisière")
	hf("combat", "corbeaux", "🐦‍⬛", "Plumes noires", "Vaincre 30 corbeaux effacés.", "tue:corbeau_efface", 30, 10, "")
	hf("combat", "dore", "✨", "Tout ce qui brille", "Attraper le Sanglier doré.", "tue:sanglier_dore", 1, 10, "Chasseur d'or")
	hf("combat", "mabon", "🐗", "La bauge de Mabon", "Vaincre le Grand Sanglier.", "tue:grand_sanglier", 1, 15, "Tombeur du Grand Sanglier")
	hf("combat", "mabon10", "👑", "Habitué de la bauge", "Vaincre 10 fois le Grand Sanglier.", "tue:grand_sanglier", 10, 25, "Fléau du Grand Sanglier")
	hf("combat", "faux_pas", "🪦", "Premier faux pas", "Tomber au combat. Ça arrive aux meilleurs.", "mort", 1, 5, "")
	hf("combat", "increvable", "💀", "Increvable", "Tomber 25 fois et se relever toujours.", "mort", 25, 10, "l'Increvable")
	hf("combat", "niveau5", "⭐", "Ça commence", "Atteindre le niveau 5.", "niveau", 5, 5, "")
	hf("combat", "niveau10", "🌟", "Sur sa voie", "Atteindre le niveau 10.", "niveau", 10, 10, "")
	hf("combat", "niveau20", "💫", "Vétéran du val", "Atteindre le niveau 20.", "niveau", 20, 15, "Vétéran")
	hf("combat", "niveau30", "🌠", "Héros d'Erydor", "Atteindre le niveau 30.", "niveau", 30, 20, "Héros d'Erydor")
	hf("combat", "niveau50", "☀️", "Légende", "Atteindre le niveau 50.", "niveau", 50, 50, "Légende vivante")

	// Exploration.
	hf("exploration", "bonjour", "👋", "Bonjour, Bourg-Lanterne", "Rencontrer 5 habitants du village.", "vus", 5, 5, "")
	hf("exploration", "tour_du_village", "🏘️", "Le tour du village", "Rencontrer les 12 habitants du val.", "vus", 12, 10, "Ami du village")
	hf("exploration", "fortune", "💰", "Bourse pleine", "Avoir 1 or en poche.", "fortune", 100*Argent, 10, "")
	hf("exploration", "epique", "🟣", "Trouvaille épique", "Obtenir un objet épique.", "rarete:4", 1, 10, "")
	hf("exploration", "legendaire", "🟠", "Une légende dans le sac", "Obtenir un objet légendaire.", "rarete:5", 1, 20, "")
	hf("exploration", "lumineux", "🌟", "Touché par le Lumenfall", "Obtenir un objet lumineux.", "rarete:6", 1, 50, "Élu du Lumenfall")

	// Métiers.
	hf("metiers", "cueillette", "🌿", "Première cueillette", "Récolter une première fois.", "recolte", 1, 5, "")
	hf("metiers", "recolteur", "⛏️", "Mains calleuses", "Récolter 250 fois.", "recolte", 250, 15, "Récolteur infatigable")
	hf("metiers", "premier_ouvrage", "🔨", "Premier ouvrage", "Fabriquer un premier objet.", "fabrique", 1, 5, "")
	hf("metiers", "artisan", "⚒️", "Artisan reconnu", "Fabriquer 100 objets.", "fabrique", 100, 15, "Artisan")
	hf("metiers", "depeceur", "🔪", "Ni poil ni plume", "Dépecer 50 bêtes.", "depece", 50, 10, "")
	hf("metiers", "maitre25", "📜", "Compagnon", "Monter un métier au niveau 25.", "metier", 25, 15, "")
	hf("metiers", "maitre50", "🏅", "Maître artisan", "Monter un métier au niveau 50.", "metier", 50, 30, "Maître artisan")

	// Social.
	hf("social", "salut", "👋", "Bien élevé", "Faire une émote.", "emote", 1, 5, "")
	hf("social", "danse", "💃", "Le pas de ton peuple", "Danser la danse de ton peuple.", "danse", 1, 5, "")
	hf("social", "bavard", "💬", "Bavard", "Écrire 200 messages.", "chat", 200, 10, "le Bavard")
	hf("social", "ami", "🤝", "Un ami dans le val", "Se faire un ami.", "amis", 1, 5, "")
	hf("social", "amis5", "🫂", "Toute une bande", "Avoir 5 amis.", "amis", 5, 10, "")
	hf("social", "groupe", "👥", "Ensemble", "Vaincre un monstre en groupe.", "groupe", 1, 5, "")
	hf("social", "groupe250", "🛡️", "Frères d'armes", "Vaincre 250 monstres en groupe.", "groupe", 250, 15, "Frère d'armes")
	hf("social", "guilde", "🏰", "Sous la bannière", "Rejoindre une guilde.", "guilde", 1, 5, "")
	hf("social", "fondateur", "🪶", "Fondateur", "Fonder une guilde au registre de Fiacre.", "fonde", 1, 10, "Fondateur")
	hf("social", "echange", "🔄", "Donnant, donnant", "Faire un échange avec un autre héros.", "echange", 1, 5, "")

	// PvP.
	hf("pvp", "duel1", "🤺", "En garde !", "Gagner un duel.", "duel_v", 1, 5, "")
	hf("pvp", "duel25", "🎖️", "Duelliste", "Gagner 25 duels.", "duel_v", 25, 15, "Duelliste")
	hf("pvp", "cercle1", "⭕", "Premier pas dans le Cercle", "Gagner un combat au Cercle des Braves.", "cercle_v", 1, 5, "")
	hf("pvp", "cercle25", "🏟️", "Habitué du Cercle", "Gagner 25 combats au Cercle.", "cercle_v", 25, 15, "Brave du Cercle")
	hf("pvp", "cercle100", "🏆", "Gloire du Cercle", "Gagner 100 combats au Cercle.", "cercle_v", 100, 30, "Gloire du Cercle")
	hf("pvp", "cote1600", "📈", "Montée en puissance", "Atteindre une cote de 1 600 au Cercle.", "cote", 1600, 10, "")
	hf("pvp", "cote1800", "🥈", "Redouté", "Atteindre une cote de 1 800 au Cercle.", "cote", 1800, 20, "le Redouté")
	hf("pvp", "cote2000", "🥇", "Invaincu", "Atteindre une cote de 2 000 au Cercle.", "cote", 2000, 40, "l'Invaincu")
	hf("pvp", "drapeau", "🚩", "Drapeau au vent", "Lever son drapeau de guerre.", "drapeau", 1, 5, "")
	hf("pvp", "guerre1", "⚔️", "Premier combat de faction", "Vaincre un héros d'une faction ennemie, drapeau levé.", "guerre", 1, 10, "")
	hf("pvp", "guerre50", "🔥", "Champion de faction", "Vaincre 50 héros ennemis, drapeau levé.", "guerre", 50, 25, "Champion de faction")

	// Les titres de saison du Cercle (les trois premiers de chaque format).
	for _, f := range CercleFormats {
		k := strconv.Itoa(f)
		Titres["saison_"+k+"_1"] = "Champion du Cercle (" + k + " c. " + k + ")"
		Titres["saison_"+k+"_2"] = "Gladiateur (" + k + " c. " + k + ")"
	}
}

// Count ajoute n au compteur key du héros, et débloque ses hauts faits.
func (a *Arena) Count(h *Fighter, key string, n int) {
	if h == nil || h.Hero == nil || h.Hero.Gloire == nil || n == 0 {
		return
	}
	g := h.Hero.Gloire
	g.C[key] += n
	g.dirty = true
	a.checkHF(h, key, g.C[key])
}

// CountMax garde le plus grand nombre atteint (niveau, cote, fortune…).
func (a *Arena) CountMax(h *Fighter, key string, v int) {
	if h == nil || h.Hero == nil || h.Hero.Gloire == nil {
		return
	}
	g := h.Hero.Gloire
	if v <= g.C[key] {
		return
	}
	g.C[key] = v
	g.dirty = true
	a.checkHF(h, key, v)
}

// CountSID : Count par numéro de héros (pour le match).
func (a *Arena) CountSID(sid int, key string, n int) { a.Count(a.Heroes[sid], key, n) }

// CountMaxSID : CountMax par numéro de héros (pour le match).
func (a *Arena) CountMaxSID(sid int, key string, v int) { a.CountMax(a.Heroes[sid], key, v) }

// Meet : le héros rencontre un habitant (pour le tour du village).
func (a *Arena) Meet(sid int, npc string) {
	h := a.Heroes[sid]
	if h == nil || h.Hero.Gloire == nil || h.Hero.Gloire.Vus[npc] {
		return
	}
	h.Hero.Gloire.Vus[npc] = true
	a.CountMax(h, "vus", len(h.Hero.Gloire.Vus))
}

func (a *Arena) checkHF(h *Fighter, key string, v int) {
	g := h.Hero.Gloire
	for _, x := range hfByKey[key] {
		if v < x.N || g.HF[x.ID] != 0 {
			continue
		}
		g.HF[x.ID] = Now()
		if x.Titre != "" {
			g.AddTitre(x.Titre)
		}
		a.emit(Event{Kind: EvGloire, Tgt: h.SID, Text: x.ID})
	}
}

// Now : l'heure en secondes Unix (remplaçable dans les tests).
var Now = func() int64 { return time.Now().Unix() }

// SetTitre : le héros choisit le titre affiché ("" : aucun).
func (a *Arena) SetTitre(sid int, id string) {
	h := a.Heroes[sid]
	if h == nil || h.Hero.Gloire == nil {
		return
	}
	g := h.Hero.Gloire
	if id != "" && !g.HasTitre(id) {
		return
	}
	g.Titre = id
	g.dirty = true
	a.emit(Event{Kind: EvLook, Src: sid})
}

// countKill : les compteurs d'un monstre vaincu.
func (a *Arena) countKill(h *Fighter, m *Fighter) {
	k := m.Mon.Kind
	a.Count(h, "tue", 1)
	a.Count(h, "tue:"+k.ID, 1)
	if k.Elem == Ombre {
		a.Count(h, "tue:efface", 1)
	}
	if g := a.groupOf[h.SID]; g != nil && len(g.Members) > 1 {
		a.Count(h, "groupe", 1)
	}
	a.CountMax(h, "fortune", h.Hero.Gold)
}

// TitreText : le texte d'un titre.
func TitreText(id string) string { return Titres[id] }

// GloireView : ce que la fiche montre des hauts faits.
func GloireView(g *Gloire) map[string]interface{} {
	if g == nil {
		return nil
	}
	titres := make([][2]string, 0, len(g.Titres))
	for _, t := range g.Titres {
		titres = append(titres, [2]string{t, Titres[t]})
	}
	sort.Slice(titres, func(i, j int) bool { return titres[i][1] < titres[j][1] })
	prog := map[string]int{}
	for _, x := range HautsFaits {
		if g.HF[x.ID] == 0 {
			prog[x.ID] = int(math.Min(float64(g.C[x.Key]), float64(x.N)))
		}
	}
	return map[string]interface{}{"hf": g.HF, "pts": g.Points(), "ti": titres, "t": g.Titre, "prog": prog,
		"ins": g.Insignes, "cote": g.Cote, "best": g.Meilleur, "dv": g.C["duel_v"], "dd": g.C["duel_d"],
		"cv": g.C["cercle_v"], "cd": g.C["cercle_d"], "gv": g.C["guerre"]}
}
