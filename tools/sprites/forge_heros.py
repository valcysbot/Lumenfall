#!/usr/bin/env python3
"""La forge des héros : dessine le héros humain en pixel art, pièce par pièce.

Le héros est un petit mannequin en 3D (capsules, ellipsoïdes, boîtes) qu'on
photographie dans la vraie vue 3/4 isométrique du jeu, pour les 8 directions et
chaque image d'animation. Chaque pièce (corps, cheveux, tunique, cape…) sort sur
sa propre planche, déjà masquée par le corps : le jeu n'a plus qu'à les empiler.

Comme tout part du même mannequin, une pièce tombe toujours pile au bon endroit,
quelle que soit la direction.

    python3 tools/sprites/forge_heros.py            # écrit client/assets/heros/humain/
    python3 tools/sprites/forge_heros.py --apercu   # + une planche d'aperçu
"""
import argparse
import colorsys
import json
import math
import os
import sys

import numpy as np
from PIL import Image

ICI = os.path.dirname(os.path.abspath(__file__))
RACINE = os.path.normpath(os.path.join(ICI, "..", ".."))
SORTIE = os.path.join(RACINE, "client", "assets", "heros", "humain")

# --- Cadre et caméra ---------------------------------------------------------

W, H = 64, 64          # taille d'une image (large pour la chute)
ANCRE = (32, 58)       # pixel des pieds (point au sol)
PITCH = math.radians(30)   # la vue 2:1 des tuiles 64 × 32
FWD = np.array([-math.cos(PITCH) / math.sqrt(2), -math.cos(PITCH) / math.sqrt(2), -math.sin(PITCH)])
RIGHT = np.array([1 / math.sqrt(2), -1 / math.sqrt(2), 0.0])
UP = np.cross(FWD, RIGHT)
LUMIERE = -RIGHT * 0.75 + UP * 0.9 - FWD * 0.55
LUMIERE /= np.linalg.norm(LUMIERE)

# Animations : nom -> (images, images par seconde, en boucle)
ANIMS = [("repos", 4, 4, True), ("marche", 8, 10, True), ("course", 8, 14, True), ("salut", 6, 5, False),
         # Le combat (Acte III).
         ("attaque", 6, 14, False), ("bouclier", 5, 12, False), ("garde", 2, 3, True),
         ("incante", 4, 6, True), ("sort", 5, 10, False), ("roulade", 6, 17, False), ("mort", 6, 8, False)]

# --- Couleurs ----------------------------------------------------------------

COULEURS_HEROS = ["#e05a47", "#f0a03c", "#e8c66a", "#8cc152", "#3fb58c", "#4fb3d9",
                  "#4a7bd0", "#8a6ad8", "#d066b0", "#f2f0ea", "#8d8a96", "#5a4636"]
CHEVEUX = {"noir": "#2e2433", "brun": "#5a3825", "chatain": "#8a5a35",
           "blond": "#e3b75c", "roux": "#c0532a", "blanc": "#ddd8d0"}
FIXES = {"peau": "#f0bf98", "yeux": "#2a2238", "pantalon": "#6e4c35", "bottes": "#4a3328",
         "gants": "#8a5a3a", "cape": "#2f6f8a", "casque": "#a4adc0", "lame": "#d5deea",
         "garde": "#d0a548", "poignee": "#4a3328", "ceinture": "#5a3a24", "boucle": "#e3c060",
         "tablier": "#e6dfcc", "bouclier": "#3d5f9e", "umbo": "#e3c060"}
# Pour les habitants : d'autres peaux, et des chapeaux.
PEAUX = {"": "#f0bf98", "mat": "#cf9670", "brun": "#8e5d40"}
CHAPEAUX = {"paille": "#dcb862", "feutre": "#4a3d52"}


def rampe(hexa):
    """4 tons (ombre profonde, ombre, base, lumière) + la couleur du contour.
    Les ombres glissent vers le violet et les lumières vers le jaune : c'est ce
    qui donne l'éclat « façon Aion »."""
    r, g, b = (int(hexa[i:i + 2], 16) / 255 for i in (1, 3, 5))
    h, s, v = colorsys.rgb_to_hsv(r, g, b)

    def ton(dh, ds, dv):
        hh = h + dh
        # Décale la teinte vers 0.72 (violet) pour les ombres, 0.14 (jaune) pour la lumière.
        return colorsys.hsv_to_rgb(hh % 1.0, min(1, max(0, s + ds)), min(1, max(0, v + dv)))

    def vers(cible, k):
        d = ((cible - h + 0.5) % 1.0) - 0.5
        return d * k

    tons = [ton(vers(0.72, 0.18), 0.12, -0.42), ton(vers(0.72, 0.08), 0.06, -0.2),
            ton(0, 0, 0), ton(vers(0.14, 0.08), -0.12, 0.14)]
    contour = ton(vers(0.72, 0.22), 0.1, -0.62)
    return [tuple(int(round(c * 255)) for c in t) for t in tons + [contour]]


# --- Formes (fonctions de distance) -----------------------------------------

def capsule(a, b, r):
    a, b = np.asarray(a, float), np.asarray(b, float)
    ba = b - a
    l2 = max(float(ba @ ba), 1e-9)

    def f(p):
        pa = p - a
        h = np.clip(pa @ ba / l2, 0, 1)
        return np.linalg.norm(pa - h[:, None] * ba, axis=1) - r
    return f


def ellipsoide(c, base, rayons):
    """Ellipsoïde de centre c, d'axes `base` (3 vecteurs unitaires) et de rayons donnés."""
    c = np.asarray(c, float)
    m = np.asarray(base, float)
    rr = np.asarray(rayons, float)

    def f(p):
        q = (p - c) @ m.T
        k0 = np.linalg.norm(q / rr, axis=1)
        k1 = np.linalg.norm(q / (rr * rr), axis=1)
        return k0 * (k0 - 1) / np.maximum(k1, 1e-9)
    return f


def boite(c, base, demi, arrondi=0.0):
    c = np.asarray(c, float)
    m = np.asarray(base, float)
    d = np.asarray(demi, float) - arrondi

    def f(p):
        q = np.abs((p - c) @ m.T) - d
        dehors = np.linalg.norm(np.maximum(q, 0), axis=1)
        dedans = np.minimum(np.max(q, axis=1), 0)
        return dehors + dedans - arrondi
    return f


def coupe(f, point, normale):
    """Garde seulement le côté de f opposé à `normale` (intersection avec un demi-espace)."""
    point = np.asarray(point, float)
    n = np.asarray(normale, float)
    n = n / np.linalg.norm(n)
    return lambda p: np.maximum(f(p), (p - point) @ n)


def sauf(f, g):
    return lambda p: np.maximum(f(p), -g(p))


def union(*fs):
    return lambda p: np.min(np.stack([f(p) for f in fs]), axis=0)


# --- Le squelette ------------------------------------------------------------

def rot(v, axe, angle):
    """Rotation de v autour d'un axe unitaire (Rodrigues)."""
    v, axe = np.asarray(v, float), np.asarray(axe, float)
    c, s = math.cos(angle), math.sin(angle)
    return v * c + np.cross(axe, v) * s + axe * (axe @ v) * (1 - c)


class Pose:
    """Positions des articulations pour une direction, une animation et un instant."""

    def __init__(self, d, anim, i, n):
        th = d * math.pi / 4
        F = np.array([math.cos(th), math.sin(th), 0.0])
        R = np.array([-F[1], F[0], 0.0])
        Z = np.array([0.0, 0.0, 1.0])
        self.F, self.R, self.Z = F, R, Z
        t = i / n
        ph = 2 * math.pi * t
        jambe = bras = coude = penche = rebond = 0.0
        salut = 0.0
        ov = {}          # bras imposés : côté -> (angle, coude, écart)
        baisse = 0.0     # les hanches descendent (à genoux)
        pli_force = 0.0  # genoux pliés
        cuisse_a = None  # angle des cuisses imposé
        self.spin = 0.0  # roulade : tout le corps tourne
        self.lame_bras = False  # l'épée suit l'avant-bras (coup d'épée)

        def onde(a, b, k):
            return a + (b - a) * max(0.0, min(1.0, k))
        if anim == "repos":
            rebond = 0.35 * math.sin(ph)
            bras = 0.04 * math.sin(ph)
            coude = 0.12
        elif anim == "marche":
            jambe = 0.42 * math.sin(ph)
            bras = -0.42 * math.sin(ph)
            coude = 0.25
            rebond = 0.9 * abs(math.cos(ph)) - 0.45
            penche = 0.04
        elif anim == "course":
            jambe = 0.62 * math.sin(ph)
            bras = -0.75 * math.sin(ph)
            coude = 1.2
            rebond = 1.8 * abs(math.cos(ph)) - 0.6
            penche = 0.2
        elif anim == "salut":
            salut = 1.0 if 0.15 < t < 0.85 else (t / 0.15 if t <= 0.15 else (1 - t) / 0.15)
            coude = 0.12
        elif anim == "attaque":
            # On arme au-dessus de l'épaule, on frappe en diagonale, on revient.
            if t < 0.34:
                k = t / 0.34
                ov[1] = (onde(0.2, 2.7, k), onde(0.3, 0.9, k), 0.25)
                penche = -0.08 * k
            elif t < 0.67:
                k = (t - 0.34) / 0.33
                ov[1] = (onde(2.7, 0.15, k), onde(0.9, 0.05, k), onde(0.25, -0.15, k))
                penche = onde(-0.08, 0.22, k)
            else:
                k = (t - 0.67) / 0.33
                ov[1] = (onde(0.15, 0.2, k), onde(0.05, 0.3, k), onde(-0.15, 0.1, k))
                penche = onde(0.22, 0.04, k)
            ov[-1] = (0.5, 1.2, 0.1)
            self.lame_bras = True
        elif anim == "bouclier":
            k = math.sin(math.pi * min(1.0, t / 0.8))
            ov[-1] = (onde(0.9, 1.55, k), onde(1.3, 0.2, k), onde(0.15, -0.1, k))
            ov[1] = (onde(0.1, -0.4, k), 0.4, 0.15)
            penche = 0.3 * k
            jambe = 0.25 * k
        elif anim == "garde":
            ov[-1] = (1.2, 1.4, -0.05)
            ov[1] = (0.2, 0.5, 0.15)
            penche = 0.1
            rebond = -0.6 + 0.2 * math.sin(ph)
        elif anim == "incante":
            ov[-1] = ov[1] = (1.0 + 0.08 * math.sin(ph), 1.0, 0.32)
            rebond = 0.3 * math.sin(ph)
        elif anim == "sort":
            k = math.sin(math.pi * t)
            ov[-1] = ov[1] = (onde(1.0, 2.1, k), onde(1.0, 0.1, k), 0.3)
            penche = -0.1 * k
        elif anim == "roulade":
            self.spin = 2 * math.pi * (t + 0.5 / n)   # un tour complet
            k = math.sin(math.pi * min(1.0, (t + 0.5 / n)))
            cuisse_a = 1.6 * k
            pli_force = 2.2 * k
            baisse = 9 * k
            ov[-1] = ov[1] = (onde(0.2, 0.9, k), onde(0.3, 1.8, k), 0.1)
            penche = 0.6 * k
        elif anim == "mort":
            k1 = min(1.0, t / 0.45)
            k2 = max(0.0, min(1.0, (t - 0.3) / 0.6))
            baisse = 11.5 * k1
            cuisse_a = 0.0
            pli_force = math.pi / 2 * k1
            penche = -1.35 * k2 * k2 + 0.2 * k1 * (1 - k2)
            ov[-1] = (onde(0.1, -0.5, k2), 0.3, onde(0.15, 0.7, k2))
            ov[1] = (onde(0.1, -0.5, k2), 0.3, onde(0.15, 0.7, k2))
        self.t, self.anim = t, anim
        self.salut = salut
        self.vague = math.sin(ph * 2) * 0.45 * salut

        def L(f, r, z):  # local -> monde
            return F * f + R * r + Z * z

        self.L = L
        self.rebond = rebond
        hanche = 24.0 + rebond - baisse
        # Le buste se penche en avant autour des hanches.
        self.penche = penche
        def buste(f, r, z):
            v = L(f, r, z - hanche)
            return rot(v, R, -penche) + Z * hanche   # incline vers l'avant
        self.buste = buste
        self.hanche = hanche

        # Jambes : cuisse + tibia, balancement autour de l'axe R.
        self.jambes = []
        for cote, sens in ((-1, 1), (1, -1)):
            a = sens * jambe if cuisse_a is None else cuisse_a
            h = L(0, cote * 3.4, hanche)
            cuisse = rot(-Z * 12.0, R, -a)          # a > 0 : jambe vers l'avant
            genou = h + cuisse
            pli = max(0.0, -a) * 0.9 + (0.25 if anim == "course" else 0.0) * max(0.0, math.sin(ph + (0 if sens > 0 else math.pi)))
            pli += pli_force
            tibia = rot(-Z * 11.5, R, -a + pli)
            cheville = genou + tibia
            # Le pied posé reste au sol.
            if cheville[2] < 1.4:
                decal = 1.4 - cheville[2]
                genou = genou + Z * decal * 0.5
                cheville = cheville + Z * decal
            pied = cheville + F * 3.0 - Z * 0.4
            self.jambes.append((h, genou, cheville, pied))

        # Bras : épaule, coude, main.
        self.bras = []
        for cote, sens in ((-1, 1), (1, -1)):
            ep = buste(0, cote * 8.6, 40.5)
            a, cd, ecart = (sens * bras, coude, 0.0) if cote not in ov else ov[cote]
            haut = rot(-Z * 9.0, R, -a)
            haut = rot(haut, F, cote * (0.12 + ecart))   # bras légèrement écartés
            co = ep + haut
            avant = rot(haut / 9.0 * 8.5, R, -cd)
            if cote < 0 and salut > 0:             # bras gauche qui salue (la main droite tient l'épée)
                haut = rot(Z * 9.0, F, -cote * 0.75) * salut + rot(-Z * 9.0, F, cote * 0.12) * (1 - salut)
                haut = haut / np.linalg.norm(haut) * 9.0
                co = ep + haut
                avant = rot(Z * 8.5, F, -cote * (0.2 + self.vague)) * salut + avant * (1 - salut)
                avant = avant / np.linalg.norm(avant) * 8.5
            main = co + avant
            self.bras.append((ep, co, main, cote))

        self.cou = (buste(0, 0, 41.5), buste(0, 0, 44.5))
        self.tete = buste(0.3, 0, 48.6)
        # Repère de la tête (elle suit le buste).
        self.tF = rot(F, R, -penche)
        self.tZ = rot(Z, R, -penche)


# --- Les pièces --------------------------------------------------------------
# Chaque pièce rend une liste de (forme, matériau). Les matériaux du corps sont
# préfixés « corps: ».

def corps(p):
    F, R, Z = p.F, p.R, p.Z
    out = []
    out.append((ellipsoide(p.buste(0, 0, 33.5), [F, R, Z], [4.6, 6.6, 8.8]), "peau"))
    out.append((ellipsoide(p.buste(0, 0, 26.0), [F, R, Z], [4.4, 6.0, 4.2]), "pantalon"))   # sous-vêtement
    out.append((capsule(*p.cou, 2.3), "peau"))
    out.append((ellipsoide(p.tete, [p.tF, R, p.tZ], [7.0, 6.8, 7.6]), "peau"))
    for k in (-1, 1):   # yeux
        out.append((ellipsoide(p.tete + p.tF * 6.1 + R * k * 2.6 + p.tZ * 0.2, [p.tF, R, p.tZ], [1.2, 1.0, 1.6]), "yeux"))
    out.append((ellipsoide(p.tete + p.tF * 6.9 - p.tZ * 1.6, [p.tF, R, p.tZ], [1.2, 1.0, 1.3]), "peau"))  # nez
    for (h, g, c, pi) in p.jambes:
        out.append((capsule(h, g, 3.0), "peau"))
        out.append((capsule(g, c, 2.5), "peau"))
        out.append((ellipsoide(pi, [F, R, Z], [3.2, 1.9, 1.3]), "peau"))
    for (e, co, m, k) in p.bras:
        out.append((capsule(e, co, 2.2), "peau"))
        out.append((capsule(co, m, 1.9), "peau"))
        out.append((ellipsoide(m, [F, R, Z], [2.2, 1.8, 2.4]), "peau"))
    return out


def tunique(p):
    F, R, Z = p.F, p.R, p.Z
    out = [(ellipsoide(p.buste(0, 0, 33.8), [F, R, Z], [5.6, 7.5, 9.6]), "tissu"),
           (coupe(ellipsoide(p.buste(0, 0, 25.5), [F, R, Z], [6.2, 7.6, 7.5]), p.buste(0, 0, 18.5), -Z), "tissu"),
           (ellipsoide(p.buste(0, 0, 25.6), [F, R, Z], [5.5, 7.3, 1.2]), "ceinture"),
           (ellipsoide(p.buste(5.4, 0, 25.6), [F, R, Z], [0.8, 1.4, 1.3]), "boucle")]
    for (e, co, m, k) in p.bras:
        out.append((capsule(e, e + (co - e) * 0.75, 3.1), "tissu"))
    return out


def pantalon(p):
    out = []
    for (h, g, c, pi) in p.jambes:
        out.append((capsule(h, g, 3.7), "pantalon"))
        out.append((capsule(g, g + (c - g) * 0.7, 3.2), "pantalon"))
    return out


def bottes(p):
    F, R, Z = p.F, p.R, p.Z
    out = []
    for (h, g, c, pi) in p.jambes:
        out.append((capsule(g + (c - g) * 0.45, c, 3.4), "bottes"))
        out.append((ellipsoide(pi, [F, R, Z], [3.9, 2.6, 1.9]), "bottes"))
    return out


def gants(p):
    F, R, Z = p.F, p.R, p.Z
    out = []
    for (e, co, m, k) in p.bras:
        out.append((capsule(co + (m - co) * 0.55, m, 2.7), "gants"))
        out.append((ellipsoide(m, [F, R, Z], [2.8, 2.4, 3.0]), "gants"))
    return out


def cape(p):
    F, R, Z = p.F, p.R, p.Z
    vent = {"repos": 0.08, "marche": 0.3, "course": 0.7, "salut": 0.1, "attaque": 0.25, "bouclier": 0.3,
            "roulade": 0.5, "mort": 0.05}.get(p.anim, 0.12)
    vent += 0.05 * math.sin(2 * math.pi * p.t * 2)
    haut = p.buste(0.4, 0, 41.5)
    # Repère de la cape : elle descend en s'évasant et le vent la pousse vers l'arrière.
    bas = rot(-Z, R, vent + p.penche)
    av = np.cross(bas, R)            # vers l'avant du héros, perpendiculaire à la cape
    av = -av if av @ F < 0 else av

    def tissu(q):
        d = q - haut
        h = d @ bas                   # distance descendue depuis les épaules
        qf, qr = d @ av, d @ R
        rf = 5.4 + 0.10 * np.maximum(h, 0)
        rr = 8.6 + 0.16 * np.maximum(h, 0)
        k = np.sqrt((qf / rf) ** 2 + (qr / rr) ** 2)
        coque = np.abs((k - 1) * np.minimum(rf, rr)) - 0.65
        coque = np.maximum(coque, qf + 0.8)            # seulement dans le dos
        coque = np.maximum(coque, -h - 0.5)            # pas au-dessus des épaules
        return np.maximum(coque, h - 28.0)             # longueur
    return [(tissu, "cape"), (capsule(p.buste(-0.5, -7.2, 41.6), p.buste(-0.5, 7.2, 41.6), 1.8), "cape")]


def casque(p):
    R = p.R
    tF, tZ = p.tF, p.tZ
    dome = ellipsoide(p.tete + tZ * 0.6, [tF, R, tZ], [8.4, 8.0, 8.4])
    dome = coupe(dome, p.tete - tZ * 1.0, -tZ)
    nasal = boite(p.tete + tF * 8.0 + tZ * 0.8, [tF, R, tZ], [0.7, 0.8, 3.2], 0.3)
    crete = boite(p.tete + tZ * 8.4, [tF, R, tZ], [6.0, 0.8, 1.2], 0.6)
    return [(sauf(dome, boite(p.tete + tF * 7.0 - tZ * 1.0, [tF, R, tZ], [3.0, 5.0, 3.6])), "casque"),
            (nasal, "casque"), (crete, "garde")]


def cheveux_court(p):
    R = p.R
    tF, tZ = p.tF, p.tZ
    calotte = ellipsoide(p.tete + tZ * 1.2 - tF * 0.6, [tF, R, tZ], [7.6, 7.4, 7.6])
    calotte = sauf(calotte, boite(p.tete + tF * 7.5 - tZ * 2.2, [tF, R, tZ], [4.0, 9.0, 4.6]))
    meche = ellipsoide(p.tete + tF * 5.2 + tZ * 4.4 + R * 1.6, [tF, R, tZ], [2.4, 3.6, 1.6])
    return [(calotte, "cheveux"), (meche, "cheveux")]


def cheveux_long(p):
    R = p.R
    tF, tZ = p.tF, p.tZ
    out = cheveux_court(p)
    dos = ellipsoide(p.tete - tF * 4.8 - tZ * 5.5, [tF, R, tZ], [2.6, 6.6, 9.0])
    out.append((dos, "cheveux"))
    return out


# --- Les pièces des habitants -----------------------------------------------

def robe(p):
    """Une robe longue (Sœur Clémence, Aëlis…) : le haut de la tunique, et une
    jupe en cloche jusqu'aux chevilles."""
    F, R, Z = p.F, p.R, p.Z
    out = tunique(p)[:1] + tunique(p)[2:]
    jupe = coupe(ellipsoide(p.L(0, 0, 13.0 + p.rebond * 0.5), [F, R, Z], [8.2, 9.4, 14.5]), p.L(0, 0, 2.2), -Z)
    jupe = coupe(jupe, p.L(0, 0, 27.0 + p.rebond), Z)
    out.append((jupe, "tissu"))
    return out


def tablier(p):
    F, R, Z = p.F, p.R, p.Z
    pan = boite(p.buste(5.6, 0, 19.5), [F, R, Z], [0.7, 5.4, 7.0], 0.4)
    bavette = boite(p.buste(5.4, 0, 31.5), [F, R, Z], [0.6, 3.8, 5.0], 0.4)
    return [(pan, "tablier"), (bavette, "tablier")]


def barbe(p):
    R = p.R
    tF, tZ = p.tF, p.tZ
    menton = ellipsoide(p.tete + tF * 4.6 - tZ * 4.4, [tF, R, tZ], [3.4, 5.4, 4.4])
    moustache = ellipsoide(p.tete + tF * 6.6 - tZ * 2.6, [tF, R, tZ], [1.2, 3.4, 1.0])
    return [(menton, "cheveux"), (moustache, "cheveux")]


def chapeau(p):
    R = p.R
    tF, tZ = p.tF, p.tZ
    bord = ellipsoide(p.tete + tZ * 4.8, [tF, R, tZ], [10.6, 10.6, 1.0])
    calotte = coupe(ellipsoide(p.tete + tZ * 4.2, [tF, R, tZ], [7.2, 7.0, 7.6]), p.tete + tZ * 4.2, -tZ)
    ruban = coupe(coupe(ellipsoide(p.tete + tZ * 4.2, [tF, R, tZ], [7.4, 7.2, 7.8]), p.tete + tZ * 5.2, -tZ), p.tete + tZ * 6.6, tZ)
    return [(bord, "chapeau"), (calotte, "chapeau"), (ruban, "ceinture")]


def epee(p):
    e, co, m, k = p.bras[1]       # main droite
    F, R, Z = p.F, p.R, p.Z
    av = (m - co) / np.linalg.norm(m - co)
    # L'épée pointe vers l'avant et le bas, en suivant un peu le mouvement du bras.
    lame_dir = F * 0.55 - Z * 0.85 + av * 0.35
    if p.lame_bras:   # coup d'épée : la lame prolonge l'avant-bras
        lame_dir = av * 1.0 + F * 0.25
    lame_dir = lame_dir / np.linalg.norm(lame_dir)
    lat = np.cross(lame_dir, R)
    lat /= np.linalg.norm(lat)
    base = m + lame_dir * 1.5
    lame = boite(base + lame_dir * 10.5, [lame_dir, R, lat], [9.5, 0.45, 1.25], 0.2)
    pointe = capsule(base + lame_dir * 19.0, base + lame_dir * 21.0, 0.6)
    garde = boite(base, [lame_dir, R, lat], [0.7, 0.9, 3.4], 0.3)
    poignee = capsule(m - lame_dir * 2.5, m + lame_dir * 0.5, 1.0)
    pommeau = ellipsoide(m - lame_dir * 3.2, [F, R, Z], [1.3, 1.3, 1.3])
    return [(union(lame, pointe), "lame"), (garde, "garde"), (poignee, "poignee"), (pommeau, "garde")]


def bouclier(p):
    """L'écu du Paladin, au bras gauche : un disque bombé avec un umbo doré."""
    e, co, m, k = p.bras[0]       # bras gauche
    F, R, Z = p.F, p.R, p.Z
    av = (m - co) / np.linalg.norm(m - co)
    # L'écu se tient à plat contre l'avant-bras, tourné vers l'extérieur.
    dehors = np.cross(av, Z) * -k
    if np.linalg.norm(dehors) < 0.3:
        dehors = -R * k
    dehors = dehors - av * (dehors @ av)
    dehors /= np.linalg.norm(dehors)
    if p.anim in ("garde", "bouclier"):
        # Bouclier levé : face à l'ennemi, bien droit devant la poitrine.
        dehors, av = F, Z
    lat = np.cross(dehors, av)
    lat /= np.linalg.norm(lat)
    av = np.cross(lat, dehors)
    c = co + (m - co) * 0.55 + dehors * 2.6
    if p.anim in ("garde", "bouclier"):
        c = m + dehors * 2.4 + R * 1.5 + Z * 1.0
    disque = ellipsoide(c, [dehors, av, lat], [1.4, 8.2, 7.2])
    umbo = ellipsoide(c + dehors * 1.2, [dehors, av, lat], [1.3, 2.0, 2.0])
    return [(disque, "bouclier"), (umbo, "umbo")]


# Ordre d'empilement dans le jeu, et ce qui cache chaque pièce.
# « enveloppe » : épaisseur ajoutée au corps pour masquer les pièces portées par-dessus les vêtements.
PIECES = {
    "corps":         {"f": None,          "enveloppe": 0.0},
    "pantalon":      {"f": pantalon,      "enveloppe": 0.0},
    "bottes":        {"f": bottes,        "enveloppe": 0.0},
    "tunique":       {"f": tunique,       "enveloppe": 0.0},
    "gants":         {"f": gants,         "enveloppe": 0.0},
    "cape":          {"f": cape,          "enveloppe": 1.2},
    "cheveux_court": {"f": cheveux_court, "enveloppe": 0.0},
    "cheveux_long":  {"f": cheveux_long,  "enveloppe": 0.6},
    "casque":        {"f": casque,        "enveloppe": 0.0},
    "epee":          {"f": epee,          "enveloppe": 1.0},
    "bouclier":      {"f": bouclier,      "enveloppe": 1.0},
    "robe":          {"f": robe,          "enveloppe": 0.0},
    "tablier":       {"f": tablier,       "enveloppe": 1.4},
    "barbe":         {"f": barbe,         "enveloppe": 0.0},
    "chapeau":       {"f": chapeau,       "enveloppe": 0.0},
}

# --- Le rendu ----------------------------------------------------------------

_xs, _ys = np.meshgrid(np.arange(W) + 0.5 - ANCRE[0], np.arange(H) + 0.5 - ANCRE[1])
ORIGINES = (_xs.reshape(-1, 1) * RIGHT + (-_ys.reshape(-1, 1)) * UP) - FWD * 80.0


def lancer(formes):
    """Lance un rayon par pixel. Renvoie (indice de forme touchée ou -1, profondeur, normale)."""
    n = ORIGINES.shape[0]
    t = np.zeros(n)
    actif = np.ones(n, bool)
    touche = np.full(n, -1)
    for _ in range(110):
        idx = np.nonzero(actif)[0]
        if idx.size == 0:
            break
        p = ORIGINES[idx] + FWD * t[idx, None]
        ds = np.stack([f(p) for f, _ in formes])
        d = ds.min(axis=0)
        ok = d < 0.04
        touche[idx[ok]] = ds[:, ok].argmin(axis=0)
        actif[idx[ok]] = False
        t[idx] += np.maximum(d, 0.02) * 0.9
        loin = t[idx] > 160
        actif[idx[loin]] = False
    normales = np.zeros((n, 3))
    hit = np.nonzero(touche >= 0)[0]
    if hit.size:
        p = ORIGINES[hit] + FWD * t[hit, None]
        e = 0.05
        g = []
        for ax in np.eye(3):
            dp = np.stack([f(p + ax * e) for f, _ in formes]).min(axis=0)
            dm = np.stack([f(p - ax * e) for f, _ in formes]).min(axis=0)
            g.append(dp - dm)
        g = np.stack(g, axis=1)
        normales[hit] = g / np.maximum(np.linalg.norm(g, axis=1, keepdims=True), 1e-9)
    return touche, t, normales


def ton_lumiere(normales):
    """Éclairage en 4 paliers (ombre profonde → lumière)."""
    l = normales @ LUMIERE
    tons = np.digitize(l, [-0.25, 0.25, 0.72])
    return tons


def gonfle(f, e):
    return (lambda p: f(p) - e) if e else f


def tourne(f, centre, axe, angle):
    """La forme f tournée de `angle` autour de l'axe passant par `centre`."""
    c, s = math.cos(-angle), math.sin(-angle)
    k = np.asarray(axe, float)

    def g(p):
        v = p - centre
        q = v * c + np.cross(k, v) * s + np.outer(v @ k, k) * (1 - c)
        return f(q + centre)
    return g


def rendu_piece(nom, pose):
    """Renvoie (matériau, ton, profondeur) par pixel pour la pièce, déjà masquée par le corps."""
    base = corps(pose)
    e = PIECES[nom]["enveloppe"]
    if nom == "corps":
        formes = base
        garde = set(range(len(base)))
    else:
        piece = PIECES[nom]["f"](pose)
        formes = [(gonfle(f, e), m) for f, m in base] + piece
        garde = set(range(len(base), len(formes)))
    if pose.spin:
        # Le héros tourne sur lui-même et décolle un peu, pour ne pas passer sous le sol.
        centre = pose.L(0, 0, 19.0)
        lift = pose.Z * 12.0 * abs(math.sin(pose.spin / 2))
        formes = [(tourne(f, centre, pose.R, pose.spin), m) for f, m in formes]
        formes = [((lambda f: lambda p: f(p - lift))(f), m) for f, m in formes]
    touche, prof, normales = lancer(formes)
    tons = ton_lumiere(normales)
    mats = np.full(touche.shape, "", dtype=object)
    for i, (f, m) in enumerate(formes):
        sel = touche == i
        if i in garde:
            mats[sel] = m
    return mats.reshape(H, W), tons.reshape(H, W), prof.reshape(H, W), (touche >= 0).reshape(H, W)


def contours(mats, prof, plein):
    """Contour fin et sombre : autour de la pièce, sauf là où le corps passe devant."""
    m = mats != ""
    out = np.full(mats.shape, "", dtype=object)
    for dy, dx in ((0, 1), (0, -1), (1, 0), (-1, 0)):
        voisin = np.roll(np.roll(m, dy, 0), dx, 1)
        vmat = np.roll(np.roll(mats, dy, 0), dx, 1)
        vprof = np.roll(np.roll(prof, dy, 0), dx, 1)
        libre = ~m & voisin & ((~plein) | (prof > vprof + 2.5))
        out[libre & (out == "")] = vmat[libre & (out == "")]
    out[0, :] = out[-1, :] = ""
    out[:, 0] = out[:, -1] = ""
    return out


def colorie(mats, tons, cont, palette):
    img = np.zeros((H, W, 4), np.uint8)
    for mat, ramp in palette.items():
        sel = mats == mat
        if sel.any():
            for k in range(4):
                s2 = sel & (tons == k)
                img[s2, :3] = ramp[k]
                img[s2, 3] = 255
        sc = cont == mat
        img[sc, :3] = ramp[4]
        img[sc, 3] = 255
    return img


def palettes_de(nom):
    """Variantes de couleur d'une pièce : nom de variante -> {matériau: rampe}."""
    f = {k: rampe(v) for k, v in FIXES.items()}
    if nom == "corps":
        return {k: {"peau": rampe(c), "yeux": f["yeux"], "pantalon": f["pantalon"]} for k, c in PEAUX.items()}
    if nom in ("tunique", "robe"):
        return {str(i): {"tissu": rampe(c), "ceinture": f["ceinture"], "boucle": f["boucle"]}
                for i, c in enumerate(COULEURS_HEROS)}
    if nom.startswith("cheveux") or nom == "barbe":
        return {k: {"cheveux": rampe(c)} for k, c in CHEVEUX.items()}
    if nom == "epee":
        return {"": {"lame": f["lame"], "garde": f["garde"], "poignee": f["poignee"]}}
    if nom == "chapeau":
        return {k: {"chapeau": rampe(c), "ceinture": f["ceinture"]} for k, c in CHAPEAUX.items()}
    if nom == "casque":
        return {"": {"casque": f["casque"], "garde": f["garde"]}}
    if nom == "bouclier":
        return {"": {"bouclier": f["bouclier"], "umbo": f["umbo"]}}
    return {"": {nom: f[nom]}}


def forge(apercu=False, seulement=None):
    os.makedirs(SORTIE, exist_ok=True)
    total = sum(a[1] for a in ANIMS)
    meta = {"image": [W, H], "ancre": list(ANCRE), "directions": 8, "anims": {}, "pieces": {}}
    col = 0
    for nom, n, fps, boucle in ANIMS:
        meta["anims"][nom] = {"debut": col, "images": n, "fps": fps, "boucle": boucle}
        col += n
    chemin_meta = os.path.join(SORTIE, "humain.json")
    if seulement and os.path.exists(chemin_meta):
        # On ne refait que quelques pièces : on garde la liste des autres.
        with open(chemin_meta) as fh:
            meta["pieces"] = json.load(fh)["pieces"]
    for nom in PIECES:
        if seulement and nom not in seulement:
            continue
        pal = palettes_de(nom)
        planches = {v: np.zeros((8 * H, total * W, 4), np.uint8) for v in pal}
        for d in range(8):
            col = 0
            for anim, n, _, _ in ANIMS:
                for i in range(n):
                    pose = Pose(d, anim, i, n)
                    mats, tons, prof, plein = rendu_piece(nom, pose)
                    cont = contours(mats, prof, plein)
                    for v, p in pal.items():
                        planches[v][d * H:(d + 1) * H, col * W:(col + 1) * W] = colorie(mats, tons, cont, p)
                    col += 1
            print(f"  {nom} : direction {d + 1}/8", file=sys.stderr)
        fichiers = []
        for v, pl in planches.items():
            fn = f"{nom}_{v}.png" if v else f"{nom}.png"
            Image.fromarray(pl).save(os.path.join(SORTIE, fn), optimize=True)
            fichiers.append(v)
        meta["pieces"][nom] = fichiers
    meta["pieces"] = {k: meta["pieces"][k] for k in PIECES if k in meta["pieces"]}
    with open(chemin_meta, "w") as fh:
        json.dump(meta, fh, ensure_ascii=False, indent=1)
    if apercu:
        apercu_planche(meta)


def apercu_planche(meta, tenue=None, chemin=None):
    """Empile une tenue complète sur les 8 directions, pour vérifier à l'œil."""
    tenue = tenue or ["corps", "pantalon", "bottes", "tunique_6", "gants", "cape", "cheveux_court_roux", "epee", "bouclier"]
    total = sum(a["images"] for a in meta["anims"].values())
    img = Image.new("RGBA", (total * W, 8 * H), (40, 36, 52, 255))
    for p in tenue:
        img.alpha_composite(Image.open(os.path.join(SORTIE, p + ".png")))
    chemin = chemin or os.path.join(ICI, "apercu.png")
    img.resize((img.width * 2, img.height * 2), Image.NEAREST).save(chemin)
    print("Aperçu :", chemin, file=sys.stderr)


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--apercu", action="store_true")
    ap.add_argument("--pieces", nargs="*")
    a = ap.parse_args()
    forge(a.apercu, a.pieces)
