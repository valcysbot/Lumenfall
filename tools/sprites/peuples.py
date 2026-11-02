"""Les données des 15 peuples d'Erydor et des couleurs du miroir de création.

Partagé par la forge (forge_peuples.py) et recopié pour le jeu dans
client/data/heros/peuples.json : taille, forme du corps, traits, teintes de peau.
"""


def degrade(ancres, n=24):
    """n couleurs réparties le long d'une suite de couleurs d'ancrage."""
    def rgb(h):
        return [int(h[i:i + 2], 16) for i in (1, 3, 5)]
    a = [rgb(h) for h in ancres]
    out = []
    for i in range(n):
        x = i * (len(a) - 1) / (n - 1)
        k = min(int(x), len(a) - 2)
        t = x - k
        c = [round(a[k][j] + (a[k + 1][j] - a[k][j]) * t) for j in range(3)]
        out.append("#%02x%02x%02x" % tuple(c))
    return out


# Faction : concorde, pacte, ligue (la bible du monde).
# forme : standard, pierre (Golems), cristal (Lumins).
# taille : facteur appliqué au héros entier (les crans de la création s'y ajoutent).
# traits : pièces dessinées en plus du corps, dans l'ordre d'empilement.
PEUPLES = {
    "humain": {"nom": ["Humain", "Humaine", "Humain·e"], "faction": "concorde", "forme": "standard", "taille": 1.0,
               "traits": [], "capitale": "Valmoran",
               "peaux": degrade(["#f8dcc6", "#f0bf98", "#d9a074", "#b47a52", "#8e5d40", "#5e3b28", "#3e2a20"])},
    "nain": {"nom": ["Nain", "Naine", "Nain·e"], "faction": "concorde", "forme": "standard", "taille": 0.8,
             "traits": [], "capitale": "Dun Brannoc",
             "peaux": degrade(["#f6d0b0", "#eab08a", "#d0906a", "#b0704e", "#8a5a3c", "#5e3d2a"])},
    "halfelin": {"nom": ["Halfelin", "Halfeline", "Halfelin·e"], "faction": "concorde", "forme": "standard", "taille": 0.7,
                 "traits": ["oreilles_rondes"], "capitale": "Mielbourg",
                 "peaux": degrade(["#fae0c8", "#f2c49c", "#dca47a", "#bc845a", "#94643e", "#6a4630"])},
    "gnome": {"nom": ["Gnome", "Gnome", "Gnome"], "faction": "concorde", "forme": "standard", "taille": 0.62,
              "traits": ["oreilles_gnome"], "capitale": "Rouagemont",
              "peaux": degrade(["#f6d6c0", "#eab896", "#d49a76", "#c8c4d8", "#a8b8d0", "#b89a80"])},
    "golem": {"nom": ["Golem", "Golem", "Golem"], "faction": "concorde", "forme": "pierre", "taille": 1.2,
              "traits": ["mousse"], "capitale": "la Forge-Mère",
              "peaux": degrade(["#c8c2b6", "#a69c8a", "#c9a77a", "#a8805a", "#a85c45", "#7d8590", "#5e6670", "#8fa38a"])},
    "elfe": {"nom": ["Elfe", "Elfe", "Elfe"], "faction": "pacte", "forme": "standard", "taille": 1.04,
             "traits": ["oreilles_elfe"], "capitale": "Elarin",
             "peaux": degrade(["#fae8d6", "#f2d0b0", "#e4b890", "#cc9c74", "#a87a56", "#e2dcb8", "#c8c8a0"])},
    "fee": {"nom": ["Fée", "Fée", "Fée"], "faction": "pacte", "forme": "standard", "taille": 0.6,
            "traits": ["ailes", "oreilles_fee"], "capitale": "Tír Glas", "flotte": True,
            "peaux": degrade(["#f8e2ec", "#f6d4e0", "#e8d8f8", "#d8e8f8", "#d8f2e4", "#f8f0d0", "#f0d8c0"])},
    "felin": {"nom": ["Félin", "Féline", "Félin·e"], "faction": "pacte", "forme": "standard", "taille": 0.96,
              "traits": ["queue_chat", "oreilles_chat"], "capitale": "Pardalis",
              "peaux": degrade(["#f6f0e6", "#f0d8a8", "#e8b46a", "#d88a3a", "#a86a3a", "#8a8a92", "#5a5260", "#2e2a32"])},
    "homme_bete": {"nom": ["Homme-bête", "Femme-bête", "Être-bête"], "faction": "pacte", "forme": "standard", "taille": 1.1,
                   "traits": ["queue_touffue", "bois"], "capitale": "Tor Cernos",
                   "peaux": degrade(["#e0d0b8", "#c9a27a", "#a8804e", "#8a6440", "#6e5038", "#4a3a30", "#8a7a6a"])},
    "sirene": {"nom": ["Triton", "Sirène", "Sirène"], "faction": "pacte", "forme": "standard", "taille": 1.0,
               "traits": ["nageoires", "oreilles_nageoires"], "capitale": "Port-Morwen",
               "peaux": degrade(["#d0f0f0", "#a8e0dc", "#7cc4d8", "#5aa0c8", "#6a88c0", "#8ad0a8", "#f0d0c8"])},
    "orque": {"nom": ["Orque", "Orque", "Orque"], "faction": "ligue", "forme": "standard", "taille": 1.15,
              "traits": ["defenses"], "capitale": "Craig-Dorn",
              "peaux": degrade(["#a8c080", "#8fb06a", "#6f9a50", "#5a7a44", "#7a8a6a", "#9a9a70", "#5e5e50"])},
    "drakeide": {"nom": ["Drakéide", "Drakéide", "Drakéide"], "faction": "ligue", "forme": "standard", "taille": 1.1,
                 "traits": ["queue_drake", "cornes"], "capitale": "Ignisval",
                 "peaux": degrade(["#e8e0d0", "#d8a040", "#c0503a", "#8a3a3a", "#8a4a9a", "#406aa0", "#4a8a60", "#2e2e3a"])},
    "revenant": {"nom": ["Revenant", "Revenante", "Revenant·e"], "faction": "ligue", "forme": "standard", "taille": 1.0,
                 "traits": [], "capitale": "Vesperine", "ames": True,
                 "peaux": degrade(["#e4e8f0", "#c8d4e0", "#aebfd4", "#9fb0c8", "#b0c8c8", "#a0a8c8", "#8a98b8"])},
    "elfe_noir": {"nom": ["Elfe noir", "Elfe noire", "Elfe noir·e"], "faction": "ligue", "forme": "standard", "taille": 1.02,
                  "traits": ["oreilles_elfe"], "capitale": "Umbrevel",
                  "peaux": degrade(["#9a8ab0", "#7a6a96", "#6a6a8a", "#5a6a8a", "#4a4a6a", "#3a3a52", "#7a5a7a"])},
    "lumin": {"nom": ["Lumin", "Lumine", "Lumin·e"], "faction": "ligue", "forme": "cristal", "taille": 0.82,
              "traits": ["eclats"], "capitale": "Clairéclat", "lueur": True,
              "peaux": degrade(["#e8faff", "#a8e8ff", "#88b8ff", "#c8a8ff", "#ffb8e8", "#ffe8a0", "#a8ffc8"])},
}

YEUX = ["#2a2238", "#5a3825", "#8a5a35", "#3a7a4a", "#3a6ab0", "#7ab0d8",
        "#8a8a96", "#e8c040", "#c04040", "#8a4ad0", "#40d0c0", "#f4f4f4"]

CHEVEUX = ["#2e2433", "#5a3825", "#8a5a35", "#e3b75c", "#c0532a", "#ddd8d0",
           "#7a2e24", "#f0e8c8", "#9a98a0", "#d07a3a", "#d8a050", "#2a3050",
           "#4a7bd0", "#3fb58c", "#e080b0", "#8a6ad8", "#d03a3a", "#40c0c8",
           "#c8d0e0", "#e8c66a", "#a0e0b0", "#b8a0e8", "#f08a3c", "#3a2a5a"]

TUNIQUES = ["#e05a47", "#f0a03c", "#e8c66a", "#8cc152", "#3fb58c", "#4fb3d9",
            "#4a7bd0", "#8a6ad8", "#d066b0", "#f2f0ea", "#8d8a96", "#5a4636"]

# Couleur par défaut de chaque matériau qui ne se choisit pas.
DEFAUTS = {
    "sous": "#6e4c35", "pantalon": "#6e4c35", "bottes": "#4a3328", "gants": "#8a5a3a", "cape": "#2f6f8a",
    "casque": "#a4adc0", "lame": "#d5deea", "garde": "#d0a548", "poignee": "#4a3328", "ceinture": "#5a3a24",
    "boucle": "#e3c060", "tablier": "#e6dfcc", "bouclier": "#3d5f9e", "umbo": "#e3c060", "bois_arme": "#8a5a35",
    "fer": "#8d96a8", "corde": "#e8dcc0", "orbe": "#8ad8ff", "verre": "#bfe8f0", "potion": "#e0507a",
    "cuivre": "#c98a4a", "plume": "#f2f0ea", "os": "#e8e0cc", "corne": "#e8dcc0", "aile": "#cfe8ff",
    "mousse": "#6a9a4a", "taches": "#a0623a", "cicatrice": "#c87a72", "peinture": "#d03a3a",
    "runes": "#6ad0f0", "larmes": "#3a3a5a", "chapeau": "#dcb862", "tissu": "#e05a47", "sourcils": "#2e2433",
    "yeux": "#2a2238", "cheveux": "#2e2433", "peau": "#f0bf98", "totem": "#c0532a", "ruban": "#d03a3a",
}
# Transparence de certains matériaux (ailes de fée).
ALPHA = {"aile": 0.72, "orbe": 0.9}

CHAPEAUX = {"paille": "#dcb862", "feutre": "#4a3d52"}
