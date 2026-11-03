package world

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

// Le client affiche les classes, les compétences et les voies sans les
// recalculer : il lit client/data/heros/classes.json, écrit à partir de
// classes.go. Ce test vérifie que la copie est à jour ; pour la réécrire :
//
//	LF_ECRIRE=1 go test ./world -run DonneesDuClient
const clientClasses = "../../client/data/heros/classes.json"

// icone choisit un pictogramme d'après ce que fait la compétence.
func icone(sk *SkillDef) string {
	has := func(e Effect) bool {
		for _, ap := range sk.Apply {
			if ap.E == e {
				return true
			}
		}
		return false
	}
	switch {
	case sk.Heal > 0 || sk.GroupHeal > 0 || has(Regen):
		return "💚"
	case has(Compagnon):
		return "🐾"
	case has(Bouclier) || has(Protection):
		return "🛡️"
	case has(Furtif):
		return "🌫️"
	case sk.Dash < 0:
		return "💨"
	case has(Provoque):
		return "📣"
	case has(Rage) || has(Inspire) || has(Aura) || has(Hate):
		return "✨"
	case has(Etourdi) || has(Gel) || has(Peur) || has(Charme) || has(Silence):
		return "💫"
	case sk.Finisher:
		return "🐉"
	case sk.Dash > 0:
		return "⚡"
	}
	if sk.Power > 0 {
		switch sk.Elem {
		case Feu:
			return "🔥"
		case Eau:
			return "💧"
		case Terre:
			return "🪨"
		case Air:
			return "🌪️"
		case Lumiere:
			return "☀️"
		case Ombre:
			return "🌑"
		}
		if sk.Radius > 0 {
			return "🌀"
		}
		if sk.Range > 0 {
			return "🎯"
		}
		return "⚔️"
	}
	return "✨"
}

func skillJSON(sk *SkillDef) map[string]any {
	return map[string]any{
		"id": sk.ID, "nom": sk.Name, "aide": sk.Desc, "niveau": sk.Level, "cd": sk.CD, "incante": sk.Cast,
		"cout": sk.Cost, "gain": sk.Gain, "cible": []string{"ennemi", "allie", "soi"}[sk.Target], "portee": sk.Range,
		"magie": sk.Magic, "element": ElementNames[sk.Elem], "zone": sk.Radius, "saut": sk.Dash, "final": sk.Finisher,
		"icone": icone(sk),
	}
}

func talentJSON(tl *Talent) map[string]any {
	return map[string]any{"id": tl.ID, "nom": tl.Name, "aide": tl.Desc, "icone": tl.Icon, "style": StyleNames[tl.Style],
		"max": tl.Max, "comp": tl.Skill}
}

func tierJSON(t *Tier) map[string]any {
	var tl []map[string]any
	for i := range t.T {
		tl = append(tl, talentJSON(&t.T[i]))
	}
	return map[string]any{"niveau": t.Level, "talents": tl}
}

func donneesClient() []byte {
	var classes []map[string]any
	for _, c := range Classes {
		var skills, specs []map[string]any
		for i := range c.Skills {
			skills = append(skills, skillJSON(&c.Skills[i]))
		}
		for i := range c.Specs {
			sp := &c.Specs[i]
			var tiers []map[string]any
			for i := range sp.Tiers {
				tiers = append(tiers, tierJSON(&sp.Tiers[i]))
			}
			specs = append(specs, map[string]any{"id": sp.ID, "nom": sp.Name, "aide": sp.Desc, "role": string(sp.Role),
				"ultime": skillJSON(&sp.Ultime), "paliers": tiers, "etoile": StyleNames[StarStyle(sp.Role)]})
		}
		r := c.Res
		classes = append(classes, map[string]any{
			"id": c.ID, "noms": c.Names, "aide": c.Desc, "armes": c.Weapons, "bouclier": c.Shield, "portee": c.Range,
			"magie": c.Magic, "element": ElementNames[c.Elem], "attaque": c.Anim, "conseil": c.Attr,
			"ressource":   map[string]any{"id": r.ID, "nom": r.Name, "max": r.Max, "chaleur": r.Heat},
			"competences": skills, "voies": specs, "commun": tierJSON(&c.Common),
		})
	}
	races := map[string]any{}
	for id, r := range Races {
		bonus := ""
		for i, v := range r.Bonus {
			if v > 0 {
				bonus = AttrNames[i]
			}
		}
		resist := ""
		if r.Resist != Physique {
			resist = ElementNames[r.Resist]
		}
		races[id] = map[string]any{"faction": r.Faction, "bonus": bonus, "resiste": resist}
	}
	out := map[string]any{
		"_note":      "Écrit par server/world/export_test.go à partir de classes.go et stats.go : ne pas retoucher à la main.",
		"classes":    classes,
		"races":      races,
		"dieux":      Gods,
		"attributs":  AttrNames,
		"effets":     EffectNames,
		"points":     PointsPerLevel,
		"reset":      FreeResetLevel,
		"voie":       SpecLevel,
		"epreuve":    VoieQuest,
		"miroir":     MiroirGold,
		"changement": SpecChangeGold(1),
		"talents":    map[string]any{"premier": TalentFirstLvl, "pas": TalentStep, "max": TalentMax, "reprise": TalentResetGold(1)},
	}
	b, _ := json.MarshalIndent(out, "", " ")
	return append(b, '\n')
}

func TestDonneesDuClient(t *testing.T) {
	want := donneesClient()
	if os.Getenv("LF_ECRIRE") != "" {
		if err := os.WriteFile(clientClasses, want, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, err := os.ReadFile(clientClasses)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatal("client/data/heros/classes.json n'est pas à jour : LF_ECRIRE=1 go test ./world -run DonneesDuClient")
	}
}
