#!/usr/bin/env bash
# Lance un serveur Lumenfall local, sans Docker : PostgreSQL déjà démarré,
# Nakama dans $NAKAMA (binaire officiel) et Go à la même version que Nakama.
#   NAKAMA=~/outils/nakama/nakama tools/dev-server.sh
set -euo pipefail
cd "$(dirname "$0")/.."
NAKAMA=${NAKAMA:-nakama}
DB=${DB:-postgres@127.0.0.1:5432/nakama}
RUN=${RUN:-.run}
GOV=$(go version -m "$(command -v "$NAKAMA")" | head -1 | awk '{print $2}')
mkdir -p "$RUN/modules"
(cd server && GOTOOLCHAIN=$GOV go build -buildmode=plugin -trimpath -o "../$RUN/modules/lumenfall.so" .)
cat > "$RUN/local.yml" <<YML
name: lumenfall-dev
data_dir: $PWD/$RUN/data
logger: { level: INFO, stdout: true }
runtime:
  path: $PWD/$RUN/modules
  env: [LF_ENV=dev, LF_ALLOW_TEST_LOGIN=true]
socket: { server_key: lumenfall-dev-key }
YML
"$NAKAMA" migrate up --database.address "$DB"
exec "$NAKAMA" --config "$RUN/local.yml" --database.address "$DB"
