# 📜 Journal de quête de Lumenfall

*Le document de suivi du projet : ce qui est fait, ce qui reste, et les décisions prises. Mis à jour à chaque livraison.*

**Version actuelle :** V0.3.1 « Premier sang » (30 octobre 2026) · **Acte en cours :** Acte III, le combat

---

## 🗺️ La grande carte des actes

| Acte | Nom | État |
|---|---|---|
| I | Les fondations : dépôt, serveur, base de données, connexion Discord, héros qui bougent ensemble, serveur de test | ✓ validé par Elowan |
| II | Le monde vivant : rendu HD-2D, héros en couches 8 directions, première région, jour / nuit, météo, PNJ | ✓ validé par Elowan |
| III | Le combat : ciblage, compétences, effets, éléments, monstres, boss, mort | ✱ en cours |
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
- ✓ **Bourg-Lanterne (V0.1.4)** : le Val de Glennmor (200 × 200 cases) remplace la Clairière. Bourg-Lanterne, sa place, son puits et son marché ; l'auberge La Chope Pleine et 2 maisons où l'on entre ; le temple de Solenn, la forge, le moulin, les fermes et le pré de Gwenn, la rivière et son pont, la Lisière Grise à l'est. Les lanternes s'allument la nuit.
- ✓ **La fabrique des cartes** : `forge_tuiles.py` (sol), `forge_decors.py` (36 décors), `gen_glennmor.py` (premier jet), puis `construire_carte.gd` en fait une scène Godot qu'on retouche à la souris, et `exporter_carte.gd` en tire les cases bloquées et les portes pour le serveur.
- ✓ **Le ciel d'Erydor (V0.1.3)** : jour et nuit sur 2 heures, la même heure pour tout le monde ; pluie, brouillard et orage décidés par le serveur pour chaque région ; torches automatiques la nuit ; réglage de luminosité de la nuit.

- ✓ **Les habitants (V0.2)** : 11 habitants de la bible vivent à Bourg-Lanterne, avec leurs horaires (le serveur les fait travailler le jour, rentrer dormir la nuit ; Corbin fait l'inverse). Bulles quand on passe, boîte de dialogue avec portrait et choix, Maître Mabon et son sanglier qui grandit. Braise vole derrière le héros et commente ce qu'il voit.

- ✓ **Clôture** : V0.2 sur le serveur de test et publiée (v0.2.0), validée par Elowan le 30 octobre 2026.

### ⚠️ Points d'attention

- **Nuits très sombres (choix d'Elowan)** : la torche s'allume toute seule la nuit, et un réglage de luminosité protège la lisibilité sur téléphone. Pour ménager les téléphones, seuls les 10 héros les plus proches portent une torche allumée.
- **Casque, gants et cape visibles (choix d'Elowan)** : presque deux fois plus de dessins par direction et par animation. Avec la forge par code, ça ne coûte que du temps de calcul (3 minutes pour tout redessiner).
- **Beaucoup de décors** : le val compte près de 7 000 décors (surtout des arbres). Godot n'affiche que ceux à l'écran, mais il faudra vérifier la fluidité sur ton téléphone ; si ça rame, on regroupera les forêts lointaines.
- **Héros de la Clairière** : la Clairière est fermée. Les héros qui y étaient réapparaissent sur la place de Bourg-Lanterne.
- **Cheveux tirés du nom** : en attendant la personnalisation de l'Acte IV, la coupe et la couleur des cheveux sont choisies à partir du nom du héros.
- **Habitants et maisons** : les habitants ne passent pas encore par les portes. Le soir, ils rentrent devant chez eux et disparaissent ; seul Maître Mabon vit à l'intérieur de son auberge.
- **Passer derrière une maison** : un héros (ou un habitant) caché derrière un toit n'est pas encore montré en silhouette. À prévoir.
- **Pixel art et écrans** : sur un écran qui n'est pas un multiple de 1280 × 720, les pixels du héros peuvent être de tailles un peu inégales. Le zoom le règle en partie ; un vrai réglage « pixels nets » viendra avec la V0.2.

---

## ⚔️ Acte III, le combat

### ✓ Quêtes accomplies

- ✓ **Le Conseil des Armes** : 25 questions, toutes répondues le 30 octobre 2026 ([archive](docs/conseil-des-armes.html)).
- ✓ **L'arène (serveur)** : `server/world/arena.go`. Le serveur décide de tout : portée, recharges, Ferveur, dégâts, critiques, esquive. Le client n'envoie que « je vise », « compétence 2 », « roulade », « bouclier », « potion ».
- ✓ **Les règles** (`server/world/combat.go`) : les 6 éléments en roue (feu > air > terre > eau > feu, lumière ↔ ombre, +50 %), 4 effets d'état (brûlure, poison, étourdissement, ralentissement), la menace pour les tanks, la mort au temple de Solenn avec 2 minutes de Faiblesse.
- ✓ **Le Paladin d'essai** : attaque de base automatique, Frappe sacrée, Coup de bouclier (étourdit), Lumière de Solenn (soin incanté), Aura de Solenn, Jugement de lumière (à distance, embrase). Ferveur, bouclier tenu au clic droit, roulade avec 2 charges d'endurance, 5 potions que Sœur Clémence remplit au village.
- ✓ **Le bestiaire du val** (`server/world/monster.go`, camps dans `maps/glennmor.monstres.json`) : sangliers (niv. 1 à 3), loups en meute (3 à 5), loups et corbeaux effacés dans la Lisière Grise (6 à 9), et le Sanglier doré qui fuit (3 % de chance). Ils flânent, remarquent les héros, appellent la meute et rentrent chez eux si l'on fuit trop loin. La nuit, les Effacés voient les torches de plus loin.
- ✓ **Le Grand Sanglier de Mabon** (`server/world/boss.go`), niveau 10, dans la Bauge au sud de la Lisière : charge et piétinement annoncés par des zones rouges ; sous la moitié de sa vie, la Flétrissure le dévore (phase 2 : il devient un Effacé, plus rapide, et des éclats gris tombent sur chaque héros).
- ✓ **XP et niveaux** : sans niveau maximum, la vie et les dégâts montent à chaque niveau ; or et trophées. Tout est sauvegardé en base avec la position.
- ✓ **La forge des bêtes** (`tools/sprites/forge_betes.py`) et 7 nouvelles animations du Paladin (attaque, coup de bouclier, garde, incantation, sort, roulade, chute) avec son écu.
- ✓ **Le tutoriel de combat** (V0.3.1, `client/scripts/combat_tutorial.gd`) : au premier monstre croisé, Braise apprend les gestes un par un (viser, frapper, bouclier, roulade, potion, autres sorts) en faisant briller le bon bouton. On peut le passer, et le revoir dans les réglages ⚙.
- ✓ **Le combat à l'écran** : cadre du héros (vie, Ferveur, XP, effets), cadre de la cible, barre de compétences (1 à 5, R, Espace), éventail pour le pouce sur téléphone, chiffres qui sautent, critiques dorés qui font trembler l'écran, ralenti sur le coup fatal d'un élite, zones rouges qui se remplissent, écran « Tu es tombé ».

### ○ Quêtes à venir

- ○ Ton avis sur la V0.3, puis les réglages (vie des monstres, dégâts, temps de recharge).
- ○ Les duels sur invitation (choix du Conseil : juste après le boss).

### ⚠️ Points d'attention

- **Équilibrage à l'aveugle** : les chiffres viennent de calculs et de tests automatiques, pas encore de vraies parties. Attends-toi à ce que certains monstres soient trop durs ou trop mous : dis-moi lesquels.
- **Tout le monde est Paladin** jusqu'à l'Acte IV : les héros existants ont reçu l'écu et les compétences du Paladin.
- **Le Grand Sanglier se joue à plusieurs** : seul au niveau 10, il faut absolument sortir des zones rouges (dans les combats simulés sans esquive, le héros seul perd 10 fois sur 10). À deux ou trois, ça passe. Il revient 5 minutes après sa chute.
- **L'équilibrage simulé** (`server/world/equilibre_test.go`) : 10 combats automatiques par camp vérifient qu'un Paladin du bon niveau gagne contre sangliers, loups, loups et corbeaux effacés. Le soin a reçu 6 s de recharge : sans elle, le Paladin était immortel.
- **Planches plus larges** : le héros est maintenant dessiné sur des images de 64 pixels de large (pour la chute), et chaque planche fait 3 712 pixels de large. C'est sous la limite de 4 096 des téléphones, mais il faudra découper les planches quand les animations des 17 classes arriveront.
- **Jeu trop ancien** : le serveur demande au moins la v0.3.0. La v0.3.1 n'ajoute que le tutoriel : une v0.3.0 peut toujours jouer.
- **Le ralenti** n'existe que sur ton écran (le monde est partagé) : il ne dure qu'une demi-seconde, sur le coup fatal d'un élite ou d'un boss.

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
| 30/10/2026 | Ciblage | Clic sur le monstre ou Tab ⭐ |
| 30/10/2026 | Attaque de base | Automatique sur la cible, les compétences à la main ⭐ |
| 30/10/2026 | Bouger en combattant | Oui, sauf pendant une incantation ⭐ |
| 30/10/2026 | Roulade | **Jauge d'endurance : 2 roulades d'affilée** (choix d'Elowan) |
| 30/10/2026 | Combat sur téléphone | Éventail de 5 compétences + roulade + cible suivante ⭐ |
| 30/10/2026 | Classe d'essai | Le Paladin seul, très soigné ⭐ |
| 30/10/2026 | Compétences | 5 : frappe, coup de bouclier, soin, aura, jugement ⭐ |
| 30/10/2026 | Ressource du Paladin | La Ferveur ⭐ |
| 30/10/2026 | Bouclier | Tenu au clic droit, bloque de face ⭐ |
| 30/10/2026 | Potions | 5, recharge de 30 s, remplies par Sœur Clémence ⭐ |
| 30/10/2026 | Effets d'état | 4 : brûlure, poison, étourdissement, ralentissement ⭐ |
| 30/10/2026 | Éléments | Les 6, en roue, +50 % ⭐ |
| 30/10/2026 | Chiffres de dégâts | Qui sautent, critiques dorés avec tremblement ⭐ |
| 30/10/2026 | Ralenti | Seulement sur le coup fatal d'un élite ou d'un boss ⭐ |
| 30/10/2026 | Attaques annoncées | Zones rouges qui se remplissent ⭐ |
| 30/10/2026 | Premiers monstres | Bêtes du val + leurs versions effacées vers la Lisière ⭐ |
| 30/10/2026 | Sortes de monstres | 4 + le Sanglier doré ⭐ |
| 30/10/2026 | Comportement | Ils flânent, attaquent de près, rentrent si l'on fuit ⭐ |
| 30/10/2026 | Premier boss | Le Grand Sanglier de Mabon, 2 phases ⭐ |
| 30/10/2026 | Menace | Dès maintenant ⭐ |
| 30/10/2026 | Mort | Temple de Solenn + Faiblesse 2 min ⭐ |
| 30/10/2026 | XP et niveaux | Dès l'Acte III, niveaux 1 à 10 dans le val ⭐ |
| 30/10/2026 | Butin | Or + trophées rares ⭐ |
| 30/10/2026 | PvP | Duels sur invitation juste après le boss ⭐ |
| 30/10/2026 | Mode hardcore | À l'Acte IV, avec la création du héros ⭐ |
| 30/10/2026 | Combat jugé « un peu compliqué » : ce qui manque | « On ne comprend pas quoi faire » → tutoriel de Braise au premier monstre ⭐ |
| 30/10/2026 | Publication des versions | Claude lance lui-même la publication (onglet Actions) après chaque fusion |

⭐ = recommandation de Claude suivie.

---

## 📦 Notes de version

### V0.3.1 · « Premier sang » (30 octobre 2026)

- 🆕 **Braise t'apprend à te battre.** Au premier monstre que tu croises, une petite carte s'ouvre sous ta vie et te montre les gestes un par un : viser, frapper, lever ton bouclier, rouler, boire une potion, puis tes autres sorts. Le bouton à presser brille, et la carte passe à l'étape suivante dès que tu as fait le geste.
- ✨ Tu peux passer le tutoriel, et le revoir quand tu veux depuis les réglages ⚙.
- 🐛 corrigé : le test de fumée du serveur de test annonçait encore la version 0.2.1 et se faisait refouler.

Captures : [sur PC](docs/captures/v0.3.1/tutoriel-pc.png) · [sur téléphone](docs/captures/v0.3.1/tutoriel-telephone.png) · [avant / après](docs/captures/v0.3.1/avant-apres.png).

### V0.3 · « Premier sang » (30 octobre 2026)

- 🆕 **Le combat !** Clique sur un monstre (ou Tab) pour le viser : ton héros y court et frappe tout seul. À toi les compétences : **1** Frappe sacrée, **2** Coup de bouclier, **3** Lumière de Solenn, **4** Aura de Solenn, **5** Jugement de lumière.
- 🆕 **La Ferveur** monte quand tu frappes ou encaisses, et paie le soin et le Jugement.
- 🆕 **Roulade** (Espace) : invincible le temps de rouler, 2 d'affilée puis tu souffles. **Bouclier** : maintiens le clic droit, il bloque les coups de face.
- 🆕 **Potions** (R) : 5 dans ta besace, Sœur Clémence la remplit quand tu rentres au village.
- 🆕 **Les monstres** : sangliers, loups en meute, et dans la Lisière Grise des loups et des corbeaux **effacés**, gris et tremblants. Ouvre l'œil : un **Sanglier doré** traîne parfois dans les champs, et il ne se laisse pas attraper facilement.
- 🆕 **Le Grand Sanglier de Mabon** t'attend dans la Bauge, au sud-est. Maître Mabon n'exagérait pas. Sors des zones rouges !
- 🆕 **XP, niveaux, or et trophées.** Si tu tombes, Sœur Clémence te relève au temple, un peu affaibli.
- 🆕 Sur téléphone, un **éventail de boutons** sous le pouce droit.
- ✨ Ton héros porte maintenant l'**écu** du Paladin, et sait frapper, se protéger, incanter, rouler… et tomber.
- ✨ Braise commente tes combats.

Captures : [premier combat](docs/captures/v0.3/combat-sanglier.png) · [la meute](docs/captures/v0.3/meute-de-loups.png) · [le Grand Sanglier](docs/captures/v0.3/grand-sanglier.png) · [son piétinement](docs/captures/v0.3/pietinement.png) · [sur téléphone](docs/captures/v0.3/telephone.png) · [le bestiaire](docs/captures/v0.3/bestiaire.png) · [le Paladin en action](docs/captures/v0.3/planche-paladin.png). Avant / après : [la place en V0.2 et le premier combat](docs/captures/v0.3/avant-apres.png).

### V0.2.1 · « Les habitants » (30 octobre 2026)

- 🐛 corrigé : un jeu trop ancien (comme la v0.1.0) affichait un écran vert dans le Val de Glennmor. Il affiche maintenant « Ton jeu est trop ancien : télécharge la dernière version ».
- ✨ Le jeu dit sa version au serveur en entrant dans le monde. Pense à prendre la v0.2.1 : les versions précédentes ne savent pas le faire.

### V0.2 · « Les habitants » (30 octobre 2026)

- 🆕 **11 habitants** à Bourg-Lanterne : Maître Mabon, Fiacre, Corbin, Aëlis, Doran, Nessa, Sœur Clémence, le Capitaine Bréval, Gwenn, et les jumeaux Tam et Tilda qui courent partout.
- 🆕 Ils ont **leurs habitudes** : au travail le jour, chez eux la nuit. Corbin, l'allumeur de lanternes, ne sort que la nuit.
- 🆕 **Clique sur un habitant pour lui parler** : ton héros marche jusqu'à lui, puis une boîte s'ouvre avec son portrait et des choix (touches 1 à 4, Échap pour partir).
- 🆕 Quand tu passes près d'eux, les habitants te lancent **une bulle**.
- 🆕 **Braise te suit** en volant et commente ce qu'il voit (la nuit, la pluie, l'auberge…). Tu peux le renvoyer dans ⚙.
- 🆕 Maître Mabon raconte l'histoire du sanglier… qui grandit à chaque fois.
- ✨ Nouvelles tenues dessinées par code : robe, tablier, barbe, chapeau de paille ou de feutre, trois couleurs de peau.
- 🐛 corrigé : un bouleau poussait dans la barrière du pré de Gwenn, et un chêne au milieu des moutons.

Captures : [la place animée](docs/captures/v0.2/la-place-animee.png) · [Maître Mabon](docs/captures/v0.2/maitre-mabon.png) · [le Capitaine Bréval](docs/captures/v0.2/capitaine-breval.png) · [les habitants et Braise](docs/captures/v0.2/les-habitants.png). Avant : [la place vide en V0.1.4](docs/captures/v0.1.4/bourg-lanterne.png).

### V0.1.4 · « Bourg-Lanterne » (30 octobre 2026)

- 🆕 Bienvenue dans le **Val de Glennmor** : 200 × 200 cases de forêts et de prairies, une rivière et son pont, le moulin, les fermes, le pré de Gwenn et ses moutons. À l'est, la **Lisière Grise**, où les couleurs s'effacent…
- 🆕 **Bourg-Lanterne**, le village d'accueil : la place pavée, le puits, le marché, le temple de Solenn, la forge de Doran et ses maisons à colombages.
- 🆕 On entre dans **l'auberge La Chope Pleine** et dans **2 maisons** : il suffit de marcher sur la porte. Dedans, il fait toujours bon (ni pluie, ni nuit noire).
- 🆕 La nuit, **les lanternes du village s'allument** toutes seules.
- ✨ Tout est dessiné par code, dans le même style que les héros : sol, arbres, maisons, meubles.
- ✨ La carte est une scène Godot : on peut la retoucher à la souris (peindre les chemins, déplacer une maison).

Captures : [Bourg-Lanterne](docs/captures/v0.1.4/bourg-lanterne.png) · [la nuit](docs/captures/v0.1.4/bourg-lanterne-nuit.png) · [La Chope Pleine](docs/captures/v0.1.4/la-chope-pleine.png) · [la rivière](docs/captures/v0.1.4/riviere.png) · [le plan du val](docs/captures/v0.1.4/plan-glennmor.png)

### V0.1.3 · « Le ciel d'Erydor » (30 octobre 2026)

- 🆕 Le jour se lève et se couche : une journée d'Erydor dure 2 heures (80 minutes de jour, 40 de nuit), et l'heure est la même pour tout le monde. Elle s'affiche en haut à gauche (☀️ 14 h, 🌙 1 h…).
- 🆕 Des nuits très noires, comme demandé : chaque héros allume sa torche tout seul à la tombée de la nuit.
- 🆕 La météo : pluie 🌧️, brouillard 🌫️ et orage ⛈️ avec éclairs. Elle est décidée par le serveur, région par région : quand il pleut chez toi, il pleut chez tes amis.
- 🆕 ⚙ Réglages : la luminosité de la nuit, pour garder la nuit jouable sur téléphone.

Captures : [le ciel en 6 images](docs/captures/v0.1.3/ciel.png) · [la nuit](docs/captures/v0.1.3/nuit.png) · [l'orage](docs/captures/v0.1.3/orage.png)

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
