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
				// log.Printf("Worker %d: failed to process payment %s: %v", w.id, event.CorrelationID, err)

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
	var mu sync.Mutex

	const maxBatchSize = 100
	paymentsBatch := make([]*models.Payment, 0, maxBatchSize)

	processBatch := func(processor string) {
		mu.Lock()
		batch := paymentsBatch
		if len(batch) > maxBatchSize {
			batch = batch[:maxBatchSize]
		}
		paymentsBatch = paymentsBatch[len(batch):]
		mu.Unlock()

		if processor != "" && len(batch) > 0 {
			log.Printf("Worker %d: retrying payments for processor %s, quantity: %d", w.id, processor, len(batch))

			for _, payment := range batch {
				if err := w.paymentService.MakePayment(ctx, payment); err != nil {
					continue
				}

				if err := w.storageService.SavePayment(ctx, payment); err != nil {
					log.Printf("Worker %d: failed to save retried payment %s: %v", w.id, payment.CorrelationID, err)
				}
			}
		}
	}

	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			processor := w.paymentService.HealthCheckService.AvailableProcessor(ctx)
			processBatch(processor)
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
			batchReady := len(paymentsBatch) >= maxBatchSize
			mu.Unlock()

			if batchReady {
				processor := w.paymentService.HealthCheckService.AvailableProcessor(ctx)
				processBatch(processor)
			}
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
