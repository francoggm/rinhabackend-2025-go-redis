package processors

import (
	"context"
	"francoggm/rinhabackend-2025-go-redis/internal/app/services"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
)

type PaymentProcessor struct {
	paymentService *services.PaymentService
	storageService *services.StorageService
}

func NewPaymentProcessor(paymentService *services.PaymentService, storageService *services.StorageService) *PaymentProcessor {
	return &PaymentProcessor{
		paymentService: paymentService,
		storageService: storageService,
	}
}

func (p *PaymentProcessor) ProcessEvent(ctx context.Context, event any) error {
	payment := event.(*models.Payment)

	if err := p.paymentService.MakePayment(ctx, payment); err != nil {
		return err
	}

	return p.storageService.SavePayment(ctx, payment)
}
