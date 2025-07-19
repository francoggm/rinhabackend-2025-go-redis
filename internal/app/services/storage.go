package services

import (
	"context"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"time"

	"github.com/redis/go-redis/v9"
)

type StorageService struct{}

func NewStorageService(cache *redis.Client) *StorageService {
	return &StorageService{}
}

func (s *StorageService) SavePayment(ctx context.Context, payment *models.Payment) error {
	return nil
}

func (s *StorageService) GetPaymentsSummary(ctx context.Context, from, to *time.Time) (map[string]*models.ProcessorSummary, error) {
	return nil, nil
}

func (s *StorageService) PurgePayments(ctx context.Context) error {
	return nil
}
