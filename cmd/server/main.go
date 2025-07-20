package main

import (
	"context"
	"fmt"
	"francoggm/rinhabackend-2025-go-redis/internal/app/server"
	"francoggm/rinhabackend-2025-go-redis/internal/app/services"
	"francoggm/rinhabackend-2025-go-redis/internal/app/workers"
	"francoggm/rinhabackend-2025-go-redis/internal/app/workers/processors"
	"francoggm/rinhabackend-2025-go-redis/internal/config"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.NewConfig()

	ctx := context.Background()

	cacheOpts := redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Cache.Host, cfg.Cache.Port),
		Password:     cfg.Cache.Password,
		DB:           0,
		MinIdleConns: 10,
		PoolTimeout:  60, // seconds
	}

	rdb := redis.NewClient(&cacheOpts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	// Worker queues
	eventsCh := make(chan any, cfg.PaymentBufferSize)

	// Services
	healthCheckService := services.NewHealthCheckService(cfg.PaymentProcessorConfig.DefaultURL, cfg.PaymentProcessorConfig.FallbackURL, rdb)
	paymentService := services.NewPaymentService(healthCheckService, cfg.PaymentProcessorConfig.DefaultURL, cfg.PaymentProcessorConfig.FallbackURL)
	storageService := services.NewStorageService(rdb)

	paymentProcessor := processors.NewPaymentProcessor(paymentService, storageService)
	paymentOrchestrator := workers.NewWorkerPool(cfg.PaymentCount, true, eventsCh, paymentProcessor)

	// Start workers in order of processing
	paymentOrchestrator.StartWorkers(ctx)

	server := server.NewServer(cfg, storageService, eventsCh)
	if err := server.Run(); err != nil {
		panic(err)
	}

	close(eventsCh)
}
