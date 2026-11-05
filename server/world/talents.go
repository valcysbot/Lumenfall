package world

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Les talents (fin de l'Acte IV, Conseil des Talents).
//
// Chaque voie a son arbre de 10 paliers, du niveau 5 au niveau 50, avec 3
// talents par palier : un pour jouer seul, un pour le groupe, un utilitaire.
// Le héros gagne 1 point par niveau à partir du niveau 5 ; un talent prend
// jusqu'à 3 points. Le palier du niveau 5 arrive avant la voie : il est
// commun aux trois arbres de la classe. Le palier du niveau 10 transforme
// l'ultime. Placer un point est gratuit ; les reprendre se fait au temple,
// contre de l'or.
//
// Les talents changent surtout les compétences (plus de dégâts, une zone,
// un effet en plus, un sort de soi qui touche tout le groupe…) et donnent
// quelques bonus chiffrés. Ils sont écrits par buildTrees à partir des
// compétences de chaque classe et du rôle de chaque voie.

const (
	NTiers         = 10 // paliers
	TalentMax      = 3  // points au plus dans un talent
	TalentFirstLvl = 5  // niveau du premier palier
	TalentStep     = 5  // niveaux entre deux paliers
)

// TalentPoints : les points de talent gagnés à ce niveau.
func TalentPoints(level int) int { return max(0, level-TalentFirstLvl+1) }

// TierLevel : le niveau où s'ouvre le palier i (0 à 9).
func TierLevel(i int) int { return TalentFirstLvl + TalentStep*i }

// TalentResetGold : le prix, au temple, pour reprendre ses points.
func TalentResetGold(level int) int { return 10 * level * Argent }

// Style : à quoi sert un talent.
type Style int

const (
	Seul   Style = iota // plus fort en solo (dégâts, soins, survie)
	Groupe              // aide le groupe
	Utile               // recharges, contrôle, déplacement
)

var StyleNames = [...]string{"seul", "groupe", "utile"}

// Bonus : un bonus chiffré, par point (0,03 = +3 %).
type Bonus struct {
	Dmg, Heal, Taken, Threat, HP, Pet, Shields float64 // Taken : dégâts reçus en moins
	Crit, Dodge, Speed, Lifelink               float64
}

func (b Bonus) scaled(n float64) Bonus {
	return Bonus{b.Dmg * n, b.Heal * n, b.Taken * n, b.Threat * n, b.HP * n, b.Pet * n, b.Shields * n,
		b.Crit * n, b.Dodge * n, b.Speed * n, b.Lifelink * n}
}

func (b Bonus) plus(o Bonus) Bonus {
	return Bonus{b.Dmg + o.Dmg, b.Heal + o.Heal, b.Taken + o.Taken, b.Threat + o.Threat, b.HP + o.HP, b.Pet + o.Pet,
		b.Shields + o.Shields, b.Crit + o.Crit, b.Dodge + o.Dodge, b.Speed + o.Speed, b.Lifelink + o.Lifelink}
}

// Talent : un talent. Les nombres sont donnés par point.
type Talent struct {
	ID, Name, Desc string
	Icon           string // un émoji, d'après ce que fait le talent
	Style          Style
	Max            int
	Skill          int // 0 : un bonus chiffré ; 1 à 8 : la compétence changée

	Power   float64 // dégâts de la compétence : +x
	HealUp  float64 // soins de la compétence : +x
	DurUp   float64 // durée de ses effets : +x
	CD      float64 // recharge : −x
	Cost    float64 // coût : −x
	Range   float64 // portée : +x cases
	Radius  float64 // zone : +x cases (une compétence sans zone en gagne une)
	Drain   float64 // part des dégâts rendue en vie : +x
	GroupUp float64 // soin du groupe autour du lanceur : +x (part de la vie)
	Add     []Apply // effets ajoutés dès le premier point
	ToGroup bool    // ses effets sur soi touchent aussi le groupe
	Bonus   Bonus
}

// Tier : un palier de l'arbre.
type Tier struct {
	Level int
	T     [3]Talent
}

// Talents : les points placés, par palier et par talent.
type Talents [NTiers][3]int

// Spent : le nombre de points placés.
func (t Talents) Spent() int {
	n := 0
	for i := range t {
		for _, v := range t[i] {
			n += v
		}
	}
	return n
}

// Tree : l'arbre de la voie (ou, sans voie, le seul palier commun).
func (c *Class) Tree(spec int) []*Tier {
	out := []*Tier{&c.Common}
	if spec >= 0 && spec < 3 {
		for i := range c.Specs[spec].Tiers {
			out = append(out, &c.Specs[spec].Tiers[i])
		}
	}
	return out
}

// CheckTalents : ces points sont-ils possibles pour ce héros ?
func CheckTalents(c *Class, spec, level int, t Talents) error {
	tree := c.Tree(spec)
	for i := range t {
		for j, v := range t[i] {
			if v < 0 {
				return errors.New("Points de talent invalides.")
			}
			if v == 0 {
				continue
			}
			if i >= len(tree) {
				return errors.New("Choisis d'abord ta voie : son arbre s'ouvre avec elle.")
			}
			if level < tree[i].Level {
				return fmt.Errorf("Ce palier s'ouvre au niveau %d.", tree[i].Level)
			}
			if v > tree[i].T[j].Max {
				return fmt.Errorf("%s prend %d points au plus.", tree[i].T[j].Name, tree[i].T[j].Max)
			}
		}
	}
	if t.Spent() > TalentPoints(level) {
		return errors.New("Pas assez de points de talent.")
	}
	return nil
}

// Within : ces points ajoutent-ils seulement à ceux d'avant (sans rien reprendre) ?
func (t Talents) Within(old Talents) bool {
	for i := range t {
		for j := range t[i] {
			if t[i][j] < old[i][j] {
				return false
			}
		}
	}
	return true
}

// KeepCommon : quand on change de voie, seul le palier commun est gardé.
func (t Talents) KeepCommon() Talents {
	var out Talents
	out[0] = t[0]
	return out
}

// Shape : les compétences d'un héros avec ses talents, et ses bonus chiffrés.
func Shape(c *Class, spec int, t Talents) ([NSkills]SkillDef, Bonus) {
	var sk [NSkills]SkillDef
	for k := 1; k <= NSkills; k++ {
		if d := c.Skill(k, spec); d != nil {
			sk[k-1] = *d
			sk[k-1].Apply = append([]Apply(nil), d.Apply...)
		}
	}
	var b Bonus
	for i, tier := range c.Tree(spec) {
		for j := range tier.T {
			n := t[i][j]
			if n <= 0 {
				continue
			}
			tl := &tier.T[j]
			b = b.plus(tl.Bonus.scaled(float64(n)))
			if tl.Skill < 1 || tl.Skill > NSkills || sk[tl.Skill-1].ID == "" {
				continue
			}
			tl.shape(&sk[tl.Skill-1], float64(n))
		}
	}
	return sk, b
}

func (tl *Talent) shape(s *SkillDef, n float64) {
	s.Power *= 1 + tl.Power*n
	s.Heal *= 1 + tl.HealUp*n
	s.GroupHeal *= 1 + tl.HealUp*n
	s.CD *= 1 - tl.CD*n
	s.Cost *= 1 - tl.Cost*n
	s.Range += tl.Range * n
	s.Radius += tl.Radius * n
	s.Drain += tl.Drain * n
	s.GroupHeal += tl.GroupUp * n
	for i := range s.Apply {
		s.Apply[i].Dur *= 1 + tl.DurUp*n
		if tl.ToGroup && s.Apply[i].Who == WSoi {
			s.Apply[i].Who = WGroupe
		}
	}
	for _, ap := range tl.Add { // les effets ajoutés durent plus longtemps à chaque point
		ap.Dur *= n
		s.Apply = append(s.Apply, ap)
	}
}

// --- L'écriture des arbres ------------------------------------------------------

func init() {
	for _, c := range Classes {
		buildTrees(c)
	}
}

// Ce que fait une compétence.
func hurts(s *SkillDef) bool { return s.Power > 0 }
func heals(s *SkillDef) bool { return s.Heal > 0 || s.GroupHeal > 0 || hasApply(s, Regen) }
func guards(s *SkillDef) bool {
	return hasApply(s, Protection) || hasApply(s, Bouclier) || hasApply(s, Provoque) || s.Threat > 0
}
func controls(s *SkillDef) bool {
	for _, e := range []Effect{Etourdi, Gel, Ralenti, Peur, Charme, Silence, Aveugle} {
		if hasApply(s, e) {
			return true
		}
	}
	return false
}
func selfBuff(s *SkillDef) bool {
	for _, ap := range s.Apply {
		if ap.Who == WSoi && ap.E != Furtif && ap.E != Compagnon && ap.E != Provoque {
			return true
		}
	}
	return false
}
func hasApply(s *SkillDef, e Effect) bool {
	for _, ap := range s.Apply {
		if ap.E == e {
			return true
		}
	}
	return false
}

// skillOrder : les compétences de la classe, de la plus utile à la moins utile pour ce rôle.
func skillOrder(c *Class, role Role) []int {
	score := func(s *SkillDef) float64 {
		v := 0.0
		switch role {
		case Degats:
			v = s.Power
		case Soin:
			v = 10*(s.Heal+s.GroupHeal) + 0.5*s.Power
			if hasApply(s, Regen) {
				v += 2
			}
		case Tank:
			v = s.Power
			if guards(s) {
				v += 3
			}
		case Soutien:
			v = 0.5 * s.Power
			if heals(s) || hasApply(s, Inspire) || hasApply(s, Aura) || hasApply(s, Rage) || hasApply(s, Bouclier) {
				v += 3
			}
		}
		return v
	}
	idx := []int{1, 2, 3, 4, 5, 6, 7}
	for i := 1; i < len(idx); i++ {
		for j := i; j > 0 && score(&c.Skills[idx[j]-1]) > score(&c.Skills[idx[j-1]-1]); j-- {
			idx[j], idx[j-1] = idx[j-1], idx[j]
		}
	}
	return idx
}

// Les noms des talents, par sorte. Ils sont choisis d'après la place du talent.
var talentNames = map[string][]string{
	"power":   {"Main lourde", "Coup affûté", "Frappe sûre", "Force brute", "Geste parfait", "Élan meurtrier", "Poigne de fer", "Coup juste", "Ardeur", "Mordant"},
	"heal":    {"Mains douces", "Source claire", "Souffle de vie", "Baume ancien", "Lumière tiède", "Onction", "Eau vive", "Toucher béni"},
	"dur":     {"Longue haleine", "Effet tenace", "Écho durable", "Persistance", "Patience", "Lente braise", "Ténacité", "Fil tendu"},
	"cleave":  {"Large balayage", "Onde de choc", "Éclats", "Ricochet", "Fracas", "Éventail", "Retombées"},
	"drain":   {"Soif", "Sang pour sang", "Vol de vie", "Reprise", "Festin", "Sève volée", "Dû rendu", "Morsure"},
	"radius":  {"Grand cercle", "Vaste portée", "Onde élargie", "Cercle ouvert", "Large souffle"},
	"mark":    {"Point faible", "Désigner la proie", "Brèche", "Faille ouverte", "Cible peinte", "Défaut de cuirasse", "Signal"},
	"inspire": {"Cri de ralliement", "Exemple à suivre", "Élan partagé", "Bannière", "Chant de marche", "Flamme commune", "Haut les cœurs"},
	"group":   {"Pour les miens", "Tous ensemble", "Partage", "Main tendue", "Un pour tous", "Cercle uni"},
	"gheal":   {"Pluie de soins", "Rosée", "Cercle de vie", "Chant apaisant", "Brise douce", "Ondée", "Halo", "Veillée"},
	"gshield": {"Égide", "Rempart commun", "Garde partagée", "Voile protecteur", "Paume ouverte"},
	"cd":      {"Vivacité", "Main rapide", "Sans répit", "Rythme", "Second souffle", "Réflexe", "Cadence", "Promptitude", "Enchaînement", "Allant"},
	"cost":    {"Économie", "Juste mesure", "Sobriété", "Épargne", "Mesure", "Frugalité"},
	"slow":    {"Entraves", "Pas lourds", "Boue", "Croc-en-jambe", "Jarret", "Lest", "Glu"},
	"range":   {"Longue portée", "Bras tendu", "Œil de faucon", "Grande allonge", "Visée lointaine"},
	"dmg":     {"Maîtrise", "Fureur froide", "Art de la guerre"},
	"taken":   {"Carapace", "Inébranlable", "Cuir épais"},
	"healp":   {"Vocation", "Cœur pur", "Grâce"},
	"life":    {"Lien vital", "Sang partagé"},
	"shields": {"Bouclier d'âme", "Garde fidèle"},
	"speed":   {"Pied léger", "Vent arrière", "Course"},
	"crit":    {"Œil vif", "Coup de chance", "Faille"},
	"dodge":   {"Feinte", "Esquive", "Ombre mouvante"},
	"ult":     {"Apogée", "Paroxysme", "Accomplissement"},
	"ultg":    {"Héritage", "Gloire partagée", "Légende commune"},
	"ultcd":   {"Retour éclair", "Prompt retour"},
	"threat":  {"Provocateur", "Regard noir", "Défi", "Bravade", "Insolence", "Cible désignée", "Tête haute", "Mauvais œil"},
}

func pct(v float64) string {
	s := fmt.Sprintf("%.1f", v*100)
	s = strings.TrimSuffix(strings.TrimSuffix(s, "0"), ".")
	return strings.Replace(s, ".", ",", 1) + " %"
}

func cases(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return strings.Replace(s, ".", ",", 1)
}

// talentIcons : l'émoji de chaque sorte de talent.
var talentIcons = map[string]string{
	"power": "⚔️", "heal": "💚", "dur": "⏳", "cleave": "💥", "drain": "🩸", "radius": "🌀", "mark": "🎯",
	"inspire": "📯", "group": "🤝", "gheal": "💞", "gshield": "🛡️", "cd": "⏱️", "cost": "💧", "slow": "🐌",
	"range": "🏹", "dmg": "🔥", "taken": "🪨", "healp": "✨", "life": "❤️", "shields": "🔰", "speed": "💨",
	"crit": "🎲", "dodge": "🌫️", "ult": "🌟", "ultg": "👑", "ultcd": "⚡", "threat": "📣",
}

var effectText = map[Effect]string{
	Marque: "l'ennemi touché reçoit 15 % de dégâts en plus", Inspire: "ton groupe gagne 10 % de dégâts et de soins",
	Ralenti: "ralentit l'ennemi", Etourdi: "étourdit l'ennemi", Bouclier: "pose un bouclier sur ton groupe",
	Regen: "ton groupe regagne de la vie chaque seconde", Rage: "ton groupe frappe 25 % plus fort",
	Provoque: "provoque les ennemis touchés", Hate: "ton groupe va 30 % plus vite", Protection: "ton groupe reçoit 30 % de dégâts en moins",
}

// describe : la phrase du talent, avec les chiffres exacts.
func (tl *Talent) describe(skill string) {
	var p []string
	per := func(s string) string {
		if tl.Max > 1 {
			return s + " par point"
		}
		return s
	}
	if tl.Power > 0 {
		p = append(p, per("+"+pct(tl.Power)+" de dégâts"))
	}
	if tl.HealUp > 0 {
		p = append(p, per("+"+pct(tl.HealUp)+" de soins"))
	}
	if tl.DurUp > 0 {
		p = append(p, per("ses effets durent "+pct(tl.DurUp)+" de plus"))
	}
	if tl.CD > 0 {
		p = append(p, per("recharge −"+pct(tl.CD)))
	}
	if tl.Cost > 0 {
		p = append(p, per("coût −"+pct(tl.Cost)))
	}
	if tl.Range > 0 {
		p = append(p, per("+"+cases(tl.Range)+" cases de portée"))
	}
	if tl.Radius > 0 {
		p = append(p, per("touche aussi les ennemis à "+cases(tl.Radius)+" cases autour"))
	}
	if tl.Drain > 0 {
		p = append(p, per("te rend "+pct(tl.Drain)+" des dégâts en vie"))
	}
	if tl.GroupUp > 0 {
		p = append(p, per("soigne ton groupe de "+pct(tl.GroupUp)+" de sa vie"))
	}
	for _, ap := range tl.Add {
		p = append(p, per(fmt.Sprintf("%s pendant %s s", effectText[ap.E], cases(ap.Dur))))
	}
	if tl.ToGroup {
		p = append(p, "ses effets sur toi touchent aussi ton groupe")
	}
	b := tl.Bonus
	for _, x := range []struct {
		v    float64
		text string
	}{{b.Dmg, "+%s de dégâts"}, {b.Heal, "+%s de soins"}, {b.Taken, "−%s de dégâts reçus"}, {b.Threat, "+%s de menace"},
		{b.HP, "+%s de vie"}, {b.Pet, "tes créatures frappent %s plus fort"}, {b.Shields, "tes boucliers absorbent %s de plus"},
		{b.Crit, "+%s de chances de coup critique"}, {b.Dodge, "+%s d'esquive"}, {b.Speed, "+%s de vitesse"},
		{b.Lifelink, "%s de tes dégâts soignent l'allié le plus blessé"}} {
		if x.v > 0 {
			p = append(p, per(fmt.Sprintf(x.text, pct(x.v))))
		}
	}
	text := strings.Join(p, ", ")
	if skill != "" {
		text = skill + " : " + text
	}
	r, n := utf8.DecodeRuneInString(text)
	tl.Desc = string(unicode.ToUpper(r)) + text[n:] + "."
}

// skillTalent : un talent qui change la compétence k, dans le style voulu.
// Les idées sont essayées dans l'ordre ; on garde la première que l'arbre n'a pas encore.
func skillTalent(c *Class, role Role, k int, style Style, seed int, used map[string]bool) Talent {
	s := c.Skill(k, -1)
	type idea struct {
		kind string
		ok   bool
		set  func(t *Talent)
	}
	var ideas []idea
	switch style {
	case Seul:
		healer := role == Soin || role == Soutien || !hurts(s)
		ideas = []idea{
			{"heal", heals(s) && healer && (s.Heal > 0 || s.GroupHeal > 0), func(t *Talent) { t.HealUp = 0.08 }},
			{"dur", (heals(s) && healer) || (role == Tank && guards(s) && len(s.Apply) > 0), func(t *Talent) { t.DurUp = 0.15 }},
			{"cleave", hurts(s) && s.Radius == 0 && s.Target == TFoe && seed%3 == 1, func(t *Talent) { t.Radius, t.Max, t.Power = 1.5, 1, 0.05 }},
			{"drain", hurts(s) && (role == Tank || seed%3 == 2), func(t *Talent) { t.Drain = 0.08 }},
			{"power", hurts(s), func(t *Talent) { t.Power = 0.08 }},
			{"dur", len(s.Apply) > 0, func(t *Talent) { t.DurUp = 0.15 }},
			{"cd", true, func(t *Talent) { t.CD = 0.08 }},
		}
	case Groupe:
		ideas = []idea{
			{"group", selfBuff(s), func(t *Talent) { t.ToGroup, t.Max = true, 1 }},
			{"gheal", heals(s), func(t *Talent) { t.GroupUp = 0.02 }},
			{"threat", role == Tank && hurts(s), func(t *Talent) { t.Add, t.Radius = []Apply{a(Provoque, 1.5, 0, WZone)}, 1.5 }},
			{"mark", hurts(s) && s.CD >= 8 && seed%2 == 0, func(t *Talent) { t.Add = []Apply{a(Marque, 1.5, 0, WCible)} }},
			{"gheal", role == Soin || role == Soutien, func(t *Talent) { t.GroupUp = 0.015 }},
			{"inspire", s.CD >= 8, func(t *Talent) { t.Add = []Apply{a(Inspire, 2, 0, WGroupe)} }},
			{"mark", hurts(s) && s.CD >= 8, func(t *Talent) { t.Add = []Apply{a(Marque, 1.5, 0, WCible)} }},
			{"gshield", true, func(t *Talent) { t.Add = []Apply{a(Bouclier, 2, 1, WGroupe)} }},
		}
	case Utile:
		ideas = []idea{
			{"radius", s.Radius > 0 && seed%3 == 0, func(t *Talent) { t.Radius = 0.5 }},
			{"range", s.Range > 3 && seed%3 == 1, func(t *Talent) { t.Range = 1.5 }},
			{"slow", hurts(s) && !controls(s) && seed%2 == 0, func(t *Talent) { t.Add = []Apply{a(Ralenti, 1, 0, WCible)} }},
			{"cost", s.Cost > 0 && seed%2 == 1, func(t *Talent) { t.Cost = 0.12 }},
			{"cd", true, func(t *Talent) { t.CD = 0.08 }},
		}
	}
	for _, id := range ideas {
		key := fmt.Sprintf("%d/%s", k, id.kind)
		if !id.ok || used[key] {
			continue
		}
		used[key] = true
		t := Talent{Style: style, Max: TalentMax, Skill: k}
		id.set(&t)
		t.Name, t.Icon = uniqueName(id.kind, seed, used), talentIcons[id.kind]
		return t
	}
	t := Talent{Style: style, Max: TalentMax, Skill: k, CD: 0.06, Icon: talentIcons["cd"]}
	t.Name = uniqueName("cd", seed, used)
	return t
}

// uniqueName : un nom de la bonne sorte que l'arbre n'a pas encore.
func uniqueName(kind string, seed int, used map[string]bool) string {
	l := talentNames[kind]
	for i := range l {
		n := l[(seed+i)%len(l)]
		if !used["nom/"+n] {
			used["nom/"+n] = true
			return n
		}
	}
	for i := 2; ; i++ {
		n := fmt.Sprintf("%s %s", l[seed%len(l)], []string{"", "", "II", "III", "IV", "V", "VI"}[min(i, 6)])
		if !used["nom/"+n] {
			used["nom/"+n] = true
			return n
		}
	}
}

// passiveTalent : le dernier palier, un bonus chiffré selon le rôle.
func passiveTalent(role Role, style Style, seed int, used map[string]bool) Talent {
	t := Talent{Style: style, Max: TalentMax}
	kind := ""
	switch style {
	case Seul:
		switch role {
		case Tank:
			t.Bonus, kind = Bonus{Taken: 0.025}, "taken"
		case Soin:
			t.Bonus, kind = Bonus{Heal: 0.03}, "healp"
		case Soutien:
			t.Bonus, kind = Bonus{Heal: 0.015, Dmg: 0.015}, "healp"
		default:
			t.Bonus, kind = Bonus{Dmg: 0.03}, "dmg"
		}
	case Groupe:
		switch role {
		case Tank:
			t.Bonus, kind = Bonus{Threat: 0.1, HP: 0.02}, "threat"
		case Degats:
			t.Bonus, kind = Bonus{Lifelink: 0.03}, "life"
		default:
			t.Bonus, kind = Bonus{Shields: 0.08, Heal: 0.01}, "shields"
		}
	case Utile:
		opts := []struct {
			b    Bonus
			kind string
		}{{Bonus{Speed: 0.03}, "speed"}, {Bonus{Crit: 0.015}, "crit"}, {Bonus{Dodge: 0.015}, "dodge"}}
		for i := range opts {
			o := opts[(seed+i)%len(opts)]
			if !used["p/"+o.kind] {
				t.Bonus, kind = o.b, o.kind
				break
			}
		}
	}
	used["p/"+kind] = true
	t.Icon = talentIcons[kind]
	t.Name = uniqueName(kind, seed, used)
	return t
}

// ultTalent : le palier du niveau 10, qui transforme l'ultime.
func ultTalent(u *SkillDef, style Style, seed int, used map[string]bool) Talent {
	t := Talent{Style: style, Max: TalentMax, Skill: 8}
	switch style {
	case Seul:
		switch {
		case u.Power > 0:
			t.Power = 0.1
		case u.Heal > 0 || u.GroupHeal > 0:
			t.HealUp = 0.1
		default:
			t.DurUp = 0.15
		}
		t.Name, t.Icon = uniqueName("ult", seed, used), talentIcons["ult"]
	case Groupe:
		switch {
		case selfBuff(u):
			t.ToGroup, t.Max = true, 1
		case u.GroupHeal > 0 || u.Heal > 0:
			t.Add = []Apply{a(Bouclier, 2, 2, WGroupe)}
		case hasApply(u, Provoque):
			t.Add = []Apply{a(Protection, 2, 0, WGroupe)}
		default:
			t.Add = []Apply{a(Inspire, 3, 0, WGroupe)}
		}
		t.Name, t.Icon = uniqueName("ultg", seed, used), talentIcons["ultg"]
	case Utile:
		t.CD = 0.1
		t.Name, t.Icon = uniqueName("ultcd", seed, used), talentIcons["ultcd"]
	}
	return t
}

func classSeed(c *Class) int {
	seed := 0
	for _, ch := range c.ID {
		seed += int(ch)
	}
	return seed
}

func (c *Class) finish(t *Tier, prefix string, i int) {
	for j := range t.T {
		tl := &t.T[j]
		tl.ID = fmt.Sprintf("%s_%d_%s", prefix, i, StyleNames[j])
		name := ""
		if tl.Skill >= 1 && tl.Skill <= 7 {
			name = c.Skills[tl.Skill-1].Name
		}
		tl.describe(name)
	}
}

// commonTier : le palier commun (niveau 5) : la compétence principale, un sort à partager, de la vitesse.
func commonTier(c *Class) (Tier, map[string]bool) {
	seed := classSeed(c)
	order := skillOrder(c, Degats)
	common := Tier{Level: TierLevel(0)}
	cu := map[string]bool{}
	common.T[Seul] = skillTalent(c, Degats, order[0], Seul, seed, cu)
	common.T[Seul].scale(0.4) // commun aux trois voies : modeste
	share := order[len(order)-1]
	for k := 1; k <= 7; k++ {
		if selfBuff(&c.Skills[k-1]) {
			share = k
			break
		}
	}
	common.T[Groupe] = skillTalent(c, Soutien, share, Groupe, seed+1, cu)
	common.T[Groupe].scale(0.6)
	common.T[Utile] = passiveTalent(Degats, Utile, 0, cu)
	c.finish(&common, c.ID, 0)
	return common, cu
}

func buildTrees(c *Class) {
	c.Common, _ = commonTier(c)
	for v := range c.Specs {
		scale := 1.0
		if f, ok := talentScale[c.ID+"/"+c.Specs[v].ID]; ok {
			scale = f
		}
		buildSpec(c, v, scale)
	}
}

// buildSpec écrit l'arbre de la voie v. scale règle la force des talents
// chiffrés (voir talents_echelle.go, mesurée par les simulations).
func buildSpec(c *Class, v int, scale float64) {
	_, cu := commonTier(c)
	sp := &c.Specs[v]
	ord := skillOrder(c, sp.Role)
	s := classSeed(c) + 7*v
	used := map[string]bool{}
	for k, b := range cu {
		used[k] = b
	}
	sp.Tiers[0] = Tier{Level: TierLevel(1)}
	for st := Seul; st <= Utile; st++ {
		sp.Tiers[0].T[st] = ultTalent(&sp.Ultime, st, s+int(st), used)
	}
	for i := 1; i < NTiers-2; i++ { // paliers 3 à 9 : les 7 compétences, de la plus utile à la voie
		tier := Tier{Level: TierLevel(i + 1)}
		k := ord[(i-1)%len(ord)]
		for st := Seul; st <= Utile; st++ {
			tier.T[st] = skillTalent(c, sp.Role, k, st, s+i+int(st), used)
		}
		sp.Tiers[i] = tier
	}
	last := Tier{Level: TierLevel(NTiers - 1)}
	for st := Seul; st <= Utile; st++ {
		last.T[st] = passiveTalent(sp.Role, st, s+v+int(st), used)
	}
	sp.Tiers[NTiers-2] = last
	for i := range sp.Tiers {
		for j := range sp.Tiers[i].T {
			sp.Tiers[i].T[j].scale(scale)
		}
		c.finish(&sp.Tiers[i], c.ID+"_"+sp.ID, i+1)
	}
	// L'ultime n'est pas une compétence de classe : on la nomme à part.
	for j := range sp.Tiers[0].T {
		sp.Tiers[0].T[j].describe(sp.Ultime.Name)
	}
}

// scale règle la force d'un talent chiffré, arrondie au demi-pour-cent.
func (tl *Talent) scale(f float64) {
	r := func(v *float64) {
		if *v > 0 {
			*v = math.Max(0.01, math.Round(*v*f*200)/200)
		}
	}
	if tl.GroupUp > 0 { // le soin de groupe touche tout le monde : il descend plus bas
		tl.GroupUp = math.Max(0.0025, math.Round(tl.GroupUp*f*400)/400)
	}
	for _, v := range []*float64{&tl.Power, &tl.HealUp, &tl.DurUp, &tl.CD, &tl.Cost, &tl.Drain,
		&tl.Bonus.Dmg, &tl.Bonus.Heal, &tl.Bonus.Taken, &tl.Bonus.Threat, &tl.Bonus.HP, &tl.Bonus.Pet, &tl.Bonus.Shields,
		&tl.Bonus.Crit, &tl.Bonus.Dodge, &tl.Bonus.Speed, &tl.Bonus.Lifelink} {
		r(v)
	}
	// Les effets ajoutés durent plus ou moins longtemps (une demi-seconde près, 1 s au moins).
	if len(tl.Add) > 0 && tl.Skill != 8 { // l'ultime, rare, garde ses effets entiers
		add := append([]Apply(nil), tl.Add...)
		for i := range add {
			add[i].Dur = math.Max(1.5, math.Round(add[i].Dur*math.Min(f, 1.5)*2)/2)
			if add[i].Pow > 0 {
				add[i].Pow = math.Max(0.2, math.Round(add[i].Pow*f*10)/10)
			}
		}
		tl.Add = add
	}
}

// RecommendTalents : les ⭐ talents conseillés pour cette voie, à ce niveau.
func RecommendTalents(c *Class, spec, level int) Talents {
	var out Talents
	tree := c.Tree(spec)
	role := Degats
	if spec >= 0 && spec < 3 {
		role = c.Specs[spec].Role
	}
	first := StarStyle(role)
	left := TalentPoints(level)
	for _, st := range []Style{first, (first + 1) % 3, (first + 2) % 3} {
		for i, tier := range tree {
			if level < tier.Level {
				break
			}
			n := min(left, tier.T[st].Max-out[i][st])
			out[i][st] += n
			left -= n
		}
	}
	return out
}

// StarStyle : le style ⭐ d'une voie.
func StarStyle(r Role) Style {
	switch r {
	case Soin, Soutien:
		return Groupe
	}
	return Seul
}
