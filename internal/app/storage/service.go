package storage

import (
	"bytes"
	"context"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"math"
	"time"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
)

const paymentsKey = "payments"

type StorageService struct {
	cache *redis.Client
}

func NewStorageService(cache *redis.Client) *StorageService {
	return &StorageService{
		cache: cache,
	}
}

func (s *StorageService) SavePayment(ctx context.Context, payment *models.Payment) error {
	payload, err := marshalPayment(payment)
	if err != nil {
		return err
	}

	return s.cache.HSet(ctx, paymentsKey, payment.CorrelationID, payload).Err()
}

func (s *StorageService) GetPaymentsSummary(ctx context.Context, from, to *time.Time) (*models.PaymentsSummary, error) {
	var paymentsSummary models.PaymentsSummary

	paymentsMap, err := s.cache.HGetAll(ctx, paymentsKey).Result()
	if err != nil {
		return nil, err
	}

	for _, data := range paymentsMap {
		var payment models.Payment

		decoder := sonic.ConfigFastest.NewDecoder(bytes.NewReader([]byte(data)))
		if err := decoder.Decode(&payment); err != nil {
			return nil, err
		}

		if !paymentWithinTime(payment.RequestedAt, from, to) {
			continue
		}

		if payment.ProcessingType == "default" {
			paymentsSummary.DefaultSummary.TotalRequests++
			paymentsSummary.DefaultSummary.TotalAmount += payment.Amount
		} else {
			paymentsSummary.FallbackSummary.TotalRequests++
			paymentsSummary.FallbackSummary.TotalAmount += payment.Amount
		}
	}

	paymentsSummary.DefaultSummary.TotalAmount = math.Round(paymentsSummary.DefaultSummary.TotalAmount*100) / 100
	paymentsSummary.FallbackSummary.TotalAmount = math.Round(paymentsSummary.FallbackSummary.TotalAmount*100) / 100

	return &paymentsSummary, nil
}

func (s *StorageService) PurgePayments(ctx context.Context) error {
	return s.cache.Del(ctx, paymentsKey).Err()
}

func marshalPayment(payment *models.Payment) ([]byte, error) {
	data, err := sonic.ConfigFastest.Marshal(payment)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func paymentWithinTime(requestedAt time.Time, from, to *time.Time) bool {
	if from != nil && requestedAt.Before(*from) {
		return false
	}

	if to != nil && requestedAt.After(*to) {
		return false
	}

	return true
}
