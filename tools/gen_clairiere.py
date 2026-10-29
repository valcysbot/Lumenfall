"""Génère la carte de test « La Clairière » (40 x 40 cases).

Légende : . herbe, f fleurs, p chemin, T arbre, R rocher, W eau.
T, R et W bloquent le passage. Usage : python3 tools/gen_clairiere.py
"""
import json, math, random, pathlib

W = H = 40
rng = random.Random(1717)
g = [["." for _ in range(W)] for _ in range(H)]

for y in range(H):
    for x in range(W):
        d = math.hypot(x - 19.5, y - 19.5)
        # Lisière de forêt irrégulière autour de la clairière.
        if d > 17 + rng.random() * 2.2 or x in (0, W - 1) or y in (0, H - 1):
            g[y][x] = "T"
        elif rng.random() < 0.06:
            g[y][x] = "f"

# Un étang au nord-est.
for y in range(H):
    for x in range(W):
        if ((x - 27) / 4.2) ** 2 + ((y - 12) / 3.0) ** 2 < 1:
            g[y][x] = "W"

# Quelques bosquets et rochers pour tester les collisions.
for cx, cy in [(10, 12), (12, 27), (28, 28), (20, 8)]:
    for dy in range(-1, 2):
        for dx in range(-1, 2):
            if rng.random() < 0.7:
                g[cy + dy][cx + dx] = "T"
for _ in range(14):
    x, y = rng.randint(6, 33), rng.randint(6, 33)
    if math.hypot(x - 20, y - 20) > 3 and g[y][x] in ".f":
        g[y][x] = "R"

# Un chemin qui traverse la clairière d'ouest en est.
for x in range(2, W - 2):
    y = 20 + round(math.sin(x / 5) * 2)
    for yy in (y, y + 1):
        g[yy][x] = "p"

# Le centre reste dégagé pour l'apparition.
for y in range(18, 23):
    for x in range(18, 23):
        if g[y][x] not in "p":
            g[y][x] = "."

data = {
    "id": "clairiere",
    "name": "La Clairière",
    "width": W,
    "height": H,
    "spawn": [20.5, 20.5],
    "blocking": "TRW",
    "rows": ["".join(r) for r in g],
}
text = json.dumps(data, ensure_ascii=False, indent=1) + "\n"
root = pathlib.Path(__file__).resolve().parent.parent
for out in [root / "server/world/maps/clairiere.json", root / "client/data/maps/clairiere.json"]:
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(text, encoding="utf-8")
    print("écrit", out)
