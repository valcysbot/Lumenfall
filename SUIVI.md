# 📜 Journal de quête de Lumenfall

*Le document de suivi du projet : ce qui est fait, ce qui reste, et les décisions prises. Mis à jour à chaque livraison.*

**Version actuelle :** V0.1.2 « Habits neufs » (30 octobre 2026) · **Acte en cours :** Acte II, le monde vivant

---

## 🗺️ La grande carte des actes

| Acte | Nom | État |
|---|---|---|
| I | Les fondations : dépôt, serveur, base de données, connexion Discord, héros qui bougent ensemble, serveur de test | ✓ validé par Elowan |
| II | Le monde vivant : rendu HD-2D, héros en couches 8 directions, première région, jour / nuit, météo, PNJ | ✱ en cours |
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

- ✓ **Les builds** : Windows (zip) et Android (APK) construits par GitHub Actions à chaque changement.
- ✓ **Le serveur de test** : VPS Hostinger KVM 1 acheté par Elowan, installé par `deploy/bootstrap-vps.sh`, mis à jour automatiquement à chaque changement du serveur, avec un test de fumée (5 faux joueurs + porte Discord) après chaque déploiement.
- ✓ **Application Discord « Lumenfall »** créée par Elowan.

- ✓ **V0.1 entre amis** : Elowan + 2 ou 3 amis dans la Clairière, sur PC et Android.
- ✓ **Bousculade (V0.1.1)** : les héros ne se traversent plus.

### ⚠️ Points d'attention

- Le trafic sortant est d'environ 19 Ko/s par joueur avec 100 joueurs au même endroit. C'est acceptable pour l'Acte I ; à l'Acte II on n'enverra à chaque joueur que ce qui se passe près de lui.
- Ton propre héros suit le serveur avec environ 0,1 s de retard voulu, pour un mouvement fluide. Une « prédiction » locale (le héros réagit instantanément) est prévue pour V0.2.
- Le serveur de test utilise une adresse gratuite en `sslip.io` avec HTTPS par Caddy. Un vrai nom de domaine viendra avant l'alpha.
- L'APK Android est signé avec une clé de test : pour le Play Store, il faudra une vraie clé de signature (Acte X).

---

## 🌗 Acte II, le monde vivant

### ✓ Quêtes accomplies

- ✓ **Le Conseil du Monde vivant** : 25 questions, toutes répondues le 29 octobre 2026 ([archive](docs/conseil-du-monde-vivant.html)).
- ✓ **Le Grimoire d'Erydor** : la bible du monde, 31 pages validées par Elowan ([bible](docs/bible-du-monde.md), [version interactive](docs/grimoire-d-erydor.html)).

- ✓ **La fabrique des dessins** : l'IA d'images n'est pas accessible depuis l'environnement de Claude ; Elowan a choisi le pixel art dessiné par code.
- ✓ **La forge des héros** (`tools/sprites/forge_heros.py`) : un mannequin 3D photographié dans la vraie vue du jeu, pièce par pièce. 10 pièces (corps, pantalon, bottes, tunique en 12 couleurs, gants, cape, cheveux courts ou longs en 6 couleurs, casque, épée), 8 directions, 4 animations (repos, marche, course, salut).
- ✓ **Habits neufs (V0.1.2)** : le pion est remplacé par le héros en couches, avec la course, le salut et le zoom.

### ○ Quêtes à venir (V0.2)

- ○ Le Val de Glennmor (≈ 200 × 200 cases) et Bourg-Lanterne : auberge La Chope Pleine + 2 maisons où l'on entre.
- ○ Jour / nuit de 2 h (80 min de jour, 40 de nuit), nuits très sombres avec torche ; pluie, brouillard, orage partagés par région.
- ○ Une dizaine de PNJ avec leurs habitudes, accueil par Maître Mabon, bulles + boîte de dialogue, Braise qui suit le héros.

### ⚠️ Points d'attention

- **Nuits très sombres (choix d'Elowan)** : la torche s'allume toute seule la nuit, et un réglage de luminosité protège la lisibilité sur téléphone.
- **Casque, gants et cape visibles (choix d'Elowan)** : presque deux fois plus de dessins par direction et par animation. Avec la forge par code, ça ne coûte que du temps de calcul (3 minutes pour tout redessiner).
- **Cheveux tirés du nom** : en attendant la personnalisation de l'Acte IV, la coupe et la couleur des cheveux sont choisies à partir du nom du héros.
- **Pixel art et écrans** : sur un écran qui n'est pas un multiple de 1280 × 720, les pixels du héros peuvent être de tailles un peu inégales. Le zoom le règle en partie ; un vrai réglage « pixels nets » viendra avec la V0.2.

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
| 29/10/2026 | Fabrication des dessins | IA + nettoyage en pixel art ⭐, impossible faute d'accès à une IA d'images |
| 29/10/2026 | Fabrication des dessins (bis) | Pixel art dessiné par code ⭐ |
| 29/10/2026 | Palette | Vive et colorée, façon Aion ⭐ |
| 29/10/2026 | Contour des sprites | Fin contour sombre ⭐ |
| 29/10/2026 | Effets HD-2D | Lumière dynamique + ombres + léger flou de profondeur ⭐ |
| 29/10/2026 | Zoom | Réglable dans une petite plage ⭐ |
| 29/10/2026 | Races dessinées à l'Acte II | Humain seulement ⭐ |
| 29/10/2026 | Pièces visibles | **Corps, tête, cheveux, torse, jambes, bottes, arme + casque, gants, cape** (choix d'Elowan) |
| 29/10/2026 | Animations | Marche, course, repos, salut ⭐ |
| 29/10/2026 | Course | Marche par défaut, touche ou bouton pour courir ⭐ |
| 29/10/2026 | Pion de l'Acte I | Remplacé pour de bon ⭐ |
| 29/10/2026 | Paysage de départ | Forêts et prairies ⭐ |
| 29/10/2026 | Taille de la première région | Environ 200 × 200 cases ⭐ |
| 29/10/2026 | Cœur de la région | Un village d'accueil ⭐ |
| 29/10/2026 | Bâtiments où l'on entre | L'auberge et 2 maisons ⭐ |
| 29/10/2026 | Construction des cartes | Éditeur de tuiles de Godot ⭐ |
| 29/10/2026 | Durée d'une journée | 2 h réelles (80 min de jour, 40 de nuit) ⭐ |
| 29/10/2026 | La nuit | **Très sombre, il faut une torche** (choix d'Elowan) |
| 29/10/2026 | Météo de l'Acte II | Pluie, brouillard, orage ⭐ |
| 29/10/2026 | Saisons | Plus tard, à l'Acte VIII ⭐ |
| 29/10/2026 | Météo partagée | Oui, par région, décidée par le serveur ⭐ |
| 29/10/2026 | Nombre de PNJ | Une dizaine, avec leurs habitudes ⭐ |
| 29/10/2026 | PNJ d'accueil | **Un aubergiste bavard** (choix d'Elowan) : Maître Mabon |
| 29/10/2026 | Dialogues | **Bulles en passant + boîte quand on parle** (choix d'Elowan) |
| 29/10/2026 | Vie des PNJ | Ils dorment la nuit et ouvrent boutique le jour ⭐ |
| 29/10/2026 | Braise dans le jeu | Il suit le héros et commente, désactivable ⭐ |
| 29/10/2026 | Bible du monde | Les 31 pages du Grimoire d'Erydor validées ⭐ |
| 29/10/2026 | 15ᵉ race | **Les Lumins**, êtres de cristal nés du Lumenfall (choix d'Elowan) |

⭐ = recommandation de Claude suivie.

---

## 📦 Notes de version

### V0.1.2 · « Habits neufs » (30 octobre 2026)

- 🆕 Adieu le pion : ton héros est un vrai aventurier humain en pixel art, dessiné pièce par pièce (tunique, pantalon, bottes, gants, cape, épée, cheveux) et vu sous 8 angles.
- 🆕 La tunique prend la couleur choisie à la création ; les cheveux (courts ou longs, 6 couleurs) viennent du nom.
- 🆕 Courir : maintiens **Maj** sur PC, ou touche le bouton 🏃 **Courir** sur téléphone. 6,5 cases par seconde au lieu de 4.
- 🆕 Saluer : bouton 👋 ou touche **H**. Tout le monde autour voit ton salut.
- 🆕 Zoom : molette sur PC, pincement ou boutons ＋ / － sur téléphone.
- ✨ L'écran de création montre ton héros qui tourne sur lui-même et change de tenue en direct.
- 🐛 Corrigé : le nom de la carte s'affichait parfois lettre par lettre, en colonne.

Captures : [avant / après](docs/captures/v0.1.2/avant-apres.png) · [la Clairière](docs/captures/v0.1.2/clairiere.png) · [création](docs/captures/v0.1.2/creation.png) · [planche du héros](docs/captures/v0.1.2/planche-heros.png)

### V0.1.1 · « Bousculade » (29 octobre 2026, serveur seulement)

- 🐛 Corrigé : les héros se traversaient. Ils se bousculent maintenant sans pouvoir passer l'un à travers l'autre (demande d'Elowan).
- ✨ S'éloigner d'un héros reste toujours possible, donc personne ne peut en coincer un autre contre un mur.
- 🧪 Test de charge : 100 faux joueurs, tick moyen 0,88 ms sur 100 ms (0,51 ms sans les collisions).

### V0.1 · « Le premier pas » (29 octobre 2026)

- 🆕 Écran titre avec connexion Discord et connexion de test.
- 🆕 Choix du héros : 3 emplacements, nom unique, 12 couleurs.
- 🆕 La Clairière en 3/4 isométrique : arbres, rochers, étang, chemin, fleurs.
- 🆕 Les autres aventuriers en direct, avec leur nom.
- 🆕 Chat général et bulles au-dessus des têtes.
- 🆕 Commandes PC (clic, ZQSD, flèches) et téléphone (joystick, toucher).
- 🆕 Braise donne des astuces sur les écrans de menu.

Captures : [écran titre](docs/captures/v0.1/titre.png) · [héros](docs/captures/v0.1/heros.png) · [création](docs/captures/v0.1/creation.png) · [la Clairière](docs/captures/v0.1/clairiere.png)
