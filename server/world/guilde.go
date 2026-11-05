package world

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// Les guildes (Acte VI) : on les fonde seul au registre de Fiacre, pour
// 10 argent (choix d'Elowan). 50 membres au plus, 5 rangs que le chef renomme,
// chacun avec ses droits ; un blason à composer ; un coffre à onglets (au
// registre de Fiacre aussi) avec son journal ; un niveau qui monte quand les
// membres jouent ensemble ; un message du jour et un petit calendrier.

const (
	GuildePrix       = 10 * Argent
	GuildeMax        = 50
	GuildeRangs      = 5
	GuildeOnglet     = 30 // places par onglet de coffre
	GuildeJournalMax = 60
	GuildeEventsMax  = 12
	GuildeMotdMax    = 200
	GuildeNiveauMax  = 10
)

// Les droits d'un rang.
const (
	DroitInviter = 1 << iota
	DroitExclure
	DroitCoffre // reprendre au coffre (déposer est permis à tous)
	DroitMotd   // message du jour
	DroitAgenda // calendrier
	DroitRangs  // changer le rang des membres (en dessous du sien)
	DroitsTous  = DroitInviter | DroitExclure | DroitCoffre | DroitMotd | DroitAgenda | DroitRangs
)

// Rang : un rang de la guilde (0 : le chef).
type Rang struct {
	Nom    string `json:"nom"`
	Droits int    `json:"dr"`
}

// Blason : l'écu de la guilde.
type Blason struct {
	Forme  int `json:"f"`  // 0 à BlasonFormes-1
	Motif  int `json:"m"`  // 0 à len(BlasonMotifs)-1
	Fond   int `json:"c1"` // couleurs, 0 à BlasonCouleurs-1
	Figure int `json:"c2"`
}

// Les écus possibles.
const (
	BlasonFormes   = 5
	BlasonCouleurs = 16
)

// BlasonMotifs : les figures (le client les dessine) ; les dernières se débloquent avec le niveau.
var BlasonMotifs = []string{"etoile", "lune", "soleil", "epee", "bouclier", "couronne", "arbre", "flamme", "goutte", "feuille",
	"loup", "sanglier", "corbeau", "dragon", "lanterne", "tour", "cle", "marteau", "fleur", "croix",
	"cerf", "ours", "serpent", "aigle", "cristal", "chene", "harpe", "ancre", "plume", "lumen"}

// MotifsAuNiveau : combien de motifs un niveau de guilde permet.
func MotifsAuNiveau(lv int) int {
	return min(len(BlasonMotifs), 12+3*(lv-1))
}

// OngletsAuNiveau : combien d'onglets de coffre (1, puis 2 au niveau 3, 3 au niveau 5).
func OngletsAuNiveau(lv int) int {
	switch {
	case lv >= 5:
		return 3
	case lv >= 3:
		return 2
	}
	return 1
}

// GuildeXP : les points pour passer du niveau lv au suivant.
func GuildeXP(lv int) int { return 400 * lv * lv }

// Evenement : une soirée prévue au calendrier.
type Evenement struct {
	T      int64  `json:"t"` // millisecondes Unix
	Titre  string `json:"ti"`
	Auteur string `json:"au"`
}

// Ligne : une ligne du journal du coffre.
type Ligne struct {
	T      int64  `json:"t"`
	Qui    string `json:"qui"`
	Action string `json:"a"` // depot, retrait, or+, or-
	Quoi   string `json:"q"`
	N      int    `json:"n"`
}

// Guilde : une guilde (les membres sont gardés avec les héros).
type Guilde struct {
	ID      int64          `json:"id"`
	Nom     string         `json:"nom"`
	Blason  Blason         `json:"bl"`
	Niveau  int            `json:"lv"`
	XP      int            `json:"xp"`
	Motd    string         `json:"motd"`
	Rangs   []Rang         `json:"rangs"`
	Agenda  []Evenement    `json:"agenda"`
	Onglets [][]*Item      `json:"onglets"`
	Argent  int            `json:"or"`
	Journal []Ligne        `json:"journal"`
	Membres map[string]int `json:"-"` // héros (identifiant) → rang, tenu à jour par le match
	dirty   bool
}

// DefaultRangs : les rangs d'une guilde neuve.
func DefaultRangs() []Rang {
	return []Rang{
		{"Chef", DroitsTous},
		{"Officier", DroitsTous},
		{"Vétéran", DroitInviter | DroitCoffre | DroitAgenda},
		{"Membre", DroitAgenda},
		{"Recrue", 0},
	}
}

// NewGuilde : une guilde neuve.
func NewGuilde(nom string, bl Blason) *Guilde {
	g := &Guilde{Nom: nom, Blason: bl, Niveau: 1, Rangs: DefaultRangs(), Membres: map[string]int{}}
	return g.Fix()
}

// Fix complète une guilde lue en base.
func (g *Guilde) Fix() *Guilde {
	if g.Niveau < 1 {
		g.Niveau = 1
	}
	if len(g.Rangs) != GuildeRangs {
		g.Rangs = DefaultRangs()
	}
	g.Rangs[0].Droits = DroitsTous
	for len(g.Onglets) < 3 {
		g.Onglets = append(g.Onglets, nil)
	}
	for i := range g.Onglets {
		if len(g.Onglets[i]) < GuildeOnglet {
			g.Onglets[i] = append(g.Onglets[i], make([]*Item, GuildeOnglet-len(g.Onglets[i]))...)
		}
	}
	if g.Membres == nil {
		g.Membres = map[string]int{}
	}
	return g
}

// Touch : la guilde a changé (à sauvegarder).
func (g *Guilde) Touch() { g.dirty = true }

// TakeDirty dit si la guilde a changé depuis la dernière sauvegarde.
func (g *Guilde) TakeDirty() bool {
	d := g.dirty
	g.dirty = false
	return d
}

// Can : le rang r a-t-il ce droit ?
func (g *Guilde) Can(r, droit int) bool {
	if r < 0 || r >= len(g.Rangs) {
		return false
	}
	return r == 0 || g.Rangs[r].Droits&droit != 0
}

// AddXP fait progresser la guilde. Il renvoie le nouveau niveau s'il change (0 sinon).
func (g *Guilde) AddXP(n int) int {
	if g.Niveau >= GuildeNiveauMax || n <= 0 {
		return 0
	}
	g.XP += n
	g.dirty = true
	up := 0
	for g.Niveau < GuildeNiveauMax && g.XP >= GuildeXP(g.Niveau) {
		g.XP -= GuildeXP(g.Niveau)
		g.Niveau++
		up = g.Niveau
	}
	if g.Niveau >= GuildeNiveauMax {
		g.XP = 0
	}
	return up
}

// Les erreurs des guildes.
var (
	ErrGuildeNom     = errors.New("Le nom d'une guilde fait de 3 à 24 lettres (lettres, espaces, apostrophes et tirets).")
	ErrGuildeBlason  = errors.New("Ce blason n'existe pas (ou son motif n'est pas encore débloqué).")
	ErrGuildePleine  = errors.New("La guilde est complète (50 membres).")
	ErrGuildeDroit   = errors.New("Ton rang ne te le permet pas.")
	ErrGuildeOnglet  = errors.New("Cet onglet du coffre n'est pas encore débloqué.")
	ErrGuildeCoffre  = errors.New("Le coffre de la guilde est plein.")
	ErrGuildeLie     = errors.New("Un objet lié ne va pas au coffre de la guilde.")
	ErrGuildeAgenda  = errors.New("Le calendrier est plein.")
	ErrGuildeRangNom = errors.New("Un nom de rang fait de 1 à 16 lettres.")
)

// ValidateGuildeNom vérifie le nom d'une guilde (et le renvoie nettoyé).
func ValidateGuildeNom(s string) (string, error) {
	s = strings.Join(strings.Fields(s), " ")
	n := utf8.RuneCountInString(s)
	if n < 3 || n > 24 {
		return "", ErrGuildeNom
	}
	for _, c := range s {
		ok := c == ' ' || c == '\'' || c == '-' || c == '’' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			strings.ContainsRune("àâäçéèêëîïôöùûüÿœæÀÂÄÇÉÈÊËÎÏÔÖÙÛÜŸŒÆ", c)
		if !ok {
			return "", ErrGuildeNom
		}
	}
	return s, nil
}

// Check vérifie un blason pour une guilde de niveau lv.
func (b Blason) Check(lv int) error {
	if b.Forme < 0 || b.Forme >= BlasonFormes || b.Motif < 0 || b.Motif >= MotifsAuNiveau(lv) ||
		b.Fond < 0 || b.Fond >= BlasonCouleurs || b.Figure < 0 || b.Figure >= BlasonCouleurs {
		return ErrGuildeBlason
	}
	return nil
}

// CleanText : un texte d'une ligne, sans caractères invisibles, coupé à max lettres.
func CleanText(s string, max int) string {
	var b strings.Builder
	n := 0
	for _, c := range s {
		if c < 32 || c == 0x7f || (c >= 0x200b && c <= 0x200f) || (c >= 0x202a && c <= 0x202e) || c == 0xfeff {
			c = ' '
		}
		if n >= max {
			break
		}
		b.WriteRune(c)
		n++
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func (g *Guilde) log(qui, action, quoi string, n int) {
	g.Journal = append(g.Journal, Ligne{T: Now() * 1000, Qui: qui, Action: action, Quoi: quoi, N: n})
	if len(g.Journal) > GuildeJournalMax {
		g.Journal = g.Journal[len(g.Journal)-GuildeJournalMax:]
	}
	g.dirty = true
}

// GuildeDepot : un objet du sac va au coffre de la guilde.
func (a *Arena) GuildeDepot(sid, i, onglet int, g *Guilde) error {
	h, inv := a.heroInv(sid)
	if h == nil || g == nil {
		return nil
	}
	if onglet < 0 || onglet >= OngletsAuNiveau(g.Niveau) {
		return ErrGuildeOnglet
	}
	it := inv.At(i)
	if it == nil || a.inTrade(h, it) {
		return nil
	}
	if it.Lie {
		return ErrGuildeLie
	}
	for k, o := range g.Onglets[onglet] {
		if o == nil {
			g.Onglets[onglet][k] = it
			inv.Sac[i] = nil
			inv.Touch()
			g.log(h.Name, "depot", it.Name(), it.Count())
			return nil
		}
	}
	return ErrGuildeCoffre
}

// GuildeRetrait : reprendre un objet du coffre de la guilde (droit « coffre »).
func (a *Arena) GuildeRetrait(sid, onglet, k, rang int, g *Guilde) error {
	h, inv := a.heroInv(sid)
	if h == nil || g == nil {
		return nil
	}
	if !g.Can(rang, DroitCoffre) {
		return ErrGuildeDroit
	}
	if onglet < 0 || onglet >= OngletsAuNiveau(g.Niveau) || k < 0 || k >= len(g.Onglets[onglet]) || g.Onglets[onglet][k] == nil {
		return nil
	}
	it := g.Onglets[onglet][k]
	name, n := it.Name(), it.Count()
	if rest := inv.Add(it); rest != nil {
		g.Onglets[onglet][k] = rest
		g.dirty = true
		return ErrSacPlein
	}
	g.Onglets[onglet][k] = nil
	g.log(h.Name, "retrait", name, n)
	return nil
}

// GuildeArgent : déposer (n > 0) ou reprendre (n < 0, droit « coffre ») de l'argent.
func (a *Arena) GuildeArgent(sid, n, rang int, g *Guilde) error {
	h, _ := a.heroInv(sid)
	if h == nil || g == nil || n == 0 {
		return nil
	}
	if n > 0 {
		n = min(n, h.Hero.Gold)
	} else {
		if !g.Can(rang, DroitCoffre) {
			return ErrGuildeDroit
		}
		n = -min(-n, g.Argent)
	}
	if n == 0 {
		return nil
	}
	h.Hero.Gold -= n
	g.Argent += n
	h.Hero.me = true
	if n > 0 {
		g.log(h.Name, "or+", Money(n), 1)
	} else {
		g.log(h.Name, "or-", Money(-n), 1)
	}
	return nil
}

// AddEvent ajoute une soirée au calendrier (les soirées passées s'effacent).
func (g *Guilde) AddEvent(e Evenement, nowMS int64) error {
	g.PruneAgenda(nowMS)
	if len(g.Agenda) >= GuildeEventsMax {
		return ErrGuildeAgenda
	}
	e.Titre = CleanText(e.Titre, 60)
	if e.Titre == "" || e.T < nowMS {
		return errors.New("Choisis un titre et une date à venir.")
	}
	g.Agenda = append(g.Agenda, e)
	for i := len(g.Agenda) - 1; i > 0 && g.Agenda[i].T < g.Agenda[i-1].T; i-- {
		g.Agenda[i], g.Agenda[i-1] = g.Agenda[i-1], g.Agenda[i]
	}
	g.dirty = true
	return nil
}

// PruneAgenda efface les soirées finies depuis plus de 6 heures.
func (g *Guilde) PruneAgenda(nowMS int64) {
	kept := g.Agenda[:0]
	for _, e := range g.Agenda {
		if e.T+6*3600*1000 >= nowMS {
			kept = append(kept, e)
		}
	}
	if len(kept) != len(g.Agenda) {
		g.dirty = true
	}
	g.Agenda = kept
}

// SetRangs renomme les rangs et règle leurs droits (le chef garde tous les siens).
func (g *Guilde) SetRangs(rs []Rang) error {
	if len(rs) != GuildeRangs {
		return ErrGuildeRangNom
	}
	for i := range rs {
		rs[i].Nom = CleanText(rs[i].Nom, 16)
		if rs[i].Nom == "" {
			return ErrGuildeRangNom
		}
		rs[i].Droits &= DroitsTous
	}
	rs[0].Droits = DroitsTous
	g.Rangs = rs
	g.dirty = true
	return nil
}
