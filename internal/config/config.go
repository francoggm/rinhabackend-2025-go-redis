package config

import (
	"os"
	"strconv"
)

type Config struct {
	Cache
	Workers
	Server
	PaymentProcessorConfig
}

type Cache struct {
	Host string
	Port string
}

type Workers struct {
	PaymentCount      int
	PaymentBufferSize int
}

type Server struct {
	Port string
}

type PaymentProcessorConfig struct {
	DefaultURL  string
	FallbackURL string
}

func NewConfig() *Config {
	return &Config{
		Cache: Cache{
			Host: getEnvString("CACHE_HOST", "localhost"),
			Port: getEnvString("CACHE_PORT", "6373"),
		},
		Workers: Workers{
			PaymentCount:      getEnvInt("PAYMENT_WORKERS_COUNT", 5),
			PaymentBufferSize: getEnvInt("PAYMENT_WORKERS_EVENTS_BUFFER_SIZE", 100),
		},
		Server: Server{
			Port: getEnvString("SERVER_PORT", "8080"),
		},
		PaymentProcessorConfig: PaymentProcessorConfig{
			DefaultURL:  getEnvString("PAYMENT_DEFAULT_URL", "http://localhost:8081"),
			FallbackURL: getEnvString("PAYMENT_FALLBACK_URL", "http://localhost:8082"),
		},
	}
}

func getEnvString(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	return value
}

func getEnvInt(key string, defaultValue int) int {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}
