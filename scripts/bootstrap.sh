#!/usr/bin/env bash
# Cyto AI — repo scaffold
# Creates the full directory tree from project-structure.txt so you can
# start dropping code in immediately. Safe to re-run (uses mkdir -p and
# only touches files if they don't already exist).
#
# Usage (from ~/Desktop/cytoAI, after `git init`):
#   bash scripts/bootstrap.sh

set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "==> Scaffolding Cyto AI in $ROOT_DIR"

# ---- directories -----------------------------------------------------------
dirs=(
  "cmd/cytoai"
  "internal/api"
  "internal/domain"
  "internal/risk"
  "internal/db/queries"
  "internal/db/migrations"
  "internal/config"
  "internal/auth"
  "pkg/cytoaiclient"
  "pkg/csvutil"
  "test/integration"
  "test/testdata"
  "configs"
  "docs"
  "examples/partner-integration"
  "api"
  "web/pages/riders"
  "web/components"
  "web/composables"
  "web/public"
  "build"
  "scripts"
  "ml/app/models"
  "ml/app/routes"
  "ml/app/schemas"
  "ml/app/services"
  "ml/app/tests"
  "ml/training/notebooks"
  "ml/data/battery/raw"
  "ml/data/repayment/raw"
  "ml/data/synthetic"
  "ml/data/sample"
  "ml/artifacts"
  ".github/workflows"
)

for d in "${dirs[@]}"; do
  mkdir -p "$d"
done

# ---- placeholder files so empty dirs survive git + give you landing spots --
touch_if_missing() {
  [ -f "$1" ] || : > "$1"
}

touch_if_missing "internal/api/.gitkeep"
touch_if_missing "internal/domain/.gitkeep"
touch_if_missing "internal/risk/.gitkeep"
touch_if_missing "pkg/cytoaiclient/.gitkeep"
touch_if_missing "test/integration/.gitkeep"
touch_if_missing "examples/partner-integration/.gitkeep"
touch_if_missing "build/.gitkeep"
touch_if_missing "ml/app/__init__.py"
touch_if_missing "ml/app/models/__init__.py"
touch_if_missing "ml/app/routes/__init__.py"
touch_if_missing "ml/app/schemas/__init__.py"
touch_if_missing "ml/app/services/__init__.py"
touch_if_missing "ml/app/tests/__init__.py"

# ---- .gitignore for data/build artifacts -----------------------------------
if [ ! -f ".gitignore" ]; then
cat > .gitignore <<'EOF'
# ML data — large/derived, not committed (except ml/data/sample/)
ml/data/battery/**
ml/data/repayment/**
ml/data/synthetic/**
!ml/data/**/.gitkeep
ml/artifacts/*.joblib

# Go
/cmd/**/cytoai
*.exe

# Node / Nuxt
web/node_modules/
web/.nuxt/
web/.output/

# Python
ml/**/__pycache__/
ml/.venv/
*.pyc

# Config secrets
configs/config.local.yaml

# OS
.DS_Store
EOF
fi

echo "==> Done. Tree created. Next: run scripts/fetch_public_data.sh to pull bootstrap datasets."