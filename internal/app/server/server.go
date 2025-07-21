package server

import (
	"fmt"
	"francoggm/rinhabackend-2025-go-redis/internal/app/server/handlers"
	"francoggm/rinhabackend-2025-go-redis/internal/app/storage"
	"francoggm/rinhabackend-2025-go-redis/internal/config"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"net/http"

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
	return http.ListenAndServe(fmt.Sprintf(":%s", s.cfg.Server.Port), s.router)
}
