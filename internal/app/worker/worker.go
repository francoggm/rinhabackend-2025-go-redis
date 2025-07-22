package worker

import (
	"context"
	"francoggm/rinhabackend-2025-go-redis/internal/app/payment"
	"francoggm/rinhabackend-2025-go-redis/internal/app/storage"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"log"
)

type Worker struct {
	id             int
	events         chan *models.Payment
	paymentService *payment.PaymentService
	storageService *storage.StorageService
}

func (w *WorkerPool) StartWorkers(ctx context.Context) {
	for _, worker := range w.workers {
		go worker.Start(ctx)
	}
}

type WorkerPool struct {
	workers []*Worker
}

func NewWorkerPool(workersCount int, events chan *models.Payment, paymentService *payment.PaymentService, storageService *storage.StorageService) *WorkerPool {
	var workers []*Worker
	for id := range workersCount {
		workers = append(workers, &Worker{
			id:             id,
			events:         events,
			paymentService: paymentService,
			storageService: storageService,
		})
	}

	return &WorkerPool{
		workers: workers,
	}
}

func (w *Worker) Start(ctx context.Context) {
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
				continue
			}

			if err := w.storageService.SavePayment(ctx, event); err != nil {
				log.Printf("Worker %d: failed to save payment %s: %v", w.id, event.CorrelationID, err)
			}
		}
	}
}
