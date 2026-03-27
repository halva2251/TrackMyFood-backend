package router

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/halva2251/trackmyfood-backend/internal/config"
	"github.com/halva2251/trackmyfood-backend/internal/handler"
	appmiddleware "github.com/halva2251/trackmyfood-backend/internal/middleware"
	"github.com/halva2251/trackmyfood-backend/internal/repository"
	"github.com/halva2251/trackmyfood-backend/internal/service"
)

func New(db *pgxpool.Pool, wg *sync.WaitGroup, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Throttle(200))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	origins := strings.Split(cfg.AllowedOrigins, ",")
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key"},
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
	producerRepo := repository.NewProducerRepo(db)
	alternativesRepo := repository.NewAlternativesRepo(db)
	userRepo := repository.NewUserRepo(db)

	// Services
	trustScoreSvc := service.NewTrustScoreService(db)
	authSvc := service.NewAuthService(cfg.JWTSecret)

	// Handlers
	scanH := handler.NewScanHandler(scanRepo, anomalyRepo)
	tempH := handler.NewTemperatureHandler(tempRepo)
	complaintH := handler.NewComplaintHandler(complaintRepo, trustScoreSvc, wg)
	recallH := handler.NewRecallHandler(recallRepo)
	producerH := handler.NewProducerHandler(producerRepo, trustScoreSvc, wg)
	altH := handler.NewAlternativesHandler(scanRepo, alternativesRepo)
	authH := handler.NewAuthHandler(userRepo, authSvc)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			slog.Error("health check db ping failed", "error", err)
			handler.WriteError(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "db": "ok"})
	})

	r.Route("/api", func(r chi.Router) {
		// Public scan endpoints with optional auth for recording scans
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.OptionalUserAuth(authSvc))
			r.Get("/scan/{barcode}", scanH.Lookup)
			r.Get("/scan/{barcode}/alternatives", altH.GetAlternatives)
		})

		r.Get("/batch/{id}/temperature", tempH.GetByBatch)
		r.Post("/complaints", complaintH.Create)

		// Auth endpoints (public)
		r.Post("/auth/register", authH.Register)
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/refresh", authH.Refresh)

		// Authenticated user endpoints
		r.Route("/user", func(r chi.Router) {
			r.Use(appmiddleware.UserAuth(authSvc))
			r.Get("/me", authH.Me)
			r.Get("/scan-history", authH.ScanHistory)
			r.Delete("/scan-history", authH.DeleteScanHistoryEntry)
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(appmiddleware.AdminAuth(cfg.AdminAPIKey))
			r.Post("/recalls", recallH.Create)
		})

		r.Route("/producer", func(r chi.Router) {
			r.Use(appmiddleware.AdminAuth(cfg.AdminAPIKey))
			r.Post("/batches", producerH.CreateBatch)
			r.Post("/batches/{id}/journey-steps", producerH.AddJourneyStep)
			r.Post("/batches/{id}/temperature-readings", producerH.AddTemperatureReading)
			r.Post("/batches/{id}/quality-checks", producerH.AddQualityCheck)
		})
	})

	return r
}
