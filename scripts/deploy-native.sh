#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"

INSTALL_DIR="${INSTALL_DIR:-/opt/geopress}"
DATA_DIR="${DATA_DIR:-/var/lib/geopress}"
ENV_FILE="${ENV_FILE:-/etc/geopress/geopress.env}"
SERVICE_NAME="${SERVICE_NAME:-geopress}"
OWNER="${OWNER-geopress:geopress}"
RUN_MIGRATIONS="${RUN_MIGRATIONS:-true}"
RESTART_SERVICE="${RESTART_SERVICE:-true}"
BUILD="${BUILD:-true}"
GIT_PULL="${GIT_PULL:-false}"
SUDO="${SUDO:-sudo}"

if [ "$(id -u)" -eq 0 ]; then
  SUDO=""
fi

if [ "${NO_SUDO:-false}" = "true" ]; then
  SUDO=""
fi

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

is_true() {
  case "$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')" in
    1|true|yes|on) return 0 ;;
    *) return 1 ;;
  esac
}

run_privileged() {
  if [ -n "$SUDO" ]; then
    "$SUDO" "$@"
    return
  fi
  "$@"
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "$1 is required" >&2
    exit 127
  fi
}

require_command cp
require_command install
require_command mkdir
require_command mv
require_command rm

if [ -n "$SUDO" ]; then
  require_command "$SUDO"
fi

if is_true "$GIT_PULL"; then
  require_command git
  echo "Pulling latest code..."
  git -C "$PROJECT_ROOT" pull --ff-only
fi

BUILD_BIN="$TMP_DIR/geopress-api"
if is_true "$BUILD"; then
  echo "Building native binary..."
  OUTPUT_DIR="$TMP_DIR/build" OUTPUT_BIN="$BUILD_BIN" "$PROJECT_ROOT/scripts/build-native.sh"
else
  BUILD_BIN="$PROJECT_ROOT/dist/geopress-api"
  if [ ! -f "$BUILD_BIN" ]; then
    echo "BUILD=false but $BUILD_BIN does not exist" >&2
    exit 1
  fi
fi

echo "Staging deployment payload..."
mkdir -p "$TMP_DIR/payload/backend"
cp "$BUILD_BIN" "$TMP_DIR/payload/geopress-api"
cp -R "$PROJECT_ROOT/scripts" "$TMP_DIR/payload/scripts"
cp -R "$PROJECT_ROOT/backend/migrations" "$TMP_DIR/payload/backend/migrations"

echo "Installing files into $INSTALL_DIR..."
run_privileged mkdir -p "$INSTALL_DIR/backend" "$DATA_DIR/runtime"
run_privileged install -m 0755 "$TMP_DIR/payload/geopress-api" "$INSTALL_DIR/geopress-api.new"
run_privileged mv "$INSTALL_DIR/geopress-api.new" "$INSTALL_DIR/geopress-api"

run_privileged rm -rf "$INSTALL_DIR/scripts.new" "$INSTALL_DIR/backend/migrations.new"
run_privileged cp -R "$TMP_DIR/payload/scripts" "$INSTALL_DIR/scripts.new"
run_privileged cp -R "$TMP_DIR/payload/backend/migrations" "$INSTALL_DIR/backend/migrations.new"
run_privileged rm -rf "$INSTALL_DIR/scripts" "$INSTALL_DIR/backend/migrations"
run_privileged mv "$INSTALL_DIR/scripts.new" "$INSTALL_DIR/scripts"
run_privileged mv "$INSTALL_DIR/backend/migrations.new" "$INSTALL_DIR/backend/migrations"
run_privileged chmod +x "$INSTALL_DIR/scripts/build-native.sh" "$INSTALL_DIR/scripts/deploy-native.sh" "$INSTALL_DIR/scripts/migrate.sh"

if [ -n "$OWNER" ]; then
  run_privileged chown -R "$OWNER" "$INSTALL_DIR" "$DATA_DIR"
fi

if [ -r "$ENV_FILE" ]; then
  set -a
  # shellcheck disable=SC1090
  . "$ENV_FILE"
  set +a
fi

if is_true "$RUN_MIGRATIONS"; then
  if [ -z "${DATABASE_URL:-}" ]; then
    echo "DATABASE_URL is required to run migrations." >&2
    echo "Set it in $ENV_FILE or export DATABASE_URL before running this script." >&2
    exit 1
  fi
  echo "Running database migrations..."
  MIGRATIONS_DIR="$INSTALL_DIR/backend/migrations" DATABASE_URL="$DATABASE_URL" "$INSTALL_DIR/scripts/migrate.sh"
fi

if is_true "$RESTART_SERVICE"; then
  require_command systemctl
  echo "Restarting $SERVICE_NAME..."
  run_privileged systemctl daemon-reload
  run_privileged systemctl restart "$SERVICE_NAME"
  run_privileged systemctl --no-pager --full status "$SERVICE_NAME" || true
fi

echo "Native deployment updated:"
echo "  binary:     $INSTALL_DIR/geopress-api"
echo "  scripts:    $INSTALL_DIR/scripts"
echo "  migrations: $INSTALL_DIR/backend/migrations"
