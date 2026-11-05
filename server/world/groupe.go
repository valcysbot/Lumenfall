package world

import (
	"math"
	"sort"
)

// Le groupe (Acte VI) : jusqu'à GroupMax héros qui partent ensemble. Le chef
// invite et exclut ; l'expérience se partage entre les membres proches, avec
// un bonus de groupe ; le butin est tiré aux dés entre eux (choix d'Elowan :
// toujours les dés, sans réglage).

const (
	GroupMax        = 5    // héros dans un groupe
	GroupInviteTime = 60.0 // secondes pour répondre à une invitation
	ShareRadius     = 40.0 // cases : les membres plus loin ne partagent pas l'expérience
	GroupXPBonus    = 0.10 // +10 % par membre en plus
	GuildXPBonus    = 0.05 // +5 % avec un membre de sa guilde (guilde de niveau 2)
)

// Group : des héros qui jouent ensemble.
type Group struct {
	ID      int
	Leader  int
	Members []int // dans l'ordre d'arrivée, le chef compris
}

// Has dit si sid fait partie du groupe.
func (g *Group) Has(sid int) bool {
	for _, m := range g.Members {
		if m == sid {
			return true
		}
	}
	return false
}

// GroupOf renvoie le groupe d'un héros (ou nil).
func (a *Arena) GroupOf(sid int) *Group { return a.groupOf[sid] }

// SameGroup dit si deux héros sont du même groupe.
func (a *Arena) SameGroup(x, y int) bool {
	g := a.groupOf[x]
	return g != nil && x != y && g.Has(y)
}

// GroupInvite : sid invite target dans son groupe (le crée s'il est seul).
func (a *Arena) GroupInvite(sid, target int) {
	h, t := a.Heroes[sid], a.Heroes[target]
	g := a.groupOf[sid]
	switch {
	case h == nil:
		return
	case t == nil || t == h:
		a.refuse(h, "Choisis d'abord un héros à inviter !")
	case g != nil && g.Leader != sid:
		a.refuse(h, "Seul le chef du groupe peut inviter.")
	case g != nil && len(g.Members) >= GroupMax:
		a.refuse(h, "Ton groupe est complet.")
	case a.groupOf[target] != nil:
		a.refuse(h, t.Name+" est déjà dans un groupe.")
	case a.Ignores != nil && a.Ignores(target, sid):
		a.refuse(h, t.Name+" n'accepte pas ton invitation.")
	case a.groupInv[target].from == sid && a.groupInv[target].left > 0:
		a.refuse(h, t.Name+" n'a pas encore répondu.")
	default:
		a.groupInv[target] = invite{from: sid, left: GroupInviteTime}
		a.emit(Event{Kind: EvGroup, Src: sid, Tgt: target, Text: "invite"})
		a.refuse(h, "Tu as invité "+t.Name+" dans ton groupe.")
	}
}

// GroupAnswer : sid accepte (ou refuse) la dernière invitation reçue.
func (a *Arena) GroupAnswer(sid int, ok bool) {
	in, has := a.groupInv[sid]
	delete(a.groupInv, sid)
	h, from := a.Heroes[sid], a.Heroes[in.from]
	if !has || in.left <= 0 || h == nil || from == nil {
		if h != nil && ok {
			a.refuse(h, "Cette invitation n'est plus valable.")
		}
		return
	}
	if !ok {
		a.refuse(from, h.Name+" décline ton invitation.")
		return
	}
	g := a.groupOf[in.from]
	switch {
	case a.groupOf[sid] != nil:
		a.refuse(h, "Tu es déjà dans un groupe.")
		return
	case g != nil && g.Leader != in.from:
		a.refuse(h, "Cette invitation n'est plus valable.")
		return
	case g != nil && len(g.Members) >= GroupMax:
		a.refuse(h, "Ce groupe est déjà complet.")
		return
	}
	if g == nil {
		a.groupID++
		g = &Group{ID: a.groupID, Leader: in.from, Members: []int{in.from}}
		a.Groups[g.ID] = g
		a.groupOf[in.from] = g
	}
	g.Members = append(g.Members, sid)
	a.groupOf[sid] = g
	a.emit(Event{Kind: EvGroup, Src: g.ID, Tgt: sid, Text: "rejoint"})
}

// GroupLeave : sid quitte son groupe. Un groupe d'un seul héros se dissout.
func (a *Arena) GroupLeave(sid int) {
	g := a.groupOf[sid]
	if g == nil {
		return
	}
	delete(a.groupOf, sid)
	for i, m := range g.Members {
		if m == sid {
			g.Members = append(g.Members[:i], g.Members[i+1:]...)
			break
		}
	}
	a.emit(Event{Kind: EvGroup, Src: g.ID, Tgt: sid, Text: "quitte"})
	if len(g.Members) <= 1 {
		for _, m := range g.Members {
			delete(a.groupOf, m)
			a.emit(Event{Kind: EvGroup, Src: g.ID, Tgt: m, Text: "quitte"})
		}
		delete(a.Groups, g.ID)
		return
	}
	if g.Leader == sid {
		g.Leader = g.Members[0]
	}
	a.emit(Event{Kind: EvGroup, Src: g.ID, Text: "maj"})
}

// GroupKick : le chef exclut un membre.
func (a *Arena) GroupKick(sid, target int) {
	g := a.groupOf[sid]
	h := a.Heroes[sid]
	if g == nil || h == nil || target == sid || !g.Has(target) {
		return
	}
	if g.Leader != sid {
		a.refuse(h, "Seul le chef du groupe peut exclure.")
		return
	}
	if t := a.Heroes[target]; t != nil {
		a.refuse(t, "Tu as été exclu du groupe.")
	}
	a.GroupLeave(target)
}

// GroupLead : le chef passe la main.
func (a *Arena) GroupLead(sid, target int) {
	g := a.groupOf[sid]
	if g == nil || g.Leader != sid || !g.Has(target) || target == sid {
		return
	}
	g.Leader = target
	a.emit(Event{Kind: EvGroup, Src: g.ID, Text: "maj"})
}

func (a *Arena) tickGroups(dt float64) {
	for sid, in := range a.groupInv {
		in.left -= dt
		if in.left <= 0 || a.Heroes[sid] == nil || a.Heroes[in.from] == nil {
			delete(a.groupInv, sid)
			continue
		}
		a.groupInv[sid] = in
	}
}

// sharers : ceux qui profitent d'un monstre tombé. Ceux qui l'ont combattu,
// et les membres de leurs groupes qui sont assez près (vivants ou non).
func (a *Arena) sharers(m *Fighter) []*Fighter {
	seen := map[int]bool{}
	var out []*Fighter
	add := func(h *Fighter) {
		if h != nil && !seen[h.SID] {
			seen[h.SID] = true
			out = append(out, h)
		}
	}
	sids := make([]int, 0, len(m.Mon.threat))
	for sid := range m.Mon.threat {
		sids = append(sids, sid)
	}
	sort.Ints(sids)
	for _, sid := range sids {
		h := a.Heroes[sid]
		if h == nil {
			continue
		}
		add(h)
		if g := a.groupOf[sid]; g != nil {
			for _, o := range g.Members {
				if f := a.Heroes[o]; f != nil && math.Hypot(f.Body.X-m.Body.X, f.Body.Y-m.Body.Y) <= ShareRadius {
					add(f)
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SID < out[j].SID })
	return out
}

// shareXP : l'expérience d'un monstre pour le héros h, quand n héros d'un même
// groupe se la partagent (n = 1 : seul). La guilde ajoute son bonus.
func (a *Arena) shareXP(m *Fighter, h *Fighter, n int, guildMate bool) int {
	xp := float64(KillXP(m.Level, h.Level, m.Mon.Kind.XPMul))
	if n > 1 {
		xp = xp * (1 + GroupXPBonus*float64(n-1)) / float64(n)
	}
	if guildMate && h.Hero.GuildLv >= 2 {
		xp *= 1 + GuildXPBonus
	}
	return int(math.Max(1, math.Round(xp)))
}

// partyCount : combien de héros du groupe de h se partagent ce monstre, et si
// l'un d'eux est de sa guilde.
func (a *Arena) partyCount(h *Fighter, among []*Fighter) (int, bool) {
	g := a.groupOf[h.SID]
	if g == nil {
		return 1, false
	}
	n, mate := 0, false
	for _, o := range among {
		if g.Has(o.SID) {
			n++
			if o != h && h.Hero.Guild != 0 && o.Hero.Guild == h.Hero.Guild {
				mate = true
			}
		}
	}
	return max(1, n), mate
}
