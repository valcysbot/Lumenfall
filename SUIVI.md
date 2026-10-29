# 📜 Journal de quête de Lumenfall

*Le document de suivi du projet : ce qui est fait, ce qui reste, et les décisions prises. Mis à jour à chaque livraison.*

**Version actuelle :** V0.1 (en préparation) · **Acte en cours :** Acte I, les fondations

---

## 🗺️ La grande carte des actes

| Acte | Nom | État |
|---|---|---|
| I | Les fondations : dépôt, serveur, base de données, connexion Discord, héros qui bougent ensemble, serveur de test | ✱ en cours |
| II | Le monde vivant : rendu HD-2D, héros en couches 8 directions, première région, jour / nuit, météo, PNJ | ○ |
| III | Le combat : ciblage, compétences, effets, éléments, monstres, boss, mort | ○ |
| IV | Les héros : 15 races, 17 classes, spécialisations, personnalisation | ○ |
| V | Objets, métiers et économie | ○ |
| VI | Social, guildes, PvP | ○ |
| VII | Donjons, raids, quêtes et histoire | ○ |
| VIII | Familiers, montures, maison, mini-jeux, événements | ○ |
| IX | Le monde complet, les 8 langues, le site, le bot | ○ |
| X | Alpha entre amis, bêta sur itch.io, 1.0 sur Steam | ○ |

---

## ⚒️ Acte I, les fondations

### ✓ Quêtes accomplies

- ✓ **Le Conseil des Fondations** : 21 questions de cadrage, toutes répondues le 29 octobre 2026 ([archive](docs/conseil-des-fondations.html)).
- ✓ **Le coffre du royaume** : dépôt `valcysbot/lumenfall` créé et branché.
- ✓ **La forge du serveur** : Nakama 3.41 + module Go + PostgreSQL. Le monde tourne à 10 calculs par seconde, et le serveur décide de tout (le client n'envoie que des intentions).
- ✓ **La Clairière** : carte de test de 40 × 40 cases (étang, rochers, bosquets, chemin), générée par `tools/gen_clairiere.py`.
- ✓ **Les héros** : 3 emplacements par compte, nom unique dans tout Erydor (accents et majuscules ignorés pour l'unicité), 12 couleurs de pion.
- ✓ **Le sceau de Discord (côté serveur)** : connexion OAuth2, testée avec un faux Discord. On garde uniquement l'identifiant, le pseudo et l'avatar.
- ✓ **La porte de test** : connexion sans Discord, refusée partout sauf sur le serveur de test.
- ✓ **Le premier pas** : client Godot 4.7, vue 3/4 isométrique, clic pour aller + ZQSD (AZERTY et QWERTY), joystick + toucher sur téléphone, noms au-dessus des têtes, chat général avec bulles.
- ✓ **L'épreuve des 100** : test de charge avec 100 faux joueurs pendant 70 s : 100/100 connectés, tick moyen **0,51 ms** sur les 100 ms disponibles, 86 Mo de mémoire.

### ✱ Quêtes en cours

- ✱ Construire le jeu pour Windows (zip) et Android (APK).
- ✱ Recette de déploiement Docker (`deploy/`) prête, à essayer sur le VPS de test.

### ○ Quêtes à venir

- ○ **Application Discord « Lumenfall »** : à créer par Elowan sur le portail développeur de Discord (guide pas à pas fourni).
- ○ **VPS de test** (environ 5 €/mois) : attend le feu vert d'Elowan avant l'achat.
- ○ **V0.1 sur PC et Android** : Elowan + 2 ou 3 amis dans la Clairière.

### ⚠️ Points d'attention

- Le trafic sortant est d'environ 19 Ko/s par joueur avec 100 joueurs au même endroit. C'est acceptable pour l'Acte I ; à l'Acte II on n'enverra à chaque joueur que ce qui se passe près de lui.
- Ton propre héros suit le serveur avec environ 0,1 s de retard voulu, pour un mouvement fluide. Une « prédiction » locale (le héros réagit instantanément) est prévue pour V0.2.
- Discord exige une adresse HTTPS pour le retour de connexion : il faudra un nom de domaine pour le serveur de test (Caddy fournit le certificat gratuitement).

---

## ⚖️ Décisions prises

| Date | Décision | Choix |
|---|---|---|
| 29/10/2026 | Moteur du client | Godot 4 (GDScript), kit officiel Nakama |
| 29/10/2026 | Serveur | Nakama + modules Go + PostgreSQL |
| 29/10/2026 | Hébergement pendant l'Acte I | Développement en local, petit VPS de test seulement pour la V0.1 ⭐ |
| 29/10/2026 | Région du serveur | Europe (France, Allemagne ou Pays-Bas) ⭐ |
| 29/10/2026 | Rythme du monde | 10 calculs par seconde ⭐ |
| 29/10/2026 | Sauvegarde du héros | Toutes les 30 s et à la déconnexion ⭐ |
| 29/10/2026 | Application Discord | Nouvelle application « Lumenfall » ⭐ |
| 29/10/2026 | Connexion de test | Oui, seulement sur le serveur de test ⭐ |
| 29/10/2026 | Données Discord gardées | Identifiant, pseudo et avatar ⭐ |
| 29/10/2026 | Héros par compte | 3 emplacements dès maintenant ⭐ |
| 29/10/2026 | Vue | 3/4 isométrique dès l'Acte I ⭐ |
| 29/10/2026 | Allure du héros (Acte I) | Pion propre et provisoire ⭐ |
| 29/10/2026 | Carte de test | Clairière avec obstacles ⭐ |
| 29/10/2026 | Déplacement PC | Clic + ZQSD ⭐ |
| 29/10/2026 | Déplacement téléphone | Joystick + toucher ⭐ |
| 29/10/2026 | Création du héros (Acte I) | **Nom + couleur de pion** (choix d'Elowan) |
| 29/10/2026 | Voir les autres | Noms + chat général avec bulles ⭐ |
| 29/10/2026 | Test de charge | 100 faux joueurs ⭐ |
| 29/10/2026 | Livraison PC | Zip Windows sur GitHub, en privé ⭐ |
| 29/10/2026 | Livraison Android | APK à installer à la main ⭐ |
| 29/10/2026 | Testeurs de l'Acte I | **Elowan + 2 ou 3 amis dès les premières versions** (choix d'Elowan) |
| 29/10/2026 | Notes de mise à jour | Une petite note chaque jour de travail ⭐ |
| 29/10/2026 | Journal de suivi | Ce fichier, dans le dépôt ⭐ |

⭐ = recommandation de Claude suivie.

---

## 📦 Notes de version

### V0.1 · « Le premier pas » (en préparation)

- 🆕 Écran titre avec connexion Discord et connexion de test.
- 🆕 Choix du héros : 3 emplacements, nom unique, 12 couleurs.
- 🆕 La Clairière en 3/4 isométrique : arbres, rochers, étang, chemin, fleurs.
- 🆕 Les autres aventuriers en direct, avec leur nom.
- 🆕 Chat général et bulles au-dessus des têtes.
- 🆕 Commandes PC (clic, ZQSD, flèches) et téléphone (joystick, toucher).
- 🆕 Braise donne des astuces sur les écrans de menu.

Captures : [écran titre](docs/captures/v0.1/titre.png) · [héros](docs/captures/v0.1/heros.png) · [création](docs/captures/v0.1/creation.png) · [la Clairière](docs/captures/v0.1/clairiere.png)
