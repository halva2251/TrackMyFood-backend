package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/halva2251/trackmyfood-backend/internal/repository"
)

// mainScanColumns lists the 19 columns returned by LookupByBarcode's primary query.
var mainScanColumns = []string{
	"p.id", "p.name", "p.category", "p.barcode",
	"pr.id", "pr.name", "pr.location", "pr.country",
	"b.id", "b.lot_number", "b.production_date", "b.expiry_date",
	"b.trust_score", "b.sub_score_cold_chain", "b.sub_score_quality",
	"b.sub_score_time_to_shelf", "b.sub_score_producer", "b.sub_score_handling",
	"b.score_calculated_at",
}

func trustScorePtr(v float64) *float64 { return &v }

func TestScanRepo_LookupByBarcode_Found(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	productID := uuid.New()
	producerID := uuid.New()
	batchID := uuid.New()
	productionDate := time.Date(2026, 3, 12, 6, 0, 0, 0, time.UTC)
	trustScore := trustScorePtr(94.0)
	lotNumber := "LOT-2026-001"
	barcode := "7610000000001"
	arrivedAt := time.Date(2026, 3, 12, 8, 0, 0, 0, time.UTC)

	// 1. Main scan query
	mock.ExpectQuery("SELECT").
		WithArgs(barcode).
		WillReturnRows(pgxmock.NewRows(mainScanColumns).AddRow(
			productID, "Organic Strawberries", "fruits", barcode,
			producerID, "Bio Hof Thurgau", "Frauenfeld", "CH",
			batchID, lotNumber, productionDate, (*time.Time)(nil),
			trustScore, trustScorePtr(100), trustScorePtr(100),
			trustScorePtr(100), trustScorePtr(100), trustScorePtr(100),
			(*time.Time)(nil),
		))

	// 2. getJourney
	mock.ExpectQuery("SELECT step_order").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"step_order", "step_type", "location", "latitude", "longitude", "arrived_at", "departed_at"}).
			AddRow(1, "harvested", "Bio Hof, Frauenfeld", (*float64)(nil), (*float64)(nil), arrivedAt, (*time.Time)(nil)))

	// 3. getRecall — no recall (ErrNoRows)
	mock.ExpectQuery("SELECT severity").
		WithArgs(batchID).
		WillReturnError(pgx.ErrNoRows)

	// 4. getCertifications
	certValidUntil := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT cert_type").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"cert_type", "issuing_body", "valid_until"}).
			AddRow("Bio", "BioSuisse", &certValidUntil))

	// 5. getSustainability
	co2 := 1.5
	water := 500.0
	transport := 50.0
	mock.ExpectQuery("SELECT co2_kg").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"co2_kg", "water_liters", "transport_km"}).
			AddRow(&co2, &water, &transport))

	repo := repository.NewScanRepo(mock)
	resp, err := repo.LookupByBarcode(context.Background(), barcode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Product.Barcode != barcode {
		t.Errorf("barcode = %q, want %q", resp.Product.Barcode, barcode)
	}
	if resp.Batch.LotNumber != lotNumber {
		t.Errorf("lot_number = %q, want %q", resp.Batch.LotNumber, lotNumber)
	}
	if resp.TrustScore.Overall != 94.0 {
		t.Errorf("overall = %v, want 94.0", resp.TrustScore.Overall)
	}
	if len(resp.Journey) != 1 {
		t.Errorf("journey len = %d, want 1", len(resp.Journey))
	}
	if resp.Recall != nil {
		t.Error("recall should be nil")
	}
	if len(resp.Certs) != 1 {
		t.Errorf("certs len = %d, want 1", len(resp.Certs))
	}
	if resp.Sustain == nil {
		t.Error("sustainability should not be nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestScanRepo_LookupByBarcode_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	barcode := "9999999999999"

	// Main query returns ErrNoRows
	mock.ExpectQuery("SELECT").
		WithArgs(barcode).
		WillReturnError(pgx.ErrNoRows)

	repo := repository.NewScanRepo(mock)
	_, err = repo.LookupByBarcode(context.Background(), barcode)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestScanRepo_LookupByBarcode_WithActiveRecall(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	productID := uuid.New()
	producerID := uuid.New()
	batchID := uuid.New()
	productionDate := time.Date(2026, 3, 1, 6, 0, 0, 0, time.UTC)
	barcode := "7610000000003"
	recalledAt := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	// 1. Main scan query
	mock.ExpectQuery("SELECT").
		WithArgs(barcode).
		WillReturnRows(pgxmock.NewRows(mainScanColumns).AddRow(
			productID, "Natural Yogurt", "dairy", barcode,
			producerID, "Swiss Dairy Co", "Bern", "CH",
			batchID, "LOT-2026-003", productionDate, (*time.Time)(nil),
			(*float64)(nil), (*float64)(nil), (*float64)(nil),
			(*float64)(nil), (*float64)(nil), (*float64)(nil),
			(*time.Time)(nil),
		))

	// 2. getJourney — empty
	mock.ExpectQuery("SELECT step_order").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"step_order", "step_type", "location", "latitude", "longitude", "arrived_at", "departed_at"}))

	// 3. getRecall — active recall
	mock.ExpectQuery("SELECT severity").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"severity", "reason", "instructions", "recalled_at", "is_active"}).
			AddRow("high", "Listeria contamination", "Do not consume. Return to store.", recalledAt, true))

	// 4. getCertifications — empty
	mock.ExpectQuery("SELECT cert_type").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"cert_type", "issuing_body", "valid_until"}))

	// 5. getSustainability — no data
	mock.ExpectQuery("SELECT co2_kg").
		WithArgs(batchID).
		WillReturnError(pgx.ErrNoRows)

	repo := repository.NewScanRepo(mock)
	resp, err := repo.LookupByBarcode(context.Background(), barcode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Recall == nil {
		t.Fatal("recall should not be nil")
	}
	if !resp.Recall.IsActive {
		t.Error("recall.is_active should be true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestScanRepo_LookupByBarcode_NoJourneyNoCerts(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	productID := uuid.New()
	producerID := uuid.New()
	batchID := uuid.New()
	productionDate := time.Date(2026, 3, 20, 6, 0, 0, 0, time.UTC)
	barcode := "7610000000004"

	// 1. Main scan query
	mock.ExpectQuery("SELECT").
		WithArgs(barcode).
		WillReturnRows(pgxmock.NewRows(mainScanColumns).AddRow(
			productID, "Mountain Flower Honey", "honey", barcode,
			producerID, "Alpine Bees", "Interlaken", "CH",
			batchID, "LOT-2026-004", productionDate, (*time.Time)(nil),
			trustScorePtr(88.0), (*float64)(nil), (*float64)(nil),
			(*float64)(nil), (*float64)(nil), (*float64)(nil),
			(*time.Time)(nil),
		))

	// 2. getJourney — empty rows
	mock.ExpectQuery("SELECT step_order").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"step_order", "step_type", "location", "latitude", "longitude", "arrived_at", "departed_at"}))

	// 3. getRecall — ErrNoRows
	mock.ExpectQuery("SELECT severity").
		WithArgs(batchID).
		WillReturnError(pgx.ErrNoRows)

	// 4. getCertifications — empty rows
	mock.ExpectQuery("SELECT cert_type").
		WithArgs(batchID).
		WillReturnRows(pgxmock.NewRows([]string{"cert_type", "issuing_body", "valid_until"}))

	// 5. getSustainability — ErrNoRows
	mock.ExpectQuery("SELECT co2_kg").
		WithArgs(batchID).
		WillReturnError(pgx.ErrNoRows)

	repo := repository.NewScanRepo(mock)
	resp, err := repo.LookupByBarcode(context.Background(), barcode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Journey) != 0 {
		t.Errorf("journey len = %d, want 0", len(resp.Journey))
	}
	if resp.Recall != nil {
		t.Error("recall should be nil")
	}
	if len(resp.Certs) != 0 {
		t.Errorf("certs len = %d, want 0", len(resp.Certs))
	}
	if resp.Sustain != nil {
		t.Error("sustainability should be nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestScanRepo_RecordScan_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	userID := uuid.New()
	batchID := uuid.New()

	mock.ExpectExec("INSERT INTO scan_history").
		WithArgs(userID, batchID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := repository.NewScanRepo(mock)
	err = repo.RecordScan(context.Background(), userID, batchID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
