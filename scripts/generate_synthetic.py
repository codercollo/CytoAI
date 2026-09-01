"""
Generate synthetic Kenya-context telemetry, repayment, and loan data matching
the Postgres schema in docs/spec.md section 4, for local pipeline testing
before real partner data or the Tier-1/Tier-2 public downloads are available.

Outputs (under ml/data/synthetic/):
    telemetry_readings.csv   -> telemetry_readings table
    repayment_events.csv     -> repayment_events table
    loans.csv                -> loans table (incl. principal_kes /
                                battery_value_kes for the RRI model's
                                loan-to-battery-value feature)

Usage:
    python scripts/generate_synthetic.py --riders 200 --days 180 [--seed 42]
"""
from __future__ import annotations

import argparse
import csv
import random
from datetime import date, timedelta
from pathlib import Path

OUT_DIR = Path(__file__).resolve().parent.parent / "ml" / "data" / "synthetic"

# Roam x 4G Capital structure used as the default synthetic loan term
DEPOSIT_KES = 25_000
DAILY_INSTALLMENT_KES = 460
TERM_MONTHS = 24

# Collateral / financing ranges used to make loan_to_battery_value_ratio a
# real, varying ratio (principal_kes / battery_value_kes) rather than a
# self-referential accounting identity. Kenya e-moto batteries are typically
# KES 120k-180k; financed bike+battery principal typically KES 220k-340k.
BATTERY_VALUE_KES_MIN = 120_000
BATTERY_VALUE_KES_MAX = 180_000
PRINCIPAL_KES_MIN = 220_000
PRINCIPAL_KES_MAX = 340_000

# Repayment behaviour calibration (see Phase-1 diagnosis): a beta(8,2)
# "reliability" distribution (mean ~0.80) keeps most riders mostly on-time,
# while only a small tail of due dates becomes a fully-missed payment. This
# makes the future-default label in train_repayment_model.py (>=2 missed
# payments in the trailing window) land in a plausible microfinance range
# (~5-25%) instead of the old ~97%.
RELIABILITY_ALPHA = 8.0
RELIABILITY_BETA = 2.0
LATE_SHARE = 0.75     # of the non-on-time mass: a few days late (transient)
PARTIAL_SHARE = 0.17  # of the non-on-time mass: partial payment
# MISSED_SHARE = 1 - LATE_SHARE - PARTIAL_SHARE = 0.08  (fully missed day)


def generate(riders: int, days: int, seed: int) -> tuple[list[dict], list[dict], list[dict]]:
    rng = random.Random(seed)
    start = date(2026, 1, 1)

    telemetry_rows = []
    repayment_rows = []
    loan_rows = []

    for r in range(riders):
        rider_id = f"rider_{r:04d}"
        battery_id = f"battery_{r:04d}"
        loan_id = f"loan_{r:04d}"

        # Each rider has a baseline reliability and activity level
        reliability = rng.betavariate(RELIABILITY_ALPHA, RELIABILITY_BETA)  # skewed toward on-time
        activity = rng.uniform(20, 120)  # km/day baseline

        # Loan/collateral values for the real loan-to-battery-value ratio.
        battery_value_kes = round(rng.uniform(BATTERY_VALUE_KES_MIN, BATTERY_VALUE_KES_MAX), -3)
        principal_kes = round(rng.uniform(PRINCIPAL_KES_MIN, PRINCIPAL_KES_MAX), -3)
        loan_rows.append(
            {
                "loan_id": loan_id,
                "rider_id": rider_id,
                "battery_id": battery_id,
                "principal_kes": principal_kes,
                "battery_value_kes": battery_value_kes,
                "term_months": TERM_MONTHS,
                "daily_installment_kes": DAILY_INSTALLMENT_KES,
                "started_at": start.isoformat(),
            }
        )

        cycle_count = 0
        for d in range(days):
            reading_date = start + timedelta(days=d)

            # Telemetry: not every rider rides every day
            if rng.random() < 0.85:
                cycle_count += 1
                distance_km = max(0.0, rng.gauss(activity, activity * 0.2))
                telemetry_rows.append(
                    {
                        "battery_id": battery_id,
                        "reading_at": reading_date.isoformat(),
                        "state_of_charge": round(rng.uniform(15, 100), 1),
                        "voltage": round(rng.uniform(48, 58), 2),
                        "temperature_c": round(rng.uniform(18, 31), 1),  # Kenyan climate norms
                        "cycle_count": cycle_count,
                        "depth_of_discharge": round(rng.uniform(20, 90), 1),
                        "distance_km_since_last": round(distance_km, 1),
                    }
                )

            # Repayment: daily installment due, status depends on rider reliability
            due_date = reading_date
            roll = rng.random()
            if roll < reliability:
                status, days_late = "on_time", 0
            elif roll < reliability + (1 - reliability) * LATE_SHARE:
                status, days_late = "late", rng.randint(1, 10)
            elif roll < reliability + (1 - reliability) * (LATE_SHARE + PARTIAL_SHARE):
                status, days_late = "partial", rng.randint(0, 5)
            else:
                status, days_late = "missed", None

            paid_date = "" if status == "missed" else (due_date + timedelta(days=days_late or 0)).isoformat()
            amount_due = DAILY_INSTALLMENT_KES
            amount_paid = 0 if status == "missed" else (amount_due // 2 if status == "partial" else amount_due)

            repayment_rows.append(
                {
                    "loan_id": loan_id,
                    "rider_id": rider_id,
                    "battery_id": battery_id,
                    "due_date": due_date.isoformat(),
                    "paid_date": paid_date,
                    "amount_due_kes": amount_due,
                    "amount_paid_kes": amount_paid,
                    "status": status,
                }
            )

    return telemetry_rows, repayment_rows, loan_rows


def write_csv(path: Path, rows: list[dict]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=list(rows[0].keys()))
        writer.writeheader()
        writer.writerows(rows)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--riders", type=int, default=200)
    parser.add_argument("--days", type=int, default=180)
    parser.add_argument("--seed", type=int, default=42)
    args = parser.parse_args()

    print(f"riders={args.riders} days={args.days} seed={args.seed}")
    telemetry_rows, repayment_rows, loan_rows = generate(args.riders, args.days, args.seed)

    telemetry_path = OUT_DIR / "telemetry_readings.csv"
    repayment_path = OUT_DIR / "repayment_events.csv"
    loans_path = OUT_DIR / "loans.csv"
    write_csv(telemetry_path, telemetry_rows)
    write_csv(repayment_path, repayment_rows)
    write_csv(loans_path, loan_rows)

    print(f"telemetry: {len(telemetry_rows):,} rows -> {telemetry_path}")
    print(f"repayments: {len(repayment_rows):,} rows -> {repayment_path}")
    print(f"loans: {len(loan_rows):,} rows -> {loans_path}")


if __name__ == "__main__":
    main()