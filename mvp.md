# Cyto AI — MVP Scope

Cyto is starting with e-boda batteries as the first vertical of a broader
PAYG asset-financing thesis — see spec.md §11 for the full picture. This
document describes only the e-boda MVP as it exists today.

## 1. Data Ingestion

The platform accepts two core data sources:

- **Repayment events** — loan payments, due dates, payment status, and payment delays.
- **Battery telemetry** — state of charge, voltage, temperature, cycle count, depth of discharge, and usage data.

Data is uploaded through CSV files and stored in **PostgreSQL** using the same schema that will later support real partner data.

---

## 2. AI / ML Engine

Cyto uses two lightweight machine-learning models that work together.

### Repayment Risk Index (RRI)

The RRI model analyzes historical repayment and rider behavior to estimate the probability of future repayment problems.

**Output:**

```text
RRI = 0–100
```

Where:

- **0 = lowest repayment risk**
- **100 = highest repayment risk**

### Battery Health Index (BHI)

The BHI model analyzes battery telemetry and usage patterns to estimate the battery's current state of health.

**Output:**

```text
BHI = 0–100
```

Where:

- **0 = severely degraded**
- **100 = excellent battery health**

Cyto also flags data anomalies for a lender to review — for example, a battery
reading that looks physically impossible or a duplicated payment. These flags
(the API calls them `anomaly_flags`) are **review signals, not rejections**.

---

## 3. CytoScore

Cyto combines the two model outputs into a single, explainable risk score.

```text
CytoScore =
0.5 × (100 − RRI)
+
0.5 × BHI
```

The resulting score ranges from **0–100**, with higher scores representing a stronger overall risk profile.

### Risk bands

```text
80–100  → Low Risk
60–79   → Medium Risk
0–59    → High Risk
```

The weighting can later be configured according to a partner's risk appetite.

---

## 4. API

Cyto exposes the scoring engine through a REST API.

```text
POST /v1/telematics
```

Ingest battery telemetry.

```text
POST /v1/repayments
```

Ingest repayment events.

```text
POST /v1/score
```

Request an on-demand score for a rider.

```text
GET /v1/score/{rider_id}
```

Retrieve the latest score for a rider and their battery.

```text
GET /v1/portfolio
```

Retrieve scores across the lender's portfolio.

---

## 5. Lender Dashboard

The dashboard gives a lender a portfolio-level view of their financed e-boda riders.

Example:

```text
200 Riders
────────────────────────────────
Rider     RRI    BHI    Score    Risk
#001      12     91     89.5     Low
#002      38     76     69.0     Medium
#003      71     54     41.5     High
```

The lender can select an individual rider to see the underlying risk indicators and contributing factors.

Example:

```text
Rider #001

CytoScore          89.5
Repayment Risk     12
Battery Health     91

Repayment
✓ 96% payments on time
✓ Low days-late average

Activity
✓ Regular swap activity

Battery
✓ Healthy degradation trajectory
✓ Normal operating temperature
✓ Moderate cycle usage

⚠ 1 flagged reading — under review
```

This gives the lender both the **overall risk score** and the information behind it.

---

## 6. End-to-End Architecture

```text
              CSV DATA
                  │
        ┌─────────┴─────────┐
        ▼                   ▼
 Repayment Events      Battery Telemetry
        │                   │
        └─────────┬─────────┘
                  ▼
             PostgreSQL
                  │
                  ▼
          Feature Engineering
                  │
        ┌─────────┴─────────┐
        ▼                   ▼
    RRI Model            BHI Model
        │                   │
        │                   │
        └─────────┬─────────┘
                  ▼
              CytoScore
                  │
                  ▼
                Go API
                  │
                  ▼
           Nuxt Dashboard
                  │
                  ▼
              Lender
```

## 7. MVP Outcome

The completed MVP should allow Cyto to take a dataset representing approximately **200 e-boda riders**, analyze their repayment behavior and battery condition, and produce an actionable portfolio of risk scores.

The core demonstration is:

> **A lender uploads their e-boda portfolio data and immediately sees which riders represent lower or higher risk, why they received that score, and the health of the battery serving as part of the financed asset.**

This establishes the core Cyto proposition:

> **Cyto turns borrower behavior and battery intelligence into a single, explainable risk signal that lenders can use before making financing decisions.**

For the MVP, do this and move on:

Keep your current BHI model. ✅
Don't retrain it yet.
In the Go/ML inference code, convert live depth_of_discharge into the 0–1 feature format the BHI model expects.
Document the conversion in the code/model contract.
Continue building:
CSV
↓
Postgres
↓
Feature transformation
↓
RRI + BHI
↓
CytoScore
↓
API
↓
Dashboard

Later, when you have real e-boda telemetry, retrain BHI using the actual telemetry feature definitions.
