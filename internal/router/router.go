package router

import (
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/halva2251/trackmyfood-backend/internal/handler"
	"github.com/halva2251/trackmyfood-backend/internal/repository"
	"github.com/halva2251/trackmyfood-backend/internal/service"
)

func New(db *pgxpool.Pool, wg *sync.WaitGroup) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-User-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Repos
	scanRepo := repository.NewScanRepo(db)
	tempRepo := repository.NewTemperatureRepo(db)
	complaintRepo := repository.NewComplaintRepo(db)
	recallRepo := repository.NewRecallRepo(db)
	anomalyRepo := repository.NewAnomalyRepo(db)

	// Services
	trustScoreSvc := service.NewTrustScoreService(db)

	// Handlers
	scanH := handler.NewScanHandler(scanRepo, anomalyRepo)
	tempH := handler.NewTemperatureHandler(tempRepo)
	complaintH := handler.NewComplaintHandler(complaintRepo, trustScoreSvc, wg)
	recallH := handler.NewRecallHandler(recallRepo)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api", func(r chi.Router) {
		r.Get("/scan/{barcode}", scanH.Lookup)
		r.Get("/batch/{id}/temperature", tempH.GetByBatch)
		r.Post("/complaints", complaintH.Create)
		r.Post("/admin/recalls", recallH.Create)
	})

	return r
}
