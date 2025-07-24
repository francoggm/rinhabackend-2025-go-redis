package main

import (
	"context"
	"fmt"
	"francoggm/rinhabackend-2025-go-redis/internal/app/healthcheck"
	"francoggm/rinhabackend-2025-go-redis/internal/app/payment"
	"francoggm/rinhabackend-2025-go-redis/internal/app/server"
	"francoggm/rinhabackend-2025-go-redis/internal/app/storage"
	"francoggm/rinhabackend-2025-go-redis/internal/app/worker"
	"francoggm/rinhabackend-2025-go-redis/internal/config"
	"francoggm/rinhabackend-2025-go-redis/internal/models"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.NewConfig()

	ctx := context.Background()

	cacheOpts := redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Cache.Port),
		Password:     cfg.Password,
		DB:           0,
		PoolSize:     250,
		MinIdleConns: 20,
		PoolTimeout:  60,
	}

	rdb := redis.NewClient(&cacheOpts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	// Worker queues
	events := make(chan *models.Payment, cfg.PaymentBufferSize)

	// Services
	healthCheckService := healthcheck.NewHealthCheckService(cfg.DefaultURL, cfg.FallbackURL, rdb)
	paymentService := payment.NewPaymentService(cfg.DefaultURL, cfg.FallbackURL, healthCheckService)
	storageService := storage.NewStorageService(rdb)

	// Start workers in order of processing
	pool := worker.NewWorkerPool(cfg.PaymentCount, events, paymentService, storageService)
	pool.StartWorkers(ctx)

	server := server.NewServer(cfg, events, storageService)
	if err := server.Run(); err != nil {
		panic(err)
	}

	close(events)
}
