package worker

import (
	"context"
	"errors"
	"francoggm/rinhabackend-2025-go-redis/internal/app/payment"
	"francoggm/rinhabackend-2025-go-redis/internal/app/storage"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"log"
	"time"
)

type Worker struct {
	id             int
	events         chan *models.Payment
	retryEvents    chan *models.Payment
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
				log.Printf("Worker %d: failed to process payment %s: %v", w.id, event.CorrelationID, err)

				if errors.Is(err, payment.ErrNoAvailableProcessor) || err == payment.ErrPaymentProcessingFailed {
					w.retryEvents <- event
				}

				continue
			}

			if err := w.storageService.SavePayment(ctx, event); err != nil {
				log.Printf("Worker %d: failed to save payment %s: %v", w.id, event.CorrelationID, err)
			}
		}
	}
}

func (w *Worker) StartRetryWork(ctx context.Context) {
	const batchSize = 500
	const cooldown = 1000 // milliseconds

	for {
		processed := 0

		for processed < batchSize {
			select {
			case <-ctx.Done():
				return
			case retryEvent, ok := <-w.retryEvents:
				if !ok {
					return
				}

				if err := w.paymentService.MakePayment(ctx, retryEvent); err != nil {
					log.Printf("Worker %d: failed to retry payment %s: %v", w.id, retryEvent.CorrelationID, err)
					continue
				}

				if err := w.storageService.SavePayment(ctx, retryEvent); err != nil {
					log.Printf("Worker %d: failed to save retried payment %s: %v", w.id, retryEvent.CorrelationID, err)
				}

				processed++
			}
		}

		log.Printf("Processed: %d", processed)

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(cooldown) * time.Millisecond):
		}
	}
}

type WorkerPool struct {
	workers []*Worker
}

func NewWorkerPool(workersCount int, events, retryEvents chan *models.Payment, paymentService *payment.PaymentService, storageService *storage.StorageService) *WorkerPool {
	var workers []*Worker
	for id := range workersCount {
		workers = append(workers, &Worker{
			id:             id,
			events:         events,
			retryEvents:    retryEvents,
			paymentService: paymentService,
			storageService: storageService,
		})
	}

	return &WorkerPool{
		workers: workers,
	}
}

func (w *WorkerPool) StartWorkers(ctx context.Context) {
	for _, worker := range w.workers {
		go worker.StartWork(ctx)
		go worker.StartRetryWork(ctx)
	}
}
