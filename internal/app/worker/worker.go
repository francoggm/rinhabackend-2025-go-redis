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
	const maxBatchSize = 500
	const batchTimeout = 200 * time.Millisecond

	var paymentsBatch []*models.Payment

	timer := time.NewTimer(batchTimeout)
	if !timer.Stop() {
		<-timer.C
	}

	flush := func() {
		if len(paymentsBatch) == 0 {
			return
		}

		batchCopy := make([]*models.Payment, len(paymentsBatch))
		copy(batchCopy, paymentsBatch)

		go w.processRetryBatch(ctx, batchCopy)
		paymentsBatch = paymentsBatch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case event, ok := <-w.retryEvents:
			if !ok {
				flush()
				return
			}

			if len(paymentsBatch) == 0 {
				timer.Reset(batchTimeout)
			}

			paymentsBatch = append(paymentsBatch, event)

			if len(paymentsBatch) >= maxBatchSize {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				flush()
			}
		case <-timer.C:
			flush()
		}
	}
}

func (w *Worker) processRetryBatch(ctx context.Context, batch []*models.Payment) {
	time.Sleep(3000 * time.Millisecond)

	processor := w.paymentService.HealthCheckService.AvailableProcessor(ctx)
	if processor == "" {
		log.Printf("Worker %d: no available processor for retry batch\n", w.id)
		return
	}

	for _, payment := range batch {
		if err := w.paymentService.MakePayment(ctx, payment); err != nil {
			continue
		}

		if err := w.storageService.SavePayment(ctx, payment); err != nil {
			log.Printf("Worker %d: failed to save payment %s after retry: %v\n", w.id, payment.CorrelationID, err)
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
