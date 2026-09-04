#!/usr/bin/env bash
# Runs ON the Contabo VPS (invoked over SSH by .github/workflows/deploy.yml).
# Assumes: Docker + Docker Compose plugin installed, repo cloned at
# /opt/cytoai, and /opt/cytoai/.env populated (see .env.example).
set -euo pipefail

APP_DIR="/opt/cytoai"
COMPOSE_FILE="build/docker-compose.prod.yml"

cd "$APP_DIR"

echo "==> Pulling latest code (for migrations + compose/Caddy config)"
git fetch origin main
git reset --hard origin/main

echo "==> Pulling latest images"
docker compose -f "$COMPOSE_FILE" --env-file .env pull api web

echo "==> Restarting stack"
docker compose -f "$COMPOSE_FILE" --env-file .env up -d

echo "==> Waiting for API to be ready"
for i in $(seq 1 30); do
	if docker compose -f "$COMPOSE_FILE" exec -T api curl -fsS http://localhost:8080/readyz >/dev/null 2>&1; then
		echo "API ready."
		break
	fi
	sleep 2
done

echo "==> Running pending migrations"
docker compose -f "$COMPOSE_FILE" --env-file .env run --rm api /app/cytoai migrate up || \
	bash "$APP_DIR/scripts/migrate.sh"

echo "==> Pruning old images"
docker image prune -f

echo "==> Deploy complete"
docker compose -f "$COMPOSE_FILE" ps