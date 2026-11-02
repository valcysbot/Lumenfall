package world

import "errors"

// L'allure d'un héros, telle que le miroir de création la règle (Acte IV).
// Les numéros renvoient aux pièces et aux couleurs de la forge des peuples
// (tools/sprites/peuples.py, client/data/heros/peuples.json).

// Ce que la forge dessine.
const (
	NGenres    = 3 // homme, femme, autre
	NTailles   = 5 // les crans de taille
	NCarrures  = 3 // mince, normale, costaud
	NVisages   = 16
	NPeaux     = 24
	NYeux      = 12
	NCoupes    = 20
	NTeintes   = 24 // couleurs de cheveux (et de marques)
	NBarbes    = 6
	NTuniques  = 12
	MaxMarques = 3
)

// Marques : les marques qu'on peut empiler sur le visage.
var Marques = []string{"taches", "cicatrice", "peinture", "runes", "ecailles", "larmes"}

// Look : l'allure choisie au miroir.
type Look struct {
	Genre   int      `json:"g"` // 0 homme, 1 femme, 2 autre
	Accord  int      `json:"a"` // 0 masculin, 1 féminin, 2 neutre
	Taille  int      `json:"t"` // 0 à 4, 2 au milieu
	Carrure int      `json:"k"` // 0 mince, 1 normale, 2 costaud
	Visage  int      `json:"v"`
	Peau    int      `json:"p"`
	Yeux    int      `json:"y"`
	Coupe   int      `json:"h"`
	Teinte  int      `json:"hc"`
	Barbe   int      `json:"b"` // -1 : aucune
	Marques [][2]int `json:"m"` // [marque, teinte], du dessous au dessus
	Tunique int      `json:"tu"`
}

// DefaultLook : une allure de départ.
func DefaultLook() Look {
	return Look{Taille: 2, Carrure: 1, Barbe: -1, Marques: [][2]int{}}
}

func in(v, n int) bool { return v >= 0 && v < n }

// Validate vérifie que chaque choix existe.
func (l Look) Validate() error {
	ok := in(l.Genre, NGenres) && in(l.Accord, NGenres) && in(l.Taille, NTailles) && in(l.Carrure, NCarrures) &&
		in(l.Visage, NVisages) && in(l.Peau, NPeaux) && in(l.Yeux, NYeux) && in(l.Coupe, NCoupes) &&
		in(l.Teinte, NTeintes) && (l.Barbe == -1 || in(l.Barbe, NBarbes)) && in(l.Tunique, NTuniques) &&
		len(l.Marques) <= MaxMarques
	if !ok {
		return errors.New("Ce choix n'existe pas au miroir.")
	}
	for _, m := range l.Marques {
		if !in(m[0], len(Marques)) || !in(m[1], NTeintes) {
			return errors.New("Cette marque n'existe pas.")
		}
	}
	return nil
}

// MiroirGold : le prix du miroir de Solenn, après le premier passage gratuit.
const MiroirGold = 50
