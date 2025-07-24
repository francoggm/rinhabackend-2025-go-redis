package worker

import (
	"context"
	"francoggm/rinhabackend-2025-go-redis/internal/app/payment"
	"francoggm/rinhabackend-2025-go-redis/internal/app/storage"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
)

type WorkerPool struct {
	workers     []*Worker
	retryWorker *RetryWorker
}

func NewWorkerPool(workersCount int, events chan *models.Payment, paymentService *payment.PaymentService, storageService *storage.StorageService) *WorkerPool {
	retryEvents := make(chan *RetryEvent, 1000)

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

	retryWorker := NewRetryWorker(99, retryEvents, paymentService, storageService)

	return &WorkerPool{
		workers:     workers,
		retryWorker: retryWorker,
	}
}

func (w *WorkerPool) StartWorkers(ctx context.Context) {
	go w.retryWorker.Start(ctx)

	for _, worker := range w.workers {
		go worker.StartWork(ctx)
	}
}
