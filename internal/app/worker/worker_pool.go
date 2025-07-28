package worker

import (
	"context"
	"francoggm/rinhabackend-2025-go-redis/internal/app/payment"
	"francoggm/rinhabackend-2025-go-redis/internal/app/storage"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
)

type WorkerPool struct {
	workers      []*Worker
	retryWorkers []*RetryWorker
}

func NewWorkerPool(workersCount int, events chan *models.Payment, paymentService *payment.PaymentService, storageService *storage.StorageService) *WorkerPool {
	retryEvents := make(chan *RetryEvent, 10000)

	var (
		workers      []*Worker
		retryWorkers []*RetryWorker
	)
	for id := range workersCount {
		workers = append(workers, NewWorker(id, events, retryEvents, paymentService, storageService))
		retryWorkers = append(retryWorkers, NewRetryWorker(id, retryEvents, paymentService, storageService))
	}

	return &WorkerPool{
		workers:      workers,
		retryWorkers: retryWorkers,
	}
}

func (w *WorkerPool) StartWorkers(ctx context.Context) {
	for _, retryWorker := range w.retryWorkers {
		go retryWorker.StartWork(ctx)
	}

	for _, worker := range w.workers {
		go worker.StartWork(ctx)
	}
}
