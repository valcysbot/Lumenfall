package world

import (
	"math/rand"
	"time"
)

// Le ciel d'Erydor : l'heure du jour, la même partout, et la météo, propre à
// chaque région.

const (
	// DayLength est la durée d'une journée complète d'Erydor (2 heures réelles).
	DayLength = 2 * time.Hour
	// DaylightLength est la part de jour : 80 minutes de jour, puis 40 de nuit.
	DaylightLength = 80 * time.Minute
)

// DayPhase renvoie l'avancée de la journée, de 0 (lever du soleil) à 1.
// Elle ne dépend que de l'heure réelle, donc tous les serveurs et toutes les
// régions sont d'accord, même après un redémarrage.
func DayPhase(t time.Time) float64 {
	ms := t.UnixMilli() % DayLength.Milliseconds()
	return float64(ms) / float64(DayLength.Milliseconds())
}

// IsNight indique s'il fait nuit à l'instant t.
func IsNight(t time.Time) bool {
	return DayPhase(t) >= float64(DaylightLength)/float64(DayLength)
}

// Les temps qu'il peut faire.
const (
	Clear = "clair"
	Rain  = "pluie"
	Fog   = "brouillard"
	Storm = "orage"
)

// Weather est la météo d'une région, décidée par le serveur.
type Weather struct {
	Kind  string
	Until time.Time // prochain changement
	rng   *rand.Rand
}

// NewWeather démarre la météo d'une région, toujours par un temps clair.
func NewWeather(now time.Time, seed int64) *Weather {
	w := &Weather{Kind: Clear, rng: rand.New(rand.NewSource(seed))}
	w.Until = now.Add(w.duration(Clear))
	return w
}

// Update change le temps quand son heure est venue. Renvoie vrai s'il a changé.
func (w *Weather) Update(now time.Time) bool {
	if now.Before(w.Until) {
		return false
	}
	next := w.pick()
	w.Until = now.Add(w.duration(next))
	changed := next != w.Kind
	w.Kind = next
	return changed
}

// Après un temps clair, on peut avoir n'importe quoi ; après de la pluie ou du
// brouillard, on revient souvent au clair ; un orage finit toujours en pluie.
func (w *Weather) pick() string {
	r := w.rng.Float64()
	switch w.Kind {
	case Clear:
		switch {
		case r < 0.40:
			return Clear
		case r < 0.70:
			return Rain
		case r < 0.90:
			return Fog
		default:
			return Storm
		}
	case Rain:
		switch {
		case r < 0.55:
			return Clear
		case r < 0.80:
			return Storm
		default:
			return Fog
		}
	case Fog:
		if r < 0.75 {
			return Clear
		}
		return Rain
	default: // orage
		return Rain
	}
}

// Combien de temps dure chaque temps : le clair dure le plus longtemps.
func (w *Weather) duration(kind string) time.Duration {
	min, span := 6, 8 // minutes
	switch kind {
	case Clear:
		min, span = 15, 20
	case Storm:
		min, span = 4, 4
	}
	return time.Duration(min+w.rng.Intn(span+1)) * time.Minute
}
