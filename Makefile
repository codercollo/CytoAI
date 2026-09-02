# Cyto AI — developer convenience targets.
#
# Assumes a POSIX shell (Git Bash / WSL / Linux / macOS) and Docker Compose v2.
# Override tools as needed, e.g.:
#   make train PYTHON=ml/.venv/Scripts/python.exe
#   make run-api DSN="postgres://user:pass@host:5432/db?sslmode=disable"

COMPOSE := docker compose -f build/docker-compose.yml

# Local service coordinates (match build/docker-compose.yml port mapping).
DSN ?= postgres://cyto:cyto@localhost:5433/cytoai?sslmode=disable
ML_URL ?= http://localhost:5001

# Prefer the project virtualenv's Python (has the ML deps). Override on other
# platforms, e.g. `make train PYTHON=python`.
PYTHON ?= ml/.venv/Scripts/python.exe

PNPM ?= pnpm
GO ?= go

help:
	@echo "Cyto AI targets:"
	@echo "  make up                Build + start the stack (postgres, api, web)"
	@echo "  make down              Stop the stack"
	@echo "  make reset             Stop the stack and wipe the database volume"
	@echo "  make ps                Show stack status"
	@echo "  make logs              Tail stack logs"
	@echo "  make build             Build all container images"
	@echo "  make migrate           Apply DB migrations"
	@echo "  make migrate-down      Roll back all DB migrations"
	@echo "  make seed-partner      Create a demo partner and print its API key"
	@echo "  make generate-data     Generate synthetic telemetry/repayment/loan/swap-event data"
	@echo "  make train             Train the BHI + RRI models (RRI now includes swap-network features + fleet BHI aggregation)"
	@echo "  make run-api           Run the Go API locally (run make run-ml in another terminal)"
	@echo "  make run-ml            Run the Flask ML sidecar locally"
	@echo "  make dev-web           Run the Nuxt dashboard against the containerized api"
	@echo "  make build-web         Build the Nuxt dashboard"
	@echo "  make test              Run Go unit tests"
	@echo "  make test-integration  Run end-to-end integration tests"
	@echo "  make test-ml           Run the Flask sidecar test suite"
	@echo "  make fmt               Format Go source"
	@echo "  make vet               Vet Go source"
	@echo "  make check             Build + vet + unit test"
	@echo "  make bootstrap         Up + migrate + generate data + train + seed partner"

.PHONY: up
up:
	$(COMPOSE) up -d --build --remove-orphans

.PHONY: down
down:
	$(COMPOSE) down --remove-orphans

.PHONY: reset
reset:
	$(COMPOSE) down -v --remove-orphans

.PHONY: ps
ps:
	$(COMPOSE) ps

.PHONY: logs
logs:
	$(COMPOSE) logs -f

.PHONY: build
build:
	$(COMPOSE) build

.PHONY: migrate
migrate:
	CYTOAI_POSTGRES_DSN="$(DSN)" bash scripts/migrate.sh up

.PHONY: migrate-down
migrate-down:
	CYTOAI_POSTGRES_DSN="$(DSN)" bash scripts/migrate.sh down

.PHONY: seed-partner
seed-partner:
	CYTOAI_POSTGRES_DSN="$(DSN)" $(GO) run ./cmd/seedpartner

.PHONY: generate-data
generate-data:
	$(PYTHON) scripts/generate_synthetic.py --riders 200 --days 180

.PHONY: train
train:
	$(PYTHON) ml/training/train_repayment_model.py
	$(PYTHON) ml/training/train_battery_model.py

# Local Go development. In the container the api service runs both the Go API
# and the ML sidecar under supervisord; locally, start the sidecar first with
# `make run-ml` in a second terminal.
.PHONY: run-api
run-api:
	CYTOAI_POSTGRES_DSN="$(DSN)" CYTOAI_ML_BASE_URL="$(ML_URL)" $(GO) run ./cmd/cytoai

.PHONY: run-ml
run-ml:
	$(PYTHON) -m ml.run

.PHONY: dev-web
dev-web:
	$(COMPOSE) rm -sf web || true
	$(COMPOSE) up -d postgres api --remove-orphans
	cd web && NUXT_API_BASE=http://localhost:8080 $(PNPM) run dev

.PHONY: build-web
build-web:
	cd web && $(PNPM) run build

.PHONY: test
test:
	$(GO) test ./...

.PHONY: test-integration
test-integration:
	TEST_DATABASE_URL="$(DSN)" CYTOAI_ML_BASE_URL="$(ML_URL)" $(GO) test ./test/integration/... -v

.PHONY: test-ml
test-ml:
	$(PYTHON) -m pytest ml/app/tests -v

.PHONY: fmt
fmt:
	gofmt -w cmd internal pkg test

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: check
check:
	$(GO) build ./...
	$(GO) vet ./...
	$(GO) test ./...

.PHONY: bootstrap
bootstrap: up migrate generate-data train seed-partner
	@echo ""
	@echo "Stack is up. Use the API key printed above in the dashboard header."
	@echo "Run the dashboard with: make dev-web"
