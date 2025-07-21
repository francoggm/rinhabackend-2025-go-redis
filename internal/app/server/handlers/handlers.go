package handlers

import (
	"francoggm/rinhabackend-2025-go-redis/internal/app/storage"
	"francoggm/rinhabackend-2025-go-redis/internal/config"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
)

type Handlers struct {
	cfg            *config.Config
	events         chan *models.Payment
	storageService *storage.StorageService
}

func NewHandlers(cfg *config.Config, events chan *models.Payment, storageService *storage.StorageService) *Handlers {
	return &Handlers{
		cfg:            cfg,
		events:         events,
		storageService: storageService,
	}
}
