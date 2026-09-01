#!/usr/bin/env bash
# Apply (or roll back) Postgres migrations in internal/db/migrations/.
#
# Usage:
#   scripts/migrate.sh up        # apply all pending migrations
#   scripts/migrate.sh down      # roll back all migrations
#   scripts/migrate.sh up 1      # apply exactly one migration
#
# The Postgres DSN is read from CYTOAI_POSTGRES_DSN when set, otherwise from
# configs/config.local.yaml (the postgres_dsn key).

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS_DIR="$ROOT_DIR/internal/db/migrations"

CMD="${1:-up}"
STEP="${2:-}"

if [ -n "${CYTOAI_POSTGRES_DSN:-}" ]; then
  DSN="$CYTOAI_POSTGRES_DSN"
else
  CONFIG_FILE="$ROOT_DIR/configs/config.local.yaml"
  if [ ! -f "$CONFIG_FILE" ]; then
    echo "error: $CONFIG_FILE not found and CYTOAI_POSTGRES_DSN not set" >&2
    exit 1
  fi
  DSN="$(grep -E '^postgres_dsn:' "$CONFIG_FILE" | head -1 | sed -e 's/^postgres_dsn:[[:space:]]*//' -e 's/"//g')"
fi

if [ -z "$DSN" ]; then
  echo "error: empty Postgres DSN (set CYTOAI_POSTGRES_DSN or configs/config.local.yaml)" >&2
  exit 1
fi

if ! command -v migrate >/dev/null 2>&1; then
  echo "error: golang-migrate CLI not found on PATH" >&2
  echo "  install: https://github.com/golang-migrate/migrate/tree/master/cmd/migrate" >&2
  exit 1
fi

if [ -n "$STEP" ]; then
  migrate -path "$MIGRATIONS_DIR" -database "$DSN" "$CMD" "$STEP"
else
  migrate -path "$MIGRATIONS_DIR" -database "$DSN" "$CMD"
fi
