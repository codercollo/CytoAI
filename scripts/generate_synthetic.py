"""
Generate synthetic Kenya-context telemetry, repayment, loan, and swap-event
data matching
the Postgres schema in docs/spec.md section 4, for local pipeline testing
before real partner data or the Tier-1/Tier-2 public downloads are available.

Outputs (under ml/data/synthetic/):
    telemetry_readings.csv   -> telemetry_readings table (leased_fixed only)
    repayment_events.csv     -> repayment_events table
    loans.csv                -> loans table (incl. principal_kes /
                                battery_value_kes and financing_model)
    swap_events.csv          -> swap_events table (swap_network riders)

Usage:
    python scripts/generate_synthetic.py --riders 200 --days 180 [--seed 42]
                                         [--swap-ratio 0.7] [--swap-pool-size 60]
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

# --- Swap-network generation (Phase 6) ---------------------------------------
# Documented synthetic correlation: a subset of swap riders are injected as
# "high-stress" — they return batteries hotter and deeper-discharged AND have
# a lower repayment reliability (higher synthetic default rate). This is what
# makes the battery_stress_profile RRI feature and the swap stress anomaly
# rule testable at all in the MVP demo. It is an ASSUMPTION, not a discovered
# real-world pattern — see docs/data-sources.md.
SWAP_STRESS_SHARE = 0.25            # fraction of swap riders injected as high-stress
STRESS_RELIABILITY_ALPHA = 5.0
STRESS_RELIABILITY_BETA = 3.0       # mean ~0.625 vs baseline ~0.80 -> higher default rate
NORMAL_TEMP_MEAN = 25.0
NORMAL_TEMP_SD = 2.5
NORMAL_DOD_MIN = 20.0
NORMAL_DOD_MAX = 80.0
STRESS_TEMP_MEAN = 32.0
STRESS_TEMP_SD = 2.5
STRESS_DOD_MIN = 55.0
STRESS_DOD_MAX = 95.0
SPORADIC_SWAP_SHARE = 0.20          # fraction of swap riders with irregular swap cadence
STATION_COUNT = 20


def generate(riders: int, days: int, seed: int, swap_ratio: float = 0.7, swap_pool_size: int = 60) -> tuple[list[dict], list[dict], list[dict], list[dict]]:
    rng = random.Random(seed)
    start = date(2026, 1, 1)

    telemetry_rows = []
    repayment_rows = []
    loan_rows = []
    swap_rows = []

    # Shared battery pool for swap_network riders: battery_id must repeat
    # across many different riders over time (the whole point of a swap
    # network) rather than being a 1:1 rider:battery mapping.
    battery_pool = [f"battery_pool_{i:03d}" for i in range(swap_pool_size)]
    pool_cycles = {b: 0 for b in battery_pool}

    swap_rider_count = int(round(riders * swap_ratio))

    for r in range(riders):
        rider_id = f"rider_{r:04d}"
        is_swap = r < swap_rider_count
        financing_model = "swap_network" if is_swap else "leased_fixed"
        battery_id = "" if is_swap else f"battery_{r:04d}"
        loan_id = f"loan_{r:04d}"

        # Each rider has a baseline reliability and activity level
        reliability = rng.betavariate(RELIABILITY_ALPHA, RELIABILITY_BETA)  # skewed toward on-time
        activity = rng.uniform(20, 120)  # km/day baseline

        # High-stress swap riders get hotter/deeper battery returns AND a lower
        # repayment reliability (the injected correlation above).
        stress_rider = is_swap and rng.random() < SWAP_STRESS_SHARE
        if stress_rider:
            reliability = rng.betavariate(STRESS_RELIABILITY_ALPHA, STRESS_RELIABILITY_BETA)

        # Swap cadence: most swap riders swap ~daily, a tail is sporadic so the
        # cadence-regularity feature has genuine signal.
        swap_interval = 1.0
        if is_swap and rng.random() < SPORADIC_SWAP_SHARE:
            swap_interval = rng.uniform(2.0, 5.0)

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
                "financing_model": financing_model,
            }
        )

        cycle_count = 0
        for d in range(days):
            reading_date = start + timedelta(days=d)

            # Telemetry: leased_fixed riders only (swap riders produce
            # swap_events instead of per-battery telemetry).
            if not is_swap and rng.random() < 0.85:
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

            # Swap events: swap_network riders draw a shared battery from the
            # pool; returned metrics depend on the stress injection.
            if is_swap and rng.random() < (1.0 / swap_interval):
                pick = rng.choice(battery_pool)
                pool_cycles[pick] += 1
                if stress_rider:
                    temp = round(max(0.0, rng.gauss(STRESS_TEMP_MEAN, STRESS_TEMP_SD)), 1)
                    dod = round(rng.uniform(STRESS_DOD_MIN, STRESS_DOD_MAX), 1)
                else:
                    temp = round(max(0.0, rng.gauss(NORMAL_TEMP_MEAN, NORMAL_TEMP_SD)), 1)
                    dod = round(rng.uniform(NORMAL_DOD_MIN, NORMAL_DOD_MAX), 1)
                swap_rows.append(
                    {
                        "rider_id": rider_id,
                        "battery_id": pick,
                        "station_id": f"ST-{rng.randint(1, STATION_COUNT):03d}",
                        "swapped_at": reading_date.isoformat(),
                        "returned_state_of_charge": round(rng.uniform(10, 60), 1),
                        "returned_temperature_c": temp,
                        "returned_cycle_count": pool_cycles[pick],
                        "returned_depth_of_discharge": dod,
                        "distance_km_since_last_swap": round(max(0.0, rng.gauss(activity, activity * 0.2)), 1),
                    }
                )

    return telemetry_rows, repayment_rows, loan_rows, swap_rows


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
    parser.add_argument("--swap-ratio", type=float, default=0.7)
    parser.add_argument("--swap-pool-size", type=int, default=60)
    args = parser.parse_args()

    print(f"riders={args.riders} days={args.days} seed={args.seed} swap_ratio={args.swap_ratio} swap_pool_size={args.swap_pool_size}")
    telemetry_rows, repayment_rows, loan_rows, swap_rows = generate(
        args.riders, args.days, args.seed, args.swap_ratio, args.swap_pool_size
    )

    telemetry_path = OUT_DIR / "telemetry_readings.csv"
    repayment_path = OUT_DIR / "repayment_events.csv"
    loans_path = OUT_DIR / "loans.csv"
    swaps_path = OUT_DIR / "swap_events.csv"
    write_csv(telemetry_path, telemetry_rows)
    write_csv(repayment_path, repayment_rows)
    write_csv(loans_path, loan_rows)
    write_csv(swaps_path, swap_rows)

    print(f"telemetry: {len(telemetry_rows):,} rows -> {telemetry_path}")
    print(f"repayments: {len(repayment_rows):,} rows -> {repayment_path}")
    print(f"loans: {len(loan_rows):,} rows -> {loans_path}")
    print(f"swaps: {len(swap_rows):,} rows -> {swaps_path}")


if __name__ == "__main__":
    main()