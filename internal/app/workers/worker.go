package workers

import (
	"context"
	"fmt"
	"francoggm/rinhabackend-2025-go-redis/internal/app/workers/processors"
)

type worker struct {
	id              int
	reenqueue       bool
	eventsCh        chan any
	eventsProcessor processors.Processor
}

func newWorker(id int, retryEvent bool, eventsCh chan any, eventsProcessor processors.Processor) *worker {
	return &worker{
		id:              id,
		reenqueue:       retryEvent,
		eventsCh:        eventsCh,
		eventsProcessor: eventsProcessor,
	}
}

func (w *worker) start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-w.eventsCh:
			if !ok {
				return
			}

			if err := w.eventsProcessor.ProcessEvent(ctx, event); err != nil {
				fmt.Println("Error processing event:", err)

				if w.reenqueue {
					w.eventsCh <- event // Retry the event
				}
			}
		}
	}
}
