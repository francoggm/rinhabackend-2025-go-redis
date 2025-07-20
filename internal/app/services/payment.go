package services

import (
	"context"
	"fmt"
	paymentclient "francoggm/rinhabackend-2025-go-redis/internal/app/services/payment_client"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"time"
)

type PaymentService struct {
	healthCheckService *HealthCheckService
	defaultClient      *paymentclient.PaymentClient
	fallbackClient     *paymentclient.PaymentClient
}

func NewPaymentService(healthCheckService *HealthCheckService, defaultURL, fallbackURL string) *PaymentService {
	return &PaymentService{
		healthCheckService: healthCheckService,
		defaultClient:      paymentclient.NewPaymentClient(defaultURL, 2*time.Second),
		fallbackClient:     paymentclient.NewPaymentClient(fallbackURL, 5*time.Second),
	}
}

func (p *PaymentService) MakePayment(ctx context.Context, payment *models.Payment) error {
	processorId := p.calculateProcessor()
	switch processorId {
	case "default":
		payment.ProcessingType = "default"
		return p.defaultClient.MakePayment(ctx, payment)
	case "fallback":
		payment.ProcessingType = "fallback"
		return p.fallbackClient.MakePayment(ctx, payment)
	}

	return fmt.Errorf("no available payment processor")
}

func (p *PaymentService) calculateProcessor() string {
	defaultHealth, err := p.healthCheckService.DefaultHealthCheck(context.Background())
	if err != nil {
		return ""
	}

	fallbackHealth, err := p.healthCheckService.FallbackHealthCheck(context.Background())
	if err != nil {
		return ""
	}

	if defaultHealth.IsFailing && fallbackHealth.IsFailing {
		return ""
	}

	if defaultHealth.IsFailing {
		return "fallback"
	}

	return "default"
}
