# 🐉 Lumenfall, les Chroniques d'Erydor

MMORPG 2D pour PC et téléphone, à vivre entre amis dans le monde d'Erydor.

![La Clairière, V0.1](docs/captures/v0.1/clairiere.png)

- 📜 **Suivi du projet, décisions et notes de version :** [SUIVI.md](SUIVI.md)

## Organisation du dépôt

| Dossier | Contenu |
|---|---|
| `client/` | Le jeu (Godot 4.7, GDScript), avec le kit Nakama pour Godot |
| `server/` | Le module serveur en Go, chargé par Nakama : connexion Discord, héros, simulation du monde |
| `server/world/` | La simulation pure (cartes, collisions, trajets, noms), testée sans Nakama |
| `deploy/` | Recette Docker pour le serveur de test et le vrai serveur (Nakama, PostgreSQL, Caddy) |
| `tools/` | Générateur de la Clairière, serveur local, faux joueurs pour le test de charge |
| `docs/` | Captures et archives des questionnaires |

## Lancer en local

1. PostgreSQL 16 qui tourne, avec une base `nakama`.
2. Le binaire officiel de Nakama 3.41, et Go (la bonne version est choisie toute seule) :
   `NAKAMA=/chemin/vers/nakama tools/dev-server.sh`
3. Ouvrir `client/` dans Godot 4.7 et lancer. Le bouton « Connexion de test » marche sur ce serveur local.

## Tests

```sh
cd server && go test ./...                         # simulation, noms, connexion Discord
cd tools/loadbots && go run . -n 100 -duree 60s   # 100 faux joueurs
```

Captures automatiques du client (Linux, sans écran) :

```sh
xvfb-run godot --path client --rendering-driver opengl3 -- --capture=world --out=clairiere.png
```

## Secrets

Les clés et mots de passe vivent dans un fichier `.env` sur le serveur, jamais dans ce dépôt. Modèle : `deploy/.env.example`.
