#!/usr/bin/env bash
# Set up and run Beego Multisite Renderer locally.
#
#   ./run.sh               use .env, or create it for the local Docker database
#   ./run.sh "<url>"       use this PostgreSQL connection URL instead
#
# Then open http://localhost:8080/
set -euo pipefail
cd "$(dirname "$0")"

command -v go >/dev/null || { echo "Go is not installed: https://go.dev/dl/" >&2; exit 1; }

if [ -n "${1:-}" ]; then
    export DATABASE_URL="$1"
elif [ ! -f .env ]; then
    echo 'DATABASE_URL=postgres://postgres:postgres@localhost:5432/beego_multisite?sslmode=disable' > .env
    echo "Created .env for the local Docker database"
fi

# Serve the publisher from conf/deployment.json at http://localhost:8080/
LOCAL_SITE_HOST="$(sed -n 's#.*"published_domain": *"https://\([^/"]*\).*#\1#p' conf/deployment.json)"
export LOCAL_SITE_HOST

echo "==> Creating tables";      go run ./cmd/dbsetup
echo "==> Inserting demo data";  go run ./cmd/dbseed
echo "==> Starting: open http://localhost:8080/  (Ctrl+C to stop)"
exec bee run