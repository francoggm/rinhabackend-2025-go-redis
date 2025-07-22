package server

import (
	"fmt"
	"francoggm/rinhabackend-2025-go-redis/internal/app/server/handlers"
	"francoggm/rinhabackend-2025-go-redis/internal/app/storage"
	"francoggm/rinhabackend-2025-go-redis/internal/config"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	cfg      *config.Config
	router   *chi.Mux
	handlers *handlers.Handlers
}

func NewServer(cfg *config.Config, events chan *models.Payment, storageService *storage.StorageService) *Server {
	srv := &Server{
		cfg:      cfg,
		router:   chi.NewRouter(),
		handlers: handlers.NewHandlers(cfg, events, storageService),
	}

	srv.registerRoutes()
	return srv
}

func (s *Server) registerRoutes() {
	s.router.Post("/payments", s.handlers.ProcessPayment)
	s.router.Get("/payments-summary", s.handlers.GetPaymentsSummary)
	s.router.Post("/purge-payments", s.handlers.PurgePayments)
}

func (s *Server) Run() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.cfg.Server.Port),
		Handler:      s.router,
		IdleTimeout:  15 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return srv.ListenAndServe()
}
