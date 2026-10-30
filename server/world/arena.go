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

// Réglages du Paladin et du combat.
const (
	MaxFerveur    = 100.0
	MaxEndurance  = 2.0 // deux roulades d'affilée
	EnduranceTime = 3.0 // secondes pour regagner une roulade
	RollDist      = 3.5
	RollTime      = 0.35
	AutoEvery     = 2.0 // l'attaque de base du Paladin
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
)

// Skill : une compétence du Paladin.
type Skill struct {
	Name  string
	CD    float64 // recharge, en secondes
	Cast  float64 // incantation (0 : instantanée)
	Cost  float64 // Ferveur dépensée
	Gain  float64 // Ferveur gagnée
	Range float64 // 0 : corps à corps
}

// Skills : la barre du Paladin, touches 1 à 5.
var Skills = [6]Skill{
	{},
	{Name: "Frappe sacrée", CD: 4, Gain: 15},
	{Name: "Coup de bouclier", CD: 10, Gain: 10},
	{Name: "Lumière de Solenn", CD: 6, Cast: 1.5, Cost: 30, Range: 10},
	{Name: "Aura de Solenn", CD: 30},
	{Name: "Jugement de lumière", CD: 8, Cost: 40, Range: 8},
}

// Numéros de compétence en dehors de la barre (pour les événements).
const (
	SkillAuto   = 0
	SkillPotion = 6
	SkillDot    = 7
)

// HeroState : ce qu'un héros a de plus qu'un monstre.
type HeroState struct {
	Target    int
	Ferveur   float64
	Endurance float64
	CD        [6]float64
	GCD       float64
	Cast      int // compétence en cours d'incantation (0 : aucune)
	CastLeft  float64
	castOn    int
	Blocking  bool
	Potions   int
	PotionCD  float64
	XP, Gold  int
	Trophies  map[string]int

	autoCD  float64
	combat  float64 // secondes depuis le dernier coup donné ou reçu
	respawn float64
	regen   float64
	shown   [3]int // Ferveur, Endurance (en dixièmes), potions au dernier envoi
	me      bool   // état privé à renvoyer au joueur
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
)

// Event flags (EvHit).
const (
	FlagCrit  = 1
	FlagHeal  = 2
	FlagBlock = 4
	FlagDodge = 8
	FlagDot   = 16
	FlagElite = 32 // EvDeath : un élite ou un boss (ralenti à l'écran)
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

	rng    *rand.Rand
	teleID int
}

// NewArena prépare l'arène d'une carte et fait apparaître ses monstres.
func NewArena(m *Map, camps []Camp, rng *rand.Rand) *Arena {
	a := &Arena{M: m, Heroes: map[int]*Fighter{}, Camps: camps, rng: rng,
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
func (a *Arena) AddHero(sid int, name string, body *Body, level, xp, gold, potions int, trophies map[string]int) *Fighter {
	if level < 1 {
		level = 1
	}
	if trophies == nil {
		trophies = map[string]int{}
	}
	f := &Fighter{SID: sid, Name: name, Body: body, Level: level, MaxHP: HeroMaxHP(level),
		Hero: &HeroState{Endurance: MaxEndurance, Potions: potions, XP: xp, Gold: gold, Trophies: trophies, combat: CombatTime, me: true}}
	f.HP = f.MaxHP
	a.Heroes[sid] = f
	return f
}

// RemoveHero retire un héros (départ) : les monstres l'oublient.
func (a *Arena) RemoveHero(sid int) {
	delete(a.Heroes, sid)
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

// UseSkill : compétence k (1 à 5) sur la cible actuelle.
func (a *Arena) UseSkill(sid, k int) {
	h := a.Heroes[sid]
	if !a.canAct(h) || k < 1 || k > 5 {
		return
	}
	hs := h.Hero
	sk := Skills[k]
	if hs.CD[k] > 0 || hs.GCD > 0 || hs.Cast != 0 {
		return // le client grise déjà le bouton : inutile d'insister
	}
	if hs.Ferveur < sk.Cost {
		a.refuse(h, "Pas assez de Ferveur !")
		return
	}
	t := a.Get(hs.Target)
	foe := t != nil && t.Mon != nil && !t.Dead && !t.Mon.gone
	w := HeroWeapon(h.Level)
	switch k {
	case 1, 2: // corps à corps
		if !foe {
			a.refuse(h, "Choisis d'abord un monstre !")
			return
		}
		if dist(h, t) > reachOf(t) {
			a.refuse(h, "Trop loin pour frapper !")
			return
		}
		h.Body.Face(t.Body.X, t.Body.Y)
		if k == 1 {
			a.hit(h, t, w*1.6, Lumiere, 1, HeroCrit, 1)
			a.emit(Event{Kind: EvAnim, Src: h.SID, Text: "attaque"})
		} else {
			a.hit(h, t, w*0.6, Physique, 2, HeroCrit, 5)
			a.addThreat(t, h, 30)
			t.Apply(Etourdi, 2, 0, h.SID)
			a.emit(Event{Kind: EvAnim, Src: h.SID, Text: "bouclier"})
		}
	case 3: // soin : un allié visé, sinon soi-même
		on := h
		if t != nil && t.Hero != nil && !t.Dead {
			on = t
		}
		if dist(h, on) > sk.Range {
			a.refuse(h, "Trop loin pour le soigner !")
			return
		}
		hs.Cast, hs.CastLeft, hs.castOn = 3, sk.Cast, on.SID
		hs.Blocking = false
		a.emit(Event{Kind: EvCast, Src: h.SID, Skill: 3, Value: int(sk.Cast * 1000)})
	case 4: // aura de groupe
		for _, o := range a.Heroes {
			if !o.Dead && dist(h, o) <= 8 {
				o.Apply(Aura, 12, 0, h.SID)
			}
		}
		a.emit(Event{Kind: EvAnim, Src: h.SID, Text: "aura"})
	case 5: // jugement, à distance
		if !foe {
			a.refuse(h, "Choisis d'abord un monstre !")
			return
		}
		if dist(h, t) > sk.Range {
			a.refuse(h, "Trop loin pour le jugement !")
			return
		}
		h.Body.Face(t.Body.X, t.Body.Y)
		a.hit(h, t, w*2.4, Lumiere, 5, HeroCrit, 1)
		t.Apply(Brulure, 4, int(math.Max(1, w*0.3)), h.SID)
		a.emit(Event{Kind: EvAnim, Src: h.SID, Text: "sort"})
	}
	if k != 3 { // le soin ne se paie qu'une fois incanté
		hs.Ferveur = math.Min(MaxFerveur, hs.Ferveur-sk.Cost+sk.Gain)
		hs.CD[k] = sk.CD
		hs.Blocking = false
	}
	hs.GCD = GCD
	hs.me = true
	h.dirty = true
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
	if skill == SkillDot {
		ev.Flags |= FlagDot
	}
	if (tgt.Hero != nil && tgt.Body.Dashing()) || (tgt.Mon != nil && tgt.Mon.evading) {
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
	if skill != SkillDot && a.rng.Float64() < crit {
		dmg *= CritMul
		ev.Flags |= FlagCrit
	}
	if tgt.Has(Aura) {
		dmg *= AuraTakenMul
	}
	if tgt.Hero != nil && tgt.Hero.Blocking && skill != SkillDot && facing(tgt, src) {
		dmg *= BlockMul
		ev.Flags |= FlagBlock
	}
	n := int(math.Max(1, math.Round(dmg)))
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
		gain := 5.0
		if ev.Flags&FlagBlock != 0 {
			gain = 8
		}
		tgt.Hero.Ferveur = math.Min(MaxFerveur, tgt.Hero.Ferveur+gain)
	}
	if tgt.Mon != nil && src.Hero != nil {
		a.addThreat(tgt, src, float64(n)*threat)
	}
	if tgt.HP <= 0 {
		a.kill(tgt, src)
	}
	return n
}

func (a *Arena) heal(src, tgt *Fighter, n, skill int) {
	if tgt.Dead {
		return
	}
	n = min(n, tgt.MaxHP-tgt.HP)
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
		h.MaxHP = HeroMaxHP(h.Level)
		if !h.Dead {
			h.HP = h.MaxHP
		}
		h.dirty = true
		a.emit(Event{Kind: EvLevel, Src: h.SID, Value: h.Level})
	}
	hs.me = true
}

// --- Le temps qui passe -------------------------------------------------------------

// Tick fait avancer le combat de dt secondes.
func (a *Arena) Tick(dt float64) {
	a.Moved = a.Moved[:0]
	a.tickTeles(dt)
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
		if e == Brulure || e == Poison {
			x.tick -= dt
			if x.tick <= 1e-6 {
				x.tick += 1
				src := a.Get(x.src)
				if src == nil {
					src = f
				}
				elem := Feu
				if e == Poison {
					elem = Terre
				}
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
	f.Body.Frozen = f.Dead || f.Has(Etourdi)
	mul := 1.0
	if f.Has(Ralenti) {
		mul *= RalentiMul
	}
	if f.Hero != nil {
		if f.Hero.Blocking {
			mul *= BlockSlow
		}
		f.Body.Mul = mul // la vitesse d'un monstre est réglée par son humeur (think)
	}
}

func (a *Arena) tickHero(h *Fighter, dt float64) {
	hs := h.Hero
	if h.Dead {
		hs.respawn -= dt
		if hs.respawn <= 0 {
			h.Dead = false
			h.HP = h.MaxHP
			h.Body.Frozen = false
			x, y := spotNear(a.M, a.Temple[0], a.Temple[1], 1.5, a.rng)
			h.Body.Teleport(x, y)
			h.Apply(Faiblesse, FaiblesseTime, 0, h.SID)
			hs.Ferveur = 0
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
	hs.Endurance = math.Min(MaxEndurance, hs.Endurance+dt/EnduranceTime)
	hs.combat += dt

	// Incantation : bouger ou être étourdi l'interrompt.
	if hs.Cast != 0 {
		if h.Body.Moving() || h.Has(Etourdi) {
			a.interrupt(h)
		} else if hs.CastLeft -= dt; hs.CastLeft <= 0 {
			hs.Cast = 0
			if on := a.Heroes[hs.castOn]; on != nil && !on.Dead && dist(h, on) <= Skills[3].Range+2 && hs.Ferveur >= Skills[3].Cost {
				hs.Ferveur -= Skills[3].Cost
				hs.CD[3] = Skills[3].CD
				hs.me = true
				a.heal(h, on, int(math.Round(float64(on.MaxHP)*0.3)), 3)
				a.emit(Event{Kind: EvAnim, Src: h.SID, Text: "sort"})
			}
		}
	}

	// L'attaque de base tourne toute seule sur la cible, à portée.
	if t := a.Get(hs.Target); t != nil && t.Mon != nil && !t.Dead && hs.autoCD <= 0 && hs.Cast == 0 && !hs.Blocking &&
		a.canAct(h) && dist(h, t) <= reachOf(t) {
		hs.autoCD = AutoEvery
		h.Body.Face(t.Body.X, t.Body.Y)
		a.hit(h, t, HeroWeapon(h.Level), Physique, SkillAuto, HeroCrit, 1)
		hs.Ferveur = math.Min(MaxFerveur, hs.Ferveur+8)
		a.emit(Event{Kind: EvAnim, Src: h.SID, Text: "attaque"})
	}

	// Hors combat : on souffle, la vie remonte et la Ferveur retombe.
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
		hs.Ferveur = math.Max(0, hs.Ferveur-4*dt)
		// Au village, Sœur Clémence remplit la besace.
		if hs.Potions < MaxPotions && math.Hypot(h.Body.X-a.Village[0], h.Body.Y-a.Village[1]) < VillageRadius {
			hs.Potions = MaxPotions
			a.emit(Event{Kind: EvError, Tgt: h.SID, Text: "Sœur Clémence remplit ta besace : 5 potions de soin !"})
		}
	}
	shown := [3]int{int(hs.Ferveur), int(hs.Endurance * 10), hs.Potions}
	if shown != hs.shown {
		hs.shown = shown
		hs.me = true
	}
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
		if h.Dead {
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
