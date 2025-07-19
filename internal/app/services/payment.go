package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

type PaymentService struct {
	httpClient              *http.Client
	defaultURL              string
	fallbackURL             string
	isDefaultHealthy        atomic.Bool
	isFallbackHealthy       atomic.Bool
	defaultMinResponseTime  atomic.Int32
	fallbackMinResponseTime atomic.Int32
}

func NewPaymentService(defaultURL, fallbackURL string) *PaymentService {
	tr := &http.Transport{
		MaxIdleConns:        30,
		MaxIdleConnsPerHost: 30,
		IdleConnTimeout:     90 * time.Second,
		MaxConnsPerHost:     50,
		DisableCompression:  true,
		DisableKeepAlives:   false,
		ForceAttemptHTTP2:   false,

		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
	}

	httpClient := &http.Client{
		Transport: tr,
	}

	service := &PaymentService{
		httpClient:  httpClient,
		defaultURL:  defaultURL,
		fallbackURL: fallbackURL,
	}

	go service.startHealthChecker()
	return service
}

func (p *PaymentService) MakePayment(ctx context.Context, payment *models.Payment) error {
	payload, err := json.Marshal(payment)
	if err != nil {
		return err
	}

	url, processingType, err := p.calculateProcessor()
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url+"/payments", bytes.NewBuffer(payload))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response from %s: %d", processingType, resp.StatusCode)
	}

	payment.ProcessingType = processingType

	return nil
}

func (p *PaymentService) calculateProcessor() (string, string, error) {
	isDefaultHealthy := p.isDefaultHealthy.Load()
	isFallbackHealthy := p.isFallbackHealthy.Load()

	defaultMinResponseTime := p.defaultMinResponseTime.Load()
	fallbackMinResponseTime := p.fallbackMinResponseTime.Load()

	if !isDefaultHealthy && !isFallbackHealthy {
		return "", "", fmt.Errorf("both payment processors are unhealthy")
	}

	if isDefaultHealthy && isFallbackHealthy {
		// 30% threshold for fallback
		if int32(float32(defaultMinResponseTime)*1.3) > fallbackMinResponseTime {
			return p.fallbackURL, "fallback", nil
		}

		return p.defaultURL, "default", nil
	} else if isDefaultHealthy {
		return p.defaultURL, "default", nil
	}

	return p.fallbackURL, "fallback", nil
}

func (p *PaymentService) startHealthChecker() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()

		p.checkDefaultHealth(ctx)
		p.checkFallbackHealth(ctx)
	}
}

func (p *PaymentService) checkDefaultHealth(ctx context.Context) {
	defaultHealthCheck, err := p.checkHealth(ctx, p.defaultURL+"/payments/service-health")
	if err != nil {
		return
	}

	p.isDefaultHealthy.Store(!defaultHealthCheck.IsFailing)
	p.defaultMinResponseTime.Store(defaultHealthCheck.MinResponseTime)
}

func (p *PaymentService) checkFallbackHealth(ctx context.Context) {
	fallbackHealthCheck, err := p.checkHealth(ctx, p.fallbackURL+"/payments/service-health")
	if err != nil {
		return
	}

	p.isFallbackHealthy.Store(!fallbackHealthCheck.IsFailing)
	p.fallbackMinResponseTime.Store(fallbackHealthCheck.MinResponseTime)
}

func (p *PaymentService) checkHealth(ctx context.Context, url string) (*models.HealthCheck, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("health check request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check failed with status code: %d", resp.StatusCode)
	}

	var healthCheck models.HealthCheck
	if err := json.NewDecoder(resp.Body).Decode(&healthCheck); err != nil {
		return nil, fmt.Errorf("failed to decode health check response: %w", err)
	}

	return &healthCheck, nil
}
