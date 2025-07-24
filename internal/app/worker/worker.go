package worker

import (
	"context"
	"errors"
	"francoggm/rinhabackend-2025-go-redis/internal/app/payment"
	"francoggm/rinhabackend-2025-go-redis/internal/app/storage"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"log"
)

type Worker struct {
	id             int
	events         chan *models.Payment
	retryEvents    chan *RetryEvent
	paymentService *payment.PaymentService
	storageService *storage.StorageService
}

func (w *Worker) StartWork(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-w.events:
			if !ok {
				return
			}

			if err := w.paymentService.MakePayment(ctx, event); err != nil {
				if errors.Is(err, payment.ErrNoAvailableProcessor) || err == payment.ErrPaymentProcessingFailed {
					w.retryEvents <- &RetryEvent{
						Payment:    event,
						RetryCount: 0,
					}
				}

				continue
			}

			if err := w.storageService.SavePayment(ctx, event); err != nil {
				log.Printf("Worker %d: failed to save payment %s: %v\n", w.id, event.CorrelationID, err)
			}
		}
	}
}
