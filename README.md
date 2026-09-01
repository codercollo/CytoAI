# Cyto AI

### Neutral risk scoring for IoT-connected, pay-as-you-go asset financing — starting with electric motorcycle batteries

CytoAI is a neutral risk-scoring layer for IoT-connected, pay-as-you-go (PAYG)
asset financing — starting with electric motorcycle batteries, the
fastest-growing and most underserved slice of a mechanism already proven at
$1.6B+ scale by M-KOPA, SunCulture, and Hello Tractor.

Today it turns two currently disconnected data streams — **battery telematics**
(how hard a battery is being used and how fast it's degrading) and
**repayment history** (how reliably a rider pays) — into a single, explainable
risk score a lender can check **before** disbursing a loan or lease, not after
it defaults.

> Batteries account for 30–40% of an electric motorcycle's total cost, and
> most Kenyan operators (Spiro, Roam, Ampersand, Arc Ride) lease batteries
> separately from the bike to keep upfront prices low. Lenders like Watu,
> Mogo, M-KOPA and 4G Capital are already underwriting thousands of these
> loans — with no shared, data-driven way to price battery degradation risk
> or rider default risk. A 2024 sector financing study by MicroSave
> Consulting explicitly recommended exactly this: digital tools for credit
> appraisal built on telematics data to reduce lender risk.

---

## What it does

1. **Ingests** battery usage data (charge cycles, temperature, depth of
   discharge) and loan repayment history from partner operators/lenders.
2. **Scores** two things with a lightweight ML model:
   - **Battery Health Index (BHI)** — predicted remaining useful life /
     state-of-health of the physical battery.
   - **Repayment Risk Index (RRI)** — probability of default based on usage
     and payment behaviour patterns, including a battery swap/usage cadence
     signal measured from telemetry already collected (no new data needed).
3. **Combines** both into a single **CytoScore** a lender can query via API
   or view on a dashboard before approving or repricing a loan.
4. **Flags anomalies for review** — fraud/anomaly checks on telemetry and repayment data
   surface as a small warning (for example, "state of charge was 140%" or a
   duplicated reading). These are review signals that protect trust in the
   score, never an automatic rejection and never a block on ingestion (the API
   field is `anomaly_flags`).

## Why this MVP is buildable fast

Real telematics + repayment data partnerships with operators take months to
negotiate — too slow for an accelerator MVP deadline. So the MVP is designed
around **small, public, freely downloadable datasets** that stand in for
partner data, using the *exact same schema* real partner data will use. This
means:

- The pipeline, database schema, and model can be built and demoed **now**.
- Swapping in a real partner's CSV export later requires **zero code
  changes** — just a compliant file.
- Training data stays in the **low tens of megabytes**, not gigabytes, so the
  model trains in minutes on a laptop or a small Contabo VPS — no GPU, no
  heavy MLOps.

### Beyond e-bikes (thesis, not built)

The e-boda MVP is the first vertical, not the last. The same mechanism — a
financed, IoT-tracked asset that is how the owner earns income, not just
something they own — extends structurally to other PAYG asset classes. None
of the following are built, validated, or supported today; they are ranked by
structural fit, not by current evidence:

1. **Electric boda bodas (e-motorcycles)** — the current MVP. Individual
   owner-operator, battery is both collateral and the asset that gates daily
   income, deposit + daily-repayment structure, swap-event telemetry already
   present. Still an MVP: prototype metrics only, no signed pilot partner yet
   (see spec.md §9).
2. **Three-wheelers / tuk-tuks** — nearly the same shape (individual
   owner-operator, deposit + daily repayment, vehicle/battery condition caps
   daily earning). The main change is telemetry granularity.
3. **E-cabs on platforms (Bolt/Uber e-motos)** — the e-boda vertical under a
   different distribution channel. Same rider/battery/swap telemetry; the only
   addition would be a "platform" tag on the loan record. No new modeling, but
   not yet validated as a separate vertical.
4. **Hired-out tractors (Hello Tractor model)** — an individual owner books
   the tractor to farmers, usage is IoT-tracked, repayment ties to bookings.
   Real adaptation needed: usage is seasonal, so "regular activity = healthy
   borrower" needs a seasonal baseline instead of a daily one.
5. **IoT-metered water-pump/borehole vending** — the least validated of the
   five. Structurally similar (pump condition and volume sold drive income),
   but not yet backed by a named, funded Kenyan operator the way the others
   are.

What makes these five — not solar panels, phones, or TVs — the credible
extension set: in each, the financed asset is how the person earns, so BHI
(asset health) and RRI (repayment behavior) reinforce each other instead of
RRI doing all the work alone.

See [`spec.md`](spec.md) for full data sourcing, schema, model, and API
details.

## Tech stack

| Layer                | Technology                                             |
| --------------------- | ------------------------------------------------------ |
| API / backend         | Go (net/http + chi router, following Alex Edwards' *Let's Go* project layout and TechSchool's *simplebank* conventions) |
| ML scoring sidecar    | Python / Flask, scikit-learn (gradient boosting, <5MB model artifact) |
| Database              | PostgreSQL                                              |
| Web dashboard         | Nuxt 4 / Vue 3                                          |
| Browser↔API routing   | Nuxt/Nitro same-origin `/v1/*` proxy                    |
| Hosting               | Contabo VPS                                             |
| CI/CD                 | GitHub Actions (`.github/workflows/test.yml`, `deploy.yml`) |
| Containers            | Docker Compose (`postgres`, `api`, `web`)              |

## Quick start (local dev)

```bash
# 1. Clone and configure
git clone https://github.com/<org>/cytoai.git
cd cytoai
cp configs/config.example.yaml configs/config.local.yaml

# 2. Build + start the stack (postgres, api, web)
make up

# 3. Pull the public bootstrap datasets (small, MBs not GBs)
./scripts/fetch_public_data.sh

# 4. Migrate, generate synthetic data, train, and seed a demo partner
make migrate generate-data train seed-partner

# 5. Open the dashboard (served at http://localhost:3000), or run it locally
#    against the containerized api backend:
make dev-web
```

The API will be available at `http://localhost:8080` and the dashboard at
`http://localhost:3000`.

## Repository layout

See [`project-structure.txt`](project-structure.txt) for the full annotated
tree.

## Documents

- [`spec.md`](spec.md) — full technical specification: data sourcing, schema,
  model design, API contract, deployment, and responsible-AI notes.
- [`mvp.md`](mvp.md) — plain-language MVP scope and dashboard examples.

## Status

**MVP / hackathon prototype**, built for the Kenya Artificial Intelligence
Accelerator Programme (GreenTech & Environmental AI pathway). Trained
initially on public proxy datasets; designed for a direct swap to real
partner telematics and repayment data with no architecture changes.

> **Prototype metrics caveat:** all model performance figures (including
> ROC-AUC) are measured on synthetic/public proxy data, not live Kenyan
> partner data, and are not production claims.

## License

TBD — proprietary during accelerator phase.
