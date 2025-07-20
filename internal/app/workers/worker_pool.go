package workers

import (
	"context"
	"francoggm/rinhabackend-2025-go-redis/internal/app/workers/processors"
)

type WorkerPool struct {
	workers         []*worker
	eventsCh        chan any
	eventsProcessor processors.Processor
}

func NewWorkerPool(workersCount int, reenqueue bool, eventsCh chan any, eventsProcessor processors.Processor) *WorkerPool {
	var workers []*worker
	for id := range workersCount {
		worker := newWorker(id, reenqueue, eventsCh, eventsProcessor)
		workers = append(workers, worker)
	}

	return &WorkerPool{
		workers:         workers,
		eventsCh:        eventsCh,
		eventsProcessor: eventsProcessor,
	}
}

func (w *WorkerPool) StartWorkers(ctx context.Context) {
	for _, worker := range w.workers {
		go worker.start(ctx)
	}
}
