"""
Train the Battery Health Index (BHI) model.

Source data: NASA PCoE Li-ion Battery Data Set (.mat files) downloaded by
scripts/fetch_public_data.sh into ml/data/battery/raw/. Falls back to the
Kaggle "cleaned" CSV mirror (ml/data/battery/raw/cleaned/) if present and
the .mat files aren't found or a given .mat file fails to parse — the NASA
archive is not perfectly uniform across batteries (a handful of files use a
slightly different MATLAB struct layout), so per-file failures are skipped
and reported rather than aborting the whole run.

Target: battery State-of-Health (SoH), defined as
    SoH = capacity_at_cycle_n / rated_capacity   (0-1, higher = healthier)
which we scale to a 0-100 Battery Health Index (BHI) at serve time.

Usage:
    python ml/training/train_battery_model.py [--data-dir ml/data/battery/raw]
                                                [--out ml/artifacts/battery_health_model.joblib]
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

import numpy as np
import pandas as pd
from sklearn.ensemble import GradientBoostingRegressor
from sklearn.metrics import mean_absolute_error, r2_score
from sklearn.model_selection import train_test_split

# Feature order is part of the model contract — the Flask sidecar and the
# Go API's request validation both depend on this exact list/order.
FEATURE_NAMES = [
    "cycle_count",
    "avg_depth_of_discharge",
    "avg_temperature_c",
    "age_days",
    "charge_rate_variance",
]
RATED_CAPACITY_AH = 2.0  # NASA PCoE cells are nominally 2Ah


def _load_mat_batteries(raw_dir: Path) -> pd.DataFrame:
    """Parse NASA PCoE .mat files into per-cycle rows. Skips unparseable files."""
    try:
        from scipy.io import loadmat
    except ImportError:
        print("scipy is required to parse .mat files (see ml/requirements.txt)", file=sys.stderr)
        return pd.DataFrame()

    mat_files = sorted(raw_dir.rglob("*.mat"))
    if not mat_files:
        return pd.DataFrame()

    rows = []
    skipped = []
    for mat_path in mat_files:
        try:
            mat = loadmat(str(mat_path), simplify_cells=True)
            # Top-level key is usually the battery id, e.g. "B0005"
            key = next(k for k in mat if not k.startswith("__"))
            cycles = mat[key]["cycle"]
            battery_id = mat_path.stem
            age_start = None
            for i, cyc in enumerate(cycles):
                if cyc.get("type") != "discharge":
                    continue
                data = cyc.get("data", {})
                capacity = data.get("Capacity")
                temps = data.get("Temperature_measured")
                voltage = data.get("Voltage_measured")
                if capacity is None:
                    continue
                capacity = float(np.ravel(capacity)[0]) if np.ndim(capacity) else float(capacity)
                temp_series = np.ravel(temps) if temps is not None else np.array([np.nan])
                volt_series = np.ravel(voltage) if voltage is not None else np.array([np.nan])
                if age_start is None:
                    age_start = i
                rows.append(
                    {
                        "battery_id": battery_id,
                        "cycle_count": i - age_start + 1,
                        "avg_depth_of_discharge": float(
                            np.clip((volt_series.max() - volt_series.min()) / max(volt_series.max(), 1e-6), 0, 1)
                        )
                        if volt_series.size
                        else np.nan,
                        "avg_temperature_c": float(np.nanmean(temp_series)),
                        "age_days": (i - age_start + 1) * 1.0,  # proxy: ~1 cycle/day
                        "charge_rate_variance": float(np.nanvar(volt_series)) if volt_series.size else np.nan,
                        "soh": capacity / RATED_CAPACITY_AH,
                    }
                )
        except Exception as exc:  # noqa: BLE001 - deliberately broad, this is a best-effort batch parse
            skipped.append((mat_path.name, str(exc)))
            continue

    if skipped:
        print(f"Skipped {len(skipped)} unparseable .mat file(s):", file=sys.stderr)
        for name, err in skipped:
            print(f"  - {name}: {err}", file=sys.stderr)

    return pd.DataFrame(rows)


def _load_cleaned_csv(raw_dir: Path) -> pd.DataFrame:
    """Fallback: Kaggle's pre-cleaned CSV mirror of the same NASA dataset."""
    cleaned_dir = raw_dir / "cleaned"
    csvs = sorted(cleaned_dir.rglob("*.csv")) if cleaned_dir.exists() else []
    if not csvs:
        return pd.DataFrame()

    frames = []
    for csv_path in csvs:
        try:
            df = pd.read_csv(csv_path)
        except Exception as exc:  # noqa: BLE001
            print(f"Skipped {csv_path.name}: {exc}", file=sys.stderr)
            continue
        cols = {c.lower(): c for c in df.columns}

        def col(*names):
            for n in names:
                if n in cols:
                    return df[cols[n]]
            return pd.Series(np.nan, index=df.index)

        out = pd.DataFrame(
            {
                "battery_id": csv_path.stem,
                "cycle_count": col("cycle", "cycle_count"),
                "avg_depth_of_discharge": col("depth_of_discharge", "dod"),
                "avg_temperature_c": col("temperature_measured", "temperature", "ambient_temperature"),
                "age_days": col("cycle", "cycle_count"),
                "charge_rate_variance": col("voltage_measured", "voltage").rolling(5, min_periods=1).var(),
                "soh": col("capacity") / RATED_CAPACITY_AH,
            }
        )
        frames.append(out)
    return pd.concat(frames, ignore_index=True) if frames else pd.DataFrame()


def load_training_frame(raw_dir: Path) -> pd.DataFrame:
    df = _load_mat_batteries(raw_dir)
    if df.empty:
        print("No usable .mat files found — falling back to cleaned CSV mirror.", file=sys.stderr)
        df = _load_cleaned_csv(raw_dir)
    if df.empty:
        sample = sorted(p.relative_to(raw_dir) for p in raw_dir.rglob("*") if p.is_file())[:15]
        hint = (
            "\n".join(f"  - {p}" for p in sample)
            if sample
            else "  (directory is empty or does not exist)"
        )
        raise SystemExit(
            f"No battery training data found under {raw_dir} (searched recursively for "
            f"*.mat and cleaned/**/*.csv). Files actually present there (first 15):\n{hint}\n"
            "Run scripts/fetch_public_data.sh first, or check whether the archive extracted "
            "under an unexpected filename/extension."
        )

    df = df.dropna(subset=["soh"])
    for col in FEATURE_NAMES:
        df[col] = df[col].fillna(df[col].median())
    df["soh"] = df["soh"].clip(0, 1.2)
    return df


def train(df: pd.DataFrame) -> tuple[GradientBoostingRegressor, dict]:
    X = df[FEATURE_NAMES]
    y = df["soh"]
    X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=42)

    model = GradientBoostingRegressor(
        n_estimators=150,
        max_depth=3,
        learning_rate=0.05,
        random_state=42,
    )
    model.fit(X_train, y_train)

    y_pred = model.predict(X_test)
    metrics = {
        "mae": float(mean_absolute_error(y_test, y_pred)),
        "r2": float(r2_score(y_test, y_pred)),
        "n_train": int(len(X_train)),
        "n_test": int(len(X_test)),
    }
    return model, metrics


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--data-dir", default="ml/data/battery/raw")
    parser.add_argument("--out", default="ml/artifacts/battery_health_model.joblib")
    args = parser.parse_args()

    import joblib

    raw_dir = Path(args.data_dir)
    df = load_training_frame(raw_dir)
    print(f"Loaded {len(df)} discharge-cycle rows from {raw_dir}")

    model, metrics = train(df)
    print(f"BHI model trained — MAE={metrics['mae']:.4f}  R2={metrics['r2']:.4f}")

    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    artifact = {
        "model": model,
        "feature_names": FEATURE_NAMES,
        "rated_capacity_ah": RATED_CAPACITY_AH,
        "metrics": metrics,
        "model_version": "bhi-v0.1",
    }
    joblib.dump(artifact, out_path)
    print(f"Saved -> {out_path} ({out_path.stat().st_size / 1024:.1f} KB)")

    metrics_path = out_path.with_suffix(".metrics.json")
    metrics_path.write_text(json.dumps(metrics, indent=2))


if __name__ == "__main__":
    main()