package world

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Couleurs de pion proposées à la création du héros (Acte I).
var HeroColors = []string{
	"#e05a47", // braise
	"#f0a03c", // ambre
	"#e8c66a", // or du Lumen
	"#8cc152", // mousse
	"#3fb58c", // émeraude
	"#4fb3d9", // lagon
	"#4a7bd0", // saphir
	"#8a6ad8", // améthyste
	"#d066b0", // orchidée
	"#f2f0ea", // ivoire
	"#8d8a96", // ardoise
	"#5a4636", // écorce
}

// MaxHeroes est le nombre d'emplacements de héros par compte.
const MaxHeroes = 3

var (
	ErrNameLength  = errors.New("Le nom doit faire entre 3 et 16 lettres.")
	ErrNameChars   = errors.New("Le nom ne peut contenir que des lettres, une apostrophe ou un trait d'union.")
	ErrNameShape   = errors.New("Le nom doit commencer et finir par une lettre, sans deux signes à la suite.")
	ErrNameBlocked = errors.New("Ce nom est réservé. Braise te suggère d'en inventer un autre !")
)

var reservedNames = []string{"admin", "moderateur", "modo", "braise", "lumenfall", "erydor", "xalmorath", "systeme", "serveur", "staff"}

// NormalizeName met en forme un nom de héros : espaces retirés, forme
// Unicode composée, première lettre en majuscule.
func NormalizeName(s string) string {
	s = norm.NFC.String(strings.TrimSpace(s))
	r := []rune(s)
	if len(r) > 0 {
		r[0] = unicode.ToUpper(r[0])
	}
	return string(r)
}

// ValidateName vérifie un nom déjà normalisé.
func ValidateName(s string) error {
	n := utf8.RuneCountInString(s)
	if n < 3 || n > 16 {
		return ErrNameLength
	}
	prevSign := false
	for i, c := range []rune(s) {
		isSign := c == '\'' || c == '-'
		if !unicode.IsLetter(c) && !isSign {
			return ErrNameChars
		}
		if isSign && (i == 0 || i == n-1 || prevSign) {
			return ErrNameShape
		}
		prevSign = isSign
	}
	key := NameKey(s)
	for _, r := range reservedNames {
		if strings.Contains(key, r) {
			return ErrNameBlocked
		}
	}
	return nil
}

// NameKey donne la clé d'unicité d'un nom : sans accents, sans signes, en
// minuscules. « Élowan » et « elowan » ne peuvent donc pas coexister.
func NameKey(s string) string {
	var b strings.Builder
	for _, c := range norm.NFD.String(s) {
		if unicode.Is(unicode.Mn, c) || c == '\'' || c == '-' {
			continue
		}
		b.WriteRune(unicode.ToLower(c))
	}
	return b.String()
}
