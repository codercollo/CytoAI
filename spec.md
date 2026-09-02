# Cyto AI — Technical Specification (MVP)

Status: Draft v0.1 — Kenya AI Accelerator submission
Owner: [team]
Last updated: 2026-08-30

---

## 1. Problem statement

CytoAI is a neutral risk-scoring layer for IoT-connected, pay-as-you-go (PAYG)
asset financing — a mechanism already proven at scale in Kenya: M-KOPA has
disbursed $1.6B+ in credit in Kenya alone, and SunCulture and Hello Tractor
run the same "asset is how the customer earns" model. CytoAI's first vertical
is electric motorcycle batteries — the fastest-growing, most underserved slice
of that mechanism.

Kenyan lenders and BaaS operators (Watu, Mogo, M-KOPA, 4G Capital, Roam,
Spiro, Ampersand, Arc Ride) are underwriting a fast-growing volume of
electric-motorcycle loans and battery leases, but have no shared, data-driven
way to answer two questions before money moves:

1. **How much life is left in the battery** — for `leased_fixed` loans it is
   the rider's collateral and the single most expensive component of the
   vehicle; for `swap_network` fleets it is the operator's pool asset that
   must be kept healthy and in rotation.
2. **How likely is this rider to keep paying** (most riders are
   informal-economy, thin-file borrowers with no traditional credit history)?

Swap networks have become the dominant distribution model and are therefore
the **primary design target**. Spiro — the category leader — accounts for
~60% of new electric-motorcycle sales in Kenya, operates 450+ battery-swap
stations, and has processed 6M+ battery swaps. In that model the battery is
not the rider's collateral, so the scoring question splits in two: fleet/pool
health for the operator, and per-rider usage stress for the lender. The
`leased_fixed` model (one battery as collateral) remains fully supported but
is **secondary**.

Industry advisory research on this exact sector (MicroSave Consulting, 2024)
recommends digital tools that use telematics data for credit appraisal and
default reduction — but no shared tool exists yet. Cyto AI is that tool,
offered as an API so any lender or operator can query a combined risk score.

Everything that follows describes the electric-motorcycle-battery MVP only.
The broader PAYG vertical thesis is future direction, not current scope — see
§11.

## 2. MVP scope

Electric boda boda batteries are vertical one of a broader PAYG thesis (§11);
the scope below is limited to that vertical and does not expand to the other
verticals in this document.

In scope for the hackathon/accelerator MVP:

- A working ingestion pipeline for three data types: battery telemetry,
  repayment events, and swap events (for swap networks).
- Two small, fast-training ML models (battery health, repayment risk) that
  combine into one score.
- A REST API (`/v1/score`) a partner can call.
- A minimal Nuxt dashboard to browse scores and manually upload CSVs.
- Bootstrapped on **public, downloadable datasets** standing in for real
  partner data (see §3), so the system is demoable without a live data
  partnership in place.

Out of scope for MVP (roadmap): live telematics streaming (MQTT/IoT
ingestion), automated lock-out/recovery integration, multi-tenant billing,
model explainability UI.

## 3. Data strategy — how data gets into the system

This is the part that has to work on day one without a signed data-sharing
agreement from a lender. The design uses three tiers, all pointed at the
**same database schema**, so tier 3 (real partner data) is a drop-in
replacement for tiers 1–2 with no code changes.

### Tier 1 — Public battery-degradation datasets (downloadable, small)

Used to train the **Battery Health Index (BHI)** model on real lithium-ion
cycle-life physics (charge/discharge curves, capacity fade, temperature
effects), since Kenya-specific telematics isn't available yet.

| Source                                                        | What it contains                                         | Approx. size                                                        |
| ------------------------------------------------------------- | -------------------------------------------------------- | ------------------------------------------------------------------- |
| NASA Prognostics Center of Excellence (PCoE) Battery Data Set | Li-ion cell charge/discharge/impedance cycles to failure | Individual battery sets ~10–50 MB, freely downloadable as .mat/.csv |
| CALCE Battery Research Group (University of Maryland)         | Li-ion cycle-life and capacity-fade datasets             | Tens of MB, CSV/Excel, public                                       |
| Public Kaggle "Battery Remaining Useful Life" datasets        | Pre-cleaned cycle-life data for RUL prediction           | A few MB                                                            |

`scripts/fetch_public_data.sh` pulls a curated subset of these (not the full
archives) — target total under ~50 MB, enough for a lightweight prototype
model, not a production-scale one.

### Tier 2 — Public credit-risk proxy datasets (downloadable, small)

Used to prototype the **Repayment Risk Index (RRI)** model's feature
engineering and calibration approach before real Kenyan repayment data is
available.

| Source                                                              | What it contains                                                                      | Approx. size                                       |
| ------------------------------------------------------------------- | ------------------------------------------------------------------------------------- | -------------------------------------------------- |
| Kaggle "Give Me Some Credit" / similar open credit-scoring datasets | Borrower features + default labels                                                    | ~5–15 MB CSV                                       |
| Zindi Africa fintech/credit challenge datasets                      | Africa-specific microfinance/mobile-money repayment data, several with Kenyan context | Typically a few MB to low tens of MB per challenge |
| FSD Kenya / CBK FinAccess Household Survey data                     | Kenya financial-inclusion survey microdata, publicly downloadable                     | Tens of MB, filterable to relevant columns         |

### Tier 3 — Synthetic Kenya-context data (generated locally, tiny)

`scripts/generate_synthetic.py` generates a small synthetic dataset that
mimics what real partner exports will look like: rider usage patterns (daily
distance, payment cadence), battery temperature exposure calibrated to Kenyan
climate norms (~18–31°C), and repayment schedules matching known local terms
(e.g., the Roam × 4G Capital structure: KSh 25,000 deposit + KSh 460/day over
24 months). This serves two purposes:

- Fills gaps where public datasets don't map cleanly to the Kenyan BaaS
  context (e.g., payment-cadence-as-usage/income-proxy).
- Produces the exact CSV schema partner data must match, so the ingestion
  pipeline and dashboard can be fully demoed end-to-end before a single real
  partner file arrives.

### Tier 4 (post-MVP) — Real partner data

Once a pilot partner (a lender or BaaS operator) is signed, they submit data
in one of two ways, both hitting the same tables:

1. **CSV upload** via the dashboard (`/web/pages/upload.vue`) or
   `POST /v1/telematics` / `POST /v1/repayments` — for partners doing daily or
   weekly batch exports. This is the primary MVP path: lightweight, no
   integration work required from the partner, works with data they already
   export from their loan management systems.
2. **API push** via partner API key for partners with live systems — same
   payload shape as the CSV, just JSON over HTTPS.

No GB-scale data is required at any stage: a meaningful pilot with a few
hundred riders and a year of daily readings is on the order of a few MB to
low tens of MB, well within what a lightweight gradient-boosted model needs
to train usefully.

## 4. Data schema (Postgres)

```sql
-- Core entities
CREATE TABLE riders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_ref TEXT,                 -- partner's own rider ID
    partner_id UUID NOT NULL REFERENCES partners(id),
    onboarded_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE batteries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_ref TEXT,
    manufacturer TEXT,
    rated_capacity_wh NUMERIC,
    commissioned_at TIMESTAMPTZ
);

CREATE TABLE telemetry_readings (
    id BIGSERIAL PRIMARY KEY,
    battery_id UUID NOT NULL REFERENCES batteries(id),
    reading_at TIMESTAMPTZ NOT NULL,
    state_of_charge NUMERIC,           -- %
    voltage NUMERIC,
    temperature_c NUMERIC,
    cycle_count INT,
    depth_of_discharge NUMERIC,        -- %
    distance_km_since_last NUMERIC
);

-- Swap-network telemetry: one row per physical battery swap (Spiro/Ampersand
-- style). battery_id here is "which unit this rider was carrying", NOT
-- collateral — it can differ swap to swap.
CREATE TABLE swap_events (
    id BIGSERIAL PRIMARY KEY,
    rider_id UUID NOT NULL REFERENCES riders(id),
    battery_id UUID NOT NULL REFERENCES batteries(id),
    station_id TEXT,
    swapped_at TIMESTAMPTZ NOT NULL,
    returned_state_of_charge NUMERIC,
    returned_temperature_c NUMERIC,
    returned_cycle_count INT,
    returned_depth_of_discharge NUMERIC,
    distance_km_since_last_swap NUMERIC
);

CREATE TABLE loans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_ref TEXT,
    rider_id UUID NOT NULL REFERENCES riders(id),
    battery_id UUID REFERENCES batteries(id),  -- NULL for swap_network (no collateral unit)
    principal_kes NUMERIC,
    battery_value_kes NUMERIC,          -- collateral value (lease); fleet unit value proxy (swap)
    term_months INT,
    daily_installment_kes NUMERIC,
    started_at DATE,
    financing_model TEXT NOT NULL DEFAULT 'leased_fixed'
        CHECK (financing_model IN ('leased_fixed','swap_network'))
);

CREATE TABLE repayment_events (
    id BIGSERIAL PRIMARY KEY,
    loan_id UUID NOT NULL REFERENCES loans(id),
    due_date DATE NOT NULL,
    paid_date DATE,
    amount_due_kes NUMERIC,
    amount_paid_kes NUMERIC,
    status TEXT CHECK (status IN ('on_time','late','missed','partial'))
);

-- Output
CREATE TABLE scores (
    id BIGSERIAL PRIMARY KEY,
    rider_id UUID REFERENCES riders(id),
    battery_id UUID REFERENCES batteries(id),
    battery_health_index NUMERIC,      -- 0-100
    repayment_risk_index NUMERIC,      -- 0-100 (lower = safer)
    cyto_score NUMERIC,               -- combined 0-100
    scored_at TIMESTAMPTZ DEFAULT now(),
    model_version TEXT
);

-- Fraud/anomaly findings (non-gating; Phase 2). One generic table flags either
-- a telemetry_reading or a repayment_event via (entity_type, entity_id).
CREATE TABLE anomaly_flags (
    id BIGSERIAL PRIMARY KEY,
    entity_type TEXT NOT NULL CHECK (entity_type IN ('telemetry_reading','repayment_event')),
    entity_id TEXT NOT NULL,
    flagged_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    rule_or_model TEXT NOT NULL,
    severity TEXT NOT NULL,
    reason TEXT NOT NULL,
    resolved BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE partners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    api_key_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);
```

CSV ingestion maps directly onto `telemetry_readings`, `repayment_events`,
and (for swap operators) `swap_events` — the files a partner needs to export
are flat versions of these tables, keyed by their own `external_ref` IDs.
Scoring branches on `loans.financing_model`: `leased_fixed` scores the rider's
own battery from `telemetry_readings`; `swap_network` scores fleet-level BHI
from aggregated `swap_events` and per-rider stress from the rider's own
`swap_events` (see §5).

## 5. Model design (lightweight, fast to prototype)

Deliberately simple models, chosen for MVP speed and interpretability over
raw accuracy — both trainable in minutes on a laptop, both under 5 MB as
serialized artifacts.

**Battery Health Index (BHI)**

- Model: scikit-learn GradientBoostingRegressor (shallow: n_estimators=150,
  max_depth=3)
- Features: cycle count, average depth of discharge, average temperature,
  age (cycle-count proxy in the current artifact), charge-rate variance
- Two BHI scopes, one model:
  - **Per-rider BHI (`leased_fixed`)** — remaining life of the single
    collateral battery, from its own `telemetry_readings`.
  - **Fleet-level BHI (`swap_network`)** — health of the pool a rider draws
    from, aggregated from the operator's `swap_events`. This is an
    operator/fleet-health signal, not a claim about any one rider's
    collateral, and is returned as `bhi_context: fleet` so the API/UI never
    present it interchangeably with per-rider BHI.
- Training-time aggregation (`aggregate_fleet_bhi()`): Phase 6 trains the
  fleet-BHI path by aggregating each battery's `swap_events` into the same
  five BHI features the per-rider model uses (latest cycle count, mean
  depth-of-discharge, mean temperature, swap-history age, charge-rate
  variance = 0 because swap_events carry no voltage snapshot). The serving
  path mirrors this in `internal/domain/swap_features.go`
  (`SwapBatteryBhiFeaturesFrom`), so inference stays consistent with training.
- Target: capacity fade % / estimated remaining cycles (from Tier-1 public
  cycle-life datasets; transferable because degradation physics is
  chemistry-driven, not geography-driven — the Kenya-specific overlay is
  temperature exposure)

**Repayment Risk Index (RRI)**

- Model: Logistic regression baseline (scikit-learn, class-weighted),
  upgrade path to gradient boosting once real repayment volume exists
- Features:
  - on-time-payment ratio
  - average days late
  - payment-cadence proxy (a usage/income proxy derived from repayment
    due/paid-date cadence — NOT telemetry-confirmed battery swaps)
  - loan-to-battery-value ratio (`principal_kes / battery_value_kes`)
  - tenure
  - telemetry-cadence proxy (swap/usage regularity: `0.5 * CV(inter-reading
gaps) + 0.5 * CV(distance_km_since_last)`; computed in
    `ml/app/services/scoring_service.py` and mirrored in
    `internal/domain/features.go`)
- Target: probability of 2+ fully-missed payments in the trailing 30% of
  each loan's repayment history (not a single late day)

For `swap_network` loans, the cadence signal above is computed from
`swap_events` timestamps instead of `telemetry_readings`, and an additional
RRI input — `battery_stress_profile` — measures how much hotter and more
deeply discharged a rider returns batteries versus the fleet average. This
stress signal is a HYPOTHESIS to validate against real repayment data once a
pilot exists, not a proven correlation.

**CytoScore (combined)**

- Deterministic, explainable weighting (not a black-box combiner) so lenders
  can see why a score landed where it did:
  `CytoScore = 0.5 * (100 - RRI) + 0.5 * BHI` (weights configurable per
  partner risk appetite; documented in `internal/risk/scoring.go`)

**Fraud / anomaly detection**

- Approach: rules-based, not a learned model. Every flag must be explainable
  to a lender/regulator ("state of charge was 140%", "cycle count went
  backwards"), which rules give directly; an isolation-forest-style model
  would output a score but not an auditable reason (see §9).
- Telemetry rules: state-of-charge out of physical bounds, temperature out of
  physical bounds, cycle-count regression, implausible SoC jump, duplicate/
  replayed readings.
- Repayment rules: negative amounts, `missed` with a paid date, `on_time`
  with a late paid date, `partial` paid >= due, duplicate events.
- Output: `anomaly_flags` (`entity_type`, `entity_id`, `rule_or_model`,
  `severity`, `reason`) returned alongside scores — never a rejection.

## 6. API contract (summary — full detail in `api/openapi.yaml`)

| Endpoint                | Method | Purpose                                                        |
| ----------------------- | ------ | -------------------------------------------------------------- |
| `/v1/telematics`        | POST   | Ingest battery telemetry (CSV or JSON batch)                   |
| `/v1/repayments`        | POST   | Ingest repayment events (CSV or JSON batch)                    |
| `/v1/riders`            | GET    | List the partner's riders                                      |
| `/v1/riders`            | POST   | Register a rider (with optional battery/loan)                  |
| `/v1/riders/{rider_id}` | GET    | Get a rider's detail, latest score, factors, and anomaly flags |
| `/v1/score/{rider_id}`  | GET    | Get latest CytoScore for a rider/battery pair                  |
| `/v1/score`             | POST   | Request an on-demand recompute for a rider                     |
| `/v1/portfolio`         | GET    | Partner's full portfolio of scores (dashboard use)             |
| `/healthz`, `/readyz`   | GET    | Liveness/readiness for deploy pipeline                         |

Auth: partner API key in `Authorization: Bearer <key>` header, hashed at
rest, scoped to that partner's own riders/batteries.

Response fields added with the swap-cadence + anomaly work (mirrored in
`api/openapi.yaml`):

- `POST /v1/score`, `GET /v1/score/{rider_id}`, and `GET /v1/portfolio`
  score objects include `anomaly_flags: []` (empty when none).
- `GET /v1/riders/{rider_id}` includes `anomaly_flags` and `factors`, and
  `factors.repayment` now includes `telemetry_cadence_proxy`.

## 7. Architecture

```
                  ┌──────────────┐
                  │  Nuxt 4 Web  │  (dashboard + CSV upload)
                  └──────┬───────┘
                         │ same-origin /v1 proxy (Nitro)
                         ▼
                  ┌──────────────┐
                  │   Go API     │  (cmd/cytoai)
                  │ ingestion +  │
                  │ auth + score │
                  │ orchestration│
                  └──────┬───────┘
                 ┌────────┴────────┐
                 ▼                 ▼
         ┌──────────────┐   ┌──────────────┐
         │  PostgreSQL  │   │ Flask ML     │
         │  (data +     │   │ sidecar      │
         │   scores)    │   │ (ml/)        │
         └──────────────┘   └──────────────┘
```

- **Go API** owns auth, validation, storage, and orchestration; calls the ML
  sidecar over an internal HTTP loopback call (supervisord co-locates both in
  the `api` container) for inference and anomaly detection.
- **Flask sidecar** is co-located with the Go API in the same container and
  keeps the Python/scikit-learn stack independently iterable.
- **Nuxt/Nitro** serves the dashboard and proxies same-origin `/v1/*` requests
  to the Go API, eliminating browser CORS. At scoring time the Go API also
  calls the sidecar's `POST /predict/anomaly` route; flags are returned
  alongside the score (non-gating).

## 8. Deployment

- **Host**: single Contabo VPS for MVP (vertical scale later if needed).
- **Containers**: `build/docker-compose.yml` runs three services: `postgres`,
  `api` (Go API + Flask ML sidecar co-located under supervisord), and `web`.
- **CI/CD**:
  - `.github/workflows/test.yml` — runs on every PR: `go test ./...`,
    `pytest ml/app/tests`, `npm run lint && npm run build` for web.
  - `.github/workflows/deploy.yml` — on merge to `main`: builds and pushes
    Docker images, SSHes into the Contabo VPS, pulls images, restarts via
    `docker compose up -d`, runs pending migrations.
- **Secrets**: partner API keys and DB credentials stored as GitHub Actions
  secrets and injected as env vars — never committed.

## 9. Responsible AI notes

- **Prototype metrics are not production claims.** Every performance figure
  reported for the MVP models (including ROC-AUC) is measured on synthetic
  and/or public proxy data, not live Kenyan partner data, and must not be
  read as real-world accuracy. Treat every score as a hackathon/accelerator
  prototype, not a deployed risk model.
- **Synthetic feature weights are hypotheses, not evidence.** The
  `battery_stress_profile` RRI feature (a rider returns batteries hotter and
  more deeply discharged than the fleet average) is trained on a synthetic,
  assumed correlation with repayment risk; it must be re-evaluated against
  real repayment outcomes before it is trusted at production confidence.
- **Population-shift and version-boundary disclosure.** The
  `rri-v0.3` → `rri-v0.4-lease` transition reflects both a feature-pipeline
  change and a training-population-size change, so scores from the two
  versions are not directly comparable, and a lender relying on score
  trends across that version boundary should treat it as a reset point, not
  a continuous series.
- **No autonomous credit decisions.** CytoScore returns a score and its
  contributing factors; the lender's own underwriting process makes the
  final call. This is stated in the API response and in partner terms.
- **Anomaly flags inform, never reject.** Fraud/anomaly detection returns
  `anomaly_flags` with a severity and reason; these are surfaced for lender
  review and never block ingestion or scoring, so a flagged reading or event
  still flows into the human-recourse process below.
- **Explainability over accuracy.** Simple, auditable models (logistic
  regression, shallow gradient boosting with documented feature weights)
  are chosen deliberately over opaque deep models so a lender — and a
  regulator — can see why a score landed where it did.
- **Data privacy.** Rider data is pseudonymized at ingestion (partner's own
  `external_ref`, not national ID); CytoScore never stores or requires
  national ID, phone number, or other PII beyond what's needed for scoring.
- **Bias monitoring.** Because thin-file, informal-economy riders are the
  target beneficiary group, the team will explicitly monitor for the model
  penalizing sparse data (new riders) rather than genuine risk signals, and
  will document a minimum-data-before-scoring threshold to avoid unfairly
  scoring new riders as high-risk purely for lack of history.
- **Human recourse.** Partners are required (contractually, for pilot
  partners) to offer riders a manual review path if a score contributes to
  a loan decline.

## 10. Milestones (accelerator timeline)

| Milestone                                                             | Target   |
| --------------------------------------------------------------------- | -------- |
| Public data pipeline + schema + synthetic generator working           | Week 1   |
| BHI + RRI prototype models trained on public/synthetic data           | Week 2   |
| Go API + Postgres + Flask sidecar integrated, deployed to Contabo     | Week 3   |
| Nuxt dashboard with CSV upload + score view                           | Week 4   |
| First pilot partner data-sharing conversation (Watu/Mogo/M-KOPA/Roam) | Post-MVP |

## 11. Future verticals (not in MVP scope)

The mechanism the e-boda MVP demonstrates — a financed, IoT-tracked asset that
is how the owner earns, not just something they own — extends structurally to
other PAYG asset classes. These are a thesis, not a capability: none are
built, validated, or supported today, and they are ranked by structural fit,
not by current evidence.

1. **Electric boda bodas (e-motorcycles)** — the current MVP vertical.
   Individual owner-operator; the battery is both collateral and the asset
   that directly gates daily income; deposit + daily-repayment structure;
   swap-event telemetry already present. Caveats: prototype metrics only and
   no signed pilot partner yet (§9).
2. **Three-wheelers / tuk-tuks** — nearly the same shape: individual
   owner-operator, deposit + daily repayment, and vehicle/battery condition
   caps daily earning capacity. Financing structure (Watu/Mogo-style asset
   finance) is nearly identical; the real change is telemetry granularity.
3. **E-cabs on platforms (Bolt/Uber e-motos)** — the current vertical under a
   different distribution channel. Same rider, same battery, same swap
   telemetry, same financier type (M-KOPA × Bolt already does this); the only
   addition would be a "platform" tag on the loan record. No new modeling,
   but not yet validated as a separate vertical.
4. **Hired-out tractors (Hello Tractor model)** — an individual owner earns by
   booking the tractor to farmers, usage is IoT-tracked, and repayment ties to
   bookings. Income-generating like e-boda. Real adaptation: usage is seasonal
   (planting/harvest), so "regular activity = healthy borrower" needs a
   seasonal baseline instead of a daily one.
5. **IoT-metered water-pump/borehole vending** — an individual vendor sells
   metered water; pump condition and volume sold drive income directly. Least
   validated of the five: a logical structural match from the PAYG pattern,
   not backed by a named, funded Kenyan operator the way the other four are.

The shared property that makes these five credible (and excludes solar panels,
phones, and TVs): in each, the financed asset is how the person earns, so BHI
(asset health) and RRI (repayment behavior) reinforce each other instead of
RRI carrying the risk signal alone.
