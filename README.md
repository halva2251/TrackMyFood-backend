# Food Flight Tracker — Backend API

This is the Go-based backend for the **Food Flight Tracker**, providing a high-performance REST API for supply chain transparency, trust score calculations, and real-time food safety alerts.

## Quick Start

### Prerequisites
- **Docker & Docker Compose**
- **Go 1.23+** (for local development)
- **Make**

### Setup
1. **Clone and Initialize**:
   ```bash
   cp .env.example .env
   ```
2. **Launch Infrastructure**:
   ```bash
   # Starts PostgreSQL and the API with hot-reload (Air)
   make up
   ```
3. **Prepare Database**:
   ```bash
   # Run migrations and seed the 4 demo scenarios
   make reset
   ```

## Architecture

The project follows a clean, interface-driven layered architecture:

- **`cmd/server/`**: Application entry point and graceful shutdown logic.
- **`internal/handler/`**: Thin HTTP layer responsible for request parsing and JSON responses.
- **`internal/service/`**: Core business logic (e.g., Trust Score calculation engine).
- **`internal/repository/`**: All SQL interactions using `pgx` (no ORM for maximum performance).
- **`internal/domain/`**: Pure data structures and domain constants.
- **`migrations/`**: Versioned PostgreSQL schema.

## API Reference

All endpoints return a standardized envelope:
```json
{
  "success": true,
  "data": { ... },
  "error": null
}
```

### 1. Scan Lookup
**`GET /api/scan/{barcode}`**

The primary "one-call" endpoint. Returns everything needed to render the product journey and trust score.

- **Parameters**: `barcode` (EAN-13 or unique QR code)
- **Headers**: `X-User-ID` (Optional UUID to record scan history)
- **Success (200 OK)**:
    - Returns `product`, `batch`, `trust_score` (overall + sub-scores), `journey`, `recall` (if any), `certifications`, and `sustainability` data.

### 2. Temperature History
**`GET /api/batch/{id}/temperature`**

Returns the full time-series temperature log for a specific batch.

- **Parameters**: `id` (Batch UUID)
- **Success (200 OK)**: List of temperature readings with timestamps and location.

### 3. File a Complaint
**`POST /api/complaints`**

Allows consumers to report issues. Filing a complaint triggers an **asynchronous trust score recalculation** for the batch.

- **Body**:
  ```json
  {
    "batch_id": "uuid",
    "user_id": "uuid",
    "complaint_type": "taste_smell | packaging_damaged | foreign_object | suspected_spoilage | other",
    "description": "text",
    "photo_url": "url"
  }
  ```

### 4. Admin: Issue Recall
**`POST /api/admin/recalls`**

Simulates a recall event. Issuing a recall **instantly overrides the trust score to 0** for the affected batch.

- **Body**:
  ```json
  {
    "batch_id": "uuid",
    "severity": "low | medium | high | critical",
    "reason": "text",
    "instructions": "text"
  }
  ```

## Trust Score Calculation

The score (0–100) is a weighted average of 5 sub-metrics:
1. **Cold Chain (30%)**: % of readings within the product's safety range.
2. **Quality Checks (25%)**: Ratio of passed vs. expected inspections.
3. **Time to Shelf (20%)**: Speed of delivery vs. optimal benchmark.
4. **Producer Track Record (15%)**: Historical performance (complaint/recall rates).
5. **Handling Steps (10%)**: Number of transfers (fewer is better).

## Demo Scenarios

Use these barcodes to test the system:

| Barcode | Product | Scenario | Expected Score |
| :--- | :--- | :--- | :--- |
| `7610000000001` | Organic Strawberries | Perfect journey, short chain | ~94 |
| `7610000000002` | Atlantic Salmon | Cold chain breach (3 spikes) | ~52 |
| `7610000000003` | Natural Yogurt | Active Critical Recall | **0** |
| `7610000000004` | Mountain Flower Honey | Highly sustainable, Fair Trade | ~88 |

## Development Commands

```bash
make test      # Run all tests with race detection and coverage
make lint      # Run golangci-lint
make build     # Build production binary
make clean     # Stop containers and wipe DB volumes
```
