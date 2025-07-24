package worker

import (
	"context"
	"errors"
	"francoggm/rinhabackend-2025-go-redis/internal/app/payment"
	"francoggm/rinhabackend-2025-go-redis/internal/app/storage"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"log"
	"sync"
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
				if errors.Is(err, payment.ErrNoAvailableProcessor) || err == payment.ErrPaymentProcessingFailed {
					w.retryEvents <- event
				}

				continue
			}

			if err := w.storageService.SavePayment(ctx, event); err != nil {
				log.Printf("Worker %d: failed to save payment %s: %v\n", w.id, event.CorrelationID, err)
			}
		}
	}
}

func (w *Worker) StartRetryWork(ctx context.Context) {
	var mu sync.Mutex

	paymentsBatch := make([]*models.Payment, 0, 500)

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			processor := w.paymentService.HealthCheckService.AvailableProcessor(ctx)

			mu.Lock()
			batchSize := len(paymentsBatch)
			mu.Unlock()

			if processor != "" && batchSize > 0 {
				mu.Lock()

				log.Printf("Worker %d: retrying payments for processor %s, quantity: %d", w.id, processor, len(paymentsBatch))
				for _, payment := range paymentsBatch {
					if err := w.paymentService.MakePayment(ctx, payment); err != nil {
						log.Printf("Worker %d: failed to retry payment %s: %v", w.id, payment.CorrelationID, err)
						continue
					}

					if err := w.storageService.SavePayment(ctx, payment); err != nil {
						log.Printf("Worker %d: failed to save retried payment %s: %v", w.id, payment.CorrelationID, err)
					}
				}

				paymentsBatch = make([]*models.Payment, 0, 500)
				mu.Unlock()
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case retryEvent, ok := <-w.retryEvents:
			if !ok {
				return
			}

			mu.Lock()
			paymentsBatch = append(paymentsBatch, retryEvent)
			mu.Unlock()
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
