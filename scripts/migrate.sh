#!/usr/bin/env sh
set -eu

DATABASE_URL="${DATABASE_URL:-postgres://geopress:geopress@localhost:5432/geopress?sslmode=disable}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-backend/migrations}"

for file in "$MIGRATIONS_DIR"/*.sql; do
  echo "Applying $file"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$file"
done
