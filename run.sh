#!/usr/bin/env bash
# One command to set up and run Beego Multisite Renderer locally:
#
#   ./run.sh
#
# 1. creates .env from .env.example (an existing .env is kept)
# 2. if DATABASE_URL in .env is empty, starts the PostgreSQL Docker
#    container "bmr-postgres" (port 55432) with a generated password
#    and writes its connection URL to .env
# 3. creates the tables and inserts the demo data (safe to repeat)
# 4. starts the server: open http://localhost:8080/
set -euo pipefail
cd "$(dirname "$0")"

CONTAINER=bmr-postgres
DB_PORT=55432
DB_NAME=beego_multisite

die() { echo "ERROR: $*" >&2; exit 1; }

# env_value KEY -> the value of KEY in .env (empty when unset)
env_value() { sed -n "s/^$1=//p" .env | tail -n 1 | tr -d '"'; }

# set_env KEY VALUE -> replaces or adds KEY=VALUE in .env
set_env() {
    if grep -q "^$1=" .env; then
        awk -v k="$1" -v v="$2" 'index($0, k "=") == 1 { $0 = k "=" v } { print }' .env > .env.tmp
        mv .env.tmp .env
    else
        echo "$1=$2" >> .env
    fi
    chmod 600 .env
}

command -v go >/dev/null || die "Go is not installed: https://go.dev/dl/"

# 1. Environment file
if [ ! -f .env ]; then
    cp .env.example .env
    echo "==> Created .env from .env.example"
fi
chmod 600 .env

# 2. Local PostgreSQL in Docker, unless .env names another database
if [ -z "$(env_value DATABASE_URL)" ] ||
   [[ "$(env_value DATABASE_URL)" == *"@localhost:${DB_PORT}/"* ]]; then
    command -v docker >/dev/null || die "Docker is not installed: https://docs.docker.com/get-docker/"
    docker info >/dev/null 2>&1 ||
        die "Docker is not running, or this user cannot use it (try: sudo usermod -aG docker \$USER, then log in again)"

    if docker container inspect "$CONTAINER" >/dev/null 2>&1; then
        echo "==> Starting PostgreSQL container $CONTAINER"
        docker start "$CONTAINER" >/dev/null
        # Reuse the password the container was created with.
        password="$(docker container inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$CONTAINER" |
            sed -n 's/^POSTGRES_PASSWORD=//p')"
    else
        echo "==> Creating PostgreSQL container $CONTAINER (first run downloads the image)"
        password="$(od -An -N16 -tx1 /dev/urandom | tr -d ' \n')"
        docker run -d --name "$CONTAINER" \
            -e POSTGRES_USER=postgres \
            -e POSTGRES_PASSWORD="$password" \
            -e POSTGRES_DB="$DB_NAME" \
            -p "127.0.0.1:${DB_PORT}:5432" \
            postgres:17 >/dev/null
    fi
    [ -n "$password" ] || die "could not read the password of container $CONTAINER"

    set_env DATABASE_URL "postgres://postgres:${password}@localhost:${DB_PORT}/${DB_NAME}?sslmode=disable"

    echo "==> Waiting for PostgreSQL"
    for _ in $(seq 1 60); do
        # TCP check: the image's temporary init server only listens on a socket.
        if docker exec "$CONTAINER" pg_isready -h 127.0.0.1 -U postgres -d "$DB_NAME" >/dev/null 2>&1; then
            ready=1
            break
        fi
        sleep 1
    done
    [ "${ready:-0}" = 1 ] || die "PostgreSQL did not start; see: docker logs $CONTAINER"
fi

# 3. Database tables and demo data
echo "==> Creating tables";     go run ./cmd/dbsetup
echo "==> Inserting demo data"; go run ./cmd/dbseed

# 4. Server
echo "==> Starting: open http://localhost:8080/  (Ctrl+C to stop)"
exec bee run