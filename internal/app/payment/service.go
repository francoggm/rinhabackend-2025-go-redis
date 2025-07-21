package payment

import (
	"context"
	"fmt"
	"francoggm/rinhabackend-2025-go-redis/internal/app/healthcheck"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"time"
)

var (
	ErrNoAvailableProcessor    = fmt.Errorf("no available payment processor")
	ErrPaymentProcessingFailed = fmt.Errorf("payment processing failed")
)

type PaymentService struct {
	healthCheckService *healthcheck.HealthCheckService
	defaultClient      *PaymentClient
	fallbackClient     *PaymentClient
}

func NewPaymentService(defaultURL, fallbackURL string, healthCheckService *healthcheck.HealthCheckService) *PaymentService {
	return &PaymentService{
		defaultClient:      NewPaymentClient(defaultURL, 2*time.Second),
		fallbackClient:     NewPaymentClient(fallbackURL, 5*time.Second),
		healthCheckService: healthCheckService,
	}
}

func (p *PaymentService) MakePayment(ctx context.Context, payment *models.Payment) error {
	processor := p.healthCheckService.AvailableProcessor(ctx)
	payment.ProcessingType = processor

	switch processor {
	case "default":
		return p.defaultClient.MakePayment(ctx, payment)
	case "fallback":
		return p.fallbackClient.MakePayment(ctx, payment)
	}

	return ErrNoAvailableProcessor
}
