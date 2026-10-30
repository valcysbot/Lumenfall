#!/usr/bin/env python3
"""La forge des bêtes : les monstres du Val de Glennmor en pixel art.

Même méthode que les héros (forge_heros.py) : un modèle 3D en formes simples,
photographié dans la vraie vue du jeu pour les 8 directions et chaque image
d'animation. Une seule planche par monstre (pas de couches) : les loups, les
sangliers, leurs cousins effacés par la Flétrissure, le corbeau effacé et le
sanglier doré.

    python3 tools/sprites/forge_betes.py              # écrit client/assets/betes/
    python3 tools/sprites/forge_betes.py loup corbeau_efface
"""
import json
import math
import os
import sys

import numpy as np
from PIL import Image

import forge_heros as fh
from forge_heros import boite, capsule, ellipsoide, rampe, rot, union, tourne

SORTIE = os.path.join(fh.RACINE, "client", "assets", "betes")

# Animations : nom -> (images, images par seconde, en boucle)
ANIMS = [("repos", 4, 4, True), ("marche", 6, 10, True), ("course", 6, 15, True),
         ("attaque", 5, 12, False), ("mort", 5, 9, False)]


def cadre(w, h, ancre):
    """Change la taille des images (les bêtes sont plus larges que hautes)."""
    fh.W, fh.H, fh.ANCRE = w, h, ancre
    xs, ys = np.meshgrid(np.arange(w) + 0.5 - ancre[0], np.arange(h) + 0.5 - ancre[1])
    fh.ORIGINES = (xs.reshape(-1, 1) * fh.RIGHT + (-ys.reshape(-1, 1)) * fh.UP) - fh.FWD * 80.0


# --- Les bêtes à quatre pattes ------------------------------------------------

LOUP = dict(long=10.5, larg=4.2, haut=5.0, garrot=13.5, patte=1.5, tete=3.6, museau=4.2, oreille="pointue",
            queue="touffue", defenses=False, criniere=False)
SANGLIER = dict(long=10.0, larg=5.6, haut=6.4, garrot=12.0, patte=1.7, tete=4.2, museau=3.6, oreille="ronde",
                queue="fine", defenses=True, criniere=True)


def quadrupede(forme, d, anim, i, n):
    th = d * math.pi / 4
    F = np.array([math.cos(th), math.sin(th), 0.0])
    R = np.array([-F[1], F[0], 0.0])
    Z = np.array([0.0, 0.0, 1.0])
    t = i / n
    ph = 2 * math.pi * t
    pas = souffle = elan = 0.0
    tete_baisse = 0.0
    machoire = 0.0
    if anim == "repos":
        souffle = 0.35 * math.sin(ph)
    elif anim in ("marche", "course"):
        pas = 0.5 if anim == "marche" else 0.85
        souffle = (0.5 if anim == "marche" else 1.2) * abs(math.cos(ph))
    elif anim == "attaque":
        k = math.sin(math.pi * t)
        elan = 4.5 * k
        tete_baisse = 0.35 * k
        machoire = 0.6 * k
    G = forme
    hz = G["garrot"] + souffle * 0.5
    c = F * elan + Z * hz
    L, Wd, Ht = G["long"], G["larg"], G["haut"]
    out = [(ellipsoide(c, [F, R, Z], [L, Wd, Ht]), "poil"),
           (ellipsoide(c - Z * Ht * 0.45 + F * 1.0, [F, R, Z], [L * 0.8, Wd * 0.8, Ht * 0.55]), "ventre")]
    if G["criniere"]:
        out.append((boite(c + Z * (Ht * 0.8) + F * 2.0, [F, R, Z], [L * 0.75, 1.0, 1.6], 0.8), "criniere"))
    # Tête et museau.
    cou = c + F * (L * 0.85) + Z * 1.5
    td = rot(F, R, -tete_baisse)
    tete = cou + td * 2.5 + Z * 1.6
    out.append((ellipsoide(tete, [td, R, Z], [G["tete"], G["tete"] * 0.85, G["tete"] * 0.9]), "poil"))
    bout = tete + td * (G["tete"] * 0.7 + G["museau"] * 0.6) - Z * 1.2
    out.append((ellipsoide(bout, [td, R, Z], [G["museau"] * 0.75, G["tete"] * 0.5, G["tete"] * 0.45]), "poil"))
    out.append((ellipsoide(bout + td * G["museau"] * 0.7 + Z * 0.3, [td, R, Z], [0.9, 1.1, 0.9]), "museau"))
    if machoire:
        md = rot(td, R, machoire)
        out.append((ellipsoide(tete + md * (G["tete"] * 0.6 + G["museau"] * 0.4) - Z * 2.6, [md, R, Z],
                               [G["museau"] * 0.6, G["tete"] * 0.4, 0.9]), "ventre"))
    for k in (-1, 1):
        out.append((ellipsoide(tete + td * (G["tete"] * 0.55) + R * k * G["tete"] * 0.5 + Z * 1.2, [td, R, Z],
                               [0.9, 0.8, 0.9]), "yeux"))
        if G["oreille"] == "pointue":
            out.append((capsule(tete - td * 0.8 + R * k * 2.0 + Z * 2.4, tete - td * 1.6 + R * k * 2.4 + Z * 6.0, 0.9), "poil"))
        else:
            out.append((ellipsoide(tete - td * 1.0 + R * k * 3.4 + Z * 2.4, [td, R, Z], [1.0, 1.8, 1.3]), "poil"))
        if G["defenses"]:
            base = bout + td * 0.8 + R * k * 1.6 - Z * 0.6
            out.append((capsule(base, base + td * 1.6 + Z * 2.4 + R * k * 0.5, 0.55), "defense"))
    # Les pattes : les diagonales vont ensemble.
    for (av, cote, phase) in ((1, -1, 0), (1, 1, math.pi), (-1, -1, math.pi), (-1, 1, 0)):
        a = pas * math.sin(ph + phase)
        h = c + F * (av * L * 0.62) + R * cote * Wd * 0.55 - Z * Ht * 0.3
        pied = F * (av * L * 0.62 + elan * 0.0) + R * cote * Wd * 0.6 + F * (hz * math.tan(a) * 0.5)
        lever = max(0.0, math.sin(ph + phase)) * pas * 2.2
        pied = pied + Z * (0.8 + lever)
        if anim == "attaque" and av > 0:
            pied = pied + F * elan * 0.9
        out.append((capsule(h, pied, G["patte"] * (1.25 if av > 0 else 1.35)), "poil"))
        out.append((ellipsoide(pied, [F, R, Z], [G["patte"] * 1.2, G["patte"], 0.9]), "museau"))
    # La queue.
    arr = c - F * L * 0.95 + Z * Ht * 0.3
    if G["queue"] == "touffue":
        rem = 0.7 * math.sin(ph * (2 if anim == "repos" else 1))
        bout_q = arr - F * 6.0 - Z * 4.5 + R * rem
        out.append((capsule(arr, bout_q, 1.8), "poil"))
        out.append((ellipsoide(bout_q - Z * 0.4, [F, R, Z], [1.6, 1.6, 1.6]), "ventre"))
    else:
        out.append((capsule(arr, arr - F * 2.5 + Z * 1.0 + R * 0.6 * math.sin(ph), 0.6), "poil"))
    if anim == "mort":
        # Il bascule sur le flanc et s'affaisse.
        k = min(1.0, (i + 1) / (n - 1))
        centre = Z * 0.0 + R * Wd
        out = [(tourne(f, centre, F, -k * math.pi / 2 * 0.95), m) for f, m in out]
        out = [((lambda f: lambda p: f(p + Z * (Wd * 0.2 * k)))(f), m) for f, m in out]
    return out


# --- Le corbeau --------------------------------------------------------------------

def corbeau(d, anim, i, n):
    th = d * math.pi / 4
    F = np.array([math.cos(th), math.sin(th), 0.0])
    R = np.array([-F[1], F[0], 0.0])
    Z = np.array([0.0, 0.0, 1.0])
    t = i / n
    ph = 2 * math.pi * t
    haut = 16.0 + 1.2 * math.sin(ph)
    avance = 0.0
    if anim == "attaque":
        k = math.sin(math.pi * t)
        haut -= 7 * k
        avance = 5 * k
    if anim == "mort":
        haut = 16.0 * (1 - min(1.0, t * 1.4)) + 2.0
    c = Z * haut + F * avance
    out = [(ellipsoide(c, [F, R, Z], [4.6, 2.6, 2.6]), "poil"),
           (ellipsoide(c - Z * 0.8, [F, R, Z], [3.6, 2.0, 1.8]), "ventre")]
    tete = c + F * 4.2 + Z * 1.6
    out.append((ellipsoide(tete, [F, R, Z], [2.3, 2.1, 2.1]), "poil"))
    out.append((capsule(tete + F * 1.6, tete + F * 4.4 - Z * 0.8, 0.7), "museau"))
    for k in (-1, 1):
        out.append((ellipsoide(tete + F * 1.2 + R * k * 1.4 + Z * 0.6, [F, R, Z], [0.7, 0.6, 0.7]), "yeux"))
    out.append((boite(c - F * 5.6 + Z * 0.4, [F, R, Z], [2.4, 2.2, 0.4], 0.3), "poil"))
    battement = 0.9 * math.sin(ph * (2 if anim != "repos" else 1))
    if anim == "mort":
        battement = -0.8
    for k in (-1, 1):
        vers = rot(R * k, F, -k * (0.2 + battement))
        normale = np.cross(F, vers)
        racine = c + Z * 1.0 + R * k * 1.6
        aile = boite(racine + vers * 5.0 - F * 0.8, [F, vers, normale], [2.6, 5.0, 0.35], 0.3)
        bout = boite(racine + vers * 9.0 - F * 2.2, [F, vers, normale], [1.6, 2.4, 0.3], 0.3)
        out.append((union(aile, bout), "aile"))
    if anim == "mort":
        k = min(1.0, (i + 1) / (n - 1))
        out = [(tourne(f, c, F, k * math.pi / 2), m) for f, m in out]
    return out


# --- Les monstres et leurs couleurs ------------------------------------------------

ECHELLE = 1.3   # les bêtes ordinaires, un peu plus grandes que nature pour rester lisibles


def agrandi(modele, k=ECHELLE):
    def m(d, anim, i, n):
        return [((lambda f: lambda p: f(p / k) * k)(f), mat) for f, mat in modele(d, anim, i, n)]
    return m


BETES = {
    "sanglier": (agrandi(lambda d, a, i, n: quadrupede(SANGLIER, d, a, i, n)),
                 {"poil": "#7a5136", "ventre": "#a07a58", "criniere": "#3e2b22", "museau": "#3a2a26",
                  "defense": "#efe6d0", "yeux": "#1d1620"}),
    "sanglier_dore": (agrandi(lambda d, a, i, n: quadrupede(SANGLIER, d, a, i, n)),
                      {"poil": "#e8b93c", "ventre": "#f6dc8a", "criniere": "#a8701c", "museau": "#7a4a1a",
                       "defense": "#fff6e0", "yeux": "#3a1d10"}),
    "loup": (agrandi(lambda d, a, i, n: quadrupede(LOUP, d, a, i, n)),
             {"poil": "#958a78", "ventre": "#c9bca6", "museau": "#2c2630", "yeux": "#e8c66a", "criniere": "#5a5048",
              "defense": "#efe6d0"}),
    # Les Effacés : la Flétrissure a bu leurs couleurs, il ne reste que leurs yeux pâles.
    "loup_efface": (agrandi(lambda d, a, i, n: quadrupede(LOUP, d, a, i, n)),
                    {"poil": "#8e8c92", "ventre": "#b8b6bc", "museau": "#4a4850", "yeux": "#f2f4ff", "criniere": "#6a686e",
                     "defense": "#d8d8dc"}),
    "corbeau_efface": (agrandi(corbeau, 1.2), {"poil": "#77767e", "ventre": "#9a99a0", "museau": "#c8c6cc", "yeux": "#f2f4ff",
                                 "aile": "#5e5d66"}),
    # Le Grand Sanglier de Mabon, deux fois et demie plus grand, puis dévoré par la Flétrissure.
    "grand_sanglier": (lambda d, a, i, n: grand(d, a, i, n),
                       {"poil": "#5e3b2a", "ventre": "#8a6248", "criniere": "#2a1d18", "museau": "#2c201e",
                        "defense": "#f6efdc", "yeux": "#e05a47"}),
    "grand_sanglier_efface": (lambda d, a, i, n: grand(d, a, i, n),
                              {"poil": "#86848c", "ventre": "#aeacb4", "criniere": "#55535c", "museau": "#3e3c44",
                               "defense": "#f2f4ff", "yeux": "#f2f4ff"}),
}

# Les grandes bêtes ont des images plus grandes : nom -> (largeur, hauteur, ancre).
GRANDES = {"grand_sanglier": (152, 112, (76, 96)), "grand_sanglier_efface": (152, 112, (76, 96))}
ECHELLE_GRAND = 3.0


def grand(d, anim, i, n):
    formes = quadrupede(dict(SANGLIER, defenses=True), d, anim, i, n)
    k = ECHELLE_GRAND
    return [((lambda f: lambda p: f(p / k) * k)(f), m) for f, m in formes]


def forge(seulement=None):
    os.makedirs(SORTIE, exist_ok=True)
    total = sum(a[1] for a in ANIMS)
    meta = {"image": [64, 48], "ancre": [32, 40], "directions": 8, "anims": {}, "betes": list(BETES),
            "tailles": {k: {"image": [w, h], "ancre": list(a)} for k, (w, h, a) in GRANDES.items()}}
    col = 0
    for nom, n, fps, boucle in ANIMS:
        meta["anims"][nom] = {"debut": col, "images": n, "fps": fps, "boucle": boucle}
        col += n
    for nom, (modele, couleurs) in BETES.items():
        if seulement and nom not in seulement:
            continue
        w, h, a = GRANDES.get(nom, (64, 48, (32, 40)))
        cadre(w, h, a)
        pal = {k: rampe(v) for k, v in couleurs.items()}
        planche = np.zeros((8 * fh.H, total * fh.W, 4), np.uint8)
        for d in range(8):
            col = 0
            for anim, n, _, _ in ANIMS:
                for i in range(n):
                    formes = modele(d, anim, i, n)
                    touche, prof, normales = fh.lancer(formes)
                    tons = fh.ton_lumiere(normales).reshape(fh.H, fh.W)
                    mats = np.full(touche.shape, "", dtype=object)
                    for k, (_, m) in enumerate(formes):
                        mats[touche == k] = m
                    mats = mats.reshape(fh.H, fh.W)
                    prof = prof.reshape(fh.H, fh.W)
                    plein = (touche >= 0).reshape(fh.H, fh.W)
                    cont = fh.contours(mats, prof, plein)
                    planche[d * fh.H:(d + 1) * fh.H, col * fh.W:(col + 1) * fh.W] = fh.colorie(mats, tons, cont, pal)
                    col += 1
        Image.fromarray(planche).save(os.path.join(SORTIE, nom + ".png"), optimize=True)
        print(f"  {nom} : prêt", file=sys.stderr)
    with open(os.path.join(SORTIE, "betes.json"), "w") as f:
        json.dump(meta, f, ensure_ascii=False, indent=1)


if __name__ == "__main__":
    forge(sys.argv[1:] or None)
