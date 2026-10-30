#!/usr/bin/env python3
"""La forge de Braise : le petit dragon qui accompagne le héros.

Même méthode que les héros (forge_heros.py) : un modèle 3D en formes simples,
photographié dans la vue du jeu pour les 8 directions. Braise vole en battant
des ailes, la queue qui ondule et une flammèche au bout.

    python3 tools/sprites/forge_braise.py     # écrit client/assets/braise/
"""
import json
import math
import os
import sys

import numpy as np
from PIL import Image

import forge_heros as fh
from forge_heros import boite, capsule, ellipsoide, rampe, rot, union

SORTIE = os.path.join(fh.RACINE, "client", "assets", "braise")
IMAGES, FPS = 8, 10
ECHELLE = 1.35
COULEURS = {"ecailles": "#e2552f", "ventre": "#f4b24c", "aile": "#f29a3e", "corne": "#f3e2c0", "os": "#7a2f3a",
            "yeux": "#2a2238", "flamme": "#ffd23a"}


def braise(d, i):
    th = d * math.pi / 4
    F = np.array([math.cos(th), math.sin(th), 0.0])
    R = np.array([-F[1], F[0], 0.0])
    Z = np.array([0.0, 0.0, 1.0])
    ph = 2 * math.pi * i / IMAGES
    c = Z * (23.0 + 1.3 * math.sin(ph))          # il flotte à hauteur d'épaule (une fois agrandi)
    out = [
        (ellipsoide(c, [F, R, Z], [6.2, 3.8, 3.8]), "ecailles"),
        (ellipsoide(c + F * 1.2 - Z * 1.4, [F, R, Z], [4.6, 2.9, 2.8]), "ventre"),
    ]
    cou = c + F * 4.6 + Z * 2.2
    tete = c + F * 7.4 + Z * 4.6
    out.append((capsule(c + F * 3.0, cou, 2.4), "ecailles"))
    out.append((ellipsoide(tete, [F, R, Z], [3.9, 3.3, 3.2]), "ecailles"))
    out.append((ellipsoide(tete + F * 3.0 - Z * 1.0, [F, R, Z], [2.5, 2.1, 1.7]), "ecailles"))
    for k in (-1, 1):
        out.append((ellipsoide(tete + F * 2.0 + R * k * 2.1 + Z * 0.9, [F, R, Z], [1.1, 1.0, 1.5]), "yeux"))
        out.append((capsule(tete - F * 1.0 + Z * 2.2 + R * k * 1.4, tete - F * 3.6 + Z * 4.8 + R * k * 2.0, 0.75), "corne"))
        out.append((ellipsoide(c + F * 2.4 + R * k * 2.2 - Z * 3.6, [F, R, Z], [1.4, 1.0, 1.6]), "ecailles"))
    # La queue ondule, avec une flammèche au bout.
    ond = 1.6 * math.sin(ph + 1.0)
    p0, p1 = c - F * 5.0, c - F * 9.5 - Z * 2.0 + R * ond
    p2 = c - F * 13.5 - Z * 1.0 + R * ond * 2.0
    out.append((union(capsule(p0, p1, 2.2), capsule(p1, p2, 1.4)), "ecailles"))
    out.append((ellipsoide(p2 - F * 1.6 + Z * 0.6, [F, R, Z], [1.8, 1.4, 2.2 + 0.4 * math.sin(ph * 2)]), "flamme"))
    # Les ailes battent autour de l'axe du corps.
    bat = 0.75 * math.sin(ph)
    for k in (-1, 1):
        vers = rot(R * k, F, -k * (0.35 + bat))       # vers le haut quand bat > 0
        normale = np.cross(F, vers)
        racine = c + Z * 2.2 + R * k * 2.0 - F * 0.6
        aile = boite(racine + vers * 5.6, [F, vers, normale], [3.4, 5.4, 0.35], 0.3)
        bout = boite(racine + vers * 9.4 - F * 1.6, [F, vers, normale], [1.8, 2.2, 0.3], 0.3)
        out.append((union(aile, bout), "aile"))
        out.append((capsule(racine, racine + vers * 11.0 - F * 0.4, 0.8), "os"))
    # Le modèle est dessiné petit, puis agrandi d'un coup.
    return [((lambda f: lambda p: f(p / ECHELLE) * ECHELLE)(f), m) for f, m in out]


def forge():
    os.makedirs(SORTIE, exist_ok=True)
    pal = {k: rampe(v) for k, v in COULEURS.items()}
    planche = np.zeros((8 * fh.H, IMAGES * fh.W, 4), np.uint8)
    for d in range(8):
        for i in range(IMAGES):
            formes = braise(d, i)
            touche, prof, normales = fh.lancer(formes)
            tons = fh.ton_lumiere(normales).reshape(fh.H, fh.W)
            mats = np.full(touche.shape, "", dtype=object)
            for k, (_, m) in enumerate(formes):
                mats[touche == k] = m
            mats = mats.reshape(fh.H, fh.W)
            prof = prof.reshape(fh.H, fh.W)
            plein = (touche >= 0).reshape(fh.H, fh.W)
            cont = fh.contours(mats, prof, plein)
            planche[d * fh.H:(d + 1) * fh.H, i * fh.W:(i + 1) * fh.W] = fh.colorie(mats, tons, cont, pal)
        print(f"  Braise : direction {d + 1}/8", file=sys.stderr)
    Image.fromarray(planche).save(os.path.join(SORTIE, "braise.png"), optimize=True)
    with open(os.path.join(SORTIE, "braise.json"), "w") as fh_:
        json.dump({"image": [fh.W, fh.H], "ancre": list(fh.ANCRE), "images": IMAGES, "fps": FPS}, fh_, indent=1)
    print("Braise est prêt :", SORTIE)


if __name__ == "__main__":
    forge()
