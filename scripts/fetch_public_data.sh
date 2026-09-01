#!/usr/bin/env bash
# Cyto AI — public dataset bootstrap
#
# Downloads small, freely-available datasets used to prototype the
# Battery Health Index (BHI) and Repayment Risk Index (RRI) models
# before real partner telematics/repayment data is available.
# See docs/data-sources.md for full source details, licenses, and
# manual-download fallbacks.
#
# Usage:
#   bash scripts/fetch_public_data.sh                # fetch everything
#   bash scripts/fetch_public_data.sh --skip-kaggle   # skip Kaggle (no API key set up yet)

set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_DIR="$ROOT_DIR/ml/data"
BATTERY_DIR="$DATA_DIR/battery/raw"
REPAYMENT_DIR="$DATA_DIR/repayment/raw"

mkdir -p "$BATTERY_DIR" "$REPAYMENT_DIR"

echo "==> Cyto AI: fetching public bootstrap datasets"
echo "    target: $DATA_DIR"
echo ""

# -----------------------------------------------------------------------------
# 1. NASA PCoE Li-ion Battery Data Set — no auth required, direct download.
#    Charge/discharge/impedance cycles for 18650 Li-ion cells run to failure.
#    ~40-50MB zipped. This is the primary source for the Battery Health Index
#    model — degradation physics (capacity fade vs. cycles/temperature) is
#    chemistry-driven, so it transfers even though the data isn't Kenyan.
# -----------------------------------------------------------------------------
NASA_URL="https://phm-datasets.s3.amazonaws.com/NASA/5.+Battery+Data+Set.zip"
NASA_MARKER="$BATTERY_DIR/.nasa_downloaded"

if [ -f "$NASA_MARKER" ]; then
  echo "==> [1/3] NASA battery dataset already present — skipping"
else
  echo "==> [1/3] Downloading NASA PCoE Li-ion Battery Data Set..."
  NASA_ZIP="$BATTERY_DIR/nasa_battery_dataset.zip"
  if curl -L --fail -o "$NASA_ZIP" "$NASA_URL"; then
    echo "    unzipping..."
    unzip -q -o "$NASA_ZIP" -d "$BATTERY_DIR"
    rm -f "$NASA_ZIP"
    touch "$NASA_MARKER"
    echo "    done -> $BATTERY_DIR"
  else
    echo "    !! download failed. Fetch manually from:"
    echo "       $NASA_URL"
    echo "       and unzip into $BATTERY_DIR"
  fi
fi
echo ""

# -----------------------------------------------------------------------------
# 2. Kaggle: pre-cleaned NASA PCoE CSV — same underlying data as (1), but
#    already flattened into ML-ready CSV instead of raw .mat files. Optional
#    but saves you writing a .mat parser for the MVP. Requires Kaggle API
#    credentials at ~/.kaggle/kaggle.json (free — Account > Settings > API
#    on kaggle.com > "Create New Token").
# -----------------------------------------------------------------------------
if [[ "${1:-}" != "--skip-kaggle" ]]; then
  if command -v kaggle >/dev/null 2>&1; then
    echo "==> [2/3] Downloading Kaggle: cleaned NASA PCoE CSV..."
    kaggle datasets download -d ckskaggle/li-ion-battery-dataset-from-nasa-pcoe \
      -p "$BATTERY_DIR/cleaned" --unzip \
      && echo "    done -> $BATTERY_DIR/cleaned" \
      || echo "    !! skipped (check ~/.kaggle/kaggle.json is set up)"
  else
    cat <<'EOF'
==> [2/3] Kaggle CLI not found — skipping cleaned-CSV download.
    Install with:  pip install kaggle
    Get a token:   kaggle.com -> Account -> Settings -> API -> Create New Token
                   save the downloaded kaggle.json to ~/.kaggle/kaggle.json
    Then re-run this script (or run --skip-kaggle to stop seeing this).
EOF
  fi
  echo ""

  # ---------------------------------------------------------------------------
  # 3. Kaggle: "Give Me Some Credit" — general credit-scoring dataset used to
  #    prototype the Repayment Risk Index feature-engineering approach before
  #    real Kenyan repayment data is available. ~5MB. NOTE: this is a Kaggle
  #    *competition* dataset — you must click "Join Competition" / accept the
  #    rules on the competition page once before the API download will work:
  #    https://www.kaggle.com/c/GiveMeSomeCredit/rules
  # ---------------------------------------------------------------------------
  if command -v kaggle >/dev/null 2>&1; then
    echo "==> [3/3] Downloading Kaggle: Give Me Some Credit (repayment proxy)..."
    if kaggle competitions download -c GiveMeSomeCredit -p "$REPAYMENT_DIR"; then
      unzip -q -o "$REPAYMENT_DIR/GiveMeSomeCredit.zip" -d "$REPAYMENT_DIR"
      rm -f "$REPAYMENT_DIR/GiveMeSomeCredit.zip"
      echo "    done -> $REPAYMENT_DIR"
    else
      echo "    !! skipped — you likely need to accept the competition rules first:"
      echo "       https://www.kaggle.com/c/GiveMeSomeCredit/rules"
    fi
  fi
else
  echo "==> --skip-kaggle passed — skipping [2/3] and [3/3]"
fi

echo ""
echo "==> Public data bootstrap complete. Sizes:"
du -sh "$DATA_DIR"/* 2>/dev/null || true
echo ""
echo "==> Next: generate synthetic Kenya-context data to fill the gaps public"
echo "    datasets don't cover (payment cadence, KES repayment terms, local temps):"
echo "       python scripts/generate_synthetic.py --riders 200 --days 180"