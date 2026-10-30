#!/usr/bin/env python3
"""Le premier jet du Val de Glennmor (bible du monde : forêts et prairies,
une rivière, Bourg-Lanterne au centre, la Lisière Grise à l'est).

Écrit tools/maps/glennmor.plan.json : le sol case par case, les décors, les
portes et le point d'apparition. Ensuite, client/outils/construire_carte.gd
transforme ce plan en scène Godot (client/cartes/glennmor.tscn), qu'on peut
retoucher à la souris dans l'éditeur de Godot.

    python3 tools/maps/gen_glennmor.py
"""
import json
import math
import os

import numpy as np

ICI = os.path.dirname(os.path.abspath(__file__))
DECORS = json.load(open(os.path.join(ICI, "..", "..", "client", "assets", "decors", "decors.json")))

W, H = 200, 200          # le val
H_TOTAL = H + 64         # + les intérieurs, cachés loin au sud, au-delà de la forêt
INTERIEURS = H + 30      # première ligne des intérieurs
rng = np.random.default_rng(1247)

# Sol : '.' herbe, ',' herbe fleurie, ':' sous-bois, 'p' chemin, 'c' pavés,
# 'W' eau, 'b' pont, 'g' herbe grise, 'f' plancher, 'v' vide.
sol = np.full((H_TOTAL, W), ".", dtype="<U1")
pris = np.zeros((H_TOTAL, W), bool)      # cases occupées par un décor
decors = []
portes = []


def bruit(x, y, s):
    return (math.sin(x * 0.11 + s) + math.sin(y * 0.13 + s * 1.7) + math.sin((x + y) * 0.07 + s * 0.3)) / 3


def libre(x, y, w=1, h=1, sols=".,:gc", passable=False):
    if x < 0 or y < 0 or x + w > W or y + h > H_TOTAL:
        return False
    zone = sol[y:y + h, x:x + w]
    return (passable or not pris[y:y + h, x:x + w].any()) and all(c in sols for c in zone.ravel())


def pose(nom, x, y, force=False, sols=".,:gc"):
    """Pose un décor dont l'emprise commence à la case (x, y)."""
    ew, eh = DECORS[nom]["emprise"]
    passable = DECORS[nom].get("passable", False)
    if not force and not libre(x, y, ew, eh, sols, passable):
        return False
    if not passable:
        pris[y:y + eh, x:x + ew] = True
    decors.append({"d": nom, "x": int(x), "y": int(y)})
    return True


def trace(points, c, largeur=1):
    """Trace un chemin (liste de points) de `largeur` cases."""
    for (x0, y0), (x1, y1) in zip(points, points[1:]):
        n = int(max(abs(x1 - x0), abs(y1 - y0)) * 2) + 1
        for i in range(n + 1):
            t = i / n
            x, y = x0 + (x1 - x0) * t, y0 + (y1 - y0) * t
            for dx in range(-largeur // 2 + 1, largeur // 2 + 1):
                for dy in range(-largeur // 2 + 1, largeur // 2 + 1):
                    xi, yi = int(round(x)) + dx, int(round(y)) + dy
                    if 0 <= xi < W and 0 <= yi < H and sol[yi, xi] != "W":
                        sol[yi, xi] = c
                    elif 0 <= xi < W and 0 <= yi < H and sol[yi, xi] == "W":
                        sol[yi, xi] = "b"


# --- Le paysage ------------------------------------------------------------------

for y in range(H):
    for x in range(W):
        n = bruit(x, y, 1.0)
        if n > 0.45 and rng.random() < 0.5:
            sol[y, x] = ","
        if x > 168 + 6 * math.sin(y * 0.09):
            sol[y, x] = "g"            # la Lisière Grise
bord = 9
for y in range(H):
    for x in range(W):
        d = min(x, y, W - 1 - x, H - 1 - y)
        if d < bord + 3 * bruit(x, y, 4.0):
            sol[y, x] = ":"
        if ((x - 28) / 40) ** 2 + ((y - 32) / 38) ** 2 < 1 + 0.5 * bruit(x, y, 2.0):
            sol[y, x] = ":"            # la forêt du nord-ouest

# La rivière, du nord au sud, à l'est du village.
for y in range(H):
    cx = 128 + 9 * math.sin(y / 23) + 4 * math.sin(y / 9)
    larg = 2.2 + 0.8 * math.sin(y / 13)
    for x in range(int(cx - larg), int(cx + larg) + 1):
        sol[y, x] = "W"

# Bourg-Lanterne : la place pavée.
PX, PY = 95, 100
for y in range(PY - 10, PY + 11):
    for x in range(PX - 12, PX + 13):
        if ((x - PX) / 12.5) ** 2 + ((y - PY) / 10.5) ** 2 <= 1:
            sol[y, x] = "c"

# Les chemins.
trace([(PX - 12, PY), (70, 101), (50, 98), (30, 103), (8, 102)], "p", 2)           # ouest
trace([(PX, PY - 10), (96, 80), (93, 60), (97, 42), (114, 36), (121, 36)], "p", 2)  # nord, vers le moulin
trace([(PX + 12, PY), (115, 100), (126, 100), (140, 101), (160, 98), (190, 99)], "p", 2)  # est, le pont
trace([(PX, PY + 10), (96, 125), (100, 145), (99, 175), (101, 192)], "p", 2)        # sud, les fermes

# --- Le village -----------------------------------------------------------------

def pose_ou_stop(nom, x, y):
    """Pose un bâtiment à (x, y), ou tout près si la place est prise."""
    essais = sorted(((dx, dy) for dx in range(-5, 6) for dy in range(-5, 6)), key=lambda d: abs(d[0]) + abs(d[1]))
    for dx, dy in essais:
        if pose(nom, x + dx, y + dy):
            return x + dx, y + dy
    raise SystemExit(f"{nom} impossible à poser près de {x},{y}")


pose("puits", PX - 1, PY - 1, force=True)
pose_ou_stop("auberge", PX - 9, PY - 17)               # La Chope Pleine, au nord de la place
pose_ou_stop("temple", PX + 14, PY - 8)                # temple de Solenn, à l'est
pose_ou_stop("forge", PX - 21, PY - 6)                 # Doran, le forgeron, à l'ouest
maisons = [(PX - 18, PY + 5), (PX + 4, PY + 12), (PX - 12, PY + 13), (PX - 20, PY - 16),
           (PX + 10, PY - 20), (PX + 13, PY + 7), (PX - 30, PY + 4), (PX + 22, PY - 13)]
for i, (x, y) in enumerate(maisons):
    maisons[i] = pose_ou_stop(f"maison_{i % 3}", x, y)
for i, (x, y) in enumerate([(PX - 7, PY + 5), (PX - 3, PY + 6), (PX + 2, PY + 5), (PX + 6, PY + 4)]):
    pose(f"etal_{i % 3}", x, y, sols="c")
for a in range(10):
    t = a / 10 * 2 * math.pi
    pose("lanterne", int(PX + 13.5 * math.cos(t)), int(PY + 11.5 * math.sin(t)))
pose("panneau", PX + 13, PY + 2)
for x, y in [(PX - 18, PY - 4), (PX - 17, PY - 4), (PX + 2, PY - 15), (PX - 5, PY - 12)]:
    pose("tonneau", x, y)
for x, y in [(PX - 19, PY - 3), (PX + 3, PY - 14)]:
    pose("caisse", x, y)

# Lanternes le long des chemins, tous les 8 pas environ.
for y in range(H):
    for x in range(W):
        if sol[y, x] == "p" and (x * 7 + y * 3) % 61 == 0 and abs(x - PX) + abs(y - PY) < 70:
            for dx, dy in ((1, 0), (0, 1), (-1, 0), (0, -1)):
                if sol[y + dy * 2, x + dx * 2] in ".,":
                    pose("lanterne", x + dx * 2, y + dy * 2)
                    break

# Le moulin, au bord de la rivière.
pose("moulin", 115, 30)

# Les fermes au sud : meules de foin, et le pré de Gwenn avec ses moutons.
for x, y in [(84, 140), (87, 142), (82, 146), (110, 132), (113, 135)]:
    pose("meule", x, y)
P0X, P0Y, PW, PH = 104, 150, 14, 10
for x in range(P0X, P0X + PW):
    pose("cloture_0", x, P0Y)
    if x != P0X + 6:
        pose("cloture_0", x, P0Y + PH)
for y in range(P0Y + 1, P0Y + PH):
    pose("cloture_1", P0X, y)
    pose("cloture_1", P0X + PW - 1, y)
for i in range(7):
    pose("mouton", P0X + 2 + (i * 5) % 10, P0Y + 2 + (i * 3) % 7)
pose("maison_1", 84, 152)                              # la ferme de Gwenn
# Le pré et sa barrière restent dégagés : pas d'arbre qui pousse au milieu.
pris[P0Y + 1:P0Y + PH, P0X + 1:P0X + PW - 1] = True
pris[P0Y + PH:P0Y + PH + 2, P0X + 6] = True

# --- La nature : forêts, bosquets, rochers ------------------------------------------

arbres_foret = ["chene_0", "chene_1", "chene_2", "pin_0", "pin_1"]
for y in range(H):
    for x in range(W):
        c = sol[y, x]
        r = rng.random()
        if c == ":":
            d = min(x, y, W - 1 - x, H - 1 - y)
            dens = 0.62 if d < 6 else 0.3
            if r < dens:
                pose(arbres_foret[rng.integers(len(arbres_foret))], x, y)
            elif r < dens + 0.05:
                pose("buisson_%d" % rng.integers(2), x, y)
        elif c in ".,":
            loin = abs(x - PX) > 22 or abs(y - PY) > 20
            if loin and bruit(x, y, 7.0) > 0.35 and r < 0.22:
                pose(["chene_0", "chene_1", "bouleau", "chene_2"][rng.integers(4)], x, y)
            elif loin and r < 0.012:
                pose("rocher", x, y)
            elif loin and r < 0.02:
                pose("buisson_%d" % rng.integers(2), x, y)
            elif loin and r < 0.023:
                pose("souche", x, y)
        elif c == "g":
            if r < 0.07:
                pose(["pin_0", "bouleau", "souche", "rocher"][rng.integers(4)], x, y)

# --- Les intérieurs (au sud, hors du val) -----------------------------------------

sol[H:, :] = "v"


def interieur(x0, y0, w, h, meubles, porte_dehors):
    """Une pièce : plancher, meubles, et une porte reliée à la porte de dehors."""
    sol[y0:y0 + h, x0:x0 + w] = "f"
    # Les murs du fond (côtés nord-est et nord-ouest de la pièce).
    for x in range(x0, x0 + w):
        pose("mur_0", x, y0 - 1, sols="v")
    for y in range(y0, y0 + h):
        pose("mur_1", x0 - 1, y, sols="v")
    for nom, dx, dy in meubles:
        pose(nom, x0 + dx, y0 + dy, sols="f")
    sortie = (x0 + w // 2, y0 + h - 1)            # case de sortie, au bas de la pièce
    pose("tapis", sortie[0], sortie[1], sols="f")
    dehors_x, dehors_y = porte_dehors
    portes.append({"de": [sortie[0], sortie[1]], "vers": [dehors_x + 0.5, dehors_y + 1.5]})
    portes.append({"de": [dehors_x, dehors_y], "vers": [sortie[0] + 0.5, sortie[1] - 0.5]})


def porte_de(nom_idx):
    """Case devant la porte du décor n° nom_idx."""
    d = decors[nom_idx]
    m = DECORS[d["d"]]
    ew, eh = m["emprise"]
    px, py = m["porte"]
    return int(math.floor(d["x"] + ew / 2 + px)), int(math.floor(d["y"] + eh / 2 + py))


def trouve(nom, x=None, y=None):
    for i, d in enumerate(decors):
        if d["d"] == nom and (x is None or (d["x"], d["y"]) == (x, y)):
            return i
    raise KeyError(nom)


interieur(10, INTERIEURS, 14, 9, [("comptoir", 2, 1), ("etagere", 1, 0), ("etagere", 6, 0), ("cheminee", 10, 0),
                            ("table", 3, 5), ("table", 7, 4), ("table", 10, 6), ("tonneau", 12, 2), ("tonneau", 0, 3)],
          porte_de(trouve("auberge")))
interieur(60, INTERIEURS, 9, 7, [("lit", 0, 1), ("table", 4, 3), ("cheminee", 5, 0), ("etagere", 3, 0)],
          porte_de(trouve("maison_0", *maisons[0])))
interieur(110, INTERIEURS, 9, 7, [("lit", 7, 1), ("table", 3, 3), ("etagere", 1, 0), ("tonneau", 0, 4)],
          porte_de(trouve("maison_1", *maisons[1])))

# Les cases de porte, dehors, doivent rester libres et sur un sol praticable.
for p in portes:
    x, y = p["de"]
    if pris[y, x] or sol[y, x] in "Wv":
        raise SystemExit(f"porte bloquée en {x},{y}")
    if sol[y, x] in ".,:":
        sol[y, x] = "p"

plan = {
    "id": "glennmor", "nom": "Le Val de Glennmor",
    "largeur": W, "hauteur": H_TOTAL,
    "apparition": [PX + 0.5, PY + 4.5],
    "interieurs": INTERIEURS,
    "sol": ["".join(r) for r in sol],
    "decors": decors, "portes": portes,
}
with open(os.path.join(ICI, "glennmor.plan.json"), "w") as fh:
    json.dump(plan, fh, ensure_ascii=False)
print(f"{len(decors)} décors, {len(portes) // 2} portes")
