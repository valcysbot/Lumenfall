#!/usr/bin/env bash
# Installation du serveur de test Lumenfall sur un VPS Ubuntu 24.04 neuf.
# À coller une seule fois dans le terminal du VPS (en root).
#
# Ce script :
#   1. installe Docker et un pare-feu (seuls SSH, HTTP et HTTPS restent ouverts) ;
#   2. crée l'utilisateur « lumenfall » qui servira aux déploiements ;
#   3. crée une clé de déploiement ET tous les mots de passe directement sur le VPS
#      (ils ne passent jamais par le chat) ;
#   4. affiche ce qu'il faut recopier dans les réglages GitHub du dépôt.
set -euo pipefail

if [ "$(id -u)" != "0" ]; then
  echo "Lance ce script en root." >&2
  exit 1
fi

echo "🐉 Braise installe le serveur de test de Lumenfall…"

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq ca-certificates curl ufw rsync >/dev/null
if ! command -v docker >/dev/null; then
  curl -fsSL https://get.docker.com | sh >/dev/null
fi

ufw allow OpenSSH >/dev/null
ufw allow 80/tcp >/dev/null
ufw allow 443/tcp >/dev/null
ufw --force enable >/dev/null

id lumenfall >/dev/null 2>&1 || useradd -m -s /bin/bash -G docker lumenfall
APP=/home/lumenfall/app
mkdir -p "$APP/deploy" /home/lumenfall/.ssh
chmod 700 /home/lumenfall/.ssh

# Clé de déploiement : la partie publique reste ici, la partie privée va dans GitHub.
KEY=/root/lumenfall-deploy-key
if [ ! -f "$KEY" ]; then
  ssh-keygen -q -t ed25519 -N "" -C "lumenfall-deploy" -f "$KEY"
fi
grep -qf "$KEY.pub" /home/lumenfall/.ssh/authorized_keys 2>/dev/null || cat "$KEY.pub" >> /home/lumenfall/.ssh/authorized_keys
chmod 600 /home/lumenfall/.ssh/authorized_keys

IP=$(curl -fsS -4 https://api.ipify.org)
HOST="$(echo "$IP" | tr . -).sslip.io"
rand() { tr -dc 'A-Za-z0-9' </dev/urandom | head -c "$1"; }

ENV="$APP/deploy/.env"
if [ ! -f "$ENV" ]; then
  cat > "$ENV" <<EOF
LF_ENV=test
LF_ALLOW_TEST_LOGIN=true
LF_DOMAIN=$HOST
POSTGRES_PASSWORD=$(rand 32)
LF_SERVER_KEY=$(rand 24)
LF_SESSION_KEY=$(rand 32)
LF_REFRESH_KEY=$(rand 32)
LF_HTTP_KEY=$(rand 32)
LF_CONSOLE_USER=elowan
LF_CONSOLE_PASSWORD=$(rand 20)
LF_DISCORD_CLIENT_ID=
LF_DISCORD_CLIENT_SECRET=
EOF
fi
chmod 600 "$ENV"
chown -R lumenfall:lumenfall /home/lumenfall
SERVER_KEY=$(grep '^LF_SERVER_KEY=' "$ENV" | cut -d= -f2)

cat <<EOF

✅ Le VPS est prêt ! Recopie maintenant ceci dans GitHub :
   https://github.com/valcysbot/lumenfall/settings/secrets/actions

── Onglet « Secrets » → New repository secret ─────────────────────
  Nom : LF_DEPLOY_SSH_KEY
  Valeur : tout le bloc ci-dessous, lignes BEGIN et END comprises

$(cat "$KEY")

── Onglet « Variables » → New repository variable (3 fois) ────────
  LF_SERVER_HOST  =  $HOST
  LF_SERVER_KEY   =  $SERVER_KEY
  LF_TEST_LOGIN   =  true

── Et dans le portail Discord (OAuth2 → Redirects) ────────────────
  https://$HOST/lumenfall/discord/callback

Le mot de passe de la console d'administration est rangé dans $ENV.
EOF
