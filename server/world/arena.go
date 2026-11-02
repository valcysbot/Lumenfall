package world

import (
	"math"
	"math/rand"
)

// L'arène : tous ceux qui se battent sur une carte (héros et monstres), et les
// règles qui les font s'affronter. Le match Nakama lui transmet les intentions
// des joueurs et diffuse les événements qu'elle produit.

// MonsterSIDBase : les monstres ont des numéros à part (les habitants sont à 100000).
const MonsterSIDBase = 200000

// Réglages du combat.
const (
	MaxEndurance  = 2.0 // deux roulades d'affilée
	EnduranceTime = 3.0 // secondes pour regagner une roulade
	RollDist      = 3.5
	RollTime      = 0.35
	AutoEvery     = 2.0 // l'attaque de base
	GCD           = 1.0
	MeleeReach    = 1.3 // + la taille du monstre
	MaxPotions    = 5
	PotionCD      = 30.0
	PotionHeal    = 0.4
	CombatTime    = 6.0 // secondes sans coup avant de souffler
	RespawnHero   = 5.0
	VillageRadius = 26.0
	HeroCrit      = 0.10
	MonsterCrit   = 0.05
	CritMul       = 1.75
	BlockMul      = 0.3
	BlockSlow     = 0.5
	BlockArc      = 70 * math.Pi / 180
	OverheatTime  = 3.0 // la Vapeur bloque tout pendant 3 s à 100
	VoieQuest     = 5   // Effacés à vaincre, au niveau 10, pour choisir sa voie
)

// Numéros de compétence en dehors de la barre (pour les événements).
const (
	SkillAuto   = 0
	SkillPotion = 9
	SkillDot    = 10
	SkillPet    = 11
	NSkills     = 8 // touches 1 à 8 (la 8 est l'ultime de la voie)
)

// HeroInit : ce qu'on sait d'un héros en entrant dans le monde.
type HeroInit struct {
	Level, XP, Gold, Potions int
	Trophies                 map[string]int
	Class                    string
	Spec                     int // -1 : pas encore de voie
	Race                     string
	Attrs                    Attrs
	Hardcore                 bool
	Quest                    int
}

// HeroState : ce qu'un héros a de plus qu'un monstre.
type HeroState struct {
	Class    *Class
	Spec     int
	Race     string
	Attrs    Attrs
	Stats    Stats
	Hardcore bool
	Quest    int // Effacés vaincus pour l'épreuve de la voie

	Target    int
	Res       float64 // la ressource de la classe
	Endurance float64
	CD        [NSkills + 1]float64
	GCD       float64
	Cast      int // compétence en cours d'incantation (0 : aucune)
	CastLeft  float64
	castOn    int
	Blocking  bool
	Potions   int
	PotionCD  float64
	XP, Gold  int
	Trophies  map[string]int

	autoCD    float64
	combat    float64 // secondes depuis le dernier coup donné ou reçu
	respawn   float64
	regen     float64
	overheat  float64
	lastSkill int
	petCD     float64
	lastX     float64
	lastY     float64
	shown     [3]int // ressource, endurance (en dixièmes), potions au dernier envoi
	me        bool   // état privé à renvoyer au joueur
}

// Weapon : la puissance de base de l'arme du héros.
func (h *Fighter) Weapon() float64 { return HeroWeapon(h.Level) }

// Power : la puissance d'une attaque, magique ou non, avec les attributs.
func (h *Fighter) Power(magic bool) float64 {
	s := h.Hero.Stats
	if magic {
		return h.Weapon() * s.Mag
	}
	return h.Weapon() * s.Phys
}

// Event kinds.
const (
	EvHit     = iota // Src frappe (ou soigne) Tgt de Value
	EvDeath          // Tgt meurt, tué par Src
	EvCast           // Src commence à incanter Skill pendant Value ms (0 : interrompu)
	EvAnim           // Src joue l'animation Text
	EvLevel          // Src passe au niveau Value
	EvLoot           // Tgt gagne Value or, XP expérience, et peut-être le trophée Text
	EvSpawn          // le monstre Src apparaît
	EvDespawn        // le corps du monstre Src disparaît
	EvRespawn        // le héros Src revient au temple
	EvError          // message privé pour Tgt
	EvTele           // une attaque annoncée au sol (Value : numéro dans Arena.Teles)
	EvPhase          // le boss Src passe à la phase Value
	EvDuel           // duel.go : Text = defi, debut, compte ou fin
	EvVoie           // l'épreuve de la voie : Text = quete, progres, pret ou choisie
)

// Event flags (EvHit).
const (
	FlagCrit     = 1
	FlagHeal     = 2
	FlagBlock    = 4
	FlagDodge    = 8
	FlagDot      = 16
	FlagElite    = 32 // EvDeath : un élite ou un boss (ralenti à l'écran)
	FlagHardcore = 64 // EvDeath : un héros hardcore tombé pour de bon
)

// Event : quelque chose que les joueurs doivent voir.
type Event struct {
	Kind, Src, Tgt, Value, Flags, Skill, XP int
	Elem                                    Element
	Text                                    string
}

// Arena réunit les combattants d'une carte.
type Arena struct {
	M        *Map
	Heroes   map[int]*Fighter
	Monsters []*Fighter
	Camps    []Camp
	Temple   [2]float64 // où reviennent les héros tombés
	Village  [2]float64 // où Sœur Clémence remplit les besaces
	Night    bool
	Events   []Event
	Moved    []*Fighter // monstres qui ont bougé pendant le dernier Tick
	Teles    []*Telegraph
	Duels    []*Duel

	rng     *rand.Rand
	teleID  int
	duelID  int
	invites map[int]invite // défi reçu par chaque héros
}

// NewArena prépare l'arène d'une carte et fait apparaître ses monstres.
func NewArena(m *Map, camps []Camp, rng *rand.Rand) *Arena {
	a := &Arena{M: m, Heroes: map[int]*Fighter{}, Camps: camps, rng: rng, invites: map[int]invite{},
		Temple: m.Spawn, Village: m.Spawn}
	for ci, c := range camps {
		for i := 0; i < c.Count; i++ {
			f := &Fighter{SID: MonsterSIDBase + len(a.Monsters), Body: &Body{Dir: rng.Intn(8)}, Mon: &Monster{Camp: ci}}
			a.Monsters = append(a.Monsters, f)
			a.spawn(f, false)
		}
	}
	a.Events = nil
	return a
}

// Get renvoie un combattant par son numéro.
func (a *Arena) Get(sid int) *Fighter {
	if f, ok := a.Heroes[sid]; ok {
		return f
	}
	if i := sid - MonsterSIDBase; i >= 0 && i < len(a.Monsters) {
		return a.Monsters[i]
	}
	return nil
}

// Visible : un monstre mort depuis longtemps n'est plus montré.
func (f *Fighter) Visible() bool { return f.Mon == nil || !f.Mon.gone }

// Golden dit si ce monstre est un Doré.
func (f *Fighter) Golden() bool { return f.Mon != nil && f.Mon.Golden }

// AddHero fait entrer un héros dans l'arène.
func (a *Arena) AddHero(sid int, name string, body *Body, in HeroInit) *Fighter {
	if in.Level < 1 {
		in.Level = 1
	}
	if in.Trophies == nil {
		in.Trophies = map[string]int{}
	}
	c := ClassByID(in.Class)
	if c == nil {
		c = ClassByID("paladin")
	}
	if in.Spec < -1 || in.Spec > 2 {
		in.Spec = -1
	}
	if _, ok := Races[in.Race]; !ok {
		in.Race = "humain"
	}
	hs := &HeroState{Class: c, Spec: in.Spec, Race: in.Race, Attrs: in.Attrs, Hardcore: in.Hardcore, Quest: in.Quest,
		Endurance: MaxEndurance, Potions: in.Potions, XP: in.XP, Gold: in.Gold, Trophies: in.Trophies,
		combat: CombatTime, me: true, Res: c.Res.Start, lastX: body.X, lastY: body.Y}
	f := &Fighter{SID: sid, Name: name, Body: body, Level: in.Level, Hero: hs}
	f.Restat()
	f.HP = f.MaxHP
	a.Heroes[sid] = f
	return f
}

// Restat recalcule les statistiques d'un héros (niveau, attributs, voie).
func (f *Fighter) Restat() {
	hs := f.Hero
	hs.Stats = ComputeStats(hs.Class, hs.Spec, hs.Race, hs.Attrs, f.Level)
	old := f.MaxHP
	f.MaxHP = hs.Stats.MaxHP
	if old > 0 && f.HP > 0 {
		f.HP = max(1, min(f.MaxHP, f.HP+f.MaxHP-old))
	}
	f.dirty = true
	hs.me = true
}

// RemoveHero retire un héros (départ) : les monstres l'oublient.
func (a *Arena) RemoveHero(sid int) {
	delete(a.Heroes, sid)
	delete(a.invites, sid)
	for _, m := range a.Monsters {
		delete(m.Mon.threat, sid)
	}
}

func (a *Arena) emit(e Event) { a.Events = append(a.Events, e) }

func (a *Arena) refuse(h *Fighter, text string) {
	a.emit(Event{Kind: EvError, Tgt: h.SID, Text: text})
}

func dist(a, b *Fighter) float64 { return math.Hypot(a.Body.X-b.Body.X, a.Body.Y-b.Body.Y) }

func reachOf(m *Fighter) float64 {
	if m.Mon != nil {
		return MeleeReach + m.Mon.Kind.Radius
	}
	return MeleeReach + HeroRadius
}

// --- Intentions des joueurs ---------------------------------------------------

// SetTarget : le joueur vise un combattant (0 : plus personne).
func (a *Arena) SetTarget(sid, target int) {
	h := a.Heroes[sid]
	if h == nil {
		return
	}
	if t := a.Get(target); t == nil || !t.Visible() {
		target = 0
	}
	h.Hero.Target = target
}

// canAct : ni mort, ni étourdi, ni en pleine roulade.
func (a *Arena) canAct(h *Fighter) bool {
	return h != nil && !h.Dead && !h.Has(Etourdi) && !h.Body.Dashing()
}

// Roll : roulade dans la direction (dx, dy), ou droit devant.
func (a *Arena) Roll(sid int, dx, dy float64) {
	h := a.Heroes[sid]
	if !a.canAct(h) {
		return
	}
	if h.Hero.Endurance < 1 {
		a.refuse(h, "Tu es essoufflé !")
		return
	}
	h.Hero.Endurance--
	a.interrupt(h)
	h.Hero.Blocking = false
	h.Body.Dash(dx, dy, RollDist, RollTime)
	h.Hero.me = true
	h.dirty = true
	a.emit(Event{Kind: EvAnim, Src: h.SID, Text: "roulade"})
}

// Block : lever (ou baisser) le bouclier.
func (a *Arena) Block(sid int, on bool) {
	h := a.Heroes[sid]
	if h == nil || h.Dead || h.Hero.Blocking == on {
		return
	}
	if on && !h.Hero.Class.Shield {
		return // seuls les porteurs de bouclier peuvent bloquer
	}
	if on {
		a.interrupt(h)
	}
	h.Hero.Blocking = on
	h.dirty = true
}

// Potion : boire une potion de soin.
func (a *Arena) Potion(sid int) {
	h := a.Heroes[sid]
	if h == nil || h.Dead {
		return
	}
	hs := h.Hero
	switch {
	case hs.Potions <= 0:
		a.refuse(h, "Plus de potions ! Sœur Clémence en a au village.")
	case hs.PotionCD > 0:
	case h.HP >= h.MaxHP:
		a.refuse(h, "Tu es déjà en pleine forme.")
	default:
		hs.Potions--
		hs.PotionCD = PotionCD
		a.heal(h, h, int(math.Round(float64(h.MaxHP)*PotionHeal)), SkillPotion)
		a.emit(Event{Kind: EvAnim, Src: h.SID, Text: "potion"})
		hs.me = true
	}
}

func (a *Arena) interrupt(h *Fighter) {
	if h.Hero.Cast != 0 {
		h.Hero.Cast = 0
		a.emit(Event{Kind: EvCast, Src: h.SID, Skill: 0})
	}
}

// --- Coups et soins ---------------------------------------------------------------

// facing : la cible regarde-t-elle vers l'attaquant (pour le bouclier) ?
func facing(t, src *Fighter) bool {
	dx, dy := src.Body.X-t.Body.X, src.Body.Y-t.Body.Y
	if dx == 0 && dy == 0 {
		return true
	}
	look := float64(t.Body.Dir) * math.Pi / 4
	d := math.Abs(math.Remainder(math.Atan2(dy, dx)-look, 2*math.Pi))
	return d <= BlockArc
}

// hit : src frappe tgt. base = dégâts avant variations ; threat multiplie la menace.
func (a *Arena) hit(src, tgt *Fighter, base float64, elem Element, skill int, crit, threat float64) int {
	if tgt.Dead || (tgt.Mon != nil && tgt.Mon.gone) {
		return 0
	}
	ev := Event{Kind: EvHit, Src: src.SID, Tgt: tgt.SID, Skill: skill, Elem: elem}
	dot := skill == SkillDot
	if dot {
		ev.Flags |= FlagDot
	}
	dodge := (tgt.Hero != nil && tgt.Body.Dashing()) || (tgt.Mon != nil && tgt.Mon.evading)
	if !dot && !dodge {
		dodge = (src.Has(Aveugle) && a.rng.Float64() < AveugleMiss) ||
			(tgt.Hero != nil && a.rng.Float64() < tgt.Hero.Stats.Dodge)
	}
	if dodge {
		ev.Flags |= FlagDodge
		a.emit(ev)
		return 0
	}
	dmg := base * (0.9 + 0.2*a.rng.Float64()) * ElementMul(elem, tgt.Elem)
	if src.Has(Faiblesse) {
		dmg *= FaiblesseMul
	}
	if src.Has(Aura) {
		dmg *= AuraDealtMul
	}
	if src.Has(Rage) {
		dmg *= RageMul
	}
	if src.Has(Inspire) {
		dmg *= InspireMul
	}
	if !dot && src.Has(Furtif) {
		dmg *= FurtifMul
		src.fx[Furtif] = effect{}
		src.dirty = true
	}
	cm := CritMul
	if src.Hero != nil {
		cm = src.Hero.Stats.CritMul
	}
	if !dot && a.rng.Float64() < crit {
		dmg *= cm
		ev.Flags |= FlagCrit
	}
	if tgt.Has(Aura) {
		dmg *= AuraTakenMul
	}
	if tgt.Has(Protection) {
		dmg *= ProtectionMul
	}
	if tgt.Has(Marque) {
		dmg *= MarqueMul
	}
	if tgt.Hero != nil {
		dmg *= tgt.Hero.Stats.Taken
		if r := tgt.Hero.Stats.Resist; r != Physique && r == elem {
			dmg *= 0.9
		}
		if tgt.Hero.Blocking && !dot && facing(tgt, src) {
			dmg *= BlockMul
			ev.Flags |= FlagBlock
		}
	}
	n := int(math.Max(1, math.Round(dmg)))
	// Un bouclier absorbe d'abord.
	if sh := &tgt.fx[Bouclier]; sh.left > 0 && sh.power > 0 {
		take := min(n, sh.power)
		sh.power -= take
		n -= take
		if sh.power <= 0 {
			*sh = effect{}
		}
		tgt.dirty = true
		if n <= 0 {
			ev.Flags |= FlagBlock
			a.emit(ev)
			return 0
		}
	}
	tgt.HP -= n
	tgt.dirty = true
	ev.Value = n
	a.emit(ev)
	for _, f := range []*Fighter{src, tgt} {
		if f.Hero != nil {
			f.Hero.combat = 0
		}
	}
	if tgt.Hero != nil {
		gain := tgt.Hero.Class.Res.OnTaken
		if ev.Flags&FlagBlock != 0 && gain > 0 {
			gain += 3
		}
		a.gainRes(tgt, gain)
	}
	if tgt.Mon != nil && tgt.Has(Charme) {
		tgt.fx[Charme] = effect{} // un coup réveille le charmé
	}
	if tgt.Mon != nil && src.Hero != nil {
		a.addThreat(tgt, src, float64(n)*threat*src.Hero.Stats.Threat)
	}
	if src.Hero != nil && src.Hero.Stats.Lifelink > 0 {
		if o := a.mostHurt(src); o != nil {
			a.heal(src, o, int(math.Round(float64(n)*src.Hero.Stats.Lifelink)), skill)
		}
	}
	if tgt.HP <= 0 && !a.duelBlow(src, tgt) {
		a.kill(tgt, src)
	}
	return n
}

// mostHurt : l'allié le plus blessé autour du héros (lui compris).
func (a *Arena) mostHurt(h *Fighter) *Fighter {
	var best *Fighter
	for _, o := range a.Heroes {
		if o.Dead || dist(h, o) > GroupRadius+2 || a.foe(h, o) {
			continue
		}
		if o.HP < o.MaxHP && (best == nil || o.Pct() < best.Pct()) {
			best = o
		}
	}
	return best
}

func (a *Arena) heal(src, tgt *Fighter, n, skill int) {
	if tgt.Dead || n <= 0 {
		return
	}
	n = min(n, tgt.MaxHP-tgt.HP)
	if n <= 0 {
		return
	}
	tgt.HP += n
	tgt.dirty = true
	a.emit(Event{Kind: EvHit, Src: src.SID, Tgt: tgt.SID, Value: n, Flags: FlagHeal, Skill: skill, Elem: Lumiere})
	// Soigner un héros énerve les monstres qui se battent contre lui.
	for _, m := range a.Monsters {
		if _, ok := m.Mon.threat[tgt.SID]; ok && !m.Dead {
			a.addThreat(m, src, float64(n)*0.5)
		}
	}
}

func (a *Arena) addThreat(m, h *Fighter, v float64) {
	if m.Mon.Kind.Flee || m.Dead || h.Dead || m.Mon.evading {
		return
	}
	if m.Mon.threat == nil {
		m.Mon.threat = map[int]float64{}
	}
	_, known := m.Mon.threat[h.SID]
	first := len(m.Mon.threat) == 0
	m.Mon.threat[h.SID] += v
	if !known && first && m.Mon.Kind.Pack {
		// La meute entend l'appel.
		for _, o := range a.Monsters {
			if o != m && !o.Dead && o.Mon.Camp == m.Mon.Camp && len(o.Mon.threat) == 0 && dist(o, m) <= PackCall {
				a.addThreat(o, h, 1)
			}
		}
	}
}

func (a *Arena) kill(f, killer *Fighter) {
	f.Dead = true
	f.HP = 0
	f.Clear()
	ev := Event{Kind: EvDeath, Src: killer.SID, Tgt: f.SID}
	if f.Mon != nil {
		mo := f.Mon
		if mo.Kind.Elite {
			ev.Flags |= FlagElite
		}
		a.emit(ev)
		a.reward(f)
		a.soulHarvest(f)
		mo.threat = nil
		mo.respawn = RespawnTime
		if mo.Kind.Respawn > 0 {
			mo.respawn = mo.Kind.Respawn
		}
		a.cancelTeles(f.SID)
		f.Body.Stop()
		return
	}
	hs := f.Hero
	a.interrupt(f)
	hs.Blocking = false
	if hs.Hardcore {
		// Hardcore : une seule vie. Le match efface le héros.
		ev.Flags |= FlagHardcore
	}
	hs.respawn = RespawnHero
	f.Body.Stop()
	f.Body.Frozen = true
	for _, m := range a.Monsters {
		delete(m.Mon.threat, f.SID)
	}
	a.emit(ev)
}

// reward : tous les héros qui ont combattu le monstre gagnent l'expérience,
// et chacun tire son propre butin.
func (a *Arena) reward(m *Fighter) {
	k := m.Mon.Kind
	for sid := range m.Mon.threat {
		h := a.Heroes[sid]
		if h == nil {
			continue
		}
		xp := KillXP(m.Level, h.Level, k.XPMul)
		gold := k.Gold[0] + a.rng.Intn(k.Gold[1]-k.Gold[0]+1)
		trophy := ""
		if a.rng.Float64() < k.TrophyP {
			trophy = k.Trophy
			h.Hero.Trophies[trophy]++
		}
		h.Hero.Gold += gold
		a.emit(Event{Kind: EvLoot, Tgt: h.SID, Src: m.SID, Value: gold, XP: xp, Text: trophy})
		a.GainXP(h, xp)
	}
}

// GainXP ajoute de l'expérience et fait monter de niveau.
func (a *Arena) GainXP(h *Fighter, xp int) {
	hs := h.Hero
	hs.XP += xp
	for hs.XP >= XPForLevel(h.Level) {
		hs.XP -= XPForLevel(h.Level)
		h.Level++
		h.Restat()
		if !h.Dead {
			h.HP = h.MaxHP
		}
		a.emit(Event{Kind: EvLevel, Src: h.SID, Value: h.Level})
		if h.Level == SpecLevel && hs.Spec < 0 {
			a.emit(Event{Kind: EvVoie, Tgt: h.SID, Text: "quete", Value: hs.Quest})
		}
	}
	hs.me = true
}

// soulHarvest : les Âmes des nécromanciens proches, et l'épreuve de la voie.
func (a *Arena) soulHarvest(m *Fighter) {
	for _, h := range a.Heroes {
		if h.Dead || dist(h, m) > 10 {
			continue
		}
		if r := h.Hero.Class.Res.OnKill; r > 0 {
			a.gainRes(h, r)
		}
	}
	effaced := m.Mon.Kind.Elem == Ombre
	if !effaced {
		return
	}
	for sid := range m.Mon.threat {
		h := a.Heroes[sid]
		if h == nil || h.Level < SpecLevel || h.Hero.Spec >= 0 || h.Hero.Quest >= VoieQuest {
			continue
		}
		h.Hero.Quest++
		h.Hero.me = true
		text := "progres"
		if h.Hero.Quest >= VoieQuest {
			text = "pret"
		}
		a.emit(Event{Kind: EvVoie, Tgt: h.SID, Text: text, Value: h.Hero.Quest})
	}
}

// --- Le temps qui passe -------------------------------------------------------------

// Tick fait avancer le combat de dt secondes.
func (a *Arena) Tick(dt float64) {
	a.Moved = a.Moved[:0]
	a.tickTeles(dt)
	a.tickDuels(dt)
	for _, h := range a.Heroes {
		a.tickEffects(h, dt)
		a.tickHero(h, dt)
	}
	for _, m := range a.Monsters {
		a.tickEffects(m, dt)
		a.think(m, dt)
		if !m.Dead && m.Body.Moving() && m.Body.Step(a.M, dt) {
			a.Moved = append(a.Moved, m)
		}
	}
}

func (a *Arena) tickEffects(f *Fighter, dt float64) {
	if f.Dead {
		return
	}
	for e := Effect(0); e < nEffects; e++ {
		x := &f.fx[e]
		if x.left <= 0 {
			continue
		}
		x.left -= dt
		switch e {
		case Brulure, Poison, Saignement, Regen:
			x.tick -= dt
			if x.tick <= 1e-6 {
				x.tick += 1
				src := a.Get(x.src)
				if src == nil {
					src = f
				}
				if e == Regen {
					a.heal(src, f, x.power, SkillDot)
					break
				}
				elem := map[Effect]Element{Brulure: Feu, Poison: Terre, Saignement: Physique}[e]
				a.hit(src, f, float64(x.power), elem, SkillDot, 0, 1)
				if f.Dead {
					return
				}
			}
		}
		if x.left <= 1e-6 {
			*x = effect{}
			f.dirty = true
		}
	}
	f.Body.Frozen = f.Dead || f.Has(Etourdi) || f.Has(Gel)
	mul := 1.0
	if f.Has(Ralenti) {
		mul *= RalentiMul
	}
	if f.Hero != nil {
		if f.Has(Hate) {
			mul *= HateMul
		}
		if f.Hero.Blocking {
			mul *= BlockSlow
		}
		f.Body.Mul = mul * f.Hero.Stats.Speed // la vitesse d'un monstre est réglée par son humeur (think)
	}
}

// gainRes ajoute (ou retire) de la ressource, dans ses limites.
func (a *Arena) gainRes(h *Fighter, v float64) {
	hs := h.Hero
	r := hs.Class.Res
	if v > 0 && r.LowHP && h.Pct() < 0.5 {
		v *= 2
	}
	hs.Res = math.Max(0, math.Min(r.Max, hs.Res+v))
}

// petPower : la force du compagnon du héros (0 : pas de compagnon).
func (h *Fighter) petPower() float64 {
	hs := h.Hero
	p := 0.0
	if h.Has(Compagnon) {
		p = float64(h.fx[Compagnon].power)
	}
	if hs.Spec >= 0 {
		if perm := hs.Class.Specs[hs.Spec].Mods.PermanentPet; perm > 0 {
			p = math.Max(p, h.Weapon()*perm*hs.Stats.Pet)
		}
	}
	return p
}

func (a *Arena) tickHero(h *Fighter, dt float64) {
	hs := h.Hero
	if h.Dead {
		hs.respawn -= dt
		if hs.respawn <= 0 && !hs.Hardcore {
			h.Dead = false
			h.HP = h.MaxHP
			h.Body.Frozen = false
			x, y := spotNear(a.M, a.Temple[0], a.Temple[1], 1.5, a.rng)
			h.Body.Teleport(x, y)
			h.Apply(Faiblesse, FaiblesseTime, 0, h.SID)
			hs.Res = hs.Class.Res.Start
			hs.lastX, hs.lastY = x, y
			h.dirty = true
			hs.me = true
			a.emit(Event{Kind: EvRespawn, Src: h.SID})
		}
		return
	}
	for k := range hs.CD {
		hs.CD[k] = math.Max(0, hs.CD[k]-dt)
	}
	hs.GCD = math.Max(0, hs.GCD-dt)
	hs.PotionCD = math.Max(0, hs.PotionCD-dt)
	hs.autoCD = math.Max(0, hs.autoCD-dt)
	hs.overheat = math.Max(0, hs.overheat-dt)
	hs.Endurance = math.Min(MaxEndurance, hs.Endurance+dt/EnduranceTime)
	hs.combat += dt

	// La ressource de la classe.
	r := hs.Class.Res
	moved := math.Hypot(h.Body.X-hs.lastX, h.Body.Y-hs.lastY)
	hs.lastX, hs.lastY = h.Body.X, h.Body.Y
	if r.Heat {
		hs.Res = math.Max(0, hs.Res-r.Regen*dt)
	} else if r.Regen > 0 {
		a.gainRes(h, r.Regen*dt)
	}
	if r.OnMove != 0 && moved > 0 && moved < 2 {
		a.gainRes(h, r.OnMove*moved)
	}
	if r.Still > 0 && !h.Body.Moving() {
		a.gainRes(h, r.Still*dt)
	}
	if hs.combat >= CombatTime && r.Decay > 0 {
		hs.Res = math.Max(0, hs.Res-r.Decay*dt)
	}

	// Incantation : bouger ou être étourdi l'interrompt.
	if hs.Cast != 0 {
		if h.Body.Moving() || h.Has(Etourdi) {
			a.interrupt(h)
		} else if hs.CastLeft -= dt; hs.CastLeft <= 0 {
			k := hs.Cast
			hs.Cast = 0
			sk := hs.Class.Skill(k, hs.Spec)
			if t := a.Get(hs.castOn); sk != nil && a.castTargetOK(h, sk, t) && a.canPay(h, sk, false) {
				a.execute(h, k, sk, t)
			} else {
				a.emit(Event{Kind: EvCast, Src: h.SID, Skill: 0})
			}
		}
	}

	// L'attaque de base tourne toute seule sur la cible, à portée.
	if t := a.Get(hs.Target); a.foe(h, t) && hs.autoCD <= 0 && hs.Cast == 0 && !hs.Blocking &&
		a.canAct(h) && dist(h, t) <= a.autoReach(h, t) {
		hs.autoCD = AutoEvery
		h.Body.Face(t.Body.X, t.Body.Y)
		c := hs.Class
		a.hit(h, t, h.Power(c.Magic)*a.championMul(h, t), c.Elem, SkillAuto, hs.Stats.Crit, 1)
		a.gainRes(h, c.Res.OnDeal)
		a.emit(Event{Kind: EvAnim, Src: h.SID, Text: c.Anim})
	}

	// Le compagnon frappe la cible du héros.
	if p := h.petPower(); p > 0 {
		hs.petCD -= dt
		if t := a.Get(hs.Target); hs.petCD <= 0 && a.foe(h, t) && dist(h, t) <= CompagnonReach {
			hs.petCD = CompagnonEvery
			a.hit(h, t, p, Physique, SkillPet, MonsterCrit, 1)
		}
	}

	// Hors combat : on souffle et la vie remonte.
	if hs.combat >= CombatTime {
		if h.HP < h.MaxHP {
			hs.regen += dt * 0.04 * float64(h.MaxHP)
			if hs.regen >= 1 {
				n := int(hs.regen)
				hs.regen -= float64(n)
				h.HP = min(h.MaxHP, h.HP+n)
				h.dirty = true
			}
		}
		// Au village, Sœur Clémence remplit la besace.
		if hs.Potions < MaxPotions && math.Hypot(h.Body.X-a.Village[0], h.Body.Y-a.Village[1]) < VillageRadius {
			hs.Potions = MaxPotions
			a.emit(Event{Kind: EvError, Tgt: h.SID, Text: "Sœur Clémence remplit ta besace : 5 potions de soin !"})
		}
	}
	shown := [3]int{int(hs.Res * 10), int(hs.Endurance * 10), hs.Potions}
	if shown != hs.shown {
		hs.shown = shown
		hs.me = true
	}
}

// autoReach : la portée de l'attaque de base.
func (a *Arena) autoReach(h *Fighter, t *Fighter) float64 {
	hs := h.Hero
	rng := hs.Class.Range
	if hs.Class.ID == "lancier" && hs.Spec == 1 {
		rng = 7 // Javelinier
	}
	if hs.Class.ID == "chaman" && hs.Spec == 2 {
		rng = 0 // Marcheur des esprits : au corps à corps
	}
	if rng > 0 {
		return rng
	}
	return reachOf(t)
}

// championMul : le Champion frappe 20 % plus fort les élites et les boss.
func (a *Arena) championMul(h, t *Fighter) float64 {
	if h.Hero.Class.ID == "guerrier" && h.Hero.Spec == 2 && t.Mon != nil && (t.Mon.Kind.Elite || t.Mon.Kind.Boss) {
		return 1.2
	}
	return 1
}

// TakeMe dit si l'état privé du héros doit être renvoyé, et remet le drapeau à zéro.
func (h *HeroState) TakeMe() bool {
	m := h.me
	h.me = false
	return m
}

// --- Les monstres -----------------------------------------------------------------

// spawn (ré)installe un monstre dans son camp.
func (a *Arena) spawn(f *Fighter, announce bool) {
	c := a.Camps[f.Mon.Camp]
	mo := f.Mon
	k := MonsterKinds[c.Kind]
	mo.Golden = c.Kind == "sanglier" && announce && a.rng.Float64() < GoldenChance
	if mo.Golden {
		k = MonsterKinds["sanglier_dore"]
	}
	mo.Kind = k
	f.Name = k.Name
	f.Elem = k.Elem
	f.Level = c.Levels[0] + a.rng.Intn(c.Levels[1]-c.Levels[0]+1)
	f.MaxHP = monsterStats(k, f.Level)
	f.HP = f.MaxHP
	f.Dead = false
	f.Clear()
	x, y := spotNear(a.M, c.At[0], c.At[1], c.Radius, a.rng)
	f.Body.Teleport(x, y)
	f.Body.Stop()
	f.Body.Frozen = false
	mo.Home = [2]float64{x, y}
	mo.threat, mo.target, mo.evading, mo.gone, mo.age = nil, 0, false, false, 0
	mo.attackCD, mo.wander = 0, 1+a.rng.Float64()*4
	mo.Phase, mo.chargeCD, mo.stompCD, mo.shardCD, mo.windup = 1, 4, 8, 6, 0
	if announce {
		a.emit(Event{Kind: EvSpawn, Src: f.SID})
	}
}

// think : l'humeur du monstre, une fois par tick.
func (a *Arena) think(f *Fighter, dt float64) {
	mo := f.Mon
	k := mo.Kind
	if f.Dead {
		mo.respawn -= dt
		if !mo.gone && mo.respawn <= math.Max(RespawnTime, k.Respawn)-CorpseTime {
			mo.gone = true
			a.emit(Event{Kind: EvDespawn, Src: f.SID})
		}
		if mo.respawn <= 0 {
			a.spawn(f, true)
		}
		return
	}
	mo.attackCD -= dt
	mo.age += dt
	if f.Has(Etourdi) {
		f.Body.Stop()
		return
	}
	if f.Has(Charme) {
		// Charmé : il oublie le combat le temps du charme.
		f.Body.Stop()
		return
	}
	if f.Has(Peur) {
		// Il fuit celui qui lui a fait peur.
		if src := a.Get(f.fx[Peur].src); src != nil {
			dx, dy := f.Body.X-src.Body.X, f.Body.Y-src.Body.Y
			if l := math.Hypot(dx, dy); l > 1e-6 {
				f.Body.Mul = k.Speed / WalkSpeed
				f.Body.SetInput(dx/l, dy/l)
				return
			}
		}
		f.Body.Stop()
		return
	}
	f.Body.Mul = k.Speed / WalkSpeed
	if f.Has(Ralenti) {
		f.Body.Mul *= RalentiMul
	}
	f.Body.Run = false

	if k.Lifetime > 0 && mo.age > k.Lifetime && len(mo.threat) == 0 {
		// Le Doré s'éclipse s'il n'est pas chassé assez vite.
		f.Dead = true
		mo.gone = true
		mo.respawn = RespawnTime
		a.emit(Event{Kind: EvDespawn, Src: f.SID})
		return
	}
	if mo.evading {
		if math.Hypot(f.Body.X-mo.Home[0], f.Body.Y-mo.Home[1]) < 1 {
			mo.evading = false
			f.Body.Stop()
		} else {
			f.Body.Mul *= 1.4
			if !f.Body.Moving() && !a.goTo(f, mo.Home[0], mo.Home[1]) {
				f.Body.Teleport(mo.Home[0], mo.Home[1])
			}
		}
		if f.HP < f.MaxHP {
			f.HP = f.MaxHP
			f.dirty = true
		}
		return
	}
	if k.Flee {
		a.flee(f, dt)
		return
	}

	// Les héros partis, morts ou trop loin sont oubliés.
	engaged := len(mo.threat) > 0
	for sid := range mo.threat {
		h := a.Heroes[sid]
		if h == nil || h.Dead || math.Hypot(h.Body.X-mo.Home[0], h.Body.Y-mo.Home[1]) > LeashRadius*1.5 {
			delete(mo.threat, sid)
		}
	}
	// Celui qui a le plus de menace est la cible ; on ne change que pour 10 % de plus.
	cur := mo.threat[mo.target]
	best, bestV := 0, 0.0
	for sid, v := range mo.threat {
		if v > bestV {
			best, bestV = sid, v
		}
	}
	if _, ok := mo.threat[mo.target]; !ok || bestV > cur*1.1 {
		mo.target = best
	}
	if f.Has(Provoque) {
		if p := a.Heroes[f.fx[Provoque].src]; p != nil && !p.Dead {
			mo.target = p.SID
		}
	}
	t := a.Heroes[mo.target]
	if t == nil {
		if engaged || f.HP < f.MaxHP {
			a.evade(f)
			return
		}
		a.lookAround(f, dt)
		return
	}
	if math.Hypot(f.Body.X-mo.Home[0], f.Body.Y-mo.Home[1]) > LeashRadius {
		a.evade(f)
		return
	}
	if k.Boss && a.bossThink(f, t, dt) {
		return
	}
	d := dist(f, t)
	if d > k.Reach && f.Has(Gel) {
		f.Body.Stop()
		return
	}
	if d > k.Reach {
		dx, dy := t.Body.X-f.Body.X, t.Body.Y-f.Body.Y
		f.Body.SetInput(dx/d, dy/d)
		return
	}
	f.Body.Stop()
	f.Body.Face(t.Body.X, t.Body.Y)
	if mo.attackCD <= 0 {
		mo.attackCD = k.Every
		if mo.Phase == 2 {
			mo.attackCD *= 0.75
		}
		a.emit(Event{Kind: EvAnim, Src: f.SID, Text: "attaque"})
		if a.hit(f, t, monsterDmg(k, f.Level), k.Elem, SkillAuto, MonsterCrit, 1) > 0 && k.OnHitP > 0 && a.rng.Float64() < k.OnHitP {
			t.Apply(k.OnHit, k.OnHitDur, int(math.Max(1, math.Round(k.OnHitPow*float64(f.Level)))), f.SID)
		}
	}
}

// lookAround : pas de cible ; un héros trop proche se fait remarquer, sinon on flâne.
func (a *Arena) lookAround(f *Fighter, dt float64) {
	mo := f.Mon
	k := mo.Kind
	reach := k.Aggro
	if a.Night && k.Elem == Ombre {
		reach += NightAggro
	}
	var near *Fighter
	nd := reach
	for _, h := range a.Heroes {
		if h.Dead || h.Has(Furtif) {
			continue
		}
		if d := dist(f, h); d < nd {
			near, nd = h, d
		}
	}
	if near != nil {
		a.addThreat(f, near, 1)
		return
	}
	a.stroll(f, dt)
}

func (a *Arena) stroll(f *Fighter, dt float64) {
	mo := f.Mon
	f.Body.Mul = 0.5
	if f.Body.Moving() {
		return
	}
	if mo.wander -= dt; mo.wander > 0 {
		return
	}
	mo.wander = 3 + a.rng.Float64()*6
	r := a.Camps[mo.Camp].Radius
	c := a.Camps[mo.Camp].At
	x, y := spotNear(a.M, c[0], c[1], r, a.rng)
	a.goTo(f, x, y)
}

// flee : le Doré s'enfuit devant les héros, sans jamais quitter son coin.
func (a *Arena) flee(f *Fighter, dt float64) {
	mo := f.Mon
	var near *Fighter
	nd := mo.Kind.Aggro
	for _, h := range a.Heroes {
		if d := dist(f, h); !h.Dead && d < nd {
			near, nd = h, d
		}
	}
	if near == nil {
		a.stroll(f, dt)
		return
	}
	dx, dy := f.Body.X-near.Body.X, f.Body.Y-near.Body.Y
	// S'il s'éloigne trop de chez lui, il vire de bord pour tourner en rond.
	hx, hy := mo.Home[0]-f.Body.X, mo.Home[1]-f.Body.Y
	if hd := math.Hypot(hx, hy); hd > LeashRadius*0.6 {
		dx, dy = dx+hx/hd*nd, dy+hy/hd*nd
	}
	l := math.Hypot(dx, dy)
	if l < 1e-6 {
		dx, dy, l = 1, 0, 1
	}
	f.Body.SetInput(dx/l, dy/l)
}

// evade : le monstre abandonne, rentre chez lui et se remet de ses blessures.
func (a *Arena) evade(f *Fighter) {
	mo := f.Mon
	mo.threat = nil
	mo.target = 0
	mo.evading = true
	if mo.Kind.Boss {
		a.resetBoss(f)
	}
	f.Clear()
	f.Body.Stop()
	a.goTo(f, mo.Home[0], mo.Home[1])
}

func (a *Arena) goTo(f *Fighter, x, y float64) bool {
	p := a.M.FindPath(f.Body.X, f.Body.Y, x, y, 3000)
	if p == nil {
		return false
	}
	f.Body.SetPath(p)
	return true
}

// InCombat dit si le héros s'est battu il y a peu.
func (h *HeroState) InCombat() bool { return h.combat < CombatTime }

// TouchMe demande de renvoyer l'état privé du héros.
func (h *HeroState) TouchMe() { h.me = true }

// Overheat : les secondes de surchauffe restantes (Vapeur).
func (h *HeroState) Overheat() float64 { return h.overheat }
