package world

import (
	"errors"
	"math"
	"strconv"
)

// Le commerce (Acte V) : l'étal d'Aelis (acheter, vendre, racheter), les
// réparations de Doran, les maîtres des métiers, l'échange face à face entre
// héros et le coffre du compte, à l'auberge. L'hôtel des ventes attendra
// Médiolane (Acte IX, choix d'Elowan).

// Réglages du commerce.
const (
	TradeReach   = 6.0 // cases pour proposer un échange
	TradeKeep    = 9.0 // au-delà, l'échange est annulé
	TradeMax     = 8   // objets par côté
	RachatMax    = 10  // ce qu'Aelis garde de côté pour le rachat
	CoffrePlaces = 60
	MiroirPerLv  = 25 // copier une apparence : 25 cuivres par niveau de la pièce
)

// --- L'étal d'Aelis --------------------------------------------------------------

// Buy : acheter n fois un article de l'étal d'Aelis.
func (a *Arena) Buy(sid int, base string, n int) {
	h, inv := a.heroInv(sid)
	if h == nil || n < 1 || n > 99 {
		return
	}
	b := Bases[base]
	ok := false
	for _, s := range Shop {
		ok = ok || s == base
	}
	if !ok || b == nil || b.Price <= 0 {
		return
	}
	if inv.Room(base) < n {
		a.refuse(h, ErrSacPlein.Error())
		return
	}
	if !a.Spend(h, b.Price*n, "achats") {
		a.refuse(h, "Il te faut "+Money(b.Price*n)+".")
		return
	}
	inv.Add(MakeStack(a.rng, base, n))
	a.emit(Event{Kind: EvItem, Tgt: sid, Text: b.Name, Flags: n})
}

// Sell : vendre à Aelis l'objet d'une case du sac.
func (a *Arena) Sell(sid, i int) {
	h, inv := a.heroInv(sid)
	if h == nil {
		return
	}
	it := inv.At(i)
	if it == nil || a.inTrade(h, it) {
		return
	}
	price := it.SellPrice()
	if it.B().Kind != KEquipement {
		price = max(1, it.B().Value/4) * it.Count()
	}
	inv.Sac[i] = nil
	inv.Rachat = append([]*Item{it}, inv.Rachat...)
	if len(inv.Rachat) > RachatMax {
		inv.Rachat = inv.Rachat[:RachatMax]
	}
	inv.Touch()
	a.Earn(h, price, "ventes")
}

// RachatPrice : le prix pour reprendre un objet vendu (le prix de vente).
func RachatPrice(it *Item) int {
	if it.B().Kind != KEquipement {
		return max(1, it.B().Value/4) * it.Count()
	}
	return it.SellPrice()
}

// Buyback : racheter un objet vendu par erreur.
func (a *Arena) Buyback(sid, k int) {
	h, inv := a.heroInv(sid)
	if h == nil || k < 0 || k >= len(inv.Rachat) {
		return
	}
	it := inv.Rachat[k]
	if inv.Free() == 0 {
		a.refuse(h, ErrSacPlein.Error())
		return
	}
	price := RachatPrice(it)
	if h.Hero.Gold < price {
		a.refuse(h, "Il te faut "+Money(price)+".")
		return
	}
	// L'argent revient à Aelis : ni gagné ni perdu pour le monde.
	h.Hero.Gold -= price
	a.Ledger.In["ventes"] -= price
	h.Hero.me = true
	inv.Rachat = append(inv.Rachat[:k], inv.Rachat[k+1:]...)
	inv.Add(it)
}

// RepairCost : ce que coûte la remise à neuf de tout ce que le héros porte et transporte.
func (inv *Inventaire) RepairCost() int {
	n := 0
	for _, it := range inv.Eq {
		if it != nil {
			n += it.RepairPrice()
		}
	}
	for _, it := range inv.Sac {
		if it != nil {
			n += it.RepairPrice()
		}
	}
	return n
}

// Repair : Doran remet tout à neuf.
func (a *Arena) Repair(sid int) {
	h, inv := a.heroInv(sid)
	if h == nil {
		return
	}
	cost := inv.RepairCost()
	if cost == 0 {
		a.refuse(h, "Tout est déjà comme neuf !")
		return
	}
	if !a.Spend(h, cost, "reparations") {
		a.refuse(h, "Doran demande "+Money(cost)+".")
		return
	}
	for _, it := range inv.Eq {
		if it != nil && it.B().Kind == KEquipement {
			it.Dur = 100
		}
	}
	for _, it := range inv.Sac {
		if it != nil && it.B().Kind == KEquipement {
			it.Dur = 100
		}
	}
	inv.Touch()
	h.Restat()
	a.refuse(h, "Doran a tout remis à neuf pour "+Money(cost)+". *Tang*")
}

// --- Les maîtres des métiers -------------------------------------------------------

// LearnMetier : apprendre un métier d'artisanat auprès de son maître.
func (a *Arena) LearnMetier(sid int, id string) error {
	h, inv := a.heroInv(sid)
	if h == nil {
		return errors.New("")
	}
	if err := inv.Learn(id); err != nil {
		return err
	}
	a.emit(Event{Kind: EvMetier, Tgt: sid, Text: id, Value: 1})
	return nil
}

// ForgetMetier : oublier un métier d'artisanat.
func (a *Arena) ForgetMetier(sid int, id string) error {
	_, inv := a.heroInv(sid)
	if inv == nil {
		return errors.New("")
	}
	return inv.Forget(id)
}

// LearnRecipe : acheter une recette au maître.
func (a *Arena) LearnRecipe(sid int, id string) error {
	h, inv := a.heroInv(sid)
	r := Recipes[id]
	if h == nil || r == nil || r.Plan != "" {
		return errors.New("Cette recette ne s'apprend pas ici.")
	}
	switch {
	case !inv.Knows(r.Metier):
		return errors.New("Il faut d'abord apprendre le métier : " + Metiers[r.Metier].Name + ".")
	case inv.Recettes[id]:
		return errors.New("Tu connais déjà cette recette.")
	case inv.Metiers[r.Metier].Lv < r.Req:
		return errors.New("Reviens au niveau " + strconv.Itoa(r.Req) + " en " + Metiers[r.Metier].Name + ".")
	}
	if !a.Spend(h, r.Price, "recettes") {
		return errors.New("Cette recette coûte " + Money(r.Price) + ".")
	}
	inv.Recettes[id] = true
	inv.Touch()
	return nil
}

// --- Le miroir de Solenn : copier une apparence -----------------------------------

// Transmog : la pièce de l'emplacement s prend l'allure d'une pièce déjà portée
// (look : « base:rareté »), contre un peu d'argent. look vide : son allure à elle.
func (a *Arena) Transmog(sid int, s Slot, look string) error {
	h, inv := a.heroInv(sid)
	if h == nil || s < 0 || s >= NSlots {
		return errors.New("")
	}
	it := inv.Eq[s]
	if it == nil {
		return errors.New("Tu ne portes rien à cet endroit.")
	}
	if look == "" {
		it.Look = ""
		inv.Touch()
		a.emit(Event{Kind: EvLook, Src: sid})
		return nil
	}
	if !inv.Apparences[look] {
		return errors.New("Tu n'as jamais porté cette pièce.")
	}
	b := Bases[lookBase(look)]
	if b == nil || b.Slot != it.B().Slot || b.Armor != it.B().Armor || b.Weapon != it.B().Weapon {
		return errors.New("Cette apparence ne va pas à cette pièce.")
	}
	price := MiroirPerLv * it.Lv
	if !a.Spend(h, price, "miroir") {
		return errors.New("Le miroir demande une offrande de " + Money(price) + ".")
	}
	it.Look = look
	inv.Touch()
	a.emit(Event{Kind: EvLook, Src: sid})
	return nil
}

func lookBase(look string) string {
	for i := len(look) - 1; i >= 0; i-- {
		if look[i] == ':' {
			return look[:i]
		}
	}
	return look
}

// --- L'échange face à face -------------------------------------------------------

// Trade : un échange entre deux héros. Chacun pose ses objets et son argent ;
// l'échange se fait quand les deux ont validé (tout changement remet à zéro).
type Trade struct {
	A, B  int
	Items map[int][]int64
	Money map[int]int
	OK    map[int]bool
	ask   bool // la demande n'a pas encore été acceptée
}

// Other : l'autre héros de l'échange.
func (t *Trade) Other(sid int) int {
	if t.A == sid {
		return t.B
	}
	return t.A
}

func (a *Arena) inTrade(h *Fighter, it *Item) bool {
	t := a.Trades[h.SID]
	if t == nil || t.ask || it == nil {
		return false
	}
	for _, id := range t.Items[h.SID] {
		if id == it.ID {
			a.refuse(h, "Cet objet est sur la table d'échange.")
			return true
		}
	}
	return false
}

// TradeAsk : proposer un échange au héros visé.
func (a *Arena) TradeAsk(sid, tgt int) {
	h, inv := a.heroInv(sid)
	o := a.Heroes[tgt]
	if h == nil || inv == nil || o == nil || o.Hero.Inv == nil || o == h {
		return
	}
	switch {
	case a.Trades[sid] != nil:
		a.refuse(h, "Tu échanges déjà avec quelqu'un.")
	case a.Trades[tgt] != nil:
		a.refuse(h, o.Name+" est occupé.")
	case h.Dead || o.Dead || h.Hero.InCombat() || o.Hero.InCombat():
		a.refuse(h, "Pas en plein combat !")
	case dist(h, o) > TradeReach:
		a.refuse(h, "Approche-toi de "+o.Name+" pour échanger.")
	default:
		t := &Trade{A: sid, B: tgt, Items: map[int][]int64{}, Money: map[int]int{}, OK: map[int]bool{}, ask: true}
		a.Trades[sid], a.Trades[tgt] = t, t
		a.emit(Event{Kind: EvTrade, Src: sid, Tgt: tgt, Text: "demande"})
	}
}

// TradeAnswer : accepter (ou refuser) la demande d'échange reçue.
func (a *Arena) TradeAnswer(sid int, ok bool) {
	t := a.Trades[sid]
	if t == nil || !t.ask || t.B != sid {
		return
	}
	if !ok {
		a.endTrade(t, "refus")
		return
	}
	t.ask = false
	a.emit(Event{Kind: EvTrade, Src: t.A, Tgt: t.B, Text: "ouvre"})
}

// TradeOffer : ce que le héros pose sur la table (objets du sac et argent).
func (a *Arena) TradeOffer(sid int, ids []string, money int) {
	h, inv := a.heroInv(sid)
	t := a.Trades[sid]
	if h == nil || t == nil || t.ask {
		return
	}
	var keep []int64
	seen := map[int64]bool{}
	for _, s := range ids {
		id := ParseID(s)
		i := inv.Find(id)
		if i < 0 || seen[id] {
			continue
		}
		if inv.Sac[i].Lie {
			a.refuse(h, inv.Sac[i].Name()+" est lié à toi : il ne s'échange pas.")
			continue
		}
		seen[id] = true
		keep = append(keep, id)
		if len(keep) >= TradeMax {
			break
		}
	}
	t.Items[sid] = keep
	t.Money[sid] = max(0, min(money, h.Hero.Gold))
	t.OK = map[int]bool{}
	a.emit(Event{Kind: EvTrade, Src: t.A, Tgt: t.B, Text: "maj"})
}

// TradeOK : le héros valide. Quand les deux ont validé, l'échange se fait.
func (a *Arena) TradeOK(sid int) {
	t := a.Trades[sid]
	if t == nil || t.ask {
		return
	}
	t.OK[sid] = true
	if !t.OK[t.A] || !t.OK[t.B] {
		a.emit(Event{Kind: EvTrade, Src: t.A, Tgt: t.B, Text: "maj"})
		return
	}
	ha, hb := a.Heroes[t.A], a.Heroes[t.B]
	take := func(h *Fighter) ([]*Item, bool) {
		var out []*Item
		for _, id := range t.Items[h.SID] {
			i := h.Hero.Inv.Find(id)
			if i < 0 {
				return nil, false
			}
			out = append(out, h.Hero.Inv.Sac[i])
		}
		return out, h.Hero.Gold >= t.Money[h.SID]
	}
	ia, okA := take(ha)
	ib, okB := take(hb)
	if !okA || !okB {
		a.endTrade(t, "annule")
		return
	}
	if ha.Hero.Inv.Free()+len(ia) < len(ib) || hb.Hero.Inv.Free()+len(ib) < len(ia) {
		a.refuse(ha, "Pas assez de place dans les sacs pour cet échange.")
		a.refuse(hb, "Pas assez de place dans les sacs pour cet échange.")
		t.OK = map[int]bool{}
		a.emit(Event{Kind: EvTrade, Src: t.A, Tgt: t.B, Text: "maj"})
		return
	}
	move := func(from, to *Fighter, items []*Item) {
		for _, it := range items {
			from.Hero.Inv.Sac[from.Hero.Inv.Find(it.ID)] = nil
		}
		from.Hero.Inv.Touch()
	}
	move(ha, hb, ia)
	move(hb, ha, ib)
	for _, it := range ib {
		ha.Hero.Inv.Add(it)
	}
	for _, it := range ia {
		hb.Hero.Inv.Add(it)
	}
	ma, mb := t.Money[t.A], t.Money[t.B]
	ha.Hero.Gold += mb - ma
	hb.Hero.Gold += ma - mb
	ha.Hero.me, hb.Hero.me = true, true
	a.endTrade(t, "fait")
}

// TradeCancel : annuler l'échange (ou la demande).
func (a *Arena) TradeCancel(sid int) {
	if t := a.Trades[sid]; t != nil {
		a.endTrade(t, "annule")
	}
}

func (a *Arena) endTrade(t *Trade, why string) {
	delete(a.Trades, t.A)
	delete(a.Trades, t.B)
	a.emit(Event{Kind: EvTrade, Src: t.A, Tgt: t.B, Text: "fin:" + why})
}

// tickTrades : un échange s'annule si l'on s'éloigne, se bat ou tombe.
func (a *Arena) tickTrades() {
	for sid, t := range a.Trades {
		if sid != t.A {
			continue
		}
		ha, hb := a.Heroes[t.A], a.Heroes[t.B]
		if ha == nil || hb == nil || ha.Dead || hb.Dead || ha.Hero.InCombat() || hb.Hero.InCombat() {
			a.endTrade(t, "annule")
		} else if dist(ha, hb) > TradeKeep {
			a.endTrade(t, "loin")
		}
	}
}

// TradeView : ce que voit chaque héros de l'échange.
func (a *Arena) TradeView(t *Trade, sid int) map[string]any {
	side := func(s int) map[string]any {
		h := a.Heroes[s]
		var items []ItemView
		if h != nil {
			viewer := a.Heroes[sid]
			for _, id := range t.Items[s] {
				if i := h.Hero.Inv.Find(id); i >= 0 {
					items = append(items, h.Hero.Inv.Sac[i].View(viewer.Hero.Class.ID))
				}
			}
		}
		name := ""
		if h != nil {
			name = h.Name
		}
		return map[string]any{"s": s, "n": name, "it": items, "or": t.Money[s], "ok": t.OK[s]}
	}
	return map[string]any{"moi": side(sid), "lui": side(t.Other(sid))}
}

// --- Le coffre du compte -----------------------------------------------------------

// Coffre : partagé entre les 3 héros d'un compte, à l'auberge.
type Coffre struct {
	Items []*Item `json:"it"`
	Money int     `json:"or"`
	dirty bool
}

// NewCoffre : un coffre vide.
func NewCoffre() *Coffre { return &Coffre{Items: make([]*Item, CoffrePlaces)} }

// Fix complète un coffre lu en base.
func (c *Coffre) Fix() {
	if len(c.Items) < CoffrePlaces {
		c.Items = append(c.Items, make([]*Item, CoffrePlaces-len(c.Items))...)
	}
	for i, it := range c.Items {
		if it != nil && it.B() == nil {
			c.Items[i] = nil
		}
	}
}

// Dirty : le coffre a changé depuis la dernière sauvegarde (et l'oublie).
func (c *Coffre) TakeDirty() bool {
	d := c.dirty
	c.dirty = false
	return d
}

// Deposit : ranger au coffre l'objet d'une case du sac.
func (a *Arena) Deposit(sid, i int, c *Coffre) {
	h, inv := a.heroInv(sid)
	if h == nil || c == nil {
		return
	}
	it := inv.At(i)
	if it == nil || a.inTrade(h, it) {
		return
	}
	for k, o := range c.Items {
		if o == nil {
			c.Items[k] = it
			inv.Sac[i] = nil
			inv.Touch()
			c.dirty = true
			return
		}
	}
	a.refuse(h, "Le coffre est plein.")
}

// Withdraw : reprendre un objet du coffre.
func (a *Arena) Withdraw(sid, k int, c *Coffre) {
	h, inv := a.heroInv(sid)
	if h == nil || c == nil || k < 0 || k >= len(c.Items) || c.Items[k] == nil {
		return
	}
	if rest := inv.Add(c.Items[k]); rest != nil {
		c.Items[k] = rest
		a.refuse(h, ErrSacPlein.Error())
	} else {
		c.Items[k] = nil
	}
	c.dirty = true
}

// CoffreMoney : déposer (n > 0) ou reprendre (n < 0) de l'argent au coffre.
func (a *Arena) CoffreMoney(sid, n int, c *Coffre) {
	h, _ := a.heroInv(sid)
	if h == nil || c == nil || n == 0 {
		return
	}
	if n > 0 {
		n = min(n, h.Hero.Gold)
	} else {
		n = -min(-n, c.Money)
	}
	h.Hero.Gold -= n
	c.Money += n
	h.Hero.me = true
	c.dirty = true
}

// ItemsView : des objets vus par un héros (nil reste nil : une case vide).
func ItemsView(items []*Item, cl string) []*ItemView {
	out := make([]*ItemView, len(items))
	for i, it := range items {
		if it != nil {
			v := it.View(cl)
			out[i] = &v
		}
	}
	return out
}

// round2 : deux décimales.
func round2(v float64) float64 { return math.Round(v*100) / 100 }
