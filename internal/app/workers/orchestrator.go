package workers

import (
	"context"
	"francoggm/rinhabackend-2025-go-redis/internal/app/workers/processors"
)

type Orchestrator struct {
	workers         []*worker
	eventsCh        chan any
	eventsProcessor processors.Processor
}

func NewOrchestrator(workersCount int, eventsCh chan any, eventsProcessor processors.Processor) *Orchestrator {
	var workers []*worker
	for id := range workersCount {
		worker := newWorker(id, eventsCh, eventsProcessor)
		workers = append(workers, worker)
	}

	return &Orchestrator{
		workers:         workers,
		eventsCh:        eventsCh,
		eventsProcessor: eventsProcessor,
	}
}

func (o *Orchestrator) StartWorkers(ctx context.Context) {
	for _, worker := range o.workers {
		go worker.start(ctx)
	}
}
