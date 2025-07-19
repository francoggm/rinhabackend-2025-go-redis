package server

import (
	"fmt"
	"francoggm/rinhabackend-2025-go-redis/internal/app/server/handlers"
	"francoggm/rinhabackend-2025-go-redis/internal/app/services"
	"francoggm/rinhabackend-2025-go-redis/internal/config"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	cfg      *config.Config
	router   *chi.Mux
	handlers *handlers.Handlers
}

func NewServer(cfg *config.Config, storageService *services.StorageService, paymentEventsCh chan any) *Server {
	srv := &Server{
		cfg:      cfg,
		router:   chi.NewRouter(),
		handlers: handlers.NewHandlers(cfg, storageService, paymentEventsCh),
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
