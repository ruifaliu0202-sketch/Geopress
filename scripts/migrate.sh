#!/usr/bin/env sh
set -eu

DATABASE_URL="${DATABASE_URL:-postgres://geopress:geopress@localhost:5432/geopress?sslmode=disable}"
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-$PROJECT_ROOT/backend/migrations}"

if ! command -v psql >/dev/null 2>&1; then
  echo "psql is required to run migrations" >&2
  exit 127
fi

if [ ! -d "$MIGRATIONS_DIR" ]; then
  echo "migrations directory not found: $MIGRATIONS_DIR" >&2
  exit 1
fi

for file in "$MIGRATIONS_DIR"/*.sql; do
  [ -e "$file" ] || {
    echo "no migration files found in $MIGRATIONS_DIR" >&2
    exit 1
  }
  echo "Applying $file"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$file"
done
