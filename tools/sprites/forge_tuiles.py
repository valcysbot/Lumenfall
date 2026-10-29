#!/usr/bin/env python3
"""La forge des tuiles : le sol d'Erydor en pixel art (losanges de 64 × 32).

Écrit client/assets/tuiles/sol.png, une planche de tuiles, et sol.json qui dit
où est chaque tuile. Les terrains qui se posent par-dessus l'herbe (chemin,
pavés, eau, herbe grise de la Flétrissure) existent en 16 versions : une par
combinaison de voisins, pour que leurs bords soient irréguliers seulement là où
ils touchent l'herbe. L'éditeur de tuiles de Godot choisit tout seul la bonne
version quand on peint (les « terrains »).

    python3 tools/sprites/forge_tuiles.py
"""
import json
import os

import numpy as np
from PIL import Image

ICI = os.path.dirname(os.path.abspath(__file__))
SORTIE = os.path.normpath(os.path.join(ICI, "..", "..", "client", "assets", "tuiles"))

TW, TH = 64, 32
COLS = 16

# Les 4 côtés d'un losange, et la case voisine de l'autre côté (dx, dy).
# Le losange de la case (x, y) a ses coins en haut (x, y), à droite (x+1, y),
# en bas (x+1, y+1) et à gauche (x, y+1).
COTES = {"ne": (0, -1), "se": (1, 0), "so": (0, 1), "no": (-1, 0)}


def hexa(h):
    return np.array([int(h[i:i + 2], 16) for i in (1, 3, 5)], float)


def bruit(seed, shape=(TH, TW)):
    return np.random.default_rng(seed).random(shape)


def losange():
    """Masque du losange, et distance (en pixels) au bord pour chaque côté."""
    ys, xs = np.mgrid[0:TH, 0:TW]
    u = (xs + 0.5 - TW / 2) / (TW / 2)
    v = (ys + 0.5 - TH / 2) / (TH / 2)
    dedans = np.abs(u) + np.abs(v) <= 1.0
    # Distance aux 4 bords, en « pixels verticaux ».
    d = {
        "ne": (1 - (u - v)) * TH / 2,
        "se": (1 - (u + v)) * TH / 2,
        "so": (1 - (-u + v)) * TH / 2,
        "no": (1 - (-u - v)) * TH / 2,
    }
    return dedans, d


DEDANS, DIST = losange()


def tons(base, n, bas=0.3, haut=0.82):
    """Trois tons autour d'une couleur, choisis par un bruit (tramage)."""
    b = hexa(base)
    sombre = b * 0.9 + np.array([-3, -1, 8])
    clair = b * 1.06 + np.array([8, 7, -4])
    out = np.where(n[..., None] < bas, sombre, np.where(n[..., None] > haut, clair, b))
    return np.clip(out, 0, 255)


def doux(n, k=2):
    """Adoucit un bruit (moyenne des voisins), pour des taches plutôt que du grain."""
    for _ in range(k):
        n = (n + np.roll(n, 1, 0) + np.roll(n, -1, 0) + np.roll(n, 1, 1) + np.roll(n, -1, 1)) / 5
    return (n - n.min()) / max(1e-9, n.max() - n.min())


def tuile_herbe(seed, base="#4c8a3f", fleurs=0, sombre=False):
    img = tons(base, doux(bruit(seed), 3), 0.22, 0.8)
    rng = np.random.default_rng(seed + 99)
    # Brins d'herbe : petits traits clairs.
    for _ in range(9):
        x, y = rng.integers(6, TW - 6), rng.integers(4, TH - 4)
        img[y, x] = hexa(base) * 1.25
        img[y - 1, x] = hexa(base) * 1.18
    couleurs = ["#f2e27a", "#e86a8a", "#9fc6ff", "#ffffff", "#c08ae8"]
    for k in range(fleurs):
        x, y = rng.integers(8, TW - 8), rng.integers(6, TH - 6)
        c = hexa(couleurs[rng.integers(len(couleurs))])
        img[y, x] = c
        img[y, x + 1] = c * 0.85
        img[y + 1, x] = hexa("#2f6a2f")
    if sombre:
        img = img * 0.78
    return img


def tuile_terre(seed):
    img = tons("#a8834f", doux(bruit(seed), 2), 0.25, 0.8)
    rng = np.random.default_rng(seed + 5)
    for _ in range(6):   # cailloux
        x, y = rng.integers(6, TW - 6), rng.integers(4, TH - 4)
        img[y, x] = hexa("#d8c49a")
        img[y, x + 1] = hexa("#8a6a40")
    return img


def tuile_paves(seed):
    ys, xs = np.mgrid[0:TH, 0:TW]
    img = tons("#a4a2ae", doux(bruit(seed), 2), 0.2, 0.85)
    # Joints : un quadrillage dans le repère du monde (les pavés suivent la grille).
    u = (xs - TW / 2) / (TW / 2)
    v = (ys - TH / 2) / (TH / 2)
    wx, wy = (u + v) * 2.0, (v - u) * 2.0
    joints = (np.abs(((wx * 2 + 0.5) % 1) - 0.5) < 0.07) | (np.abs(((wy * 2 + (np.floor(wx * 2) % 2) * 0.5) % 1) - 0.5) < 0.07)
    img[joints] = hexa("#5e5c6c")
    return img


def tuile_eau(seed, frame=0):
    ys, xs = np.mgrid[0:TH, 0:TW]
    n = bruit(seed)
    img = tons("#3a78b8", n * 0.5 + 0.25)
    vague = np.sin((xs * 0.35 + ys * 0.7 + frame * 2.1 + seed)) > 0.93
    img[vague] = hexa("#8cc4ee")
    return img


def tuile_plancher(seed):
    ys, xs = np.mgrid[0:TH, 0:TW]
    n = bruit(seed)
    img = tons("#9a6a42", n * 0.6 + 0.2)
    u = (xs - TW / 2) / (TW / 2)
    v = (ys - TH / 2) / (TH / 2)
    wy = (v - u) * 2.0
    lames = np.abs(((wy * 3) % 1) - 0.5) > 0.44
    img[lames] = hexa("#5e3c24")
    return img


def tuile_grise(seed):
    """Herbe de la Lisière Grise : les couleurs s'effacent."""
    img = tuile_herbe(seed, "#7d867a", fleurs=2)
    g = img.mean(axis=2, keepdims=True)
    return img * 0.25 + g * 0.75


def tuile_vide():
    return np.zeros((TH, TW, 3)) + hexa("#0b0a10")


def avec_bords(img, masque, seed, rebord):
    """Applique la forme du losange, et des bords irréguliers sur les côtés
    dont le voisin n'est pas du même terrain. `masque` : côtés reliés."""
    rng = np.random.default_rng(seed)
    alpha = DEDANS.copy()
    out = img.copy()
    for cote in COTES:
        if cote in masque:
            continue
        d = DIST[cote]
        # Profondeur du bord : 2 à 5 pixels, irrégulière.
        ys, xs = np.mgrid[0:TH, 0:TW]
        ondule = 2.5 + 1.6 * np.sin(xs * 0.55 + seed) + rng.random((TH, TW)) * 1.2
        alpha &= d > ondule
        lisiere = (d > ondule) & (d <= ondule + 1.2)
        if rebord is not None:
            out[lisiere & DEDANS] = out[lisiere & DEDANS] * 0.6 + hexa(rebord) * 0.4
    return out, alpha


def rgba(img, alpha):
    a = np.zeros((TH, TW, 4), np.uint8)
    a[..., :3] = np.clip(img, 0, 255).astype(np.uint8)
    a[..., 3] = np.where(alpha, 255, 0)
    return a


def forge():
    os.makedirs(SORTIE, exist_ok=True)
    tuiles = []    # (nom, image rgba, infos)

    def ajoute(nom, img, alpha=None, **infos):
        tuiles.append((nom, rgba(img, DEDANS if alpha is None else alpha), infos))

    for i in range(4):
        ajoute(f"herbe_{i}", tuile_herbe(10 + i))
    for i in range(2):
        ajoute(f"herbe_fleurie_{i}", tuile_herbe(20 + i, fleurs=5))
    for i in range(2):
        ajoute(f"sous_bois_{i}", tuile_herbe(30 + i, "#3d6e37", sombre=True))
    for i in range(2):
        ajoute(f"plancher_{i}", tuile_plancher(40 + i))
    ajoute("vide", tuile_vide(), bloque=True)

    # Terrains à bords : 16 versions chacun.
    terrains = {
        "chemin": (tuile_terre, "#6e5634", False),
        "paves": (tuile_paves, "#4a4856", False),
        "eau": (tuile_eau, "#e8f4ff", True),
        "gris": (tuile_grise, None, False),
    }
    noms = list(COTES)
    for t, (f, rebord, bloque) in terrains.items():
        for m in range(16):
            masque = [noms[k] for k in range(4) if m & (1 << k)]
            img, alpha = avec_bords(f(100 + m * 7 + len(t)), masque, m * 13 + len(t), rebord)
            ajoute(f"{t}_{m}", img, alpha, terrain=t, relie=masque, bloque=bloque)

    rows = (len(tuiles) + COLS - 1) // COLS
    planche = np.zeros((rows * TH, COLS * TW, 4), np.uint8)
    meta = {"taille": [TW, TH], "colonnes": COLS, "cotes": COTES, "tuiles": {}}
    for k, (nom, img, infos) in enumerate(tuiles):
        cx, cy = k % COLS, k // COLS
        planche[cy * TH:(cy + 1) * TH, cx * TW:(cx + 1) * TW] = img
        meta["tuiles"][nom] = dict(atlas=[cx, cy], **infos)
    Image.fromarray(planche).save(os.path.join(SORTIE, "sol.png"), optimize=True)
    with open(os.path.join(SORTIE, "sol.json"), "w") as fh:
        json.dump(meta, fh, ensure_ascii=False, indent=1)
    print(f"{len(tuiles)} tuiles écrites dans {SORTIE}")


if __name__ == "__main__":
    forge()
