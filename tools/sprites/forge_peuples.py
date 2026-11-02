#!/usr/bin/env python3
"""La forge des peuples : les 15 races d'Erydor, pièce par pièce (Acte IV).

Même méthode que forge_heros.py (un mannequin 3D photographié dans la vraie vue
du jeu), mais chaque planche est écrite en « couleurs à remplir » : le pixel ne
dit pas sa couleur, il dit quel matériau il porte (peau, yeux, cheveux, tissu…)
et dans quel ton (ombre profonde, ombre, base, lumière, contour). Le jeu remplit
les couleurs au moment de l'affichage, avec un petit shader. C'est ce qui permet
24 teintes de peau, 24 couleurs de cheveux et 12 couleurs d'yeux sans dessiner
une planche par couleur.

Codage d'un pixel : rouge = (n° de matériau × 5 + ton) × 6, alpha = 255.

    python3 tools/sprites/forge_peuples.py               # tout (environ 15 min sur 4 cœurs)
    python3 tools/sprites/forge_peuples.py --pieces cheveux_03 visage_00
    python3 tools/sprites/forge_peuples.py --apercu      # planches d'aperçu dans tools/sprites/
"""
import argparse
import json
import math
import os
import sys
from multiprocessing import Pool

import numpy as np
from PIL import Image

ICI = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, ICI)
import forge_heros as fh  # noqa: E402
from forge_heros import boite, capsule, ellipsoide, coupe, sauf, union, rot, rampe, Pose  # noqa: E402
import peuples as pp  # noqa: E402

SORTIE = os.path.join(fh.RACINE, "client", "assets", "heros", "peuples")
DONNEES = os.path.join(fh.RACINE, "client", "data", "heros", "peuples.json")
W, H = fh.W, fh.H

ANIMS = fh.ANIMS + [("tir", 6, 12, False)]
COLONNES = 32            # images par rangée de planche (2 048 pixels, sous la limite des téléphones)
CARRURES = {"mince": 0.85, "normale": 1.0, "costaud": 1.2}

PALIER_CONTOUR = 4


# --- Le corps ---------------------------------------------------------------

def corps_standard(p, yeux=False):
    F, R, Z, k = p.F, p.R, p.Z, p.k
    out = [(ellipsoide(p.buste(0, 0, 33.5), [F, R, Z], [4.6 * k, 6.6 * k, 8.8]), "peau"),
           (ellipsoide(p.buste(0, 0, 26.0), [F, R, Z], [4.4 * k, 6.0 * k, 4.2]), "sous"),
           (capsule(*p.cou, 2.3 * (0.5 + 0.5 * k)), "peau"),
           (ellipsoide(p.tete, [p.tF, R, p.tZ], [7.0, 6.8, 7.6]), "peau")]
    if yeux:
        for s in (-1, 1):
            out.append((ellipsoide(p.tete + p.tF * 6.1 + R * s * 2.6 + p.tZ * 0.2, [p.tF, R, p.tZ], [1.2, 1.0, 1.6]), "yeux"))
    lk = 0.6 + 0.4 * k
    for (h, g, c, pi) in p.jambes:
        out.append((capsule(h, g, 3.0 * lk), "peau"))
        out.append((capsule(g, c, 2.5 * lk), "peau"))
        out.append((ellipsoide(pi, [F, R, Z], [3.2, 1.9, 1.3]), "peau"))
    for (e, co, m, s) in p.bras:
        out.append((capsule(e, co, 2.2 * lk), "peau"))
        out.append((capsule(co, m, 1.9 * lk), "peau"))
        out.append((ellipsoide(m, [F, R, Z], [2.2, 1.8, 2.4]), "peau"))
    return out


def corps_pierre(p):
    """Golem : des blocs de pierre polis, des épaules en rochers, de gros poings."""
    F, R, Z, k = p.F, p.R, p.Z, p.k
    out = [(boite(p.buste(0, 0, 33.0), [F, R, Z], [4.2 * k, 6.0 * k, 8.4], 1.6), "peau"),
           (boite(p.buste(0, 0, 25.6), [F, R, Z], [4.0 * k, 5.6 * k, 3.8], 1.2), "sous"),
           (boite(p.buste(0, 0, 43.0), [F, R, Z], [1.8, 1.8, 2.0], 0.6), "peau"),
           (ellipsoide(p.tete, [p.tF, R, p.tZ], [7.0, 6.8, 7.6]), "peau"),
           # Une rune qui luit au milieu de la poitrine.
           (ellipsoide(p.buste(4.1 * k, 0, 35.0), [F, R, Z], [0.6, 1.2, 1.6]), "runes")]
    for s in (-1, 1):
        out.append((ellipsoide(p.buste(0, s * 7.6, 40.5), [F, R, Z], [3.4, 3.2, 3.0]), "peau"))
    lk = 0.6 + 0.4 * k
    for (h, g, c, pi) in p.jambes:
        out.append((capsule(h, g, 3.1 * lk), "peau"))
        out.append((capsule(g, c, 2.7 * lk), "peau"))
        out.append((boite(pi, [F, R, Z], [3.2, 2.2, 1.4], 0.6), "peau"))
    for (e, co, m, s) in p.bras:
        out.append((capsule(e, co, 2.3 * lk), "peau"))
        out.append((capsule(co, m, 2.1 * lk), "peau"))
        out.append((boite(m, [F, R, Z], [2.3, 2.0, 2.4], 0.8), "peau"))
    return out


def octaedre(c, base, rayons):
    c = np.asarray(c, float)
    m = np.asarray(base, float)
    rr = np.asarray(rayons, float)

    def f(q):
        v = np.abs((q - c) @ m.T) / rr
        return (v.sum(axis=1) - 1) * rr.min() * 0.57
    return f


def corps_cristal(p):
    """Lumin : un corps de cristal taillé en facettes."""
    F, R, Z, k = p.F, p.R, p.Z, p.k
    out = [(octaedre(p.buste(0, 0, 34.0), [F, R, Z], [6.0 * k, 8.6 * k, 11.0]), "peau"),
           (octaedre(p.buste(0, 0, 25.5), [F, R, Z], [5.6 * k, 7.6 * k, 5.4]), "sous"),
           (capsule(*p.cou, 1.8), "peau"),
           (ellipsoide(p.tete, [p.tF, R, p.tZ], [7.0, 6.8, 7.6]), "peau")]
    lk = 0.6 + 0.4 * k
    for (h, g, c, pi) in p.jambes:
        out.append((capsule(h, g, 2.7 * lk), "peau"))
        out.append((capsule(g, c, 2.2 * lk), "peau"))
        out.append((octaedre(pi, [F, R, Z], [3.4, 2.2, 1.6]), "peau"))
    for (e, co, m, s) in p.bras:
        out.append((capsule(e, co, 2.0 * lk), "peau"))
        out.append((capsule(co, m, 1.7 * lk), "peau"))
        out.append((octaedre(m, [F, R, Z], [2.4, 2.0, 2.6]), "peau"))
    return out


CORPS = {"standard": corps_standard, "pierre": corps_pierre, "cristal": corps_cristal}


def T(p, f, r, z):
    """Un point dans le repère de la tête."""
    return p.tete + p.tF * f + p.R * r + p.tZ * z


def tete_base(p):
    return [p.tF, p.R, p.tZ]


# --- Les visages (16) ---------------------------------------------------------

YEUX_FORMES = [
    {"r": [1.2, 1.0, 1.6], "ecart": 2.6, "z": 0.2},    # ronds
    {"r": [1.3, 1.25, 2.0], "ecart": 2.7, "z": 0.3},   # grands
    {"r": [1.0, 1.35, 1.1], "ecart": 2.7, "z": 0.5},   # en amande
    {"r": [1.0, 1.0, 0.85], "ecart": 2.4, "z": 0.2},   # étroits
]
# (sourcils, nez) : sourcils 0 aucun, 1 fins, 2 épais, 3 froncés ; nez 0 petit, 1 moyen, 2 fort, 3 long.
VISAGE_DETAILS = [(1, 1), (2, 2), (3, 0), (0, 3)]


def visage(n):
    yf = YEUX_FORMES[n // 4]
    sourcils, nez = VISAGE_DETAILS[n % 4]

    def f(p):
        b = tete_base(p)
        out = []
        for s in (-1, 1):
            out.append((ellipsoide(T(p, 6.1, s * yf["ecart"], yf["z"]), b, yf["r"]), "yeux"))
            if sourcils:
                ep = {1: [0.6, 1.5, 0.45], 2: [0.9, 1.8, 0.75], 3: [0.8, 1.7, 0.6]}[sourcils]
                bb = b
                if sourcils == 3:   # froncés : penchés vers le nez
                    bb = [p.tF, rot(p.R, p.tF, s * 0.4), rot(p.tZ, p.tF, s * 0.4)]
                out.append((ellipsoide(T(p, 5.9, s * (yf["ecart"] + 0.2), yf["z"] + 2.5), bb, ep), "sourcils"))
        nz = {0: (6.9, -1.6, [0.9, 0.8, 0.9]), 1: (6.9, -1.6, [1.2, 1.0, 1.3]),
              2: (6.8, -1.5, [1.7, 1.4, 1.5]), 3: (7.2, -1.2, [2.1, 0.9, 1.0])}[nez]
        out.append((ellipsoide(T(p, nz[0], 0, nz[1]), b, nz[2]), "peau"))
        return out
    return f


# --- Les marques (en couches) -------------------------------------------------

def marque(nom):
    def f(p):
        b = tete_base(p)
        out = []
        if nom == "taches":
            for s in (-1, 1):
                for (df, dr, dz) in ((0, 2.2, -1.6), (0, 3.4, -1.0), (-0.5, 4.2, -2.0)):
                    out.append((ellipsoide(T(p, 6.0 + df, s * dr, dz), b, [0.8, 0.45, 0.45]), "taches"))
        elif nom == "cicatrice":
            bb = [p.tF, rot(p.R, p.tF, 0.9), rot(p.tZ, p.tF, 0.9)]
            out.append((ellipsoide(T(p, 6.1, 2.6, 0.6), bb, [1.1, 0.35, 2.6]), "cicatrice"))
        elif nom == "peinture":
            for s in (-1, 1):
                for dz in (-1.2, -2.6):
                    out.append((ellipsoide(T(p, 5.6, s * 3.6, dz), b, [1.2, 1.6, 0.45]), "peinture"))
        elif nom == "runes":
            for (dr, dz) in ((0, 4.6), (-1.6, 4.0), (1.6, 4.0)):
                out.append((ellipsoide(T(p, 6.2, dr, dz), b, [0.8, 0.5, 0.5]), "runes"))
        elif nom == "ecailles":
            for s in (-1, 1):
                for (dr, dz) in ((4.6, -0.5), (4.9, -1.8), (4.2, -2.8)):
                    out.append((ellipsoide(T(p, 4.8, s * dr, dz), b, [0.9, 0.8, 0.6]), "peau"))
        elif nom == "larmes":
            for s in (-1, 1):
                out.append((ellipsoide(T(p, 6.0, s * 2.6, -2.2), b, [0.7, 0.4, 1.1]), "larmes"))
        return out
    return f


MARQUES = ["taches", "cicatrice", "peinture", "runes", "ecailles", "larmes"]


# --- Les traits des peuples ---------------------------------------------------

def trait(nom):
    def f(p):
        b = tete_base(p)
        F, R, Z = p.F, p.R, p.Z
        out = []
        if nom in ("oreilles_elfe", "oreilles_fee", "oreilles_rondes", "oreilles_gnome", "oreilles_nageoires"):
            for s in (-1, 1):
                if nom == "oreilles_elfe":   # longues, pointées vers l'arrière et le haut
                    d = rot(rot(p.R * s, p.tZ, -s * 0.55), p.tF, s * 0.5)
                    out.append((capsule(T(p, 0.2, s * 6.0, 0.4), T(p, 0, s * 6.0, 0) + d * 5.6, 0.8), "peau"))
                    out.append((ellipsoide(T(p, 0, s * 6.6, 0.4), [p.tF, d, np.cross(p.tF, d)], [1.6, 2.4, 0.7]), "peau"))
                elif nom == "oreilles_fee":
                    d = rot(rot(p.R * s, p.tZ, -s * 0.5), p.tF, s * 0.75)
                    out.append((capsule(T(p, 0.2, s * 6.0, 0.6), T(p, 0, s * 6.0, 0.6) + d * 3.8, 0.7), "peau"))
                elif nom == "oreilles_rondes":
                    out.append((ellipsoide(T(p, 0.3, s * 6.7, 0.4), b, [1.6, 1.1, 2.0]), "peau"))
                elif nom == "oreilles_gnome":
                    d = rot(p.R * s, p.tZ, -s * 0.3)
                    out.append((ellipsoide(T(p, 0, s * 7.8, 0.8), [p.tF, d, p.tZ], [2.2, 2.6, 3.0]), "peau"))
                else:   # nageoires d'oreilles des sirènes et tritons
                    d = rot(p.R * s, p.tZ, -s * 0.6)
                    out.append((ellipsoide(T(p, -0.8, s * 7.0, 1.0), [p.tF, d, p.tZ], [2.8, 0.5, 2.6]), "peau"))
        elif nom == "oreilles_chat":
            for s in (-1, 1):
                pointe = T(p, -0.6, s * 4.4, 11.4)
                out.append((capsule(T(p, -0.2, s * 3.8, 6.2), pointe, 1.9), "peau"))
                out.append((ellipsoide(T(p, -0.4, s * 4.0, 7.8), b, [1.2, 2.2, 2.6]), "peau"))
        elif nom == "bois":   # Hommes-bêtes : des bois de cerf, l'héritage de Cernos
            for s in (-1, 1):
                a = T(p, -0.5, s * 3.6, 5.6)
                m = a + p.R * s * 3.0 + p.tZ * 4.5
                h = m + p.R * s * 1.6 + p.tZ * 4.0 - p.tF * 0.8
                out += [(capsule(a, m, 0.95), "corne"), (capsule(m, h, 0.8), "corne"),
                        (capsule(m + (h - m) * 0.2, m + p.tF * 2.2 + p.tZ * 2.4, 0.7), "corne"),
                        (capsule(a + (m - a) * 0.5, a + (m - a) * 0.5 - p.tF * 1.6 + p.tZ * 3.0 + p.R * s * 0.4, 0.65), "corne")]
        elif nom == "cornes":   # Drakéides : deux cornes qui filent vers l'arrière
            for s in (-1, 1):
                a = T(p, 1.0, s * 3.8, 5.2)
                m = a - p.tF * 3.4 + p.tZ * 2.4 + p.R * s * 0.8
                h = m - p.tF * 3.0 + p.tZ * 0.2
                out += [(capsule(a, m, 1.25), "corne"), (capsule(m, h, 0.8), "corne")]
        elif nom == "defenses":   # Orques
            for s in (-1, 1):
                out.append((capsule(T(p, 5.4, s * 2.6, -4.6), T(p, 7.0, s * 3.0, -2.6), 0.6), "corne"))
        elif nom == "eclats":   # Lumins : des cristaux qui poussent sur la tête
            for (df, dr, dz, ln) in ((0.5, 0, 6.6, 4.2), (-1.5, -2.8, 5.8, 3.4), (-1.5, 2.8, 5.8, 3.4),
                                     (-3.5, -1.2, 5.0, 3.0), (-3.5, 1.4, 4.8, 2.6)):
                c = T(p, df, dr, dz)
                d = (c - p.tete) / np.linalg.norm(c - p.tete)
                lat = np.cross(d, p.R)
                lat /= np.linalg.norm(lat)
                out.append((octaedre(c + d * ln * 0.45, [d, p.R, lat], [ln, 1.1, 1.1]), "peau"))
        elif nom == "mousse":   # Golems : de la mousse et deux cailloux sur le crâne
            calotte = coupe(ellipsoide(p.tete + p.tZ * 1.0, b, [7.3, 7.1, 7.6]), p.tete + p.tZ * 3.4, -p.tZ)
            out.append((calotte, "mousse"))
            out.append((ellipsoide(T(p, -2.0, 3.0, 7.0), b, [1.6, 1.4, 1.2]), "peau"))
        elif nom == "ailes":   # Fées : deux paires d'ailes qui battent sans cesse
            batt = 0.55 + 0.45 * math.sin(2 * math.pi * p.t * 2)
            for s in (-1, 1):
                ep = p.buste(-3.6, s * 1.6, 38.0)
                for (dz, lg, haut, dx) in ((3.0, 9.5, 6.0, 0.0), (-4.0, 6.5, 4.0, 0.2)):
                    d = rot(rot(-p.F, p.Z, s * (0.6 + 0.6 * batt)), p.F, -s * 0.15)
                    d = d / np.linalg.norm(d)
                    up = rot(p.Z, d, dx)
                    ep2 = np.cross(d, up)
                    out.append((ellipsoide(ep + d * lg * 0.85 + p.Z * dz * 0.6, [d, up, ep2], [lg, haut, 0.35]), "aile"))
        elif nom in ("queue_chat", "queue_drake", "queue_touffue"):
            base = p.buste(-4.0 * p.k, 0, 23.5)
            ond = math.sin(2 * math.pi * p.t) * (0.35 if p.anim in ("marche", "course") else 0.2)
            pts = [base]
            d = rot(-p.F * 0.85 - p.Z * 0.5, p.Z, ond)
            nseg = 4 if nom != "queue_touffue" else 2
            for i in range(nseg):
                d = rot(d, p.Z, ond * 0.6)
                d = d + p.Z * (0.25 if nom == "queue_chat" else (0.05 if nom == "queue_drake" else 0.2))
                d = d / np.linalg.norm(d)
                pts.append(pts[-1] + d * (3.2 if nom != "queue_drake" else 3.6))
            for i in range(nseg):
                r = {"queue_chat": 1.1, "queue_drake": 2.3 - i * 0.5, "queue_touffue": 1.4}[nom]
                out.append((capsule(pts[i], pts[i + 1], r), "peau"))
            if nom == "queue_touffue":
                out.append((ellipsoide(pts[-1], [p.F, p.R, p.Z], [2.4, 2.2, 2.4]), "cheveux"))
        elif nom == "nageoires":   # Sirènes et tritons : de petites nageoires aux chevilles et aux avant-bras
            for (h, g, c, pi) in p.jambes:
                d = (c - g) / np.linalg.norm(c - g)
                out.append((ellipsoide(c - p.F * 1.6 + p.Z * 1.2, [d, p.F, np.cross(d, p.F)], [2.4, 2.0, 0.4]), "peau"))
            for (e, co, m, s) in p.bras:
                d = (m - co) / np.linalg.norm(m - co)
                n = np.cross(d, p.Z)
                if np.linalg.norm(n) < 0.2:
                    n = p.R
                n = n / np.linalg.norm(n)
                out.append((ellipsoide(co + (m - co) * 0.5 - np.cross(d, n) * 1.6, [d, n, np.cross(d, n)], [2.6, 0.4, 1.6]), "peau"))
        return out
    return f


TRAITS = ["oreilles_elfe", "oreilles_fee", "oreilles_rondes", "oreilles_gnome", "oreilles_nageoires", "oreilles_chat",
          "bois", "cornes", "defenses", "eclats", "mousse", "ailes", "queue_chat", "queue_drake", "queue_touffue",
          "nageoires"]


# --- Les cheveux (20) et les barbes (6) ----------------------------------------

def calotte(p, gonfle=0.0, front=2.2, haut=1.2):
    b = tete_base(p)
    c = ellipsoide(p.tete + p.tZ * haut - p.tF * 0.6, b, [7.6 + gonfle, 7.4 + gonfle, 7.6 + gonfle])
    return sauf(c, boite(T(p, 7.5, 0, -front), b, [4.0, 9.0, 4.6]))


def coiffure(n):
    def f(p):
        b = tete_base(p)
        out = []

        def e(fz, r, rad, m="cheveux"):
            out.append((ellipsoide(T(p, *fz), b, rad), m))

        def cap(a, z, r):
            out.append((capsule(T(p, *a), T(p, *z), r), "cheveux"))
        if n == 0:     # ras
            out.append((coupe(ellipsoide(p.tete + p.tZ * 0.6 - p.tF * 0.4, b, [7.2, 7.0, 7.4]), T(p, 0, 0, 1.4), -p.tZ), "cheveux"))
        elif n == 1:   # court
            out.append((calotte(p), "cheveux"))
            e((5.2, 1.6, 4.4), 0, [2.4, 3.6, 1.6])
        elif n == 2:   # en brosse, hérissé
            out.append((calotte(p, -0.2), "cheveux"))
            for (df, dr) in ((3, 0), (0, -3), (0, 3), (-3, 0), (1, -1.5), (1, 1.5), (-2, -2.5), (-2, 2.5)):
                c = T(p, df, dr, 6.0)
                d = (c - p.tete) / np.linalg.norm(c - p.tete)
                out.append((capsule(c, c + d * 3.0, 1.0), "cheveux"))
        elif n == 3:   # crête
            out.append((coupe(ellipsoide(p.tete + p.tZ * 0.4 - p.tF * 0.4, b, [7.1, 6.9, 7.3]), T(p, 0, 0, 1.8), -p.tZ), "cheveux"))
            out.append((boite(T(p, -0.5, 0, 7.4), b, [6.4, 1.0, 2.6], 0.9), "cheveux"))
        elif n == 4:   # boucles
            out.append((calotte(p), "cheveux"))
            for (df, dr, dz) in ((3, -4, 5), (3, 4, 5), (0, -6, 3), (0, 6, 3), (-4, -5, 3), (-4, 5, 3), (-5, 0, 5), (2, 0, 7), (-2, -3, 7), (-2, 3, 7)):
                e((df, dr, dz), 0, [2.4, 2.4, 2.4])
        elif n == 5:   # afro
            out.append((sauf(ellipsoide(p.tete + p.tZ * 3.2 - p.tF * 1.2, b, [10.0, 10.4, 9.4]), boite(T(p, 7.6, 0, -0.6), b, [5.0, 12.0, 5.0])), "cheveux"))
        elif n == 6:   # mi-longs, jusqu'à la mâchoire
            out.append((calotte(p), "cheveux"))
            for s in (-1, 1):
                e((0.5, s * 6.0, -2.0), 0, [4.2, 2.0, 5.0])
            e((-4.6, 0, -2.4), 0, [2.6, 6.4, 5.6])
        elif n == 7:   # longs
            out.append((calotte(p), "cheveux"))
            e((5.2, 1.6, 4.4), 0, [2.4, 3.6, 1.6])
            e((-4.8, 0, -5.5), 0, [2.6, 6.6, 9.0])
        elif n == 8:   # très longs, jusqu'à la taille
            out.append((calotte(p), "cheveux"))
            e((-4.6, 0, -9.0), 0, [2.6, 6.8, 13.5])
            for s in (-1, 1):
                e((1.0, s * 5.8, -3.5), 0, [2.4, 1.6, 6.5])
        elif n == 9:   # queue-de-cheval
            out.append((calotte(p), "cheveux"))
            cap((-6.4, 0, 3.0), (-9.0, 0, -4.5), 1.8)
            e((-9.2, 0, -6.0), 0, [1.8, 1.6, 2.6])
            e((-6.6, 0, 3.2), 0, [1.2, 1.4, 1.2], "ruban")
        elif n == 10:  # chignon
            out.append((calotte(p), "cheveux"))
            e((-6.0, 0, 4.2), 0, [3.2, 3.2, 3.0])
        elif n == 11:  # couettes
            out.append((calotte(p), "cheveux"))
            for s in (-1, 1):
                cap((-1.5, s * 6.6, 1.5), (-2.5, s * 9.0, -4.0), 1.7)
                e((-1.5, s * 6.8, 1.6), 0, [1.0, 1.2, 1.0], "ruban")
        elif n == 12:  # une tresse dans le dos
            out.append((calotte(p), "cheveux"))
            for i in range(6):
                e((-6.4 - i * 0.5, 0, 1.0 - i * 2.4), 0, [1.6, 1.8, 1.6])
        elif n == 13:  # deux tresses devant les épaules
            out.append((calotte(p), "cheveux"))
            for s in (-1, 1):
                for i in range(5):
                    e((1.0 - i * 0.2, s * (6.2 + i * 0.3), -1.0 - i * 2.4), 0, [1.4, 1.5, 1.4])
        elif n == 14:  # carré
            out.append((calotte(p, 0.4, 2.8), "cheveux"))
            out.append((coupe(sauf(ellipsoide(p.tete - p.tF * 0.4, b, [8.0, 8.0, 8.6]), boite(T(p, 7.0, 0, -3.0), b, [4.0, 10.0, 6.0])), T(p, 0, 0, -4.6), p.tZ * -1), "cheveux"))
        elif n == 15:  # mèche de côté
            out.append((calotte(p), "cheveux"))
            bb = [p.tF, rot(p.R, p.tF, 0.6), rot(p.tZ, p.tF, 0.6)]
            out.append((ellipsoide(T(p, 5.4, -2.2, 2.6), bb, [2.0, 5.0, 2.2]), "cheveux"))
            e((1.0, -6.0, -1.0), 0, [3.0, 1.8, 4.6])
        elif n == 16:  # dreadlocks
            out.append((calotte(p), "cheveux"))
            for i, a in enumerate(np.linspace(-1.9, 1.9, 7)):
                d0 = T(p, -math.cos(a) * 5.6 + 0.5, math.sin(a) * 6.4, 2.0)
                cap(((d0 - p.tete) @ p.tF, (d0 - p.tete) @ p.R, 2.0), ((d0 - p.tete) @ p.tF - 1.0, (d0 - p.tete) @ p.R * 1.15, -6.5 - (i % 2) * 1.5), 1.05)
        elif n == 17:  # chignon haut de guerrier
            out.append((coupe(ellipsoide(p.tete + p.tZ * 0.6 - p.tF * 0.4, b, [7.2, 7.0, 7.4]), T(p, 0, 0, 1.4), -p.tZ), "cheveux"))
            cap((-1.0, 0, 6.8), (-1.5, 0, 9.6), 1.6)
            e((-1.6, 0, 10.4), 0, [2.0, 2.0, 1.6])
        elif n == 18:  # sauvages
            out.append((calotte(p, 0.6), "cheveux"))
            e((-5.0, 0, -4.0), 0, [3.4, 7.6, 8.0])
            for s in (-1, 1):
                cap((0, s * 6.4, 2.0), (-1.0, s * 9.2, -1.0), 1.4)
                cap((-3.0, s * 6.0, 4.0), (-5.0, s * 8.6, 6.0), 1.2)
            e((5.4, 0.8, 4.6), 0, [2.4, 4.6, 1.6])
        elif n == 19:  # frange droite et cheveux lisses aux épaules
            out.append((calotte(p, 0.2, 1.4), "cheveux"))
            out.append((boite(T(p, 6.2, 0, 3.6), b, [1.0, 5.6, 1.4], 0.6), "cheveux"))
            e((-4.4, 0, -4.6), 0, [2.8, 7.2, 7.4])
            for s in (-1, 1):
                e((1.0, s * 6.2, -2.6), 0, [2.6, 1.6, 5.2])
        return out
    return f


NB_COIFFURES = 20
NOMS_COIFFURES = ["Ras", "Court", "En brosse", "Crête", "Boucles", "Afro", "Mi-longs", "Longs", "Très longs",
                  "Queue-de-cheval", "Chignon", "Couettes", "Tresse", "Deux tresses", "Carré", "Mèche de côté",
                  "Dreadlocks", "Chignon de guerrier", "Sauvages", "Frange droite"]


def barbe(n):
    def f(p):
        b = tete_base(p)
        out = []
        if n == 0:     # courte
            out.append((ellipsoide(T(p, 4.6, 0, -4.4), b, [3.4, 5.4, 4.4]), "cheveux"))
            out.append((ellipsoide(T(p, 6.6, 0, -2.6), b, [1.2, 3.4, 1.0]), "cheveux"))
        elif n == 1:   # longue
            out.append((ellipsoide(T(p, 4.8, 0, -5.6), b, [3.4, 5.2, 6.0]), "cheveux"))
            out.append((ellipsoide(T(p, 6.6, 0, -2.6), b, [1.2, 3.4, 1.0]), "cheveux"))
        elif n == 2:   # tressée, à la naine
            out.append((ellipsoide(T(p, 4.6, 0, -4.4), b, [3.4, 5.4, 4.2]), "cheveux"))
            for i in range(4):
                out.append((ellipsoide(T(p, 5.6 + i * 0.3, 0, -7.6 - i * 2.0), b, [1.6, 1.8, 1.4]), "cheveux"))
            out.append((ellipsoide(T(p, 6.6, 0, -2.6), b, [1.2, 3.4, 1.0]), "cheveux"))
            out.append((ellipsoide(T(p, 6.4, 0, -14.4), b, [1.0, 1.2, 0.8]), "boucle"))
        elif n == 3:   # moustache
            for s in (-1, 1):
                out.append((ellipsoide(T(p, 6.6, s * 1.8, -2.6), [p.tF, rot(p.R, p.tF, s * 0.35), rot(p.tZ, p.tF, s * 0.35)], [1.1, 2.4, 0.9]), "cheveux"))
        elif n == 4:   # bouc
            out.append((ellipsoide(T(p, 6.0, 0, -5.4), b, [1.8, 1.8, 2.2]), "cheveux"))
            out.append((ellipsoide(T(p, 6.6, 0, -2.6), b, [1.0, 2.6, 0.8]), "cheveux"))
        elif n == 5:   # favoris
            for s in (-1, 1):
                out.append((ellipsoide(T(p, 2.6, s * 5.6, -2.6), b, [2.0, 1.6, 3.6]), "cheveux"))
        return out
    return f


NOMS_BARBES = ["Courte", "Longue", "Tressée", "Moustache", "Bouc", "Favoris"]


# --- Les vêtements (refaits pour chaque carrure) -------------------------------

def tunique(p):
    F, R, Z, k = p.F, p.R, p.Z, p.k
    out = [(ellipsoide(p.buste(0, 0, 33.8), [F, R, Z], [5.6 * k, 7.5 * k, 9.6]), "tissu"),
           (coupe(ellipsoide(p.buste(0, 0, 25.5), [F, R, Z], [6.2 * k, 7.6 * k, 7.5]), p.buste(0, 0, 18.5), -Z), "tissu"),
           (ellipsoide(p.buste(0, 0, 25.6), [F, R, Z], [5.5 * k, 7.3 * k, 1.2]), "ceinture"),
           (ellipsoide(p.buste(5.4 * k, 0, 25.6), [F, R, Z], [0.8, 1.4, 1.3]), "boucle")]
    lk = 0.6 + 0.4 * k
    for (e, co, m, s) in p.bras:
        out.append((capsule(e, e + (co - e) * 0.75, 3.1 * lk), "tissu"))
    return out


def pantalon(p):
    lk = 0.6 + 0.4 * p.k
    out = []
    for (h, g, c, pi) in p.jambes:
        out.append((capsule(h, g, 3.7 * lk), "pantalon"))
        out.append((capsule(g, g + (c - g) * 0.7, 3.2 * lk), "pantalon"))
    return out


def bottes(p):
    F, R, Z = p.F, p.R, p.Z
    lk = 0.6 + 0.4 * p.k
    out = []
    for (h, g, c, pi) in p.jambes:
        out.append((capsule(g + (c - g) * 0.45, c, 3.4 * lk), "bottes"))
        out.append((ellipsoide(pi, [F, R, Z], [3.9, 2.6, 1.9]), "bottes"))
    return out


def gants(p):
    F, R, Z = p.F, p.R, p.Z
    lk = 0.6 + 0.4 * p.k
    out = []
    for (e, co, m, s) in p.bras:
        out.append((capsule(co + (m - co) * 0.55, m, 2.7 * lk), "gants"))
        out.append((ellipsoide(m, [F, R, Z], [2.8, 2.4, 3.0]), "gants"))
    return out


def cape(p):
    k = p.k
    out = fh.cape(p)
    if k != 1.0:
        # La cape s'écarte un peu plus des épaules d'un héros costaud.
        out = [((lambda f: (lambda q: f(_ecarte(p, q, 1.0 / k)) * min(1.0, k)))(f), m) for f, m in out]
    return out


def _ecarte(p, q, s):
    """Resserre (s > 1) ou écarte (s < 1) les points autour de l'axe du buste, à l'horizontale."""
    axe0 = p.L(0, 0, 0)
    d = q - axe0
    z = d @ p.Z
    horiz = d - np.outer(z, p.Z)
    return axe0 + np.outer(z, p.Z) + horiz * s


def robe(p):
    F, R, Z = p.F, p.R, p.Z
    out = tunique(p)[:1] + tunique(p)[2:]
    k = p.k
    jupe = coupe(ellipsoide(p.L(0, 0, 13.0 + p.rebond * 0.5), [F, R, Z], [8.2 * k, 9.4 * k, 14.5]), p.L(0, 0, 2.2), -Z)
    jupe = coupe(jupe, p.L(0, 0, 27.0 + p.rebond), Z)
    out.append((jupe, "tissu"))
    return out


def tablier(p):
    F, R, Z, k = p.F, p.R, p.Z, p.k
    return [(boite(p.buste(5.6 * k, 0, 19.5), [F, R, Z], [0.7, 5.4 * k, 7.0], 0.4), "tablier"),
            (boite(p.buste(5.4 * k, 0, 31.5), [F, R, Z], [0.6, 3.8 * k, 5.0], 0.4), "tablier")]


VETEMENTS = {"tunique": tunique, "pantalon": pantalon, "bottes": bottes, "gants": gants, "cape": cape,
             "robe": robe, "tablier": tablier}


# --- Les armes des 17 classes --------------------------------------------------

def _main(p, cote):
    e, co, m, s = p.bras[0 if cote < 0 else 1]
    av = (m - co) / np.linalg.norm(m - co)
    return co, m, av


def _repere(d, R):
    d = d / np.linalg.norm(d)
    lat = np.cross(d, R)
    if np.linalg.norm(lat) < 1e-3:
        lat = np.cross(d, np.array([0.0, 0.0, 1.0]))
    lat /= np.linalg.norm(lat)
    return d, np.cross(lat, d), lat


def _dir_lame(p):
    co, m, av = _main(p, 1)
    d = p.F * 0.55 - p.Z * 0.85 + av * 0.35
    if p.lame_bras:
        d = av * 1.0 + p.F * 0.25
    return m, d / np.linalg.norm(d)


def _dir_hampe(p):
    """Bâtons et lances : tenus droits au repos, pointés vers l'avant pour frapper."""
    co, m, av = _main(p, 1)
    if p.anim in ("attaque", "sort"):
        d = av + p.F * 0.4 + p.Z * 0.3
    elif p.anim in ("incante",):
        d = p.Z * 1.0 + p.F * 0.35
    elif p.anim == "mort":
        d = p.F * 0.2 - p.Z * 0.3 + p.R
    else:
        d = p.Z * 1.0 + p.F * 0.18 + av * 0.12
    return m, d / np.linalg.norm(d)


def arme(nom):
    def f(p):
        F, R, Z = p.F, p.R, p.Z
        out = []
        if nom in ("epee", "epee_longue", "rapiere", "masse", "hache"):
            m, d = _dir_lame(p)
            d, n, lat = _repere(d, R)
            base = m + d * 1.5
            ln = {"epee": 9.5, "epee_longue": 12.0, "rapiere": 10.5, "masse": 7.0, "hache": 7.5}[nom]
            if nom in ("epee", "epee_longue"):
                out.append((union(boite(base + d * (ln + 1.0), [d, R, lat], [ln, 0.45, 1.25 if nom == "epee" else 1.4], 0.2),
                                  capsule(base + d * (ln * 2 + 0.5), base + d * (ln * 2 + 2.0), 0.6)), "lame"))
                out.append((boite(base, [d, R, lat], [0.7, 0.9, 3.4 if nom == "epee" else 4.2], 0.3), "garde"))
            elif nom == "rapiere":
                out.append((capsule(base + d * 1.0, base + d * (ln * 2), 0.45), "lame"))
                out.append((ellipsoide(base + d * 0.4, [d, R, lat], [0.8, 2.0, 2.0]), "garde"))
            elif nom == "masse":
                out.append((capsule(base, base + d * ln * 1.6, 0.7), "bois_arme"))
                out.append((ellipsoide(base + d * ln * 1.75, [d, R, lat], [2.6, 2.4, 2.4]), "fer"))
                for s in (-1, 1):
                    out.append((capsule(base + d * ln * 1.75 + lat * s * 2.4, base + d * ln * 1.75 + lat * s * 3.6, 0.6), "fer"))
            else:
                out.append((capsule(base, base + d * ln * 1.7, 0.7), "bois_arme"))
                out.append((boite(base + d * ln * 1.45 + lat * 2.2, [d, R, lat], [2.6, 0.45, 2.2], 0.3), "lame"))
            out.append((capsule(m - d * 2.5, m + d * 0.5, 1.0), "poignee"))
            out.append((ellipsoide(m - d * 3.2, [F, R, Z], [1.3, 1.3, 1.3]), "garde"))
            if nom == "hache":   # le Berserker en a une dans chaque main
                co, m2, av2 = _main(p, -1)
                d2 = F * 0.55 - Z * 0.85 + av2 * 0.35
                d2, n2, lat2 = _repere(d2, R)
                out.append((capsule(m2 - d2 * 2.0, m2 + d2 * 12.0, 0.7), "bois_arme"))
                out.append((boite(m2 + d2 * 10.4 + lat2 * 2.2, [d2, R, lat2], [2.6, 0.45, 2.2], 0.3), "lame"))
        elif nom == "dagues":
            for cote in (-1, 1):
                co, m, av = _main(p, cote)
                d = av if (p.lame_bras or p.anim == "attaque") else F * 0.5 - Z * 0.8 + av * 0.3
                d, n, lat = _repere(d, R)
                out.append((boite(m + d * 4.2, [d, R, lat], [3.4, 0.4, 0.9], 0.2), "lame"))
                out.append((boite(m + d * 0.9, [d, R, lat], [0.5, 0.7, 2.0], 0.2), "garde"))
        elif nom in ("lance", "baton", "baton_mage", "baton_noueux", "totem", "faux"):
            m, d = _dir_hampe(p)
            d, n, lat = _repere(d, R)
            bas, haut = m - d * 9.0, m + d * (17.0 if nom in ("lance", "faux") else 13.0)
            out.append((capsule(bas, haut, 0.75 if nom != "baton_noueux" else 0.95), "bois_arme"))
            if nom == "lance":
                out.append((octaedre(haut + d * 2.6, [d, n, lat], [3.4, 0.5, 1.4]), "lame"))
                out.append((ellipsoide(haut, [d, n, lat], [0.6, 1.0, 1.0]), "ruban"))
            elif nom == "baton_mage":
                out.append((ellipsoide(haut + d * 1.8, [d, n, lat], [1.8, 1.8, 1.8]), "orbe"))
                for s in (-1, 1):
                    out.append((capsule(haut - d * 0.8, haut + d * 2.2 + lat * s * 1.8, 0.5), "garde"))
            elif nom == "baton_noueux":
                out.append((ellipsoide(haut + d * 1.0, [d, n, lat], [1.8, 1.6, 1.6]), "bois_arme"))
                out.append((ellipsoide(haut + d * 0.2 + lat * 1.6, [d, n, lat], [1.2, 0.8, 1.4]), "mousse"))
                out.append((capsule(haut - d * 1.0, haut + d * 1.6 + n * 2.0, 0.5), "bois_arme"))
            elif nom == "totem":
                out.append((boite(haut + d * 1.0, [d, n, lat], [2.2, 1.6, 1.6], 0.4), "totem"))
                for s in (-1, 1):
                    out.append((capsule(haut + d * 0.6 + lat * s * 1.4, haut + d * 0.2 + lat * s * 3.2 - d * 2.4, 0.45), "plume"))
            elif nom == "faux":
                out.append((ellipsoide(haut - n * 4.0 - d * 1.2, [n, d, lat], [5.0, 1.2, 0.4]), "lame"))
            else:   # bâton de moine : bagues de cuivre
                for t in (0.1, 0.9):
                    out.append((capsule(bas + (haut - bas) * t, bas + (haut - bas) * (t + 0.04), 1.0), "cuivre"))
        elif nom == "arc":
            co, m, av = _main(p, -1)
            d = Z * 1.0 + F * 0.1 if p.anim == "tir" else -Z * 0.6 + F * 0.6 + av * 0.2
            d, n, lat = _repere(d, R)
            avant = F if p.anim == "tir" else n
            for s in (-1, 1):
                a = m + d * s * 0.6
                mi = m + d * s * 5.2 + avant * 1.4
                b = m + d * s * 9.6
                out += [(capsule(a, mi, 0.65), "bois_arme"), (capsule(mi, b, 0.55), "bois_arme")]
            tire = 0.0
            if p.anim == "tir" and p.t < 0.75:
                tire = min(1.0, p.t / 0.6) * 5.0
            for s in (-1, 1):
                out.append((capsule(m + d * s * 9.6, m - avant * tire, 0.18), "corde"))
        elif nom == "carquois":
            c = p.buste(-5.0 * p.k, 2.6, 34.0)
            d = rot(Z, F, 0.4)
            d, n, lat = _repere(d, R)
            out.append((capsule(c - d * 5.0, c + d * 4.0, 1.9), "cuir_carquois"))
            for s in (-1, 0, 1):
                out.append((capsule(c + d * 4.0 + lat * s * 0.8, c + d * 6.6 + lat * s * 0.8, 0.5), "plume"))
        elif nom == "luth":
            c = p.buste(-5.4 * p.k, 0.6, 31.0)
            d = rot(Z, F, -0.55)
            d, n, lat = _repere(d, R)
            out.append((ellipsoide(c - d * 2.0, [d, F, np.cross(d, F)], [3.8, 1.3, 3.2]), "bois_arme"))
            out.append((capsule(c + d * 1.0, c + d * 8.0, 0.6), "poignee"))
            out.append((ellipsoide(c - d * 1.8 + F * -1.2, [d, F, np.cross(d, F)], [0.9, 0.3, 0.9]), "poignee"))
        elif nom == "orbe":
            co, m, av = _main(p, -1)
            bob = 0.6 * math.sin(2 * math.pi * p.t * 2)
            c = m + Z * (4.2 + bob) + F * 1.0
            if p.anim in ("sort", "incante"):
                c = m + F * 3.0 + Z * 2.0
            out.append((ellipsoide(c, [F, R, Z], [2.2, 2.2, 2.2]), "orbe"))
            out.append((ellipsoide(c, [F, R, Z], [2.8, 2.8, 0.4]), "garde"))
        elif nom == "fioles":
            co, m, av = _main(p, 1)
            c = m + Z * 1.6 + F * 0.6
            out.append((ellipsoide(c, [F, R, Z], [1.5, 1.5, 1.7]), "potion"))
            out.append((capsule(c + Z * 1.4, c + Z * 2.6, 0.55), "verre"))
            for s in (-1, 1):   # deux fioles à la ceinture
                b = p.buste(2.0 * p.k, s * 6.8 * p.k, 24.0)
                out.append((ellipsoide(b, [F, R, Z], [1.0, 1.0, 1.4]), "potion"))
                out.append((capsule(b + Z * 1.2, b + Z * 2.0, 0.4), "verre"))
        elif nom == "arquebuse":
            co, m, av = _main(p, 1)
            d = F if p.anim in ("tir", "attaque", "sort") else F * 0.6 - Z * 0.8
            d, n, lat = _repere(d, R)
            out.append((boite(m - d * 1.0 - n * 0.8, [d, n, lat], [3.4, 1.4, 0.8], 0.4), "bois_arme"))
            out.append((capsule(m + d * 1.0, m + d * 12.0, 0.75), "fer"))
            out.append((capsule(m + d * 11.2, m + d * 12.4, 1.05), "cuivre"))
            out.append((boite(m + d * 3.6 + n * 0.9, [d, n, lat], [1.0, 0.6, 0.6], 0.2), "cuivre"))
        elif nom in ("bouclier", "bouclier_rond"):
            out = fh.bouclier(p)
            if nom == "bouclier_rond":
                out = [(f, {"bouclier": "bois_arme", "umbo": "fer"}[m]) for f, m in out]
        return out
    return f


ARMES = ["epee", "bouclier", "bouclier_rond", "hache", "lance", "epee_longue", "baton", "baton_mage", "faux",
         "orbe", "fioles", "arquebuse", "masse", "baton_noueux", "totem", "rapiere", "luth", "dagues", "arc", "carquois"]

DEFAUTS_ARMES = {"cuir_carquois": "#7a4a2a"}


def casque(p):
    return fh.casque(p)


def chapeau(p):
    return fh.chapeau(p)


# --- Le catalogue des pièces --------------------------------------------------
# nom de planche -> (fonction, forme du corps qui cache, carrure, enveloppe)

def catalogue():
    cat = {}
    for forme in CORPS:
        for nomc, k in CARRURES.items():
            cat[f"corps_{forme}_{nomc}"] = ("corps", forme, k, 0.0)
    for n in range(16):
        cat[f"visage_{n:02d}"] = (visage(n), "standard", 1.0, 0.0)
    for m in MARQUES:
        cat[f"marque_{m}"] = (marque(m), "standard", 1.0, 0.0)
    for t in TRAITS:
        cat[f"trait_{t}"] = (trait(t), "standard", 1.0, 0.4 if t.startswith("queue") or t == "ailes" else 0.0)
    for n in range(NB_COIFFURES):
        cat[f"cheveux_{n:02d}"] = (coiffure(n), "standard", 1.0, 0.6 if n in (6, 7, 8, 12, 14, 16, 18, 19) else 0.0)
    for n in range(len(NOMS_BARBES)):
        cat[f"barbe_{n:02d}"] = (barbe(n), "standard", 1.0, 0.0)
    for v, fn in VETEMENTS.items():
        for nomc, k in CARRURES.items():
            env = 1.2 if v == "cape" else (1.4 if v == "tablier" else 0.0)
            cat[f"{v}_{nomc}"] = (fn, "standard", k, env)
    cat["casque"] = (casque, "standard", 1.0, 0.0)
    cat["chapeau"] = (chapeau, "standard", 1.0, 0.0)
    for a in ARMES:
        cat[f"arme_{a}"] = (arme(a), "standard", 1.0, 1.0)
    return cat


# --- Le rendu -----------------------------------------------------------------

def rendu(nom, spec, pose):
    fn, forme, k, env = spec
    if fn == "corps":
        formes = CORPS[forme](pose)
        garde = set(range(len(formes)))
    else:
        base = corps_standard(pose)
        piece = fn(pose)
        formes = [(fh.gonfle(f, env), m) for f, m in base] + piece
        garde = set(range(len(base), len(formes)))
    if not formes or len(garde) == 0:
        return None
    if pose.spin:
        centre = pose.L(0, 0, 19.0)
        lift = pose.Z * 12.0 * abs(math.sin(pose.spin / 2))
        formes = [(fh.tourne(f, centre, pose.R, pose.spin), m) for f, m in formes]
        formes = [((lambda f: lambda q: f(q - lift))(f), m) for f, m in formes]
    touche, prof, normales = fh.lancer(formes)
    tons = fh.ton_lumiere(normales)
    mats = np.full(touche.shape, "", dtype=object)
    for i, (f, m) in enumerate(formes):
        if i in garde:
            mats[touche == i] = m
    return mats.reshape(H, W), tons.reshape(H, W), prof.reshape(H, W), (touche >= 0).reshape(H, W)


def code(mats, tons, cont, slots):
    img = np.zeros((H, W, 4), np.uint8)
    for i, mat in enumerate(slots):
        sel = mats == mat
        for t in range(4):
            s2 = sel & (tons == t)
            img[s2, 0] = (i * 5 + t) * 6
            img[s2, 3] = 255
        sc = cont == mat
        img[sc, 0] = (i * 5 + PALIER_CONTOUR) * 6
        img[sc, 3] = 255
    return img


def total_images():
    return sum(a[1] for a in ANIMS)


def taille_planche():
    n = total_images()
    rangs = (n + COLONNES - 1) // COLONNES
    return COLONNES * W, rangs * 8 * H


def case(col, d):
    """Coin haut-gauche de l'image n° col (dans l'ordre des animations), direction d."""
    return (col % COLONNES) * W, (col // COLONNES) * 8 * H + d * H


def forge_piece(nom):
    spec = catalogue()[nom]
    k = spec[2]
    largeur, hauteur = taille_planche()
    rendus = []
    for d in range(8):
        col = 0
        for anim, n, _, _ in ANIMS:
            for i in range(n):
                pose = Pose(d, anim, i, n, k)
                r = rendu(nom, spec, pose)
                rendus.append((col, d, r))
                col += 1
    slots = []
    for _, _, r in rendus:
        if r is None:
            continue
        for m in np.unique(r[0]):
            if m and m not in slots:
                slots.append(m)
    # Un ordre stable : celui du dictionnaire des défauts quand c'est possible.
    ordre = ["peau", "sous", "yeux", "sourcils", "cheveux", "tissu", "ceinture", "boucle"]
    slots.sort(key=lambda m: (ordre.index(m) if m in ordre else 99, m))
    assert len(slots) <= 8, (nom, slots)
    planche = np.zeros((hauteur, largeur, 4), np.uint8)
    for col, d, r in rendus:
        if r is None:
            continue
        mats, tons, prof, plein = r
        cont = fh.contours(mats, prof, plein)
        x, y = case(col, d)
        planche[y:y + H, x:x + W] = code(mats, tons, cont, slots)
    Image.fromarray(planche).save(os.path.join(SORTIE, nom + ".png"), optimize=True)
    print(f"  {nom} : {slots}", file=sys.stderr)
    return nom, slots


def ecrire_donnees(pieces):
    os.makedirs(os.path.dirname(DONNEES), exist_ok=True)
    meta = {"image": [W, H], "ancre": list(fh.ANCRE), "directions": 8, "colonnes": COLONNES, "anims": {},
            "pieces": pieces}
    col = 0
    for nom, n, fps, boucle in ANIMS:
        meta["anims"][nom] = {"debut": col, "images": n, "fps": fps, "boucle": boucle}
        col += n
    with open(os.path.join(SORTIE, "planches.json"), "w") as fh_:
        json.dump(meta, fh_, ensure_ascii=False, indent=1, sort_keys=True)
    donnees = {
        "peuples": pp.PEUPLES, "yeux": pp.YEUX, "cheveux": pp.CHEVEUX, "tuniques": pp.TUNIQUES,
        "defauts": dict(pp.DEFAUTS, **DEFAUTS_ARMES), "alpha": pp.ALPHA, "chapeaux": pp.CHAPEAUX,
        "coiffures": NOMS_COIFFURES, "barbes": NOMS_BARBES, "marques": MARQUES, "carrures": list(CARRURES),
    }
    with open(DONNEES, "w") as fh_:
        json.dump(donnees, fh_, ensure_ascii=False, indent=1)


# --- Aperçu (pour vérifier à l'œil) -------------------------------------------

def decode(planche, couleurs):
    """Remplit une planche codée avec une couleur par matériau."""
    a = np.asarray(planche)
    idx = (a[..., 0].astype(int) + 3) // 6
    slot, ton = idx // 5, idx % 5
    out = np.zeros_like(a)
    for i, hexa in enumerate(couleurs):
        ramp = rampe(hexa)
        for t in range(5):
            sel = (a[..., 3] > 0) & (slot == i) & (ton == t)
            out[sel, :3] = ramp[t]
            out[sel, 3] = 255
    return out


def compose(couches, col, d, meta):
    """couches : [(pièce, {matériau: couleur})] -> image RGBA d'une case."""
    img = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    x, y = case(col, d)
    for piece, couleurs in couches:
        pl = Image.open(os.path.join(SORTIE, piece + ".png")).crop((x, y, x + W, y + H))
        slots = meta["pieces"][piece]
        defs = dict(pp.DEFAUTS, **DEFAUTS_ARMES)
        c = [couleurs.get(s, defs.get(s, "#ff00ff")) for s in slots]
        img.alpha_composite(Image.fromarray(decode(pl, c)))
    return img


def heros_type(peuple, n):
    """Une tenue d'exemple pour l'aperçu."""
    P = pp.PEUPLES[peuple]
    forme = P["forme"]
    carr = ["mince", "normale", "costaud"][n % 3]
    peau = P["peaux"][(n * 5) % 24]
    cheveux = pp.CHEVEUX[(n * 7) % 24]
    tissu = pp.TUNIQUES[n % 12]
    cs = {"peau": peau, "yeux": pp.YEUX[(n * 3) % 12], "sourcils": cheveux, "cheveux": cheveux, "tissu": tissu}
    couches = [f"corps_{forme}_{carr}", f"pantalon_{carr}", f"bottes_{carr}"]
    couches += [f"trait_{t}" for t in P["traits"] if t.startswith(("queue", "ailes", "nageoires"))]
    couches += [f"tunique_{carr}", f"visage_{n % 16:02d}"]
    if forme == "standard" and peuple not in ("fee",):
        couches.append(f"cheveux_{(n * 3) % 20:02d}")
    couches += [f"trait_{t}" for t in P["traits"] if not t.startswith(("queue", "ailes", "nageoires"))]
    if peuple == "nain":
        couches.append("barbe_02")
    if peuple == "fee":
        couches.append(f"cheveux_{(n * 3) % 20:02d}")
    couches.append(["arme_epee", "arme_baton_mage", "arme_arc", "arme_lance", "arme_dagues"][n % 5])
    return [(c, cs) for c in couches]


def apercus():
    with open(os.path.join(SORTIE, "planches.json")) as f:
        meta = json.load(f)
    peuples = list(pp.PEUPLES)
    # 1. Les 15 peuples, de face (direction 1), à leur taille.
    pl = Image.new("RGBA", (W * 3 * 15, H * 3 + 30), (40, 36, 52, 255))
    for i, pe in enumerate(peuples):
        img = compose(heros_type(pe, i), 0, 1, meta)
        t = pp.PEUPLES[pe]["taille"]
        g = img.resize((int(W * 3 * t), int(H * 3 * t)), Image.NEAREST)
        pl.alpha_composite(g, (i * W * 3 + (W * 3 - g.width) // 2, H * 3 - g.height + 10 - int(12 * 3 * (1 - t))))
    pl.save(os.path.join(ICI, "apercu_peuples.png"))
    # 2. Les 20 coiffures et les 16 visages.
    pl = Image.new("RGBA", (W * 2 * 20, H * 2 * 2), (40, 36, 52, 255))
    cs = {"peau": "#f0bf98", "cheveux": "#c0532a", "sourcils": "#c0532a", "yeux": "#3a6ab0", "tissu": "#4a7bd0"}
    for n in range(20):
        c = [(x, cs) for x in ("corps_standard_normale", "tunique_normale", "visage_00", f"cheveux_{n:02d}")]
        pl.alpha_composite(compose(c, 0, 1, meta).resize((W * 2, H * 2), Image.NEAREST), (n * W * 2, 0))
    for n in range(16):
        c = [(x, cs) for x in ("corps_standard_normale", "tunique_normale", f"visage_{n:02d}", "cheveux_00")]
        pl.alpha_composite(compose(c, 0, 1, meta).resize((W * 2, H * 2), Image.NEAREST), (n * W * 2, H * 2))
    pl.save(os.path.join(ICI, "apercu_miroir.png"))
    print("Aperçus : tools/sprites/apercu_peuples.png, apercu_miroir.png", file=sys.stderr)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--apercu", action="store_true")
    ap.add_argument("--pieces", nargs="*")
    ap.add_argument("-j", type=int, default=os.cpu_count())
    a = ap.parse_args()
    os.makedirs(SORTIE, exist_ok=True)
    cat = catalogue()
    noms = a.pieces if a.pieces is not None else list(cat)
    chemin = os.path.join(SORTIE, "planches.json")
    pieces = {}
    if os.path.exists(chemin):
        with open(chemin) as f:
            pieces = json.load(f).get("pieces", {})
    if noms:
        with Pool(a.j) as pool:
            for nom, slots in pool.imap_unordered(forge_piece, noms):
                pieces[nom] = slots
    pieces = {k: pieces[k] for k in cat if k in pieces}
    ecrire_donnees(pieces)
    if a.apercu:
        apercus()


if __name__ == "__main__":
    main()
