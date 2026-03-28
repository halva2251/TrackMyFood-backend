# GEMINI.md

This file provides guidance to Gemini when working with code in this repository.

## Development Commands

```bash
# Start everything (DB + API) in Docker
make up

# Stop containers
make down

# Stop + wipe database volume
make clean

# Run migrations against local DB
make migrate

# Seed demo data
make seed

# Full reset: drop schema, recreate, seed
make reset

# Local dev with hot reload (requires: go install github.com/air-verse/air@latest)
make dev

# Build binary
make build

# Run tests
make test

# Run a single test
make test-one T=TestTrustScore

# Lint and vet
make lint
make vet
```

**Environment**: Copy `.env.example` to `.env` for local development. Docker Compose sets its own env vars.

## Architecture

Go backend using chi router, pgx (raw SQL), PostgreSQL. No ORM.

```
cmd/server/main.go           – entry point, config, DB pool, graceful shutdown
internal/
  config/                     – env var parsing
  domain/                     – pure structs with JSON tags (no methods, no DB)
  handler/                    – HTTP handlers (thin: parse request → call repo → write JSON)
  repository/                 – all SQL queries live here
  service/trust_score.go      – trust score calculation engine
  router/                     – chi router setup + middleware
migrations/                   – SQL schema (up/down)
seed/                         – demo data for 4 scenarios
```

**Key pattern**: `handler/` is thin glue. Business logic lives in `service/`. All SQL lives in `repository/`. Domain structs serve as both DB scan targets and JSON response shapes.

**API envelope**: All endpoints return `{"success": bool, "data": ..., "error": "..."}` via `handler.WriteJSON` / `handler.WriteError`.

**Trust score**: Cached on the `batches` table. Recalculated via `service.CalculateTrustScore()` when data changes (complaint filed, etc.). Recall overrides score to 0.

## Key Endpoints

- `GET /api/scan/{barcode}` – main endpoint, returns everything in one call (product, batch, trust score, journey, recall, certs, sustainability)
- `POST /api/scan/{barcode}/chat` – ask AI questions about this specific product batch
- `GET /api/batch/{id}/temperature` – cold chain time-series
- `POST /api/complaints` – file complaint, triggers async score recalc
- `POST /api/admin/recalls` – create recall, zero score, return affected users

## Demo Barcodes

| Barcode | Product | Expected Score |
|---------|---------|----------------|
| 7610000000001 | Organic Strawberries (perfect) | ~94 |
| 7610000000002 | Atlantic Salmon (sketchy) | ~52 |
| 7610000000003 | Natural Yogurt (recalled) | 0 |
| 7610000000004 | Mountain Flower Honey (sustainable) | ~88 |

---

# Food Flight Tracker — Project Brief

## What is this?

We are building the **backend** for a mobile app called **Food Flight Tracker** — a consumer-facing product that lets people scan a food product's barcode and instantly see a **trust score** for that specific batch, along with its full journey from farm to shelf.

Think of it as **Yuka, but for the supply chain**. Yuka scores a product based on its ingredients and nutritional value. We score a **specific batch** based on what happened to it during production, storage, transport, and delivery — cold chain integrity, quality inspections, handling steps, and the producer's track record.

This is a hackathon project for a challenge by **Autexis** (a Swiss automation company) called *"Track my Food – vom Feld bis ins Verkaufsregal."*

## The core idea

A consumer scans a barcode or QR code on a food product. The app shows:

1. **A trust score (0–100)** for this exact batch — not the brand, not the product line, but this specific LOT/batch. The score is color-coded (green/orange/red) with a label like "Excellent," "Good," "Fair," "Poor," or "Critical."

2. **Sub-score breakdown** — the trust score is composed of transparent, explainable sub-scores:
   - **Cold chain integrity (weight: 30%)** — percentage of the journey where temperature stayed within the acceptable range for this product category. Deviations are penalized proportionally to duration and severity.
   - **Quality checks passed (weight: 25%)** — ratio of inspections passed to inspections expected. Missing checks count as failures.
   - **Time to shelf (weight: 20%)** — ratio of actual time from production to store vs. the optimal benchmark for this product category. Faster = better.
   - **Producer track record (weight: 15%)** — rolling historical score based on the producer's recall frequency, complaint rate, and average cold chain compliance across all their batches.
   - **Handling steps (weight: 10%)** — number of transfers/handoffs the batch went through. Fewer = better. Benchmarked per product category.

3. **Journey visualization data** — the sequence of steps the batch traveled through (e.g., harvested at farm → processed at facility → stored in cold warehouse → transported by truck → arrived at retailer). Each step has a location, timestamps, and the type of step.

4. **Recall alerts** — if the batch has an active recall, this overrides everything. The API must clearly flag this. On the app side, the screen turns red with instructions for the consumer.

5. **Consumer complaints** — users can report problems (taste/smell off, packaging damaged, foreign object, suspected spoilage, other). Complaints can include a photo and a text description. Complaints feed back into the producer's track record score over time.

6. **Scan history & push notifications** — the system records which users scanned which batches. If a recall is issued on a batch after a user scanned it, they should be notifiable (the mobile app handles push delivery, but the backend needs to support looking up affected users).

### Optional features (implement if time allows)

- **Cold chain visualization data** — time-series temperature readings for the batch's journey, with acceptable min/max range per reading. The app renders this as a chart.
- **Sustainability info** — CO₂ footprint (kg), water usage (liters), transport distance (km) for the batch.
- **Certifications** — Bio, Fair Trade, Demeter, etc. Each certification has a type, issuing body, and validity date.
- **Anomaly detection** — flag batches where metrics deviate significantly (>2 standard deviations) from historical averages for that product category, even if no single metric is in the red. This is basic statistics (z-score), not ML.

## What we are building right now

**Only the backend.** The mobile app (React Native / Expo) will be built separately. The backend needs to expose a clean REST API that the mobile app can consume.

The most important endpoint is the scan lookup: given a barcode (EAN) or QR code value, return everything the app needs to render the trust score screen in a **single API response** — product info, batch info, pre-calculated trust score, all sub-scores, journey steps, certifications, active recall status. One call, one render. No waterfalls.

## Data model

The central entity is **Batch** — everything hangs off a specific batch, not a product. A product has many batches. Each batch has journey steps, temperature readings, quality checks, and potentially a recall, complaints, certifications, and sustainability data.

Here are the key entities and their relationships:

- **Producer** → has many Products
- **Product** → has many Batches (identified by LOT number)
- **Batch** → has many JourneySteps, TemperatureReadings, QualityChecks, Complaints, Certifications
- **Batch** → has one optional Recall
- **Batch** → has one optional Sustainability record
- **User** → has many Complaints, many ScanHistory entries
- **ScanHistory** → links Users to Batches (records who scanned what and when)

### Trust score calculation

The trust score is a **cached, pre-calculated value** stored on the Batch record. It gets recalculated whenever underlying data changes (new temperature reading, quality check result, complaint filed). The score engine should be a standalone internal function/service that:

1. Queries all relevant sub-score data for the batch
2. Calculates each sub-score on a 0–100 scale
3. Applies the weighted formula: `TrustScore = 0.30×ColdChain + 0.25×QualityChecks + 0.20×TimeToShelf + 0.15×ProducerTrackRecord + 0.10×HandlingSteps`
4. If a sub-score has no data (e.g., no temperature readings exist), exclude it and redistribute its weight proportionally among the remaining sub-scores
5. Writes the result back to the batch record

The score should also be recalculated when a complaint is filed (since it affects the producer track record sub-score).

## Demo data

The backend should be seeded with realistic demo data for 4 product scenarios. These will be scanned with printed QR codes during the hackathon demo:

1. **"The perfect batch"** — organic strawberries from a Thurgau farm. Short journey (farm → cold storage → retailer in <24h), all quality checks passed, temperature always in range. Trust score ~94.

2. **"The sketchy batch"** — imported salmon. Cold chain broken for 2 hours during transport (temperature spiked to 12°C when max is 4°C), one quality check failed, longer journey with 5 handling steps. Trust score ~52.

3. **"The recalled product"** — a dairy product (yogurt) with an active recall due to contamination. Trust score drops to near-zero. The recall record should have severity, reason, and consumer instructions.

4. **"The sustainable choice"** — locally produced honey with excellent sustainability metrics (low CO₂, short transport), Bio and Fair Trade certifications, all checks passed. Trust score ~88.

Each product needs a realistic barcode (EAN-13 format) or a unique QR code identifier that the scan endpoint can look up.

## Constraints

- We are a small team (4–5 people), and the hackathon is ~24 hours
- The backend should use **Go** or **C#/.NET** — the team knows both, make a recommendation based on what fits best
- Use **PostgreSQL** as the database
- The API should be RESTful with JSON responses
- Prioritize getting the scan → trust score → journey flow working end-to-end over adding every optional feature
- The codebase should be clean enough to present to a jury — they may look at the code

## What success looks like

At the end, we should be able to:
1. Hit `GET /api/scan/{barcode}` and get back a full JSON payload with the trust score, sub-scores, journey, and recall status
2. Hit `GET /api/batch/{id}/temperature` and get cold chain time-series data
3. Hit `POST /api/complaints` to file a complaint with a photo
4. Hit `POST /api/admin/recalls` to trigger a recall (for live demo purposes)
5. Have all 4 demo products seeded and scannable
6. Have the trust score recalculate correctly when data changes
