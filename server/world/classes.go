package world

// Les 17 classes d'Erydor et leurs 51 voies (Acte IV, Conseil des Héros).
//
// Chaque classe a sa ressource, son arme, 7 compétences (débloquées du
// niveau 1 au niveau 9) et 3 voies. La voie se choisit au niveau 10 : elle
// apporte un bonus permanent et la compétence ultime (touche 8).
//
// Tout est décrit ici, sous forme de données ; arena.go et skills.go font
// tourner ces données sans rien savoir de chaque classe.

// Target : sur qui se lance une compétence.
type Target int

const (
	TFoe  Target = iota // un ennemi visé
	TAlly               // un allié visé, ou soi-même
	TSelf               // soi-même (et, avec Radius, la zone autour de soi)
)

// Who : à qui va un effet.
type Who int

const (
	WCible  Who = iota // la cible de la compétence
	WSoi               // le lanceur
	WGroupe            // le lanceur et les alliés à moins de GroupRadius cases
	WZone              // tous les ennemis touchés
)

// GroupRadius : la portée des sorts de groupe, en cases.
const GroupRadius = 8.0

// Apply : un effet posé par une compétence. Pow est multiplié par la
// puissance du héros (dégâts par seconde, bouclier, soins par seconde…).
type Apply struct {
	E   Effect
	Dur float64
	Pow float64
	Who Who
}

// SkillDef : une compétence.
type SkillDef struct {
	ID, Name, Desc string
	Level          int     // niveau pour l'apprendre
	CD, Cast       float64 // recharge et incantation, en secondes
	Cost, Gain     float64 // ressource dépensée et gagnée (pour la Vapeur, Cost réchauffe)
	Target         Target
	Range          float64 // 0 : corps à corps
	Power          float64 // dégâts : × la puissance de l'arme
	Magic          bool    // un sort (le Silence l'empêche, l'Intelligence le renforce)
	Elem           Element
	Radius         float64 // > 0 : touche tous les ennemis autour de la cible (ou de soi)
	Heal           float64 // soin : part de la vie maximale de la cible
	GroupHeal      float64 // soin de groupe autour du lanceur (part de la vie maximale)
	Apply          []Apply
	Threat         float64 // menace en plus (multiplicateur)
	Dash           float64 // saut vers la cible (> 0) ou en arrière (< 0), en cases
	Execute        bool    // dégâts doublés sous 30 % de vie
	Drain          float64 // part des dégâts rendue en vie au lanceur
	Finisher       bool    // Moine : dépense tout le Souffle, +60 % par point
	Anim           string
}

// Mods : le bonus permanent d'une voie.
type Mods struct {
	Dmg, Heal, Taken, Threat, HP, Pet float64 // multiplicateurs (0 compte comme 1)
	Crit, Dodge                         float64 // chances en plus
	Speed                               float64 // vitesse en plus (0,1 = +10 %)
	Lifelink                            float64 // part des dégâts qui soigne l'allié le plus blessé
	PermanentPet                        float64 // un compagnon toujours là, de cette puissance
	Shields                             float64 // boucliers posés plus solides
}

// Role : tank, soin, dégâts, soutien.
type Role string

const (
	Tank    Role = "tank"
	Soin    Role = "soin"
	Degats  Role = "degats"
	Soutien Role = "soutien"
)

// Spec : une voie.
type Spec struct {
	ID, Name, Desc string
	Role           Role
	Mods           Mods
	Ultime         SkillDef
}

// Resource : la ressource d'une classe.
type Resource struct {
	ID, Name                  string
	Max, Start                float64
	Regen                     float64 // par seconde, toujours
	Decay                     float64 // par seconde, hors combat
	OnDeal, OnTaken           float64 // par coup donné (attaque de base) et reçu
	OnMove                    float64 // par case parcourue
	Still                     float64 // par seconde sans bouger
	OnKill                    float64 // quand un ennemi proche tombe
	Heat                      bool    // la Vapeur : les compétences réchauffent, Regen refroidit
	LowHP                     bool    // la Furie : double gain sous la moitié de la vie
}

// Class : une classe.
type Class struct {
	ID      string
	Names   [3]string // masculin, féminin, neutre
	Desc    string
	Res     Resource
	Weapons []string // pièces d'arme dessinées (client)
	Shield  bool     // peut lever un bouclier (clic droit)
	Range   float64  // portée de l'attaque de base (0 : corps à corps)
	Elem    Element  // élément de l'attaque de base
	Magic   bool     // l'attaque de base est magique
	Anim    string   // animation de l'attaque de base
	Attr    [9]int   // répartition conseillée (poids par attribut)
	Skills  [7]SkillDef
	Specs   [3]Spec
}

// Le niveau où l'on choisit sa voie, et où l'ultime arrive.
const SpecLevel = 10

func a(e Effect, dur, pow float64, who Who) Apply { return Apply{E: e, Dur: dur, Pow: pow, Who: who} }

// Attributs, dans l'ordre : Force, Intelligence, Agilité, Vitesse, Esquive,
// Vitalité, Chance, Endurance, Précision.

var (
	resRage    = Resource{ID: "rage", Name: "Rage", Max: 100, OnDeal: 8, OnTaken: 6, Decay: 5}
	resFurie   = Resource{ID: "furie", Name: "Furie", Max: 100, OnDeal: 6, OnTaken: 9, Decay: 5, LowHP: true}
	resElan    = Resource{ID: "elan", Name: "Élan", Max: 100, OnDeal: 5, OnMove: 5, Decay: 3}
	resDragon  = Resource{ID: "souffle_draconique", Name: "Souffle draconique", Max: 100, Start: 100, Regen: 5}
	resFerveur = Resource{ID: "ferveur", Name: "Ferveur", Max: 100, OnDeal: 8, OnTaken: 5, Decay: 4}
	resMoine   = Resource{ID: "souffle", Name: "Souffle intérieur", Max: 5}
	resMana    = Resource{ID: "mana", Name: "Mana", Max: 100, Start: 100, Regen: 5}
	resAmes    = Resource{ID: "ames", Name: "Âmes", Max: 10, OnKill: 3, Decay: 0.1}
	resLien    = Resource{ID: "lien", Name: "Lien", Max: 100, Start: 100, Regen: 6}
	resReactif = Resource{ID: "reactifs", Name: "Réactifs", Max: 6, Start: 6, Regen: 1.0 / 3}
	resVapeur  = Resource{ID: "vapeur", Name: "Vapeur", Max: 100, Regen: 12, Heat: true}
	resGrace   = Resource{ID: "grace", Name: "Grâce", Max: 100, Start: 100, Regen: 5}
	resSeve    = Resource{ID: "seve", Name: "Sève", Max: 100, Start: 100, Regen: 5}
	resEsprits = Resource{ID: "esprits", Name: "Esprits", Max: 100, Start: 100, Regen: 5}
	resHarmo   = Resource{ID: "harmonie", Name: "Harmonie", Max: 100, Decay: 3}
	resEnergie = Resource{ID: "energie", Name: "Énergie", Max: 100, Start: 100, Regen: 12}
	resConcent = Resource{ID: "concentration", Name: "Concentration", Max: 100, Start: 50, Still: 15, OnMove: -4}
)

// Des ultimes et des bonus qui reviennent souvent.
var (
	modsTank = Mods{Taken: 0.82, Threat: 1.6, HP: 1.15}
	modsSoin = Mods{Heal: 1.3}
)

func ultTank(id, name, desc string) SkillDef {
	return SkillDef{ID: id, Name: name, Desc: desc, Level: SpecLevel, CD: 90, Target: TSelf, Radius: 6, Threat: 10,
		Apply: []Apply{a(Protection, 8, 0, WSoi), a(Bouclier, 10, 5, WSoi), a(Provoque, 6, 0, WZone)}, Anim: "garde"}
}

func ultGroupHeal(id, name, desc string, part float64) SkillDef {
	return SkillDef{ID: id, Name: name, Desc: desc, Level: SpecLevel, CD: 90, Cast: 1.5, Target: TSelf, GroupHeal: part,
		Apply: []Apply{a(Regen, 10, 0.5, WGroupe)}, Magic: true, Elem: Lumiere, Anim: "sort"}
}

// Classes : les 17 classes, dans l'ordre de l'écran de création.
var Classes = []*Class{
	{
		ID: "guerrier", Names: [3]string{"Guerrier", "Guerrière", "Guerrier·e"},
		Desc: "Maître de toutes les armes, il encaisse ou frappe fort. Sa Rage monte à chaque coup.",
		Res: resRage, Weapons: []string{"epee", "bouclier_rond"}, Shield: true, Anim: "attaque",
		Attr: [9]int{4, 0, 1, 1, 1, 3, 1, 2, 1},
		Skills: [7]SkillDef{
			{ID: "frappe_heroique", Name: "Frappe héroïque", Desc: "Un grand coup d'épée qui te rend un peu de vie.", Level: 1, CD: 4, Gain: 15, Power: 1.6, Drain: 0.4, Anim: "attaque"},
			{ID: "charge", Name: "Charge", Desc: "Fonce sur l'ennemi et l'étourdit 1 s.", Level: 1, CD: 12, Range: 8, Dash: 8, Power: 0.8, Gain: 20, Apply: []Apply{a(Etourdi, 1, 0, WCible)}, Anim: "attaque"},
			{ID: "tourbillon", Name: "Tourbillon", Desc: "Frappe tous les ennemis autour de toi.", Level: 2, CD: 8, Cost: 30, Target: TSelf, Radius: 2.5, Power: 1.2, Anim: "attaque"},
			{ID: "cri_de_guerre", Name: "Cri de guerre", Desc: "Ton groupe frappe 25 % plus fort pendant 10 s.", Level: 3, CD: 30, Target: TSelf, Apply: []Apply{a(Rage, 10, 0, WGroupe)}, Anim: "salut"},
			{ID: "coup_de_bouclier", Name: "Coup de bouclier", Desc: "Étourdit 2 s et attire la colère de l'ennemi.", Level: 5, CD: 10, Power: 0.6, Threat: 5, Gain: 10, Apply: []Apply{a(Etourdi, 2, 0, WCible)}, Anim: "bouclier"},
			{ID: "brise_armure", Name: "Brise-armure", Desc: "L'ennemi reçoit 15 % de dégâts en plus pendant 8 s.", Level: 7, CD: 10, Cost: 20, Power: 1.0, Apply: []Apply{a(Marque, 8, 0, WCible)}, Anim: "attaque"},
			{ID: "mur_de_fer", Name: "Mur de fer", Desc: "Tu reçois 30 % de dégâts en moins pendant 8 s.", Level: 9, CD: 45, Target: TSelf, Apply: []Apply{a(Protection, 8, 0, WSoi)}, Anim: "garde"},
		},
		Specs: [3]Spec{
			{ID: "rempart", Name: "Rempart", Role: Tank, Desc: "Le mur au bouclier : plus de vie, moins de dégâts reçus, beaucoup de menace.", Mods: Mods{Taken: 0.82, Threat: 1.6, HP: 1.15},
				Ultime: ultTank("dernier_rempart", "Dernier rempart", "Protection, bouclier, et tous les ennemis proches n'ont d'yeux que pour toi.")},
			{ID: "gladiateur", Name: "Gladiateur", Role: Degats, Desc: "Deux lames, des coups critiques plus fréquents.", Mods: Mods{Dmg: 1.15, Crit: 0.05},
				Ultime: SkillDef{ID: "danse_des_lames", Name: "Danse des lames", Desc: "Une tornade d'acier qui fait saigner tout autour de toi.", Level: SpecLevel, CD: 60, Cost: 40, Target: TSelf, Radius: 3, Power: 3.0, Apply: []Apply{a(Saignement, 6, 0.4, WZone)}, Anim: "attaque"}},
			{ID: "champion", Name: "Champion", Role: Degats, Desc: "Il provoque les plus forts : 20 % de dégâts en plus contre les élites et les boss.", Mods: Mods{Dmg: 1.1},
				Ultime: SkillDef{ID: "defi_du_champion", Name: "Défi du champion", Desc: "Un coup terrible qui étourdit et provoque.", Level: SpecLevel, CD: 60, Power: 4.0, Threat: 5, Apply: []Apply{a(Etourdi, 1.5, 0, WCible), a(Provoque, 6, 0, WCible)}, Anim: "attaque"}},
		},
	},
	{
		ID: "berserker", Names: [3]string{"Berserker", "Berserker", "Berserker"},
		Desc: "Deux haches et aucune prudence. Plus il saigne, plus sa Furie monte.",
		Res: resFurie, Weapons: []string{"hache"}, Anim: "attaque",
		Attr: [9]int{5, 0, 1, 1, 0, 3, 1, 2, 1},
		Skills: [7]SkillDef{
			{ID: "taillade", Name: "Taillade", Desc: "Un coup de hache rapide.", Level: 1, CD: 3, Gain: 10, Power: 1.4, Anim: "attaque"},
			{ID: "saut_sauvage", Name: "Saut sauvage", Desc: "Bondit sur l'ennemi et ralentit tout autour.", Level: 1, CD: 14, Range: 7, Dash: 7, Radius: 2, Power: 1.0, Apply: []Apply{a(Ralenti, 3, 0, WZone)}, Anim: "attaque"},
			{ID: "hurlement", Name: "Hurlement", Desc: "Les ennemis proches fuient de peur 3 s.", Level: 2, CD: 25, Target: TSelf, Radius: 4, Apply: []Apply{a(Peur, 3, 0, WZone)}, Anim: "salut"},
			{ID: "saignee", Name: "Saignée", Desc: "Une entaille qui saigne 8 s.", Level: 3, CD: 8, Cost: 20, Power: 0.8, Apply: []Apply{a(Saignement, 8, 0.4, WCible)}, Anim: "attaque"},
			{ID: "rage_sanguinaire", Name: "Rage sanguinaire", Desc: "+25 % de dégâts 10 s, et tu reprends 10 % de ta vie.", Level: 5, CD: 40, Target: TSelf, Heal: 0.1, Apply: []Apply{a(Rage, 10, 0, WSoi)}, Anim: "salut"},
			{ID: "execution", Name: "Exécution", Desc: "Énorme, et deux fois plus fort sur un ennemi presque à terre.", Level: 7, CD: 12, Cost: 40, Power: 3.0, Execute: true, Anim: "attaque"},
			{ID: "inarretable", Name: "Inarrêtable", Desc: "Plus rapide et plus solide pendant 6 s.", Level: 9, CD: 45, Target: TSelf, Apply: []Apply{a(Hate, 6, 0, WSoi), a(Protection, 6, 0, WSoi)}, Anim: "salut"},
		},
		Specs: [3]Spec{
			{ID: "sang_bete", Name: "Sang-bête", Role: Degats, Desc: "La bête prend le dessus : plus fort et plus rapide.", Mods: Mods{Dmg: 1.15, Speed: 0.1},
				Ultime: SkillDef{ID: "forme_de_la_bete", Name: "Forme de la bête", Desc: "15 s de rage, de vitesse, et 20 % de vie regagnée.", Level: SpecLevel, CD: 90, Target: TSelf, Heal: 0.2, Apply: []Apply{a(Rage, 15, 0, WSoi), a(Hate, 15, 0, WSoi)}, Anim: "salut"}},
			{ID: "briseur", Name: "Briseur", Role: Degats, Desc: "Des coups qui brisent les armures, et des critiques plus lourds.", Mods: Mods{Dmg: 1.12, Crit: 0.04},
				Ultime: SkillDef{ID: "briseur_de_monts", Name: "Briseur de monts", Desc: "Un coup à fendre une montagne, qui étourdit 3 s.", Level: SpecLevel, CD: 60, Cost: 50, Power: 5.0, Apply: []Apply{a(Etourdi, 3, 0, WCible)}, Anim: "attaque"}},
			{ID: "inebranlable", Name: "Inébranlable", Role: Tank, Desc: "Plus il est blessé, plus il tient.", Mods: Mods{Taken: 0.8, Threat: 1.5, HP: 1.2},
				Ultime: ultTank("indomptable", "Indomptable", "Rien ne t'abat : protection, bouclier, et tous les ennemis proches sur toi.")},
		},
	},
	{
		ID: "lancier", Names: [3]string{"Lancier", "Lancière", "Lancier·e"},
		Desc: "La lance tient l'ennemi à distance. Son Élan monte quand il bouge.",
		Res: resElan, Weapons: []string{"lance"}, Anim: "attaque",
		Attr: [9]int{4, 0, 2, 2, 1, 2, 1, 1, 1},
		Skills: [7]SkillDef{
			{ID: "estoc", Name: "Estoc", Desc: "Un coup de lance qui porte loin.", Level: 1, CD: 3, Gain: 10, Range: 2.5, Power: 1.5, Anim: "attaque"},
			{ID: "balayage", Name: "Balayage", Desc: "La hampe fauche et ralentit les ennemis autour.", Level: 1, CD: 8, Target: TSelf, Radius: 2.5, Power: 1.0, Apply: []Apply{a(Ralenti, 2, 0, WZone)}, Anim: "attaque"},
			{ID: "javelot", Name: "Javelot", Desc: "Lance un javelot à distance.", Level: 2, CD: 6, Cost: 20, Range: 9, Power: 1.6, Anim: "attaque"},
			{ID: "bond", Name: "Bond", Desc: "Saute sur l'ennemi et frappe autour de l'impact.", Level: 3, CD: 14, Range: 8, Dash: 8, Radius: 2, Power: 1.2, Anim: "attaque"},
			{ID: "hampe_tournoyante", Name: "Hampe tournoyante", Desc: "La lance tournoie et frappe fort tout autour.", Level: 5, CD: 15, Cost: 30, Target: TSelf, Radius: 3, Power: 1.8, Anim: "attaque"},
			{ID: "garde_basse", Name: "Garde basse", Desc: "Protection 5 s, et ton Élan remonte.", Level: 7, CD: 25, Target: TSelf, Gain: 30, Apply: []Apply{a(Protection, 5, 0, WSoi)}, Anim: "garde"},
			{ID: "percee", Name: "Percée", Desc: "Un coup qui traverse et fait saigner.", Level: 9, CD: 12, Cost: 40, Range: 2.5, Power: 2.8, Apply: []Apply{a(Saignement, 6, 0.4, WCible)}, Anim: "attaque"},
		},
		Specs: [3]Spec{
			{ID: "hallebardier", Name: "Hallebardier", Role: Degats, Desc: "Les grandes zones : chaque coup compte double quand ils sont nombreux.", Mods: Mods{Dmg: 1.15},
				Ultime: SkillDef{ID: "moisson_d_acier", Name: "Moisson d'acier", Desc: "Fauche tout à 4 cases et étourdit 1 s.", Level: SpecLevel, CD: 60, Cost: 50, Target: TSelf, Radius: 4, Power: 3.0, Apply: []Apply{a(Etourdi, 1, 0, WZone)}, Anim: "attaque"}},
			{ID: "javelinier", Name: "Javelinier", Role: Degats, Desc: "Les lancers : ton attaque de base porte à 7 cases.", Mods: Mods{Dmg: 1.1},
				Ultime: SkillDef{ID: "pluie_de_javelots", Name: "Pluie de javelots", Desc: "Une pluie de javelots sur une zone lointaine.", Level: SpecLevel, CD: 60, Cost: 50, Range: 10, Radius: 3, Power: 3.0, Anim: "attaque"}},
			{ID: "sentinelle", Name: "Sentinelle", Role: Tank, Desc: "Le gardien du passage.", Mods: modsTank,
				Ultime: ultTank("mur_de_lances", "Mur de lances", "Un mur de lances : protection, bouclier, et les ennemis proches sur toi.")},
		},
	},
	{
		ID: "chevalier_dragon", Names: [3]string{"Chevalier-dragon", "Chevalière-dragon", "Chevalier·e-dragon"},
		Desc: "Lié à un dragonneau, il crache le feu et porte des écailles. Son Souffle draconique se recharge seul.",
		Res: resDragon, Weapons: []string{"epee_longue"}, Elem: Feu, Anim: "attaque",
		Attr: [9]int{3, 1, 1, 1, 1, 3, 1, 2, 1},
		Skills: [7]SkillDef{
			{ID: "frappe_d_ecailles", Name: "Frappe d'écailles", Desc: "Un coup d'épée longue.", Level: 1, CD: 3.5, Power: 1.5, Anim: "attaque"},
			{ID: "souffle_ardent", Name: "Souffle ardent", Desc: "Crache le feu devant toi : brûle 4 s.", Level: 1, CD: 8, Cost: 25, Range: 5, Radius: 2, Power: 1.4, Magic: true, Elem: Feu, Apply: []Apply{a(Brulure, 4, 0.3, WZone)}, Anim: "sort"},
			{ID: "plongeon", Name: "Plongeon du dragon", Desc: "Plonge sur l'ennemi et frappe autour.", Level: 2, CD: 14, Range: 8, Dash: 8, Radius: 2, Power: 1.2, Elem: Feu, Anim: "attaque"},
			{ID: "ecailles_durcies", Name: "Écailles durcies", Desc: "Un bouclier d'écailles et 4 s de protection.", Level: 3, CD: 30, Target: TSelf, Apply: []Apply{a(Bouclier, 10, 3, WSoi), a(Protection, 4, 0, WSoi)}, Anim: "garde"},
			{ID: "rugissement", Name: "Rugissement", Desc: "Les ennemis proches fuient 2 s.", Level: 5, CD: 25, Target: TSelf, Radius: 5, Apply: []Apply{a(Peur, 2, 0, WZone)}, Anim: "salut"},
			{ID: "griffe_flamme", Name: "Griffe-flamme", Desc: "Un coup de feu au corps à corps.", Level: 7, CD: 10, Cost: 30, Power: 2.2, Elem: Feu, Anim: "attaque"},
			{ID: "appel_du_dragonneau", Name: "Appel du dragonneau", Desc: "Ton dragonneau se bat à tes côtés 20 s.", Level: 9, CD: 60, Target: TSelf, Apply: []Apply{a(Compagnon, 20, 0.5, WSoi)}, Anim: "salut"},
		},
		Specs: [3]Spec{
			{ID: "ecaille", Name: "Écaille", Role: Tank, Desc: "Une armure d'écailles qui encaisse tout.", Mods: modsTank,
				Ultime: ultTank("rempart_d_ecailles", "Rempart d'écailles", "Tes écailles deviennent un rempart, et les ennemis proches viennent à toi.")},
			{ID: "souffle", Name: "Souffle", Role: Degats, Desc: "Le feu des dragons : 15 % de dégâts en plus.", Mods: Mods{Dmg: 1.15},
				Ultime: SkillDef{ID: "souffle_du_grand_ver", Name: "Souffle du grand ver", Desc: "Un torrent de flammes sur une large zone.", Level: SpecLevel, CD: 60, Cost: 60, Range: 6, Radius: 3.5, Power: 3.5, Magic: true, Elem: Feu, Apply: []Apply{a(Brulure, 6, 0.5, WZone)}, Anim: "sort"}},
			{ID: "lien_du_dragonneau", Name: "Lien du dragonneau", Role: Degats, Desc: "Ton dragonneau ne te quitte plus et se bat toujours avec toi.", Mods: Mods{Dmg: 1.05, PermanentPet: 0.6},
				Ultime: SkillDef{ID: "envol_commun", Name: "Envol commun", Desc: "Ton dragonneau grandit 15 s : il frappe trois fois plus fort, et toi aussi.", Level: SpecLevel, CD: 60, Target: TSelf, Apply: []Apply{a(Compagnon, 15, 1.5, WSoi), a(Rage, 15, 0, WSoi)}, Anim: "salut"}},
		},
	},
	{
		ID: "paladin", Names: [3]string{"Paladin", "Paladine", "Paladin·e"},
		Desc: "Le serment de Solenn : il se soigne, protège le groupe, encaisse et tape fort. Sa Ferveur monte au contact.",
		Res: resFerveur, Weapons: []string{"epee", "bouclier"}, Shield: true, Anim: "attaque",
		Attr: [9]int{3, 2, 0, 1, 1, 3, 1, 2, 1},
		Skills: [7]SkillDef{
			{ID: "frappe_sacree", Name: "Frappe sacrée", Desc: "Un coup d'épée chargé de lumière.", Level: 1, CD: 4, Gain: 15, Power: 1.6, Elem: Lumiere, Anim: "attaque"},
			{ID: "coup_de_bouclier", Name: "Coup de bouclier", Desc: "Étourdit 2 s et attire la colère de l'ennemi.", Level: 2, CD: 10, Gain: 10, Power: 0.6, Threat: 5, Apply: []Apply{a(Etourdi, 2, 0, WCible)}, Anim: "bouclier"},
			{ID: "lumiere_de_solenn", Name: "Lumière de Solenn", Desc: "Soigne un allié (ou toi) de 30 % de sa vie.", Level: 1, CD: 6, Cast: 1.5, Cost: 30, Target: TAlly, Range: 10, Heal: 0.3, Magic: true, Elem: Lumiere, Anim: "sort"},
			{ID: "aura_de_solenn", Name: "Aura de Solenn", Desc: "Ton groupe reçoit 15 % de dégâts en moins et frappe 10 % plus fort.", Level: 3, CD: 30, Target: TSelf, Apply: []Apply{a(Aura, 12, 0, WGroupe)}, Anim: "salut"},
			{ID: "jugement", Name: "Jugement de lumière", Desc: "Un rayon de lumière à distance qui embrase.", Level: 5, CD: 8, Cost: 40, Range: 8, Power: 2.4, Magic: true, Elem: Lumiere, Apply: []Apply{a(Brulure, 4, 0.3, WCible)}, Anim: "sort"},
			{ID: "benediction", Name: "Bénédiction", Desc: "Un bouclier de lumière sur un allié.", Level: 7, CD: 20, Cost: 20, Target: TAlly, Range: 10, Magic: true, Apply: []Apply{a(Bouclier, 10, 3, WCible)}, Anim: "sort"},
			{ID: "consecration", Name: "Consécration", Desc: "Le sol brûle de lumière autour de toi et soigne le groupe.", Level: 9, CD: 15, Cost: 30, Target: TSelf, Radius: 3, Power: 1.4, GroupHeal: 0.1, Magic: true, Elem: Lumiere, Anim: "sort"},
		},
		Specs: [3]Spec{
			{ID: "bouclier_de_l_aube", Name: "Bouclier de l'Aube", Role: Tank, Desc: "Le tank de Solenn : ton bouclier bloque encore mieux.", Mods: Mods{Taken: 0.82, Threat: 1.6, HP: 1.15},
				Ultime: SkillDef{ID: "aube_inviolable", Name: "Aube inviolable", Desc: "Ton groupe est protégé 8 s, et les ennemis proches viennent à toi.", Level: SpecLevel, CD: 120, Target: TSelf, Radius: 6, Threat: 10,
					Apply: []Apply{a(Protection, 8, 0, WGroupe), a(Bouclier, 10, 6, WSoi), a(Provoque, 6, 0, WZone)}, Anim: "garde"}},
			{ID: "heraut", Name: "Héraut", Role: Soutien, Desc: "La voix de Solenn : tes soins et tes auras portent plus loin.", Mods: Mods{Heal: 1.2, Shields: 1.3},
				Ultime: SkillDef{ID: "chant_de_solenn", Name: "Chant de Solenn", Desc: "Soigne tout le groupe de 35 % et l'inspire 15 s.", Level: SpecLevel, CD: 90, Cast: 1.0, Target: TSelf, GroupHeal: 0.35, Magic: true, Elem: Lumiere, Apply: []Apply{a(Inspire, 15, 0, WGroupe)}, Anim: "sort"}},
			{ID: "croise", Name: "Croisé", Role: Degats, Desc: "La lumière qui frappe : 20 % de dégâts en plus.", Mods: Mods{Dmg: 1.2},
				Ultime: SkillDef{ID: "lance_de_solenn", Name: "Lance de Solenn", Desc: "Une lance de lumière qui transperce et étourdit.", Level: SpecLevel, CD: 60, Cost: 50, Range: 8, Power: 4.5, Magic: true, Elem: Lumiere, Apply: []Apply{a(Etourdi, 1.5, 0, WCible)}, Anim: "sort"}},
		},
	},
	{
		ID: "moine", Names: [3]string{"Moine", "Moniale", "Moine"},
		Desc: "Poings et bâton. Ses coups remplissent son Souffle intérieur, que ses coups finaux libèrent.",
		Res: resMoine, Weapons: []string{"baton"}, Anim: "attaque",
		Attr: [9]int{2, 1, 3, 2, 2, 2, 1, 1, 1},
		Skills: [7]SkillDef{
			{ID: "paume_du_tigre", Name: "Paume du tigre", Desc: "Un coup rapide : +1 Souffle.", Level: 1, CD: 2.5, Gain: 1, Power: 1.2, Anim: "attaque"},
			{ID: "pied_tourbillon", Name: "Coup de pied tourbillon", Desc: "Frappe autour de toi : +1 Souffle.", Level: 1, CD: 8, Gain: 1, Target: TSelf, Radius: 2.5, Power: 1.0, Anim: "attaque"},
			{ID: "dragon_de_jade", Name: "Dragon de jade", Desc: "Coup final : dépense tout ton Souffle, +60 % de dégâts par point.", Level: 2, CD: 6, Power: 1.0, Finisher: true, Elem: Air, Anim: "attaque"},
			{ID: "brume_apaisante", Name: "Brume apaisante", Desc: "Soigne un allié et le soigne encore 6 s.", Level: 3, CD: 8, Cast: 1.0, Target: TAlly, Range: 8, Heal: 0.2, Magic: true, Elem: Eau, Apply: []Apply{a(Regen, 6, 0.3, WCible)}, Anim: "sort"},
			{ID: "pas_du_vent", Name: "Pas du vent", Desc: "Glisse jusqu'à l'ennemi, puis cours plus vite 4 s.", Level: 5, CD: 12, Range: 7, Dash: 7, Apply: []Apply{a(Hate, 4, 0, WSoi)}, Anim: "roulade"},
			{ID: "paralysie", Name: "Paralysie", Desc: "Un point de pression : l'ennemi est étourdi 3 s.", Level: 7, CD: 20, Power: 0.3, Apply: []Apply{a(Etourdi, 3, 0, WCible)}, Anim: "attaque"},
			{ID: "peau_de_fer", Name: "Peau de fer", Desc: "6 s de protection, et ta vie remonte.", Level: 9, CD: 45, Target: TSelf, Apply: []Apply{a(Protection, 6, 0, WSoi), a(Regen, 6, 0.4, WSoi)}, Anim: "garde"},
		},
		Specs: [3]Spec{
			{ID: "poing_de_vent", Name: "Poing de vent", Role: Degats, Desc: "Des combos d'une vitesse folle.", Mods: Mods{Dmg: 1.15, Speed: 0.05},
				Ultime: SkillDef{ID: "cent_poings", Name: "Cent poings", Desc: "Une pluie de coups : dépense tout ton Souffle.", Level: SpecLevel, CD: 60, Power: 3.0, Finisher: true, Elem: Air, Anim: "attaque"}},
			{ID: "brume_sereine", Name: "Brume sereine", Role: Soin, Desc: "Soins par la brume et le toucher.", Mods: modsSoin,
				Ultime: ultGroupHeal("renouveau", "Renouveau", "Une brume qui soigne tout le groupe de 40 %.", 0.4)},
			{ID: "pilier", Name: "Pilier", Role: Tank, Desc: "Immobile comme une montagne.", Mods: Mods{Taken: 0.8, Threat: 1.5, HP: 1.2},
				Ultime: ultTank("la_montagne", "La montagne", "Tu deviens la montagne : protection, bouclier, et tout le monde te regarde.")},
		},
	},
	{
		ID: "mage", Names: [3]string{"Mage", "Mage", "Mage"},
		Desc: "Le feu, la glace et la foudre. Son Mana se recharge tout seul.",
		Res: resMana, Weapons: []string{"baton_mage"}, Range: 8, Magic: true, Elem: Feu, Anim: "sort",
		Attr: [9]int{0, 5, 1, 1, 1, 2, 1, 1, 2},
		Skills: [7]SkillDef{
			{ID: "trait_de_feu", Name: "Trait de feu", Desc: "Une flèche de feu.", Level: 1, Cast: 1.2, Cost: 8, Range: 9, Power: 1.6, Magic: true, Elem: Feu, Anim: "sort"},
			{ID: "eclair_de_givre", Name: "Éclair de givre", Desc: "Glace l'ennemi : ralenti 4 s.", Level: 1, CD: 6, Cost: 15, Range: 9, Power: 1.2, Magic: true, Elem: Eau, Apply: []Apply{a(Ralenti, 4, 0, WCible)}, Anim: "sort"},
			{ID: "nova_de_glace", Name: "Nova de glace", Desc: "Gèle sur place les ennemis autour de toi 3 s.", Level: 2, CD: 15, Cost: 20, Target: TSelf, Radius: 3, Power: 0.8, Magic: true, Elem: Eau, Apply: []Apply{a(Gel, 3, 0, WZone)}, Anim: "sort"},
			{ID: "boule_de_feu", Name: "Boule de feu", Desc: "Explose sur la cible et brûle tout autour.", Level: 3, CD: 8, Cast: 1.8, Cost: 25, Range: 9, Radius: 2, Power: 2.2, Magic: true, Elem: Feu, Apply: []Apply{a(Brulure, 4, 0.3, WZone)}, Anim: "sort"},
			{ID: "clignement", Name: "Clignement", Desc: "Disparaît et réapparaît 5 cases en arrière.", Level: 5, CD: 15, Target: TSelf, Dash: -5, Magic: true, Anim: "roulade"},
			{ID: "chaine_d_eclairs", Name: "Chaîne d'éclairs", Desc: "La foudre saute d'ennemi en ennemi.", Level: 7, CD: 10, Cost: 25, Range: 9, Radius: 3, Power: 1.6, Magic: true, Elem: Air, Anim: "sort"},
			{ID: "barriere_de_mana", Name: "Barrière de mana", Desc: "Un bouclier de mana.", Level: 9, CD: 30, Target: TSelf, Magic: true, Apply: []Apply{a(Bouclier, 10, 4, WSoi)}, Anim: "sort"},
		},
		Specs: [3]Spec{
			{ID: "pyromancien", Name: "Pyromancien", Role: Degats, Desc: "Le feu qui brûle en zone : 20 % de dégâts en plus.", Mods: Mods{Dmg: 1.2},
				Ultime: SkillDef{ID: "meteore", Name: "Météore", Desc: "Un météore s'écrase sur la zone et l'embrase.", Level: SpecLevel, CD: 60, Cast: 2, Cost: 50, Range: 10, Radius: 3, Power: 5.0, Magic: true, Elem: Feu, Apply: []Apply{a(Brulure, 6, 0.5, WZone)}, Anim: "sort"}},
			{ID: "givremage", Name: "Givremage", Role: Degats, Desc: "La glace qui gèle et protège : un peu moins de dégâts reçus.", Mods: Mods{Dmg: 1.1, Taken: 0.9},
				Ultime: SkillDef{ID: "blizzard", Name: "Blizzard", Desc: "Une tempête de glace : gel 3 s, ralenti 6 s.", Level: SpecLevel, CD: 60, Cost: 50, Range: 10, Radius: 4, Power: 3.0, Magic: true, Elem: Eau, Apply: []Apply{a(Gel, 3, 0, WZone), a(Ralenti, 6, 0, WZone)}, Anim: "sort"}},
			{ID: "tempetier", Name: "Tempêtier", Role: Degats, Desc: "La foudre, rapide et précise : plus de coups critiques.", Mods: Mods{Dmg: 1.15, Crit: 0.05},
				Ultime: SkillDef{ID: "tempete", Name: "Tempête", Desc: "La foudre tombe sur toute la zone et étourdit 1 s.", Level: SpecLevel, CD: 60, Cost: 50, Range: 10, Radius: 4, Power: 3.5, Magic: true, Elem: Air, Apply: []Apply{a(Etourdi, 1, 0, WZone)}, Anim: "sort"}},
		},
	},
	{
		ID: "necromancien", Names: [3]string{"Nécromancien", "Nécromancienne", "Nécromancien·ne"},
		Desc: "Il parle aux esprits et lève les os. Chaque ennemi qui tombe près de lui lui donne des Âmes.",
		Res: resAmes, Weapons: []string{"faux"}, Range: 7, Magic: true, Elem: Ombre, Anim: "sort",
		Attr: [9]int{0, 5, 1, 1, 1, 2, 1, 2, 1},
		Skills: [7]SkillDef{
			{ID: "trait_d_ombre", Name: "Trait d'ombre", Desc: "Un trait d'ombre : +1 Âme.", Level: 1, Cast: 1.0, Gain: 1, Range: 9, Power: 1.4, Magic: true, Elem: Ombre, Anim: "sort"},
			{ID: "malediction", Name: "Malédiction", Desc: "Ronge l'ennemi 10 s : +1 Âme.", Level: 1, CD: 6, Gain: 1, Range: 9, Magic: true, Elem: Ombre, Apply: []Apply{a(Poison, 10, 0.5, WCible)}, Anim: "sort"},
			{ID: "lever_les_os", Name: "Lever les os", Desc: "Un squelette se bat pour toi 20 s.", Level: 2, CD: 30, Cost: 3, Target: TSelf, Magic: true, Apply: []Apply{a(Compagnon, 20, 0.6, WSoi)}, Anim: "sort"},
			{ID: "drain_de_vie", Name: "Drain de vie", Desc: "Vole la vie de l'ennemi.", Level: 3, CD: 10, Cast: 1.5, Range: 8, Power: 1.6, Drain: 0.6, Magic: true, Elem: Ombre, Anim: "incante"},
			{ID: "faux_spectrale", Name: "Faux spectrale", Desc: "La faux fauche tout autour de toi.", Level: 5, CD: 8, Target: TSelf, Radius: 2.5, Power: 1.3, Elem: Ombre, Anim: "attaque"},
			{ID: "terreur", Name: "Terreur", Desc: "L'ennemi fuit de peur 4 s.", Level: 7, CD: 25, Cost: 2, Range: 8, Magic: true, Apply: []Apply{a(Peur, 4, 0, WCible)}, Anim: "sort"},
			{ID: "pacte_d_os", Name: "Pacte d'os", Desc: "Une armure d'os te protège.", Level: 9, CD: 40, Target: TSelf, Magic: true, Apply: []Apply{a(Bouclier, 10, 4, WSoi)}, Anim: "sort"},
		},
		Specs: [3]Spec{
			{ID: "maitre_des_os", Name: "Maître des Os", Role: Degats, Desc: "Une petite armée : tes squelettes frappent deux fois plus fort.", Mods: Mods{Pet: 2.0, Dmg: 1.05},
				Ultime: SkillDef{ID: "armee_des_os", Name: "Armée des os", Desc: "Toute une armée de squelettes se lève 20 s.", Level: SpecLevel, CD: 90, Cost: 5, Target: TSelf, Magic: true, Apply: []Apply{a(Compagnon, 20, 2.0, WSoi)}, Anim: "sort"}},
			{ID: "moissonneur", Name: "Moissonneur", Role: Degats, Desc: "La faux qui draine la vie : 20 % de dégâts en plus.", Mods: Mods{Dmg: 1.2},
				Ultime: SkillDef{ID: "moisson", Name: "Moisson", Desc: "Fauche toute la zone et vole leur vie.", Level: SpecLevel, CD: 60, Cost: 5, Target: TSelf, Radius: 4, Power: 4.0, Drain: 0.3, Magic: true, Elem: Ombre, Anim: "attaque"}},
			{ID: "passeur", Name: "Passeur", Role: Soin, Desc: "Le gardien des esprits : tes coups soignent l'allié le plus blessé.", Mods: Mods{Heal: 1.3, Lifelink: 0.6},
				Ultime: ultGroupHeal("rappel_des_ames", "Rappel des âmes", "Les esprits soignent tout le groupe de 40 %.", 0.4)},
		},
	},
	{
		ID: "invocateur", Names: [3]string{"Invocateur", "Invocatrice", "Invocateur·rice"},
		Desc: "Ses créatures se battent pour lui. Son Lien se partage entre elles.",
		Res: resLien, Weapons: []string{"orbe"}, Range: 8, Magic: true, Elem: Air, Anim: "sort",
		Attr: [9]int{0, 5, 1, 1, 1, 2, 1, 2, 1},
		Skills: [7]SkillDef{
			{ID: "projectile_d_arcane", Name: "Projectile d'arcane", Desc: "Une boule d'énergie pure.", Level: 1, Cast: 1.0, Cost: 5, Range: 9, Power: 1.3, Magic: true, Elem: Air, Anim: "sort"},
			{ID: "invocation_esprit", Name: "Invocation : esprit", Desc: "Un esprit se bat pour toi 20 s.", Level: 1, CD: 20, Cost: 30, Target: TSelf, Magic: true, Apply: []Apply{a(Compagnon, 20, 0.6, WSoi)}, Anim: "incante"},
			{ID: "ordre_d_attaque", Name: "Ordre d'attaque", Desc: "Tes créatures se jettent sur la cible.", Level: 2, CD: 8, Cost: 15, Range: 9, Power: 1.6, Magic: true, Elem: Air, Anim: "sort"},
			{ID: "sceau_de_lien", Name: "Sceau de lien", Desc: "Ralentit 5 s et marque l'ennemi : +15 % de dégâts reçus.", Level: 3, CD: 15, Range: 9, Magic: true, Apply: []Apply{a(Ralenti, 5, 0, WCible), a(Marque, 8, 0, WCible)}, Anim: "sort"},
			{ID: "bouclier_d_esprit", Name: "Bouclier d'esprit", Desc: "Un esprit protège un allié.", Level: 5, CD: 25, Target: TAlly, Range: 10, Magic: true, Apply: []Apply{a(Bouclier, 10, 3, WCible)}, Anim: "sort"},
			{ID: "explosion_d_esprit", Name: "Explosion d'esprit", Desc: "Un esprit explose sur la zone.", Level: 7, CD: 12, Cost: 25, Range: 9, Radius: 2.5, Power: 2.0, Magic: true, Elem: Air, Anim: "sort"},
			{ID: "echange", Name: "Échange", Desc: "Tu échanges ta place avec un esprit, plus loin en arrière.", Level: 9, CD: 20, Target: TSelf, Dash: -6, Magic: true, Apply: []Apply{a(Hate, 3, 0, WSoi)}, Anim: "roulade"},
		},
		Specs: [3]Spec{
			{ID: "demoniste", Name: "Démoniste", Role: Degats, Desc: "Des démons tenus en laisse : 15 % de dégâts en plus.", Mods: Mods{Dmg: 1.15, Pet: 1.2},
				Ultime: SkillDef{ID: "demon_majeur", Name: "Démon majeur", Desc: "Un démon majeur se bat pour toi 20 s.", Level: SpecLevel, CD: 90, Cost: 40, Target: TSelf, Magic: true, Apply: []Apply{a(Compagnon, 20, 2.2, WSoi)}, Anim: "incante"}},
			{ID: "dresseur_de_chimeres", Name: "Dresseur de chimères", Role: Degats, Desc: "Des bêtes magiques : tes créatures frappent 50 % plus fort.", Mods: Mods{Pet: 1.5},
				Ultime: SkillDef{ID: "chimere", Name: "Chimère", Desc: "Une chimère surgit, frappe et terrifie les ennemis proches.", Level: SpecLevel, CD: 90, Cost: 40, Target: TSelf, Radius: 5, Magic: true, Apply: []Apply{a(Compagnon, 20, 1.5, WSoi), a(Peur, 3, 0, WZone)}, Anim: "incante"}},
			{ID: "lieur_d_elements", Name: "Lieur d'éléments", Role: Degats, Desc: "Il invoque les éléments eux-mêmes : 15 % de dégâts en plus.", Mods: Mods{Dmg: 1.15},
				Ultime: SkillDef{ID: "convergence", Name: "Convergence", Desc: "Feu et glace se rencontrent : brûle et gèle toute la zone.", Level: SpecLevel, CD: 60, Cost: 50, Range: 9, Radius: 3.5, Power: 4.0, Magic: true, Elem: Feu, Apply: []Apply{a(Brulure, 5, 0.5, WZone), a(Gel, 2, 0, WZone)}, Anim: "sort"}},
		},
	},
	{
		ID: "alchimiste", Names: [3]string{"Alchimiste", "Alchimiste", "Alchimiste"},
		Desc: "Fioles, bombes et potions. Ses Réactifs se rechargent un par un.",
		Res: resReactif, Weapons: []string{"fioles"}, Range: 7, Elem: Terre, Anim: "attaque",
		Attr: [9]int{0, 4, 2, 1, 1, 2, 1, 1, 2},
		Skills: [7]SkillDef{
			{ID: "fiole_acide", Name: "Fiole acide", Desc: "Une fiole qui ronge.", Level: 1, Cost: 1, Range: 8, Power: 1.6, Elem: Terre, Anim: "attaque"},
			{ID: "potion_lancee", Name: "Potion lancée", Desc: "Soigne un allié de 18 %.", Level: 1, CD: 4, Cost: 1, Target: TAlly, Range: 9, Heal: 0.18, Anim: "attaque"},
			{ID: "bombe_incendiaire", Name: "Bombe incendiaire", Desc: "Explose et brûle la zone.", Level: 2, CD: 8, Cost: 2, Range: 8, Radius: 2, Power: 1.4, Elem: Feu, Apply: []Apply{a(Brulure, 4, 0.3, WZone)}, Anim: "attaque"},
			{ID: "elixir_de_vigueur", Name: "Élixir de vigueur", Desc: "Un allié se soigne 10 s et court plus vite.", Level: 3, CD: 25, Target: TAlly, Range: 9, Apply: []Apply{a(Regen, 10, 0.6, WCible), a(Hate, 5, 0, WCible)}, Anim: "attaque"},
			{ID: "gaz_aveuglant", Name: "Gaz aveuglant", Desc: "Les ennemis de la zone ratent un coup sur deux 4 s.", Level: 5, CD: 18, Cost: 2, Range: 8, Radius: 2.5, Apply: []Apply{a(Aveugle, 4, 0, WZone)}, Anim: "attaque"},
			{ID: "mixture_explosive", Name: "Mixture explosive", Desc: "Une grosse explosion.", Level: 7, CD: 12, Cost: 3, Range: 8, Radius: 3, Power: 2.4, Elem: Feu, Anim: "attaque"},
			{ID: "transmutation", Name: "Transmutation", Desc: "Tu changes ta peau en pierre et retrouves tous tes Réactifs.", Level: 9, CD: 40, Target: TSelf, Gain: 6, Apply: []Apply{a(Bouclier, 10, 4, WSoi)}, Anim: "sort"},
		},
		Specs: [3]Spec{
			{ID: "apothicaire", Name: "Apothicaire", Role: Soin, Desc: "Des potions de soin plus fortes.", Mods: modsSoin,
				Ultime: ultGroupHeal("grande_panacee", "Grande panacée", "Une pluie de potions soigne tout le groupe de 40 %.", 0.4)},
			{ID: "artificier", Name: "Artificier", Role: Degats, Desc: "Les bombes : 20 % de dégâts en plus.", Mods: Mods{Dmg: 1.2},
				Ultime: SkillDef{ID: "feu_d_artifice", Name: "Feu d'artifice", Desc: "Un bouquet final qui étourdit toute la zone.", Level: SpecLevel, CD: 60, Cost: 4, Range: 9, Radius: 4, Power: 4.0, Elem: Feu, Apply: []Apply{a(Etourdi, 1, 0, WZone)}, Anim: "attaque"}},
			{ID: "empoisonneur", Name: "Empoisonneur", Role: Degats, Desc: "Le maître des poisons.", Mods: Mods{Dmg: 1.15},
				Ultime: SkillDef{ID: "nuee_toxique", Name: "Nuée toxique", Desc: "Un nuage de poison qui ralentit toute la zone.", Level: SpecLevel, CD: 60, Cost: 4, Range: 9, Radius: 4, Power: 1.0, Elem: Terre, Apply: []Apply{a(Poison, 10, 1.0, WZone), a(Ralenti, 8, 0, WZone)}, Anim: "attaque"}},
		},
	},
	{
		ID: "ingenieur", Names: [3]string{"Ingénieur", "Ingénieure", "Ingénieur·e"},
		Desc: "Arquebuse, tourelles et vapeur. Chaque compétence fait chauffer sa Vapeur : attention à la surchauffe !",
		Res: resVapeur, Weapons: []string{"arquebuse"}, Range: 8, Anim: "tir",
		Attr: [9]int{1, 3, 3, 1, 1, 2, 1, 1, 2},
		Skills: [7]SkillDef{
			{ID: "tir_de_precision", Name: "Tir de précision", Desc: "Un tir d'arquebuse. +10 de Vapeur.", Level: 1, Cost: 10, Range: 9, Power: 1.6, Anim: "tir"},
			{ID: "tourelle", Name: "Tourelle", Desc: "Pose une tourelle qui tire 15 s. +25 de Vapeur.", Level: 1, CD: 20, Cost: 25, Target: TSelf, Apply: []Apply{a(Compagnon, 15, 0.7, WSoi)}, Anim: "salut"},
			{ID: "grenade", Name: "Grenade", Desc: "Explose sur la zone. +20 de Vapeur.", Level: 2, CD: 8, Cost: 20, Range: 8, Radius: 2.5, Power: 1.4, Elem: Feu, Anim: "attaque"},
			{ID: "filet", Name: "Filet", Desc: "Cloue l'ennemi sur place 3 s. +10 de Vapeur.", Level: 3, CD: 15, Cost: 10, Range: 8, Apply: []Apply{a(Gel, 3, 0, WCible)}, Anim: "tir"},
			{ID: "reparation", Name: "Réparation d'urgence", Desc: "Soigne un allié de 20 %. +20 de Vapeur.", Level: 5, CD: 20, Cost: 20, Target: TAlly, Range: 8, Heal: 0.2, Anim: "salut"},
			{ID: "salve", Name: "Salve", Desc: "Trois tirs d'un coup. +30 de Vapeur.", Level: 7, CD: 10, Cost: 30, Range: 9, Power: 2.4, Anim: "tir"},
			{ID: "refroidissement", Name: "Refroidissement", Desc: "Vide toute ta Vapeur, et tu cours plus vite 3 s.", Level: 9, CD: 30, Target: TSelf, Gain: 100, Apply: []Apply{a(Hate, 3, 0, WSoi)}, Anim: "salut"},
		},
		Specs: [3]Spec{
			{ID: "artilleur", Name: "Artilleur", Role: Degats, Desc: "Le canon : 20 % de dégâts en plus.", Mods: Mods{Dmg: 1.2},
				Ultime: SkillDef{ID: "grand_canon", Name: "Grand canon", Desc: "Un boulet énorme qui étourdit toute la zone.", Level: SpecLevel, CD: 60, Cost: 40, Range: 10, Radius: 3, Power: 5.0, Elem: Feu, Apply: []Apply{a(Etourdi, 1.5, 0, WZone)}, Anim: "tir"}},
			{ID: "batisseur_de_tourelles", Name: "Bâtisseur de tourelles", Role: Degats, Desc: "Des tourelles qui tirent 50 % plus fort.", Mods: Mods{Pet: 1.5, Dmg: 1.05},
				Ultime: SkillDef{ID: "batterie", Name: "Batterie", Desc: "Toute une batterie de tourelles pendant 20 s.", Level: SpecLevel, CD: 90, Cost: 40, Target: TSelf, Apply: []Apply{a(Compagnon, 20, 2.0, WSoi)}, Anim: "salut"}},
			{ID: "pilote_d_armure", Name: "Pilote d'armure", Role: Tank, Desc: "Une armure à vapeur qui encaisse au contact.", Mods: Mods{Taken: 0.8, Threat: 1.5, HP: 1.2},
				Ultime: ultTank("armure_pleine_vapeur", "Armure à pleine vapeur", "L'armure siffle : protection, bouclier, et tous les ennemis proches sur toi.")},
		},
	},
	{
		ID: "pretre", Names: [3]string{"Prêtre", "Prêtresse", "Prêtre"},
		Desc: "Le vrai soigneur d'Erydor. Sa Grâce se recharge toute seule.",
		Res: resGrace, Weapons: []string{"masse"}, Range: 7, Magic: true, Elem: Lumiere, Anim: "sort",
		Attr: [9]int{0, 5, 0, 1, 1, 3, 1, 2, 1},
		Skills: [7]SkillDef{
			{ID: "chatiment", Name: "Châtiment", Desc: "Un trait de lumière.", Level: 1, Cast: 1.2, Cost: 6, Range: 9, Power: 1.4, Magic: true, Elem: Lumiere, Anim: "sort"},
			{ID: "soin", Name: "Soin", Desc: "Soigne un allié (ou toi) de 25 %.", Level: 1, Cast: 1.5, Cost: 15, Target: TAlly, Range: 10, Heal: 0.25, Magic: true, Elem: Lumiere, Anim: "sort"},
			{ID: "renouveau", Name: "Renouveau", Desc: "L'allié se soigne tout seul pendant 12 s.", Level: 2, CD: 6, Cost: 10, Target: TAlly, Range: 10, Magic: true, Apply: []Apply{a(Regen, 12, 0.5, WCible)}, Anim: "sort"},
			{ID: "bouclier_de_foi", Name: "Bouclier de foi", Desc: "Un bouclier de lumière sur un allié.", Level: 3, CD: 8, Cost: 15, Target: TAlly, Range: 10, Magic: true, Apply: []Apply{a(Bouclier, 10, 2.5, WCible)}, Anim: "sort"},
			{ID: "priere_de_groupe", Name: "Prière de groupe", Desc: "Soigne tout le groupe de 25 %.", Level: 5, CD: 15, Cast: 2, Cost: 35, Target: TSelf, GroupHeal: 0.25, Magic: true, Elem: Lumiere, Anim: "incante"},
			{ID: "sceau_de_solenn", Name: "Sceau de Solenn", Desc: "Une lumière aveuglante : l'ennemi rate un coup sur deux 4 s.", Level: 7, CD: 25, Range: 8, Magic: true, Apply: []Apply{a(Aveugle, 4, 0, WCible)}, Anim: "sort"},
			{ID: "ange_gardien", Name: "Ange gardien", Desc: "Soigne un allié de 50 % et le protège 6 s.", Level: 9, CD: 90, Target: TAlly, Range: 10, Heal: 0.5, Magic: true, Elem: Lumiere, Apply: []Apply{a(Protection, 6, 0, WCible)}, Anim: "sort"},
		},
		Specs: [3]Spec{
			{ID: "oracle", Name: "Oracle", Role: Soin, Desc: "Les gros soins.", Mods: modsSoin,
				Ultime: ultGroupHeal("miracle", "Miracle", "Un miracle soigne tout le groupe de 60 %.", 0.6)},
			{ID: "veilleur", Name: "Veilleur", Role: Soin, Desc: "Les boucliers posés avant les coups : 50 % plus solides.", Mods: Mods{Heal: 1.15, Shields: 1.5},
				Ultime: SkillDef{ID: "egide", Name: "Égide", Desc: "Un grand bouclier sur tout le groupe.", Level: SpecLevel, CD: 90, Target: TSelf, Magic: true, Apply: []Apply{a(Bouclier, 12, 5, WGroupe)}, Anim: "sort"}},
			{ID: "pretre_du_silence", Name: "Prêtre du Silence", Role: Soutien, Desc: "Le prêtre des dieux disparus : il coupe les sorts ennemis et inspire le groupe.", Mods: Mods{Dmg: 1.1, Heal: 1.1},
				Ultime: SkillDef{ID: "le_grand_silence", Name: "Le grand silence", Desc: "Les ennemis proches se taisent et sont étourdis 2 s ; ton groupe est inspiré.", Level: SpecLevel, CD: 90, Target: TSelf, Radius: 5, Magic: true,
					Apply: []Apply{a(Silence, 6, 0, WZone), a(Etourdi, 2, 0, WZone), a(Inspire, 12, 0, WGroupe)}, Anim: "sort"}},
		},
	},
	{
		ID: "druide", Names: [3]string{"Druide", "Druidesse", "Druide"},
		Desc: "Fidèle de Cernos : il soigne par les racines ou prend forme animale. Sa Sève se recharge toute seule.",
		Res: resSeve, Weapons: []string{"baton_noueux"}, Range: 7, Magic: true, Elem: Terre, Anim: "sort",
		Attr: [9]int{1, 4, 1, 1, 1, 3, 1, 2, 1},
		Skills: [7]SkillDef{
			{ID: "epines", Name: "Épines", Desc: "Une volée d'épines.", Level: 1, Cast: 1.0, Cost: 6, Range: 9, Power: 1.4, Magic: true, Elem: Terre, Anim: "sort"},
			{ID: "racines", Name: "Racines", Desc: "Des racines clouent l'ennemi 3 s.", Level: 1, CD: 12, Cost: 15, Range: 9, Power: 0.4, Magic: true, Elem: Terre, Apply: []Apply{a(Gel, 3, 0, WCible)}, Anim: "sort"},
			{ID: "fleur_de_vie", Name: "Fleur de vie", Desc: "Soigne un allié (ou toi) de 22 %.", Level: 2, Cast: 1.5, Cost: 15, Target: TAlly, Range: 10, Heal: 0.22, Magic: true, Elem: Terre, Anim: "sort"},
			{ID: "griffes_sauvages", Name: "Griffes sauvages", Desc: "Des griffes poussent : un coup qui fait saigner.", Level: 3, CD: 6, Power: 1.6, Apply: []Apply{a(Saignement, 6, 0.3, WCible)}, Anim: "attaque"},
			{ID: "nuee_de_lucioles", Name: "Nuée de lucioles", Desc: "Des lucioles brûlantes sur la zone.", Level: 5, CD: 10, Cost: 20, Range: 9, Radius: 2.5, Power: 1.2, Magic: true, Elem: Air, Anim: "sort"},
			{ID: "ecorce", Name: "Écorce", Desc: "La peau d'un allié devient écorce : 30 % de dégâts en moins 6 s.", Level: 7, CD: 30, Target: TAlly, Range: 10, Magic: true, Apply: []Apply{a(Protection, 6, 0, WCible)}, Anim: "sort"},
			{ID: "floraison", Name: "Floraison", Desc: "Tout fleurit : soigne le groupe et continue 8 s.", Level: 9, CD: 20, Cost: 30, Target: TSelf, GroupHeal: 0.18, Magic: true, Elem: Terre, Apply: []Apply{a(Regen, 8, 0.3, WGroupe)}, Anim: "incante"},
		},
		Specs: [3]Spec{
			{ID: "forme_d_ours", Name: "Forme d'ours", Role: Tank, Desc: "Tu encaisses comme un ours.", Mods: Mods{Taken: 0.8, Threat: 1.6, HP: 1.2},
				Ultime: ultTank("colere_de_l_ours", "Colère de l'ours", "Un rugissement d'ours : protection, bouclier, et les ennemis proches sur toi.")},
			{ID: "forme_de_lynx", Name: "Forme de lynx", Role: Degats, Desc: "Tu chasses comme un lynx : plus fort, plus de critiques.", Mods: Mods{Dmg: 1.2, Crit: 0.05},
				Ultime: SkillDef{ID: "bond_du_lynx", Name: "Bond du lynx", Desc: "Bondit sur la proie et la lacère.", Level: SpecLevel, CD: 60, Range: 8, Dash: 8, Power: 4.0, Apply: []Apply{a(Saignement, 8, 0.5, WCible)}, Anim: "attaque"}},
			{ID: "coeur_de_la_foret", Name: "Cœur de la forêt", Role: Soin, Desc: "Les racines et les fleurs soignent plus fort.", Mods: modsSoin,
				Ultime: ultGroupHeal("bosquet_sacre", "Bosquet sacré", "Un bosquet pousse et soigne tout le groupe de 45 %.", 0.45)},
		},
	},
	{
		ID: "chaman", Names: [3]string{"Chaman", "Chamane", "Chaman·e"},
		Desc: "Il appelle les esprits des éléments avec ses totems.",
		Res: resEsprits, Weapons: []string{"totem"}, Range: 7, Magic: true, Elem: Air, Anim: "sort",
		Attr: [9]int{1, 4, 1, 1, 1, 3, 1, 2, 1},
		Skills: [7]SkillDef{
			{ID: "eclair", Name: "Éclair", Desc: "Un éclair.", Level: 1, Cast: 1.2, Cost: 8, Range: 9, Power: 1.5, Magic: true, Elem: Air, Anim: "sort"},
			{ID: "vague_de_soins", Name: "Vague de soins", Desc: "Soigne un allié (ou toi) de 22 %.", Level: 1, Cast: 1.5, Cost: 15, Target: TAlly, Range: 10, Heal: 0.22, Magic: true, Elem: Eau, Anim: "sort"},
			{ID: "totem_de_pierre", Name: "Totem de pierre", Desc: "Ton groupe reçoit 30 % de dégâts en moins 6 s.", Level: 2, CD: 30, Cost: 15, Target: TSelf, Magic: true, Apply: []Apply{a(Protection, 6, 0, WGroupe)}, Anim: "salut"},
			{ID: "horion_de_flammes", Name: "Horion de flammes", Desc: "Brûle l'ennemi.", Level: 3, CD: 8, Cost: 15, Range: 9, Power: 1.2, Magic: true, Elem: Feu, Apply: []Apply{a(Brulure, 6, 0.4, WCible)}, Anim: "sort"},
			{ID: "chaine_de_foudre", Name: "Chaîne de foudre", Desc: "La foudre saute d'ennemi en ennemi.", Level: 5, CD: 10, Cost: 25, Range: 9, Radius: 3, Power: 1.6, Magic: true, Elem: Air, Anim: "sort"},
			{ID: "loup_esprit", Name: "Loup-esprit", Desc: "Un loup-esprit se bat avec toi 20 s.", Level: 7, CD: 45, Target: TSelf, Magic: true, Apply: []Apply{a(Compagnon, 20, 0.8, WSoi)}, Anim: "incante"},
			{ID: "pluie_guerisseuse", Name: "Pluie guérisseuse", Desc: "Une pluie soigne le groupe, et continue 8 s.", Level: 9, CD: 20, Cost: 30, Target: TSelf, GroupHeal: 0.15, Magic: true, Elem: Eau, Apply: []Apply{a(Regen, 8, 0.3, WGroupe)}, Anim: "incante"},
		},
		Specs: [3]Spec{
			{ID: "voix_des_orages", Name: "Voix des orages", Role: Degats, Desc: "La foudre et la terre : 20 % de dégâts en plus.", Mods: Mods{Dmg: 1.2},
				Ultime: SkillDef{ID: "colere_du_ciel", Name: "Colère du ciel", Desc: "Le ciel s'abat sur la zone et étourdit.", Level: SpecLevel, CD: 60, Cost: 50, Range: 10, Radius: 4, Power: 4.5, Magic: true, Elem: Air, Apply: []Apply{a(Etourdi, 1.5, 0, WZone)}, Anim: "sort"}},
			{ID: "source_vive", Name: "Source vive", Role: Soin, Desc: "L'eau et les totems soignent plus fort.", Mods: modsSoin,
				Ultime: ultGroupHeal("maree_de_vie", "Marée de vie", "Une marée soigne tout le groupe de 50 %.", 0.5)},
			{ID: "marcheur_des_esprits", Name: "Marcheur des esprits", Role: Degats, Desc: "Au corps à corps avec un loup-esprit.", Mods: Mods{Dmg: 1.15, Pet: 1.3},
				Ultime: SkillDef{ID: "meute_d_esprits", Name: "Meute d'esprits", Desc: "Toute une meute d'esprits court avec toi 20 s.", Level: SpecLevel, CD: 90, Target: TSelf, Magic: true, Apply: []Apply{a(Compagnon, 20, 2.0, WSoi), a(Hate, 10, 0, WSoi)}, Anim: "incante"}},
		},
	},
	{
		ID: "barde", Names: [3]string{"Barde", "Barde", "Barde"},
		Desc: "Il soigne, charme et frappe en musique. Enchaîner les chants remplit son Harmonie.",
		Res: resHarmo, Weapons: []string{"rapiere", "luth"}, Range: 6, Magic: true, Elem: Air, Anim: "sort",
		Attr: [9]int{0, 4, 2, 1, 1, 2, 2, 1, 1},
		Skills: [7]SkillDef{
			{ID: "note_percante", Name: "Note perçante", Desc: "Une note qui blesse : +15 d'Harmonie.", Level: 1, Gain: 15, Range: 8, Power: 1.2, Magic: true, Elem: Air, Anim: "sort"},
			{ID: "chant_de_vaillance", Name: "Chant de vaillance", Desc: "Ton groupe est inspiré 12 s : +10 % de dégâts et de soins.", Level: 1, CD: 20, Gain: 20, Target: TSelf, Magic: true, Apply: []Apply{a(Inspire, 12, 0, WGroupe)}, Anim: "incante"},
			{ID: "ballade_apaisante", Name: "Ballade apaisante", Desc: "Soigne un allié de 20 % : +15 d'Harmonie.", Level: 2, CD: 6, Cast: 1.5, Gain: 15, Target: TAlly, Range: 10, Heal: 0.2, Magic: true, Elem: Eau, Anim: "incante"},
			{ID: "berceuse", Name: "Berceuse", Desc: "L'ennemi est charmé 5 s : il ne se bat plus.", Level: 3, CD: 20, Gain: 10, Range: 8, Magic: true, Apply: []Apply{a(Charme, 5, 0, WCible)}, Anim: "incante"},
			{ID: "dissonance", Name: "Dissonance", Desc: "Une fausse note qui blesse et ralentit autour de toi.", Level: 5, CD: 10, Gain: 15, Target: TSelf, Radius: 3, Power: 1.4, Magic: true, Elem: Air, Apply: []Apply{a(Ralenti, 3, 0, WZone)}, Anim: "sort"},
			{ID: "hymne_de_resistance", Name: "Hymne de résistance", Desc: "Ton groupe reçoit 30 % de dégâts en moins 6 s.", Level: 7, CD: 30, Gain: 20, Target: TSelf, Magic: true, Apply: []Apply{a(Protection, 6, 0, WGroupe)}, Anim: "incante"},
			{ID: "crescendo", Name: "Crescendo", Desc: "Dépense 60 d'Harmonie en un grand coup de musique.", Level: 9, CD: 15, Cost: 60, Range: 8, Power: 3.0, Magic: true, Elem: Air, Anim: "sort"},
		},
		Specs: [3]Spec{
			{ID: "conteur", Name: "Conteur", Role: Soutien, Desc: "Il endort et charme : tes chants de groupe durent plus longtemps.", Mods: Mods{Dmg: 1.05, Heal: 1.1, Shields: 1.2},
				Ultime: SkillDef{ID: "conte_enchante", Name: "Conte enchanté", Desc: "Tous les ennemis proches sont charmés 6 s ; ton groupe est inspiré.", Level: SpecLevel, CD: 90, Target: TSelf, Radius: 6, Magic: true, Apply: []Apply{a(Charme, 6, 0, WZone), a(Inspire, 15, 0, WGroupe)}, Anim: "incante"}},
			{ID: "virtuose", Name: "Virtuose", Role: Degats, Desc: "Il frappe à distance par le son : 20 % de dégâts en plus.", Mods: Mods{Dmg: 1.2},
				Ultime: SkillDef{ID: "grand_final", Name: "Grand final", Desc: "Un accord final qui étourdit toute la zone.", Level: SpecLevel, CD: 60, Cost: 50, Range: 9, Radius: 4, Power: 4.5, Magic: true, Elem: Air, Apply: []Apply{a(Etourdi, 1, 0, WZone)}, Anim: "sort"}},
			{ID: "troubadour", Name: "Troubadour", Role: Soin, Desc: "Ses chansons soignent.", Mods: modsSoin,
				Ultime: ultGroupHeal("refrain_de_la_vie", "Refrain de la vie", "Un refrain qui soigne tout le groupe de 45 %.", 0.45)},
		},
	},
	{
		ID: "voleur", Names: [3]string{"Voleur", "Voleuse", "Voleur·se"},
		Desc: "Furtivité, poisons et coups bas. Son Énergie remonte très vite.",
		Res: resEnergie, Weapons: []string{"dagues"}, Anim: "attaque",
		Attr: [9]int{2, 0, 4, 2, 2, 1, 2, 1, 2},
		Skills: [7]SkillDef{
			{ID: "coup_sournois", Name: "Coup sournois", Desc: "Un coup de dague.", Level: 1, Cost: 30, Power: 1.4, Anim: "attaque"},
			{ID: "lames_empoisonnees", Name: "Lames empoisonnées", Desc: "Empoisonne l'ennemi 10 s.", Level: 1, CD: 12, Cost: 20, Power: 0.6, Apply: []Apply{a(Poison, 10, 0.5, WCible)}, Anim: "attaque"},
			{ID: "disparition", Name: "Disparition", Desc: "Tu deviens invisible 6 s : les monstres t'oublient, et ton prochain coup frappe double.", Level: 2, CD: 25, Target: TSelf, Apply: []Apply{a(Furtif, 6, 0, WSoi)}, Anim: "roulade"},
			{ID: "pas_de_l_ombre", Name: "Pas de l'ombre", Desc: "Apparaît dans le dos de l'ennemi et frappe.", Level: 3, CD: 15, Range: 8, Dash: 8, Power: 1.2, Elem: Ombre, Anim: "attaque"},
			{ID: "eventail_de_lames", Name: "Éventail de lames", Desc: "Des lames volent tout autour de toi.", Level: 5, CD: 8, Cost: 35, Target: TSelf, Radius: 2.5, Power: 1.2, Anim: "attaque"},
			{ID: "coup_aveuglant", Name: "Coup aveuglant", Desc: "L'ennemi rate un coup sur deux 4 s.", Level: 7, CD: 18, Cost: 20, Power: 0.5, Apply: []Apply{a(Aveugle, 4, 0, WCible)}, Anim: "attaque"},
			{ID: "evasion", Name: "Évasion", Desc: "4 s de vitesse et de protection.", Level: 9, CD: 40, Target: TSelf, Apply: []Apply{a(Hate, 4, 0, WSoi), a(Protection, 4, 0, WSoi)}, Anim: "roulade"},
		},
		Specs: [3]Spec{
			{ID: "assassin", Name: "Assassin", Role: Degats, Desc: "Furtivité et poisons : 20 % de dégâts en plus.", Mods: Mods{Dmg: 1.2},
				Ultime: SkillDef{ID: "assassinat", Name: "Assassinat", Desc: "Un coup mortel, deux fois plus fort sur un ennemi presque à terre.", Level: SpecLevel, CD: 60, Cost: 50, Power: 6.0, Execute: true, Anim: "attaque"}},
			{ID: "bretteur", Name: "Bretteur", Role: Degats, Desc: "Deux lames et des esquives : 10 % d'esquive en plus.", Mods: Mods{Dmg: 1.15, Dodge: 0.1},
				Ultime: SkillDef{ID: "mille_entailles", Name: "Mille entailles", Desc: "Une pluie d'entailles qui fait saigner.", Level: SpecLevel, CD: 60, Cost: 50, Power: 4.0, Apply: []Apply{a(Saignement, 8, 0.6, WCible)}, Anim: "attaque"}},
			{ID: "ombrelame", Name: "Ombrelame", Role: Degats, Desc: "Il saute d'ombre en ombre.", Mods: Mods{Dmg: 1.15, Speed: 0.05},
				Ultime: SkillDef{ID: "danse_des_ombres", Name: "Danse des ombres", Desc: "Surgit au milieu des ennemis et frappe tout autour.", Level: SpecLevel, CD: 60, Cost: 50, Range: 8, Dash: 8, Radius: 3, Power: 3.5, Elem: Ombre, Anim: "attaque"}},
		},
	},
	{
		ID: "archer", Names: [3]string{"Archer", "Archère", "Archer·e"},
		Desc: "Arcs et arbalètes. Sa Concentration monte quand il reste immobile.",
		Res: resConcent, Weapons: []string{"arc", "carquois"}, Range: 9, Anim: "tir",
		Attr: [9]int{1, 0, 4, 1, 2, 2, 1, 1, 3},
		Skills: [7]SkillDef{
			{ID: "tir_vise", Name: "Tir visé", Desc: "Un tir soigneusement visé.", Level: 1, Cast: 1.0, Cost: 15, Range: 10, Power: 1.8, Anim: "tir"},
			{ID: "fleche_rapide", Name: "Flèche rapide", Desc: "Un tir sans viser : +10 de Concentration.", Level: 1, CD: 4, Gain: 10, Range: 10, Power: 1.0, Anim: "tir"},
			{ID: "piege_a_loup", Name: "Piège à loup", Desc: "Cloue l'ennemi sur place 3 s.", Level: 2, CD: 18, Range: 8, Power: 0.4, Apply: []Apply{a(Gel, 3, 0, WCible)}, Anim: "tir"},
			{ID: "salve", Name: "Salve", Desc: "Une volée de flèches sur une zone.", Level: 3, CD: 10, Cost: 30, Range: 10, Radius: 2.5, Power: 1.4, Anim: "tir"},
			{ID: "bond_en_arriere", Name: "Bond en arrière", Desc: "Saute 5 cases en arrière.", Level: 5, CD: 14, Target: TSelf, Dash: -5, Anim: "roulade"},
			{ID: "marque_du_chasseur", Name: "Marque du chasseur", Desc: "L'ennemi reçoit 15 % de dégâts en plus 10 s.", Level: 7, CD: 15, Range: 10, Apply: []Apply{a(Marque, 10, 0, WCible)}, Anim: "tir"},
			{ID: "fleche_perforante", Name: "Flèche perforante", Desc: "Une flèche qui traverse et fait saigner.", Level: 9, CD: 12, Cost: 40, Range: 11, Power: 2.8, Apply: []Apply{a(Saignement, 6, 0.4, WCible)}, Anim: "tir"},
		},
		Specs: [3]Spec{
			{ID: "tireur_d_elite", Name: "Tireur d'élite", Role: Degats, Desc: "Les tirs de très loin : plus fort, plus de critiques.", Mods: Mods{Dmg: 1.2, Crit: 0.05},
				Ultime: SkillDef{ID: "tir_du_siecle", Name: "Tir du siècle", Desc: "Une flèche qu'on racontera longtemps.", Level: SpecLevel, CD: 60, Cast: 2, Cost: 50, Range: 12, Power: 6.5, Anim: "tir"}},
			{ID: "rodeur", Name: "Rôdeur", Role: Degats, Desc: "Il chasse avec un compagnon qui ne le quitte jamais.", Mods: Mods{Dmg: 1.05, PermanentPet: 0.6},
				Ultime: SkillDef{ID: "appel_de_la_meute", Name: "Appel de la meute", Desc: "Toute une meute chasse avec toi 20 s.", Level: SpecLevel, CD: 90, Target: TSelf, Apply: []Apply{a(Compagnon, 20, 2.0, WSoi), a(Hate, 10, 0, WSoi)}, Anim: "salut"}},
			{ID: "trappeur", Name: "Trappeur", Role: Degats, Desc: "Pièges et ralentissements.", Mods: Mods{Dmg: 1.1},
				Ultime: SkillDef{ID: "champ_de_pieges", Name: "Champ de pièges", Desc: "Toute la zone se couvre de pièges : gel 4 s et ralenti 8 s.", Level: SpecLevel, CD: 60, Cost: 40, Range: 9, Radius: 4, Power: 2.0, Apply: []Apply{a(Gel, 4, 0, WZone), a(Ralenti, 8, 0, WZone)}, Anim: "tir"}},
		},
	},
}

// ClassByID retrouve une classe (nil si elle n'existe pas).
func ClassByID(id string) *Class {
	for _, c := range Classes {
		if c.ID == id {
			return c
		}
	}
	return nil
}

// Skill renvoie la compétence k (1 à 8) d'une classe pour la voie donnée
// (-1 : pas encore de voie). La touche 8 est l'ultime de la voie.
func (c *Class) Skill(k, spec int) *SkillDef {
	if k >= 1 && k <= 7 {
		return &c.Skills[k-1]
	}
	if k == 8 && spec >= 0 && spec < 3 {
		return &c.Specs[spec].Ultime
	}
	return nil
}
