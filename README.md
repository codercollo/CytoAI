# CytoAI

**Explainable risk scoring for IoT-connected, pay-as-you-go asset financing.**

CytoAI combines **asset telematics** and **repayment behavior** into a single, explainable risk signal for lenders and PAYG asset operators.

The MVP focuses on **electric motorcycle batteries in Kenya**, where battery health, usage patterns, swap activity, and repayment behavior can provide complementary signals for financing decisions.

## What It Does

CytoAI turns two data streams into actionable risk signals:

* **Battery Health Index (BHI)** — estimates battery health and remaining life.
* **Repayment Risk Index (RRI)** — estimates repayment risk from payment behavior and usage regularity.
* **CytoScore** — deterministic 0–100 score combining BHI and RRI.
* **Anomaly Flags** — surfaces unusual usage or repayment patterns for human review.

The system supports both major battery-financing models:

| Model                    | Battery Context    | Primary Use             |
| ------------------------ | ------------------ | ----------------------- |
| **Swap network**         | Fleet/pool health  | Operations & fleet risk |
| **Fixed/leased battery** | Individual battery | Asset & credit risk     |

For swap networks, BHI is explicitly returned as `fleet` context so fleet health is never confused with an individual battery's condition.

## Who It's For

### Lenders

Use telematics and repayment behavior as additional signals when evaluating financing risk.

### Asset Operators

Monitor battery health, identify abnormal usage patterns, and determine which assets may need maintenance or removal from circulation.

## Architecture

```text
                 ┌──────────────────┐
                 │ Battery Telemetry │
                 └────────┬─────────┘
                          │
                 ┌────────▼─────────┐
                 │                  │
                 │    CytoAI API    │
                 │                  │
                 └────────┬─────────┘
                          │
          ┌───────────────┼────────────────┐
          │               │                │
          ▼               ▼                ▼
     Battery Health   Repayment Risk   Anomaly Detection
          │               │                │
          └───────────────┼────────────────┘
                          ▼
                     CytoScore
                          │
                 ┌────────┴────────┐
                 ▼                 ▼
              Lenders           Operators
```

The MVP uses public and synthetic proxy datasets with the **same schema intended for partner data**, allowing real telematics and repayment exports to be introduced without changing the core pipeline.

## Tech Stack

| Layer          | Technology                  |
| -------------- | --------------------------- |
| API / Backend  | Go, `net/http`, chi         |
| ML Scoring     | Python, Flask, scikit-learn |
| Database       | PostgreSQL                  |
| Dashboard      | Nuxt 4, Vue 3               |
| Routing        | Nuxt/Nitro `/v1/*` proxy    |
| Infrastructure | Docker Compose              |
| Hosting        | Contabo VPS                 |
| CI/CD          | GitHub Actions              |

## Quick Start

### Prerequisites

* Go
* Docker & Docker Compose
* Python
* PostgreSQL
* Make

Clone and configure:

```bash
git clone https://github.com/<org>/cytoai.git
cd cytoai

cp configs/config.example.yaml configs/config.local.yaml
```

Start the stack:

```bash
make up
```

Fetch the public datasets:

```bash
./scripts/fetch_public_data.sh
```

Initialize the development environment:

```bash
make migrate generate-data train seed-partner
```

The services are available at:

```text
API:       http://localhost:8080
Dashboard: http://localhost:3000
```

For frontend development:

```bash
make dev-web
```

## Repository

```text
cytoai/
├── api/                  # Go API
├── ml/                   # Python scoring service
├── web/                  # Nuxt dashboard
├── db/                   # PostgreSQL schema & migrations
├── configs/              # Application configuration
├── scripts/              # Data and development scripts
├── .github/workflows/    # CI/CD
├── spec.md               # Technical specification
├── mvp.md                # MVP scope
└── project-structure.txt # Annotated repository tree
```

## Data & Model

The MVP is intentionally lightweight:

* Public proxy datasets for initial development
* Synthetic data for controlled scenarios
* Training datasets measured in MBs rather than GBs
* CPU-friendly model training
* Small model artifacts suitable for low-cost infrastructure

Real partner data can replace the proxy datasets while preserving the same ingestion and scoring interfaces.

## Status

**MVP / Hackathon Prototype**

CytoAI is currently a prototype developed for the **Kenya Artificial Intelligence Accelerator Programme — GreenTech & Environmental AI pathway**.

Model performance metrics are based on **synthetic and public proxy data** and should not be interpreted as production performance or validated Kenyan partner results.

## Roadmap

The current MVP focuses on electric motorcycle batteries.

The underlying architecture is designed to potentially extend to other IoT-connected PAYG assets such as:

* Electric tuk-tuks
* E-cabs
* Hired-out tractors
* IoT-metered water infrastructure

These extensions are **future opportunities, not currently supported or validated products**.

## Documentation

* [`spec.md`](spec.md) — Technical specification, data sources, schema, models, API, deployment, and responsible-AI considerations.
* [`mvp.md`](mvp.md) — MVP scope and product examples.
* [`project-structure.txt`](project-structure.txt) — Full annotated repository structure.

## License

**TBD** — proprietary during the accelerator phase.
