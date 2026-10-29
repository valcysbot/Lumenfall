#!/bin/sh
# Démarrage de Nakama dans Docker : écrit la configuration à partir des
# variables d'environnement (le fichier .env du VPS), migre la base, lance.
set -e
DB="nakama:${POSTGRES_PASSWORD}@postgres:5432/nakama"
CONF=/nakama/data/lumenfall.yml
cat > "$CONF" <<YML
name: lumenfall
data_dir: /nakama/data
logger:
  level: INFO
  stdout: true
socket:
  server_key: "${LF_SERVER_KEY}"
session:
  token_expiry_sec: 604800
  encryption_key: "${LF_SESSION_KEY}"
  refresh_encryption_key: "${LF_REFRESH_KEY}"
runtime:
  path: /nakama/data/modules
  http_key: "${LF_HTTP_KEY}"
  env:
    - "LF_ENV=${LF_ENV}"
    - "LF_ALLOW_TEST_LOGIN=${LF_ALLOW_TEST_LOGIN}"
    - "LF_PUBLIC_URL=https://${LF_DOMAIN}"
    - "LF_DISCORD_CLIENT_ID=${LF_DISCORD_CLIENT_ID}"
    - "LF_DISCORD_CLIENT_SECRET=${LF_DISCORD_CLIENT_SECRET}"
console:
  port: 7351
  username: "${LF_CONSOLE_USER}"
  password: "${LF_CONSOLE_PASSWORD}"
YML
chmod 600 "$CONF"
/nakama/nakama migrate up --database.address "$DB"
exec /nakama/nakama --config "$CONF" --database.address "$DB"
