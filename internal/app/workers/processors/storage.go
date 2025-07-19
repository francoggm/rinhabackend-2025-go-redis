package processors

import (
	"context"
	"francoggm/rinhabackend-2025-go-redis/internal/app/services"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
)

type StorageProcessor struct {
	service *services.StorageService
}

func NewStorageProcessor(service *services.StorageService) *StorageProcessor {
	return &StorageProcessor{
		service: service,
	}
}

func (p *StorageProcessor) ProcessEvent(ctx context.Context, event any) error {
	payment := event.(*models.Payment)
	return p.service.SavePayment(ctx, payment)
}
