#!/usr/bin/env python3
"""La forge des métiers (Acte V) : les points de récolte et les ateliers.

Même méthode que la forge des décors (petites maquettes 3D photographiées dans
la vue du jeu). Les points de récolte : filons de cuivre et de fer, plantes
(sauge, lin, lysimaque, chardon gris, fleur de lune), bois à couper (chêne,
arbre gris). Les ateliers : enclume, établi du joaillier, alambic, métier à
tisser, chevalet du tanneur, marmite.

Écrit client/assets/metiers/<nom>.png et metiers.json (ancre).

    python3 tools/sprites/forge_metiers.py
"""
import json
import math
import os
import sys

import numpy as np
from PIL import Image

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import forge_decors as fd  # noqa: E402
from forge_decors import b, cone, cylindre, capsule, ellipsoide, coupe, BASE, CASE  # noqa: E402
from forge_heros import rampe, boite  # noqa: E402

SORTIE = os.path.normpath(os.path.join(fd.ICI, "..", "..", "client", "assets", "metiers"))

fd.COULEURS.update({
    "cuivre": "#d8803c", "minerai_fer": "#b4c0d4", "sauge": "#8fb07a", "lin": "#5a86e0",
    "tige": "#4f8a3c", "lysimaque": "#f0c840", "chardon": "#a894c0", "gris": "#a8a4ae",
    "lune": "#cfe4ff", "ecorce_grise": "#8a8690", "cuivre_fonce": "#9a5a30", "verre": "#9ad0e0",
    "cuir": "#a0683a", "laine": "#f4f0e8", "soupe": "#c88a3a",
})
fd.RAMPES.update({k: rampe(v) for k, v in fd.COULEURS.items() if k not in fd.RAMPES})


def filon(minerai):
    f = [(ellipsoide([0, 0, 7], BASE, [18, 15, 13]), "roche"),
         (ellipsoide([10, 7, 4], BASE, [10, 8, 7]), "roche"),
         (ellipsoide([-11, 6, 3], BASE, [7, 6, 5]), "pierre2")]
    # Des éclats de minerai qui sortent de la roche.
    for i, (x, y, z, r) in enumerate([(-6, 8, 14, 4.2), (4, 11, 10, 3.6), (8, -2, 18, 3.2), (-2, 2, 19, 3.0), (13, 10, 7, 2.8)]):
        f.append((ellipsoide([x, y, z], BASE, [r, r, r * 1.3]), minerai))
    return f


def plante(fleur, tige="tige", haute=False):
    k = 1.25 if haute else 1.0
    f = [(ellipsoide([0, 0, 5], BASE, [15, 13, 6]), tige)]
    for i in range(7):
        a = i * math.tau / 7 + 0.3
        x, y = 9 * math.cos(a), 9 * math.sin(a)
        f.append((capsule([x * 0.4, y * 0.4, 2], [x, y, 21 * k + (i % 2) * 4], 1.5), tige))
        f.append((ellipsoide([x, y, 22 * k + (i % 2) * 4], BASE, [4.2, 4.2, 4.2]), fleur))
    return f


def bois(ecorce, feuille):
    """Un tronc abattu, la hache plantée dedans, des copeaux."""
    d = np.array([1.0, -0.35, 0.0])
    d /= np.linalg.norm(d)
    f = [(capsule((-d * 22 + [0, 0, 8]).tolist(), (d * 22 + [0, 0, 8]).tolist(), 8), ecorce),
         (ellipsoide((d * 26 + [0, 0, 8]).tolist(), BASE, [3, 7, 7]), "bois"),
         (capsule([4, -2, 15], [8, 2, 32], 1.2), "tronc"),
         (b(9, 3, 30, 4, 1, 3, 0.5), "fer")]
    for x, y in [(-12, 12), (12, 12), (-2, 16)]:
        f.append((ellipsoide([x, y, 1], BASE, [3, 2, 1]), "bois"))
    f.append((ellipsoide([-24, 9, 3], BASE, [4, 3, 3]), feuille))  # une branche tombée
    return f


def enclume():
    return [(b(0, 0, 9, 9, 7, 9, 1), "tronc"),
            (b(0, 0, 21, 14, 6, 3, 1), "fer"),
            (cone([16, 0, 18], 4, 8), "fer"),
            (b(-12, 9, 2, 5, 3, 2), "fer"),
            (capsule([-6, 4, 25], [-14, 12, 25], 1.2), "tronc"),
            (b(-15, 13, 25, 2.5, 2.5, 2), "fer"),
            (cylindre([22, 10, 6], 6, 6), "fer"), (cylindre([22, 10, 12.5], 5.2, 0.6), "eau")]


def etabli():
    f = [(b(0, 0, 18, 20, 11, 2, 1), "bois"),
         (b(-17, -8, 8, 2, 2, 9), "tronc"), (b(17, -8, 8, 2, 2, 9), "tronc"),
         (b(-17, 8, 8, 2, 2, 9), "tronc"), (b(17, 8, 8, 2, 2, 9), "tronc"),
         (cylindre([-8, 0, 22], 4, 2), "pierre"), (capsule([6, -2, 21], [12, 4, 21], 0.9), "fer")]
    for x, y, m in [(-12, 6, "or"), (0, 6, "verre"), (10, -6, "cuivre")]:
        f.append((ellipsoide([x, y, 21.5], BASE, [2.4, 2.4, 2.4]), m))
    f.append((b(0, 12, 26, 18, 1.2, 6, 0.5), "tronc"))
    return f


def alambic():
    return [(b(0, 0, 3, 14, 12, 3, 1), "pierre2"),
            (b(0, 0, 7, 6, 6, 1.5), "braise"),
            (ellipsoide([0, 0, 18], BASE, [10, 10, 10]), "cuivre"),
            (cone([0, 0, 26], 5, 9), "cuivre"),
            (capsule([0, 0, 34], [14, 6, 30], 1.5), "cuivre_fonce"),
            (capsule([14, 6, 30], [16, 8, 14], 1.5), "cuivre_fonce"),
            (ellipsoide([17, 9, 8], BASE, [5, 5, 6]), "verre")]


def metier_tisser():
    f = [(b(-18, 0, 20, 2, 2, 20), "tronc"), (b(18, 0, 20, 2, 2, 20), "tronc"),
         (b(0, 0, 38, 20, 2, 2), "tronc"), (b(0, 0, 8, 20, 2, 2), "tronc"),
         (b(0, 0.5, 23, 15, 0.8, 13), "lin"),
         (b(0, 6, 4, 16, 5, 4, 1), "bois")]
    for x in (-10, -3, 4, 11):
        f.append((capsule([x, 0, 9], [x, 0, 37], 0.4), "laine"))
    f.append((ellipsoide([-22, 8, 5], BASE, [5, 5, 5]), "laine"))
    return f


def chevalet():
    return [(capsule([-14, -4, 0], [-4, 0, 34], 1.8), "tronc"), (capsule([14, -4, 0], [4, 0, 34], 1.8), "tronc"),
            (capsule([0, 8, 0], [0, 1, 32], 1.8), "tronc"),
            (b(0, -2.5, 20, 12, 0.8, 12, 3), "cuir"),
            (cylindre([18, 8, 7], 6, 7), "bois"), (cylindre([18, 8, 14.2], 5.2, 0.6), "cuir")]


def marmite():
    f = [(capsule([-14, 0, 0], [-10, 0, 30], 1.5), "fer"), (capsule([14, 0, 0], [10, 0, 30], 1.5), "fer"),
         (capsule([-11, 0, 30], [11, 0, 30], 1.2), "fer"),
         (coupe(ellipsoide([0, 0, 14], BASE, [11, 11, 10]), [0, 0, 20], [0, 0, 1]), "fer"),
         (cylindre([0, 0, 19.5], 9, 0.6), "soupe")]
    for i in range(5):
        a = i * math.tau / 5
        f.append((capsule([7 * math.cos(a), 7 * math.sin(a), 1], [2 * math.cos(a), 2 * math.sin(a), 3], 1.6), "bois"))
    f.append((ellipsoide([0, 0, 2], BASE, [4, 4, 2.5]), "braise"))
    return f


PIECES = {
    "filon_cuivre": lambda: filon("cuivre"),
    "filon_fer": lambda: filon("minerai_fer"),
    "sauge": lambda: plante("sauge", "sauge"),
    "lin": lambda: plante("lin"),
    "lysimaque": lambda: plante("lysimaque", haute=True),
    "chardon_gris": lambda: plante("chardon", "gris", haute=True),
    "fleur_de_lune": lambda: plante("lune", "ecorce_grise"),
    "chene": lambda: bois("tronc", "feuille"),
    "arbre_gris": lambda: bois("ecorce_grise", "gris"),
    "forge": enclume,
    "joaillerie": etabli,
    "alchimie": alambic,
    "couture": metier_tisser,
    "cuir": chevalet,
    "cuisine": marmite,
}


def forge(seulement=None):
    os.makedirs(SORTIE, exist_ok=True)
    chemin = os.path.join(SORTIE, "metiers.json")
    meta = json.load(open(chemin)) if os.path.exists(chemin) else {}
    for nom, fn in PIECES.items():
        if seulement and nom not in seulement:
            continue
        w, h = 140, 160
        ancre = (w // 2, int(h * 0.75))
        img = fd.photo(fn(), w, h, ancre)
        img, ancre = fd.rogne(img, ancre)
        Image.fromarray(img).save(os.path.join(SORTIE, nom + ".png"), optimize=True)
        meta[nom] = {"ancre": [int(ancre[0]), int(ancre[1])]}
        print(f"  {nom} : {img.shape[1]} × {img.shape[0]}", file=sys.stderr)
    with open(chemin, "w") as fh:
        json.dump(dict(sorted(meta.items())), fh, ensure_ascii=False, indent=1)


if __name__ == "__main__":
    forge(sys.argv[1:] or None)
