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
		PoolSize:     cfg.Workers.StorageCount,
		MinIdleConns: 10,
		PoolTimeout:  60, // seconds
	}

	rdb := redis.NewClient(&cacheOpts)
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		panic(err)
	}

	// Worker queues
	paymentEventsCh := make(chan any, cfg.PaymentBufferSize)
	storageEventsCh := make(chan any, cfg.StorageBufferSize)

	// Services
	paymentService := services.NewPaymentService(cfg.PaymentProcessorConfig.DefaultURL, cfg.PaymentProcessorConfig.FallbackURL)
	storageService := services.NewStorageService(rdb)

	// Worker processors
	paymentProcessor := processors.NewPaymentProcessor(paymentService, storageEventsCh)
	storageProcessor := processors.NewStorageProcessor(storageService)

	// Worker orchestrators
	paymentOrchestrator := workers.NewOrchestrator(cfg.PaymentCount, paymentEventsCh, paymentProcessor)
	storageOrchestrator := workers.NewOrchestrator(cfg.StorageCount, storageEventsCh, storageProcessor)

	// Start workers in order of processing
	storageOrchestrator.StartWorkers(ctx)
	paymentOrchestrator.StartWorkers(ctx)

	server := server.NewServer(cfg, storageService, paymentEventsCh)
	if err := server.Run(); err != nil {
		panic(err)
	}

	close(paymentEventsCh)
	close(storageEventsCh)
}
