package worker

import (
	"context"
	"francoggm/rinhabackend-2025-go-redis/internal/app/payment"
	"francoggm/rinhabackend-2025-go-redis/internal/app/storage"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"log"
	"math"
	"math/rand"
	"time"
)

type RetryEvent struct {
	Payment    *models.Payment
	RetryCount int
}

type RetryWorker struct {
	id             int
	retryEvents    chan *RetryEvent
	paymentService *payment.PaymentService
	storageService *storage.StorageService
}

func NewRetryWorker(id int, retryEvents chan *RetryEvent, ps *payment.PaymentService, ss *storage.StorageService) *RetryWorker {
	return &RetryWorker{
		id:             id,
		retryEvents:    retryEvents,
		paymentService: ps,
		storageService: ss,
	}
}

func (w *RetryWorker) Start(ctx context.Context) {
	log.Printf("Worker %d: starting retry worker\n", w.id)

	const maxBatchSize = 20
	const batchTimeout = 200 * time.Millisecond

	var batch []*RetryEvent

	timer := time.NewTimer(batchTimeout)
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}

	flush := func() {
		if len(batch) == 0 {
			return
		}

		batchCopy := make([]*RetryEvent, len(batch))
		copy(batchCopy, batch)

		go w.processBatch(ctx, batchCopy)
		batch = batch[:0]
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

			if len(batch) == 0 {
				timer.Reset(batchTimeout)
			}

			batch = append(batch, event)
			if len(batch) >= maxBatchSize {
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

func (w *RetryWorker) processBatch(ctx context.Context, batch []*RetryEvent) {
	processor := w.paymentService.HealthCheckService.AvailableProcessor(ctx)
	if processor == "" {
		log.Printf("Worker %d: no available processor for retry batch\n", w.id)

		for _, item := range batch {
			time.AfterFunc(500*time.Millisecond, func() {
				w.retryEvents <- item
			})
		}
		return
	}

	for _, item := range batch {
		if err := w.paymentService.MakePayment(ctx, item.Payment); err != nil {
			item.RetryCount++
			const maxRetries = 5

			if item.RetryCount >= maxRetries {
				log.Printf("Worker %d: payment %s failed after %d retries, giving up\n", w.id, item.Payment.CorrelationID, item.RetryCount)
				continue
			}

			backoff := time.Duration(math.Pow(2, float64(item.RetryCount))) * 100 * time.Millisecond
			jitter := time.Duration(rand.Intn(100)) * time.Millisecond
			delay := backoff + jitter

			time.AfterFunc(delay, func() {
				w.retryEvents <- item
			})
			continue
		}

		if err := w.storageService.SavePayment(ctx, item.Payment); err != nil {
			log.Printf("Worker %d: failed to save payment %s after retry: %v\n", w.id, item.Payment.CorrelationID, err)
		}
	}
}
