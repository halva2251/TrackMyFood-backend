package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/halva2251/trackmyfood-backend/internal/domain"
	"github.com/halva2251/trackmyfood-backend/internal/handler"
	"github.com/halva2251/trackmyfood-backend/internal/middleware"
)

type mockComplaintRepo struct {
	createFunc func(ctx context.Context, c domain.Complaint) (domain.Complaint, error)
}

func (m *mockComplaintRepo) Create(ctx context.Context, c domain.Complaint) (domain.Complaint, error) {
	return m.createFunc(ctx, c)
}

type mockScoreRecalculator struct {
	called bool
}

func (m *mockScoreRecalculator) Recalculate(_ context.Context, _ uuid.UUID) error {
	m.called = true
	return nil
}

// withUserID injects user ID into request context, simulating UserAuth middleware.
func withUserID(r *http.Request, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

func TestComplaintHandler_Create(t *testing.T) {
	batchID := uuid.MustParse("00000000-0000-0000-0002-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0004-000000000001")

	successMock := &mockComplaintRepo{
		createFunc: func(_ context.Context, c domain.Complaint) (domain.Complaint, error) {
			c.ID = uuid.New()
			c.CreatedAt = time.Now()
			return c, nil
		},
	}

	tests := []struct {
		name       string
		body       string
		repo       *mockComplaintRepo
		authed     bool
		wantStatus int
		wantErr    string
	}{
		{
			name:       "success",
			body:       fmt.Sprintf(`{"batch_id":"%s","complaint_type":"taste_smell","description":"smells bad"}`, batchID),
			repo:       successMock,
			authed:     true,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "all valid complaint types - packaging_damaged",
			body:       fmt.Sprintf(`{"batch_id":"%s","complaint_type":"packaging_damaged"}`, batchID),
			repo:       successMock,
			authed:     true,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "all valid complaint types - foreign_object",
			body:       fmt.Sprintf(`{"batch_id":"%s","complaint_type":"foreign_object"}`, batchID),
			repo:       successMock,
			authed:     true,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "all valid complaint types - suspected_spoilage",
			body:       fmt.Sprintf(`{"batch_id":"%s","complaint_type":"suspected_spoilage"}`, batchID),
			repo:       successMock,
			authed:     true,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "all valid complaint types - other",
			body:       fmt.Sprintf(`{"batch_id":"%s","complaint_type":"other"}`, batchID),
			repo:       successMock,
			authed:     true,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "unauthenticated",
			body:       fmt.Sprintf(`{"batch_id":"%s","complaint_type":"other"}`, batchID),
			repo:       successMock,
			authed:     false,
			wantStatus: http.StatusUnauthorized,
			wantErr:    "authentication required",
		},
		{
			name:       "invalid json",
			body:       `{invalid`,
			repo:       successMock,
			authed:     true,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid request body",
		},
		{
			name:       "invalid batch_id",
			body:       `{"batch_id":"bad","complaint_type":"other"}`,
			repo:       successMock,
			authed:     true,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid batch_id",
		},
		{
			name:       "invalid complaint type",
			body:       fmt.Sprintf(`{"batch_id":"%s","complaint_type":"invalid_type"}`, batchID),
			repo:       successMock,
			authed:     true,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid complaint_type",
		},
		{
			name: "repo error",
			body: fmt.Sprintf(`{"batch_id":"%s","complaint_type":"other"}`, batchID),
			repo: &mockComplaintRepo{
				createFunc: func(_ context.Context, _ domain.Complaint) (domain.Complaint, error) {
					return domain.Complaint{}, fmt.Errorf("db error")
				},
			},
			authed:     true,
			wantStatus: http.StatusInternalServerError,
			wantErr:    "failed to create complaint",
		},
		{
			name:       "invalid photo_url — not a url",
			body:       fmt.Sprintf(`{"batch_id":"%s","complaint_type":"other","photo_url":"not-a-url"}`, batchID),
			repo:       successMock,
			authed:     true,
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid photo_url",
		},
		{
			name:       "invalid photo_url — non-http scheme",
			body:       fmt.Sprintf(`{"batch_id":"%s","complaint_type":"other","photo_url":"ftp://example.com/photo.jpg"}`, batchID),
			repo:       successMock,
			authed:     true,
			wantStatus: http.StatusBadRequest,
			wantErr:    "photo_url must use http or https",
		},
		{
			name:       "valid https photo_url",
			body:       fmt.Sprintf(`{"batch_id":"%s","complaint_type":"other","photo_url":"https://example.com/photo.jpg"}`, batchID),
			repo:       successMock,
			authed:     true,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "valid http photo_url",
			body:       fmt.Sprintf(`{"batch_id":"%s","complaint_type":"other","photo_url":"http://example.com/photo.jpg"}`, batchID),
			repo:       successMock,
			authed:     true,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "empty photo_url string — passes validation",
			body:       fmt.Sprintf(`{"batch_id":"%s","complaint_type":"other","photo_url":""}`, batchID),
			repo:       successMock,
			authed:     true,
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := &mockScoreRecalculator{}
			h := handler.NewComplaintHandler(tt.repo, scorer, &sync.WaitGroup{})

			req := httptest.NewRequest(http.MethodPost, "/api/complaints", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.authed {
				req = withUserID(req, userID)
			}
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
