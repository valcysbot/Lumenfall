#!/usr/bin/env python3
"""La forge des décors : arbres, rochers, maisons, auberge, lanternes…

Même méthode que la forge des héros : chaque décor est une petite maquette 3D
photographiée dans la vue 3/4 du jeu, avec le même éclairage, les mêmes
contours sombres et les mêmes couleurs vives. Une case du sol mesure
CASE unités de côté.

Écrit client/assets/decors/<nom>.png et decors.json (ancre, emprise au sol en
cases, lumière éventuelle).

    python3 tools/sprites/forge_decors.py
"""
import json
import math
import os
import sys

import numpy as np
from PIL import Image

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from forge_heros import (FWD, RIGHT, UP, LUMIERE, rampe, capsule, ellipsoide,  # noqa: E402
                         boite, coupe, union, contours)

ICI = os.path.dirname(os.path.abspath(__file__))
SORTIE = os.path.normpath(os.path.join(ICI, "..", "..", "client", "assets", "decors"))
CASE = 64 / math.sqrt(2)          # une case = 45,25 unités (64 pixels de large à l'écran)
X, Y, Z = np.eye(3)
BASE = [X, Y, Z]

COULEURS = {
    "tronc": "#7a5236", "feuille": "#4f9a3c", "feuille2": "#3f8a4a", "pin": "#2f7a52",
    "bouleau": "#e8e2d4", "roche": "#8e8c9c", "mousse": "#6aa04a",
    "mur": "#efe0c4", "colombage": "#6a4228", "toit": "#c0533a", "toit2": "#4a6ab8",
    "toit3": "#8a5aa8", "pierre": "#b0aab8", "pierre2": "#8c8698", "fenetre": "#3a3450",
    "vitre": "#ffd27a", "porte": "#7a4a2a", "or": "#e8c050", "bois": "#9a6a42",
    "fer": "#5a5866", "flamme": "#ffcf5a", "toile": "#e8d8b0", "toile2": "#d0503a",
    "toile3": "#3a8ac0", "eau": "#3a78b8", "paille": "#e0c070", "laine": "#f4f0e8",
    "braise": "#ff7a30", "enseigne": "#2a2238",
}
RAMPES = {k: rampe(v) for k, v in COULEURS.items()}


def lancer(formes, origines, pas=180, limite=900.0):
    n = origines.shape[0]
    t = np.zeros(n)
    actif = np.ones(n, bool)
    touche = np.full(n, -1)
    for _ in range(pas):
        idx = np.nonzero(actif)[0]
        if idx.size == 0:
            break
        p = origines[idx] + FWD * t[idx, None]
        ds = np.stack([f(p) for f, _ in formes])
        d = ds.min(axis=0)
        ok = d < 0.06
        touche[idx[ok]] = ds[:, ok].argmin(axis=0)
        actif[idx[ok]] = False
        t[idx] += np.maximum(d * 0.8, 0.05)
        actif[idx[t[idx] > limite]] = False
    normales = np.zeros((n, 3))
    hit = np.nonzero(touche >= 0)[0]
    if hit.size:
        p = origines[hit] + FWD * t[hit, None]
        g = []
        for ax in np.eye(3):
            dp = np.stack([f(p + ax * 0.08) for f, _ in formes]).min(axis=0)
            dm = np.stack([f(p - ax * 0.08) for f, _ in formes]).min(axis=0)
            g.append(dp - dm)
        g = np.stack(g, axis=1)
        normales[hit] = g / np.maximum(np.linalg.norm(g, axis=1, keepdims=True), 1e-9)
    return touche, t, normales, origines + FWD * t[:, None]


def photo(formes, w, h, ancre):
    """Rend les formes ; le point (0, 0, 0) tombe sur le pixel `ancre`."""
    xs, ys = np.meshgrid(np.arange(w) + 0.5 - ancre[0], np.arange(h) + 0.5 - ancre[1])
    orig = xs.reshape(-1, 1) * RIGHT + (-ys.reshape(-1, 1)) * UP - FWD * 450.0
    touche, prof, normales, pos = lancer(formes, orig)
    l = normales @ LUMIERE
    tons = np.digitize(l, [-0.25, 0.25, 0.72]).reshape(h, w)
    mats = np.full(touche.shape, "", dtype=object)
    for i, (_, m) in enumerate(formes):
        mats[touche == i] = m
    # Motifs : rangs de tuiles sur les toits, assises de pierres, planches.
    z = pos[:, 2]
    rang = (np.mod(z, 7.0) < 1.3)
    for m, motif in (("toit", rang), ("toit2", rang), ("toit3", rang), ("paille", np.mod(z, 5.0) < 1.0),
                     ("pierre", np.mod(z, 9.0) < 1.1), ("pierre2", np.mod(z, 9.0) < 1.1)):
        sel = (mats == m) & motif & (tons.ravel() > 0)
        tons.ravel()[sel] -= 1
    mats = mats.reshape(h, w)
    prof = prof.reshape(h, w)
    plein = (touche >= 0).reshape(h, w)
    cont = contours(mats, prof, plein)
    img = np.zeros((h, w, 4), np.uint8)
    for m in set(mats.ravel()) | set(cont.ravel()):
        if not m:
            continue
        r = RAMPES[m]
        for k in range(4):
            s = (mats == m) & (tons == k)
            img[s, :3] = r[k]
            img[s, 3] = 255
        img[cont == m, :3] = r[4]
        img[cont == m, 3] = 255
    # La flamme et les vitres brillent : pas d'ombre sur elles.
    for m in ("flamme", "vitre", "braise"):
        s = mats == m
        img[s, :3] = RAMPES[m][3]
    return img


def rogne(img, ancre):
    """Retire les bords vides, en gardant l'ancre juste."""
    a = img[..., 3] > 0
    ys, xs = np.nonzero(a)
    x0, x1, y0, y1 = xs.min(), xs.max() + 1, ys.min(), ys.max() + 1
    return img[y0:y1, x0:x1], (ancre[0] - x0, ancre[1] - y0)


# --- Formes utiles -------------------------------------------------------------

def cone(c, r, h):
    """Cône de base c (sur le sol), rayon r, hauteur h."""
    c = np.asarray(c, float)
    k = r / h

    def f(p):
        q = p - c
        rr = np.hypot(q[:, 0], q[:, 1])
        cote = (rr - k * (h - q[:, 2])) / math.sqrt(1 + k * k)
        return np.maximum(np.maximum(cote, -q[:, 2]), q[:, 2] - h)
    return f


def cylindre(c, rayon, demi_h):
    """Cylindre vertical de centre c."""
    c = np.asarray(c, float)

    def f(p):
        q = p - c
        return np.maximum(np.hypot(q[:, 0], q[:, 1]) - rayon, np.abs(q[:, 2]) - demi_h)
    return f


def toit(c, dx, dy, z0, haut, long_x=True):
    """Toit à deux pans au-dessus d'un rectangle (faîtage le long de x ou de y)."""
    c = np.asarray(c, float)

    def f(p):
        q = p - c
        travers = np.abs(q[:, 1]) / dy if long_x else np.abs(q[:, 0]) / dx
        pente = (q[:, 2] - z0) - haut * (1 - travers)
        pente = pente / math.sqrt(1 + (haut / (dy if long_x else dx)) ** 2)
        bx = np.abs(q[:, 0]) - dx
        by = np.abs(q[:, 1]) - dy
        return np.maximum.reduce([pente, z0 - q[:, 2], bx, by])
    return f


def b(cx, cy, cz, hx, hy, hz, r=0.0):
    return boite([cx, cy, cz], BASE, [hx, hy, hz], r)


# --- Les décors ------------------------------------------------------------------
# Chaque décor : (formes, emprise en cases (largeur x, profondeur y), infos).

def chene(v=0):
    rng = np.random.default_rng(v)
    k = 1.6
    f = [(capsule([0, 0, 0], [2, -1, 46 * k], 6.5 * k), "tronc")]
    for i in range(7):
        a = i * 0.9 + v
        cx, cy = 18 * k * math.cos(a) * rng.uniform(0.6, 1), 18 * k * math.sin(a) * rng.uniform(0.6, 1)
        f.append((ellipsoide([cx, cy, k * (58 + rng.uniform(-6, 10))], BASE, [17 * k, 17 * k, 15 * k]), "feuille" if i % 2 else "feuille2"))
    f.append((ellipsoide([0, 0, 78 * k], BASE, [21 * k, 21 * k, 18 * k]), "feuille"))
    return f, (1, 1), {}


def pin(v=0):
    k = 1.7
    f = [(capsule([0, 0, 0], [0, 0, 30 * k], 4.5 * k), "tronc")]
    for i, (z, rr) in enumerate([(18, 24), (40, 19), (60, 14)]):
        f.append((cone([0, 0, z * k], rr * k, 34 * k), "pin"))
    return f, (1, 1), {}


def bouleau(v=0):
    k = 1.6
    f = [(capsule([0, 0, 0], [-2 * k, 1 * k, 60 * k], 3.8 * k), "bouleau"),
         (ellipsoide([-1 * k, 0, 66 * k], BASE, [14 * k, 14 * k, 20 * k]), "feuille"),
         (ellipsoide([6 * k, 4 * k, 56 * k], BASE, [9 * k, 9 * k, 9 * k]), "feuille2")]
    return f, (1, 1), {}


def buisson(v=0):
    f = [(ellipsoide([0, 0, 9], BASE, [16, 14, 11]), "feuille2"),
         (ellipsoide([7, -5, 13], BASE, [9, 9, 8]), "feuille")]
    if v == 1:
        f += [(ellipsoide([10, 6, 14], BASE, [2, 2, 2]), "toile2"), (ellipsoide([-6, 9, 12], BASE, [2, 2, 2]), "toile2")]
    return f, (1, 1), {}


def rocher(v=0):
    f = [(ellipsoide([0, 0, 6], BASE, [17, 14, 12]), "roche"),
         (ellipsoide([8, 6, 4], BASE, [9, 8, 7]), "roche"),
         (coupe(ellipsoide([-2, -2, 10], BASE, [15, 12, 9]), [0, 0, 14], [0, 0, -1]), "mousse")]
    return f, (1, 1), {}


def souche(v=0):
    return [(capsule([0, 0, 0], [0, 0, 8], 9), "tronc")], (1, 1), {}


def lanterne(v=0):
    f = [(capsule([0, 0, 0], [0, 0, 70], 2.4), "fer"),
         (b(0, 0, 71, 6, 6, 1.2), "fer"),
         (ellipsoide([0, 0, 79], BASE, [4.5, 4.5, 6.5]), "flamme"),
         (cone([0, 0, 86], 8, 8), "fer"),
         (b(0, 0, 1.5, 6, 6, 1.5, 0.5), "pierre2")]
    for sx in (-1, 1):
        for sy in (-1, 1):
            f.append((capsule([sx * 5.5, sy * 5.5, 71], [sx * 5.5, sy * 5.5, 86], 0.9), "fer"))
    return f, (1, 1), {"lumiere": [0, 0, 79]}


def cloture(v=0):
    """Barrière d'une case, le long de x (v=0) ou de y (v=1)."""
    L = CASE / 2
    if v == 0:
        f = [(b(0, 0, 9, L, 1.2, 1.3), "bois"), (b(0, 0, 17, L, 1.2, 1.3), "bois"),
             (b(-L + 2, 0, 11, 2, 2, 11), "tronc"), (b(L - 2, 0, 11, 2, 2, 11), "tronc")]
    else:
        f = [(b(0, 0, 9, 1.2, L, 1.3), "bois"), (b(0, 0, 17, 1.2, L, 1.3), "bois"),
             (b(0, -L + 2, 11, 2, 2, 11), "tronc"), (b(0, L - 2, 11, 2, 2, 11), "tronc")]
    return f, (1, 1), {}


def puits(v=0):
    f = [(cylindre([0, 0, 9], 26, 9), "pierre"),
         (cylindre([0, 0, 18], 20, 0.5), "eau"),
         (b(-20, 0, 34, 2.5, 2.5, 34), "tronc"), (b(20, 0, 34, 2.5, 2.5, 34), "tronc"),
         (toit([0, 0, 0], 28, 18, 64, 16), "toit"),
         (capsule([-18, 0, 50], [18, 0, 50], 2), "bois")]
    return f, (2, 2), {}


def maison(v=0):
    """Maison à colombages de 4 × 4 cases, porte au sud-ouest (côté +y)."""
    d = 2 * CASE - 6
    hm = 98
    toits = ["toit", "toit2", "toit3"]
    f = [(b(0, 0, hm / 2, d, d, hm / 2), "mur")]
    for sx in (-1, 1):
        for sy in (-1, 1):
            f.append((b(sx * d, sy * d, hm / 2, 3, 3, hm / 2), "colombage"))
    f.append((b(0, d, hm * 0.55, d, 2.2, 2.4), "colombage"))
    f.append((b(d, 0, hm * 0.55, 2.2, d, 2.4), "colombage"))
    # Porte (côté +y) et fenêtres.
    f.append((b(-12, d + 1, 31, 13, 2.4, 31), "porte"))
    f.append((b(-12, d + 2, 63, 15, 2, 2), "colombage"))
    for x in (30, 60):
        f.append((b(x, d + 1.6, 40, 8, 2.4, 9), "vitre"))
        f.append((b(x, d + 0.8, 40, 10.5, 1.8, 11.5), "colombage"))
    for y in (-40, 20):
        f.append((b(d + 1.6, y, 40, 2.4, 8, 9), "vitre"))
        f.append((b(d + 0.8, y, 40, 1.8, 10.5, 11.5), "colombage"))
    f.append((toit([0, 0, 0], d + 10, d + 10, hm, 58, long_x=(v % 2 == 0)), toits[v % 3]))
    f.append((b(-40, -30, hm + 50, 8, 8, 24), "pierre2"))
    return f, (4, 4), {"porte": [-12 / CASE, 2.0]}


def auberge(v=0):
    """La Chope Pleine : 6 × 5 cases, deux étages, enseigne."""
    dx, dy = 3 * CASE - 6, 2.5 * CASE - 6
    hm = 130
    f = [(b(0, 0, 30, dx, dy, 30), "pierre"),
         (b(0, 0, 30 + 50, dx - 2, dy - 2, 50), "mur")]
    for sx in (-1, 0, 1):
        f.append((b(sx * dx, dy, hm / 2, 3, 3, hm / 2), "colombage"))
    for sy in (-1, 0, 1):
        f.append((b(dx, sy * dy, hm / 2, 3, 3, hm / 2), "colombage"))
    f.append((b(0, dy, 62, dx, 2.4, 2.6), "colombage"))
    f.append((b(dx, 0, 62, 2.4, dy, 2.6), "colombage"))
    f.append((b(0, dy + 1, 30, 15, 2.4, 30), "porte"))
    for x in (-80, -45, 45, 80):
        for zz in (32, 92):
            f.append((b(x, dy + 1.6, zz, 9, 2.4, 10), "vitre"))
            f.append((b(x, dy + 0.8, zz, 11.5, 1.8, 12.5), "colombage"))
    for y in (-50, 0, 50):
        f.append((b(dx + 1.6, y, 92, 2.4, 9, 10), "vitre"))
        f.append((b(dx + 0.8, y, 92, 1.8, 11.5, 12.5), "colombage"))
    f.append((toit([0, 0, 0], dx + 12, dy + 12, hm, 70), "toit"))
    f.append((b(60, -40, hm + 60, 9, 9, 30), "pierre2"))
    # Enseigne : une chope dorée sur un panneau.
    f.append((capsule([40, dy + 4, 70], [40, dy + 22, 70], 1.5), "fer"))
    f.append((b(40, dy + 20, 56, 1.2, 9, 11, 1), "enseigne"))
    f.append((capsule([40, dy + 21.5, 52], [40, dy + 21.5, 61], 4.5), "or"))
    return f, (6, 5), {"porte": [0, 2.5]}


def temple(v=0):
    """Temple de Solenn : pierre blanche, disque solaire doré."""
    dx, dy = 2.5 * CASE - 6, 2.5 * CASE - 6
    f = [(b(0, 0, 4, dx + 8, dy + 8, 4), "pierre2"),
         (b(0, 0, 50, dx, dy, 46), "pierre")]
    for k in (-1, 1):
        f.append((capsule([k * 40, dy + 10, 8], [k * 40, dy + 10, 92], 5), "pierre"))
        f.append((capsule([dx + 10, k * 40, 8], [dx + 10, k * 40, 92], 5), "pierre"))
    f.append((b(0, dy + 1, 34, 16, 2.4, 30), "porte"))
    f.append((toit([0, 0, 0], dx + 18, dy + 18, 96, 60), "pierre2"))
    f.append((capsule([0, dy + 14, 132], [0, dy + 16, 132], 16), "or"))
    f.append((capsule([0, dy + 17, 132], [0, dy + 18, 132], 9), "flamme"))
    return f, (5, 5), {"porte": [0, 2.5]}


def forge_atelier(v=0):
    dx, dy = 2 * CASE - 6, 1.5 * CASE - 6
    f = [(b(0, 0, 36, dx, dy, 36), "pierre2"),
         (toit([0, 0, 0], dx + 8, dy + 8, 72, 34), "toit2"),
         (b(-50, -20, 110, 11, 11, 40), "pierre2"),
         (b(-50, -20, 152, 8, 8, 2), "braise"),
         (b(30, dy + 24, 14, 12, 6, 6), "fer"), (b(30, dy + 24, 5, 6, 4, 5), "fer"),
         (b(-20, dy + 1, 22, 16, 2.4, 22), "braise")]
    return f, (4, 3), {"lumiere": [-20, dy + 20, 20]}


def moulin(v=0):
    f = [(cylindre([0, 0, 50], 50, 50), "pierre"),
         (cone([0, 0, 100], 58, 50), "toit")]
    for k in range(4):
        a = k * math.pi / 2 + 0.4
        d = np.array([math.cos(a), 0, math.sin(a)])
        centre = np.array([0, 56, 110])
        f.append((boite(centre + d * 46, [d, Y, np.cross(d, Y)], [44, 1.5, 9], 1), "toile"))
    f.append((capsule([0, 50, 110], [0, 60, 110], 5), "bois"))
    f.append((b(0, 49, 20, 12, 2.4, 20), "porte"))
    return f, (4, 4), {}


def etal(v=0):
    toile = ["toile2", "toile3", "toile"][v % 3]
    f = [(b(0, 0, 14, 40, 16, 14), "bois"),
         (b(-38, 14, 30, 2, 2, 30), "tronc"), (b(38, 14, 30, 2, 2, 30), "tronc"),
         (b(-38, -14, 30, 2, 2, 30), "tronc"), (b(38, -14, 30, 2, 2, 30), "tronc"),
         (toit([0, 0, 0], 46, 22, 58, 12), toile)]
    for k in range(4):
        f.append((ellipsoide([-27 + k * 18, 4, 31], BASE, [6, 6, 4]), ["toile2", "or", "feuille", "paille"][k]))
    return f, (2, 1), {}


def tonneau(v=0):
    return [(cylindre([0, 0, 14], 11, 13), "bois"), (cylindre([0, 0, 8], 11.8, 1), "fer"),
            (cylindre([0, 0, 20], 11.8, 1), "fer")], (1, 1), {}


def caisse(v=0):
    return [(b(0, 0, 13, 13, 13, 13, 1), "bois"), (b(0, 0, 13, 13.6, 2, 13.6), "tronc")], (1, 1), {}


def panneau(v=0):
    return [(capsule([0, 0, 0], [0, 0, 44], 2.4), "tronc"),
            (b(6, 0, 40, 14, 1.6, 6, 1), "bois"), (b(-4, 2, 30, 1.6, 12, 5, 1), "bois")], (1, 1), {}


def meule(v=0):
    return [(ellipsoide([0, 0, 14], BASE, [22, 22, 22]), "paille")], (1, 1), {}


def mouton(v=0):
    return [(ellipsoide([0, 0, 16], BASE, [14, 9, 9]), "laine"),
            (ellipsoide([14, 0, 20], BASE, [5, 4, 5]), "enseigne"),
            (capsule([-8, -4, 0], [-8, -4, 9], 1.8), "enseigne"), (capsule([8, 4, 0], [8, 4, 9], 1.8), "enseigne"),
            (capsule([-8, 4, 0], [-8, 4, 9], 1.8), "enseigne"), (capsule([8, -4, 0], [8, -4, 9], 1.8), "enseigne")], (1, 1), {}


# Intérieurs.
def comptoir(v=0):
    L = CASE * 1.5
    return [(b(0, 0, 17, L, 10, 17), "bois"), (b(0, 0, 35, L + 2, 12, 2), "tronc"),
            (capsule([-20, 0, 37], [-20, 0, 45], 3.5), "or"), (capsule([10, 0, 37], [10, 0, 45], 3.5), "or")], (3, 1), {}


def table(v=0):
    return [(cylindre([0, 0, 23], 20, 1.5), "bois"), (capsule([0, 0, 0], [0, 0, 22], 3), "tronc"),
            (capsule([6, 4, 25], [6, 4, 31], 3), "or")], (1, 1), {}


def lit(v=0):
    return [(b(0, 0, 9, 18, 34, 9), "bois"), (b(0, 4, 19, 16, 28, 3, 2), "toile2"),
            (b(0, -26, 21, 14, 6, 4, 2), "laine"), (b(0, -35, 18, 18, 2, 18), "tronc")], (1, 2), {}


def cheminee(v=0):
    return [(b(0, 0, 40, 30, 12, 40), "pierre2"), (b(0, 9, 16, 16, 5, 14), "enseigne"),
            (ellipsoide([0, 15, 10], BASE, [9, 3, 8]), "flamme")], (2, 1), {"lumiere": [0, 20, 14]}


def etagere(v=0):
    f = [(b(0, 0, 40, 26, 6, 40), "bois")]
    for z in (20, 44, 68):
        f.append((b(0, 4, z, 24, 3, 1.5), "tronc"))
        f.append((b(-10, 4, z + 6, 4, 2, 5), "toile3"))
        f.append((b(8, 4, z + 6, 5, 2, 5), "toile2"))
    return f, (1, 1), {}


def mur(v=0):
    """Mur du fond d'une pièce, une case de long, le long de x (v=0) ou de y (v=1)."""
    L = CASE / 2
    hx, hy = (L, 4) if v == 0 else (4, L)
    f = [(b(0, 0, 50, hx, hy, 50), "mur"), (b(0, 0, 6, hx + 0.5, hy + 0.5, 6), "pierre2"),
         (b(0, 0, 97, hx + 0.5, hy + 0.5, 3), "colombage")]
    return f, (1, 1), {}


def tapis(v=0):
    return [(b(0, 0, 0.6, 18, 12, 0.6, 0.3), "toile2"), (b(0, 0, 0.9, 13, 7, 0.4), "or")], (1, 1), {"passable": True}


DECORS = {
    "mur": (mur, 2), "tapis": (tapis, 1),
    "chene": (chene, 3), "pin": (pin, 2), "bouleau": (bouleau, 1), "buisson": (buisson, 2),
    "rocher": (rocher, 1), "souche": (souche, 1), "lanterne": (lanterne, 1), "cloture": (cloture, 2),
    "puits": (puits, 1), "maison": (maison, 3), "auberge": (auberge, 1), "temple": (temple, 1),
    "forge": (forge_atelier, 1), "moulin": (moulin, 1), "etal": (etal, 3), "tonneau": (tonneau, 1),
    "caisse": (caisse, 1), "panneau": (panneau, 1), "meule": (meule, 1), "mouton": (mouton, 1),
    "comptoir": (comptoir, 1), "table": (table, 1), "lit": (lit, 1), "cheminee": (cheminee, 1),
    "etagere": (etagere, 1),
}


def forge(seulement=None):
    os.makedirs(SORTIE, exist_ok=True)
    chemin_meta = os.path.join(SORTIE, "decors.json")
    meta = json.load(open(chemin_meta)) if os.path.exists(chemin_meta) else {}
    for nom, (fn, variantes) in DECORS.items():
        if seulement and nom not in seulement:
            continue
        for v in range(variantes):
            formes, emprise, infos = fn(v)
            w = int(max(emprise) * 64 + 120)
            h = int(w * 1.4)
            ancre = (w // 2, int(h * 0.78))
            img = photo(formes, w, h, ancre)
            img, ancre = rogne(img, ancre)
            cle = f"{nom}_{v}" if variantes > 1 else nom
            Image.fromarray(img).save(os.path.join(SORTIE, cle + ".png"), optimize=True)
            m = {"ancre": [int(ancre[0]), int(ancre[1])], "emprise": list(emprise)}
            if "lumiere" in infos:
                lx, ly, lz = infos["lumiere"]
                p = np.array([lx, ly, lz])
                m["lumiere"] = [round(float(p @ RIGHT), 1), round(float(-(p @ UP)), 1)]
            if "porte" in infos:
                m["porte"] = infos["porte"]
            if infos.get("passable"):
                m["passable"] = True
            meta[cle] = m
            print(f"  {cle} : {img.shape[1]} × {img.shape[0]}", file=sys.stderr)
    with open(chemin_meta, "w") as fh:
        json.dump(dict(sorted(meta.items())), fh, ensure_ascii=False, indent=1)


if __name__ == "__main__":
    forge(sys.argv[1:] or None)
