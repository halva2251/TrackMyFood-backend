package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
	"github.com/halva2251/trackmyfood-backend/internal/handler"
)

type mockRecallRepo struct {
	createFunc       func(ctx context.Context, recall domain.Recall) (domain.Recall, error)
	zeroScoreFunc    func(ctx context.Context, batchID uuid.UUID) error
	affectedFunc     func(ctx context.Context, batchID uuid.UUID) ([]domain.User, error)
}

func (m *mockRecallRepo) Create(ctx context.Context, recall domain.Recall) (domain.Recall, error) {
	return m.createFunc(ctx, recall)
}

func (m *mockRecallRepo) ZeroBatchScore(ctx context.Context, batchID uuid.UUID) error {
	if m.zeroScoreFunc != nil {
		return m.zeroScoreFunc(ctx, batchID)
	}
	return nil
}

func (m *mockRecallRepo) GetAffectedUsers(ctx context.Context, batchID uuid.UUID) ([]domain.User, error) {
	if m.affectedFunc != nil {
		return m.affectedFunc(ctx, batchID)
	}
	return nil, nil
}

func TestRecallHandler_Create(t *testing.T) {
	batchID := uuid.MustParse("00000000-0000-0000-0002-000000000001")

	successMock := &mockRecallRepo{
		createFunc: func(_ context.Context, r domain.Recall) (domain.Recall, error) {
			r.ID = uuid.New()
			r.RecalledAt = time.Now()
			r.IsActive = true
			return r, nil
		},
	}

	tests := []struct {
		name       string
		body       string
		repo       *mockRecallRepo
		wantStatus int
		wantErr    string
	}{
		{
			name:       "success - critical",
			body:       fmt.Sprintf(`{"batch_id":"%s","severity":"critical","reason":"contamination","instructions":"return to store"}`, batchID),
			repo:       successMock,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "success - low severity",
			body:       fmt.Sprintf(`{"batch_id":"%s","severity":"low","reason":"labeling error","instructions":"check label"}`, batchID),
			repo:       successMock,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "success - medium severity",
			body:       fmt.Sprintf(`{"batch_id":"%s","severity":"medium","reason":"minor issue","instructions":"check product"}`, batchID),
			repo:       successMock,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "success - high severity",
			body:       fmt.Sprintf(`{"batch_id":"%s","severity":"high","reason":"serious issue","instructions":"do not consume"}`, batchID),
			repo:       successMock,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid json",
			body:       `{bad`,
			repo:       successMock,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid request body",
		},
		{
			name:       "invalid batch_id",
			body:       `{"batch_id":"bad","severity":"critical","reason":"x","instructions":"y"}`,
			repo:       successMock,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid batch_id",
		},
		{
			name:       "invalid severity",
			body:       fmt.Sprintf(`{"batch_id":"%s","severity":"extreme","reason":"x","instructions":"y"}`, batchID),
			repo:       successMock,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid severity",
		},
		{
			name:       "empty reason",
			body:       fmt.Sprintf(`{"batch_id":"%s","severity":"critical","reason":"","instructions":"y"}`, batchID),
			repo:       successMock,
			wantStatus: http.StatusBadRequest,
			wantErr:    "reason and instructions are required",
		},
		{
			name:       "empty instructions",
			body:       fmt.Sprintf(`{"batch_id":"%s","severity":"critical","reason":"x","instructions":""}`, batchID),
			repo:       successMock,
			wantStatus: http.StatusBadRequest,
			wantErr:    "reason and instructions are required",
		},
		{
			name: "create error",
			body: fmt.Sprintf(`{"batch_id":"%s","severity":"critical","reason":"x","instructions":"y"}`, batchID),
			repo: &mockRecallRepo{
				createFunc: func(_ context.Context, _ domain.Recall) (domain.Recall, error) {
					return domain.Recall{}, fmt.Errorf("duplicate recall")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to create recall",
		},
		{
			name: "zero score error",
			body: fmt.Sprintf(`{"batch_id":"%s","severity":"critical","reason":"x","instructions":"y"}`, batchID),
			repo: &mockRecallRepo{
				createFunc: func(_ context.Context, r domain.Recall) (domain.Recall, error) {
					r.ID = uuid.New()
					r.RecalledAt = time.Now()
					r.IsActive = true
					return r, nil
				},
				zeroScoreFunc: func(_ context.Context, _ uuid.UUID) error {
					return fmt.Errorf("update failed")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to update batch score",
		},
		{
			name: "get affected users error",
			body: fmt.Sprintf(`{"batch_id":"%s","severity":"critical","reason":"x","instructions":"y"}`, batchID),
			repo: &mockRecallRepo{
				createFunc: func(_ context.Context, r domain.Recall) (domain.Recall, error) {
					r.ID = uuid.New()
					r.RecalledAt = time.Now()
					r.IsActive = true
					return r, nil
				},
				affectedFunc: func(_ context.Context, _ uuid.UUID) ([]domain.User, error) {
					return nil, fmt.Errorf("query failed")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to get affected users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handler.NewRecallHandler(tt.repo)

			req := httptest.NewRequest(http.MethodPost, "/api/admin/recalls", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.Create(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			var resp handler.Response
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if tt.wantErr != "" {
				if resp.Success {
					t.Error("expected success = false")
				}
				if resp.Error != tt.wantErr {
					t.Errorf("error = %q, want %q", resp.Error, tt.wantErr)
				}
			} else {
				if !resp.Success {
					t.Errorf("expected success = true, got error: %s", resp.Error)
				}
			}
		})
	}
}

func TestRecallHandler_Create_ReturnsAffectedUsers(t *testing.T) {
	batchID := uuid.MustParse("00000000-0000-0000-0002-000000000001")

	displayName := "Test User"
	mock := &mockRecallRepo{
		createFunc: func(_ context.Context, r domain.Recall) (domain.Recall, error) {
			r.ID = uuid.New()
			r.RecalledAt = time.Now()
			r.IsActive = true
			return r, nil
		},
		affectedFunc: func(_ context.Context, _ uuid.UUID) ([]domain.User, error) {
			return []domain.User{
				{ID: uuid.New(), Email: "user@test.ch", DisplayName: &displayName, CreatedAt: time.Now()},
			}, nil
		},
	}

	h := handler.NewRecallHandler(mock)

	body := fmt.Sprintf(`{"batch_id":"%s","severity":"critical","reason":"contamination","instructions":"return to store"}`, batchID)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/recalls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Recall        domain.Recall `json:"recall"`
			AffectedUsers []domain.User `json:"affected_users"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Data.AffectedUsers) != 1 {
		t.Errorf("affected users = %d, want 1", len(resp.Data.AffectedUsers))
	}
	if resp.Data.AffectedUsers[0].Email != "user@test.ch" {
		t.Errorf("email = %q, want %q", resp.Data.AffectedUsers[0].Email, "user@test.ch")
	}
	if resp.Data.Recall.Severity != "critical" {
		t.Errorf("severity = %q, want %q", resp.Data.Recall.Severity, "critical")
	}
}
