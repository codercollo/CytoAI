# Data sources

The MVP trains on three tiers, described in `spec.md` §3:

1. **Public battery-degradation data** (NASA PCoE, CALCE, Kaggle RUL) for the
   Battery Health Index (BHI) model.
2. **Public credit-risk proxy data** (Kaggle "Give Me Some Credit", Zindi,
   FSD Kenya/CBK FinAccess) for the Repayment Risk Index (RRI) model.
3. **Locally generated synthetic Kenya-context data** (`ml/data/synthetic/`)
   that mimics real partner exports.

### Locally generated swap_events (Phase 6)

`scripts/generate_synthetic.py` now also writes `ml/data/synthetic/swap_events.csv`
(matching the `swap_events` table exactly). It models a shared battery pool
(default 60 batteries) rotating across the synthetic swap_network riders, so
`battery_id` deliberately repeats across many riders over time.

**Injected correlation (documented ASSUMPTION, not an observed pattern):**
~25% of swap_network riders (`SWAP_STRESS_SHARE`) are generated as "high-stress":
they return batteries hotter (mean ~32°C vs ~25°C) and deeper-discharged
(55–95% vs 20–80% DoD) AND are assigned a lower repayment reliability
(beta(5,3), mean ~0.625, vs the baseline beta(8,2), mean ~0.80). This is what
makes the `battery_stress_profile` RRI feature and the swap stress anomaly rule
testable in the MVP demo. Re-validate against real partner data before trusting
the trained weight — see `spec.md` §9.

The swap-cadence feature (`telemetry_cadence_proxy`) is derived from
`telemetry_readings.csv` for leased_fixed riders and from `swap_events.csv` for
swap_network riders (single source of truth: `ml/app/services/scoring_service.py`).
Anomaly-detection thresholds are sanity bounds calibrated against the synthetic
ranges (Kenyan climate norms ~18–31°C), not a new external corpus.
