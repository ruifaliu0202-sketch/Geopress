#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"

FRONTEND_DIR="$PROJECT_ROOT/frontend"
BACKEND_DIR="$PROJECT_ROOT/backend"
EMBED_DIST_DIR="$BACKEND_DIR/internal/web/dist"
OUTPUT_DIR="${OUTPUT_DIR:-$PROJECT_ROOT/dist}"
OUTPUT_BIN="${OUTPUT_BIN:-$OUTPUT_DIR/geopress-api}"

mkdir -p "$OUTPUT_DIR"

cd "$FRONTEND_DIR"
npm run build

rm -rf "$EMBED_DIST_DIR"
mkdir -p "$EMBED_DIST_DIR"
cp -R "$FRONTEND_DIR/dist/." "$EMBED_DIST_DIR/"

cd "$BACKEND_DIR"
go build -trimpath -ldflags="-s -w" -o "$OUTPUT_BIN" ./cmd/api

printf 'Built %s\n' "$OUTPUT_BIN"
