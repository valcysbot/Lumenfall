#!/usr/bin/env python3
"""Les points de récolte et les ateliers du Val de Glennmor (Acte V).

Écrit server/world/maps/glennmor.metiers.json : des filons au pied des rochers,
des chênes à couper à l'orée des bois, des plantes dans les prairies ; les
filons de fer, le chardon, les arbres gris et la fleur de lune dans la Lisière
Grise, à l'est. Les ateliers sont au village, près de leurs maîtres.

Chaque point est sur une case libre, entourée de cases libres, et relié au
village à pied. Le tirage est fixe (graine), pour qu'on obtienne toujours le
même fichier.

    python3 tools/maps/gen_metiers.py
"""
import json
import math
import os
import random
from collections import deque

ICI = os.path.dirname(os.path.abspath(__file__))
RACINE = os.path.join(ICI, "..", "..")
CARTE = os.path.join(RACINE, "server", "world", "maps", "glennmor.json")
PLAN = os.path.join(ICI, "glennmor.plan.json")
SORTIE = os.path.join(RACINE, "server", "world", "maps", "glennmor.metiers.json")

carte = json.load(open(CARTE))
plan = json.load(open(PLAN))
rows = carte["rows"]
W, H = carte["width"], carte["indoors_from"]
VILLAGE = (95.5, 104.5)
rng = random.Random(5150)


def libre(x, y):
    return 0 <= x < W and 0 <= y < len(rows) and rows[y][x] != "#"


def degage(x, y):
    """Une case libre, avec ses 8 voisines libres : facile à atteindre et à cliquer."""
    return all(libre(x + dx, y + dy) for dx in (-1, 0, 1) for dy in (-1, 0, 1))


# Les cases reliées au point d'apparition, à pied (sans les portes).
sx, sy = int(carte["spawn"][0]), int(carte["spawn"][1])
relie = set([(sx, sy)])
file = deque([(sx, sy)])
while file:
    x, y = file.popleft()
    for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
        q = (x + dx, y + dy)
        if q not in relie and q[1] < H and libre(*q):
            relie.add(q)
            file.append(q)

decors = {}
for d in plan["decors"]:
    decors.setdefault(d["d"], []).append((d["x"], d["y"]))


def pres_de(sorte_decor):
    """Les cases dégagées et reliées juste à côté d'un décor de cette sorte."""
    out = set()
    for nom, pos in decors.items():
        if not nom.startswith(sorte_decor):
            continue
        for (x, y) in pos:
            for dx in range(-3, 4):
                for dy in range(-3, 4):
                    q = (x + dx, y + dy)
                    if q in relie and degage(*q) and 2 <= abs(dx) + abs(dy) <= 4:
                        out.add(q)
    return sorted(out)


def prairie():
    return sorted(q for q in relie if degage(*q) and all(degage(q[0] + dx, q[1] + dy) for dx in (-2, 2) for dy in (-2, 2)))


def zone(q, est):
    x, y = q
    if math.hypot(x - VILLAGE[0], y - VILLAGE[1]) < 30 or y < 8 or y > H - 8 or x < 8:
        return False
    return x >= 165 if est else x < 158


poses = []


def placer(sorte, n, cases, est, ecart=7.0):
    cases = [q for q in cases if zone(q, est)]
    rng.shuffle(cases)
    mis = 0
    for (x, y) in cases:
        if mis >= n:
            break
        if any(math.hypot(x - px, y - py) < ecart for (_, (px, py)) in poses):
            continue
        poses.append((sorte, (x + 0.5, y + 0.5)))
        mis += 1
    if mis < n:
        raise SystemExit(f"pas assez de place pour {sorte} ({mis}/{n})")


rochers = pres_de("rocher")
arbres = pres_de("chene") + pres_de("pin") + pres_de("bouleau")
pres = prairie()
placer("filon_cuivre", 14, rochers, False)
placer("chene", 14, arbres, False)
placer("sauge", 12, pres, False)
placer("lin", 12, pres, False)
placer("lysimaque", 8, pres, False)
placer("filon_fer", 8, rochers, True, 6)
placer("arbre_gris", 8, arbres + pres, True, 6)
placer("chardon_gris", 8, pres, True, 6)
placer("fleur_de_lune", 5, pres, True, 6)


def pres_du_point(x0, y0, interieur=False):
    """La case dégagée la plus proche d'un point (au village)."""
    best = None
    for dy in range(-6, 7):
        for dx in range(-6, 7):
            q = (int(x0) + dx, int(y0) + dy)
            ok = degage(*q) and (interieur or q in relie)
            if ok and not any(math.hypot(q[0] + 0.5 - a[0], q[1] + 0.5 - a[1]) < 2.5 for a in ateliers_pris):
                d = math.hypot(dx, dy)
                if best is None or d < best[0]:
                    best = (d, q)
    if best is None:
        raise SystemExit(f"pas de place pour un atelier près de {x0}, {y0}")
    q = best[1]
    ateliers_pris.append((q[0] + 0.5, q[1] + 0.5))
    return [q[0] + 0.5, q[1] + 0.5]


ateliers_pris = []
# Près de Doran (la forge et l'établi du joaillier), de Nessa (l'alambic),
# au marché pour la couture et le cuir (Gwenn y descend vendre sa laine),
# et dans la cuisine de La Chope Pleine, près de Mabon.
ateliers = [
    {"metier": "forge", "at": pres_du_point(78.5, 100.5)},
    {"metier": "joaillerie", "at": pres_du_point(74.5, 101.5)},
    {"metier": "alchimie", "at": pres_du_point(102.5, 72.5)},
    {"metier": "couture", "at": pres_du_point(90.5, 110.5)},
    {"metier": "cuir", "at": pres_du_point(94.5, 111.5)},
    {"metier": "cuisine", "at": pres_du_point(11.5, 232.5, interieur=True)},
]

donnees = {
    "_note": "Les points de récolte et les ateliers du Val de Glennmor (Acte V). Écrit par tools/maps/gen_metiers.py ; le serveur vérifie au démarrage que chaque point est praticable.",
    "noeuds": [{"sorte": s, "at": [x, y]} for s, (x, y) in poses],
    "ateliers": ateliers,
}
with open(SORTIE, "w") as f:
    f.write("{\n")
    f.write(f' "_note": {json.dumps(donnees["_note"], ensure_ascii=False)},\n')
    f.write(' "noeuds": [\n')
    f.write(",\n".join("  " + json.dumps(n, ensure_ascii=False) for n in donnees["noeuds"]))
    f.write("\n ],\n")
    f.write(' "ateliers": [\n')
    f.write(",\n".join("  " + json.dumps(a, ensure_ascii=False) for a in ateliers))
    f.write("\n ]\n}\n")
print(f"{len(poses)} points de récolte, {len(ateliers)} ateliers → {os.path.relpath(SORTIE, RACINE)}")
