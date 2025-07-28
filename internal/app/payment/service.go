package payment

import (
	"context"
	"fmt"
	"francoggm/rinhabackend-2025-go-redis/internal/app/healthcheck"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/valyala/fasthttp"
)

var (
	ErrNoAvailableProcessor    = fmt.Errorf("no available payment processor")
	ErrPaymentProcessingFailed = fmt.Errorf("payment processing failed")
)

type PaymentService struct {
	defaultUrl  string
	fallbackUrl string
	client      *fasthttp.Client

	HealthCheckService *healthcheck.HealthCheckService
}

func NewPaymentService(defaultHost, fallbackHost string, healthCheckService *healthcheck.HealthCheckService) *PaymentService {
	client := &fasthttp.Client{
		MaxConnsPerHost: 1000,
	}

	return &PaymentService{
		defaultUrl:         defaultHost + "/payments",
		fallbackUrl:        fallbackHost + "/payments",
		client:             client,
		HealthCheckService: healthCheckService,
	}
}

func (p *PaymentService) MakePayment(ctx context.Context, payment *models.Payment) error {
	processor := p.HealthCheckService.AvailableProcessor(ctx)
	payment.ProcessingType = processor

	switch processor {
	case "default":
		return p.innerPayment(p.defaultUrl, payment)
	case "fallback":
		return p.innerPayment(p.fallbackUrl, payment)
	}

	return ErrNoAvailableProcessor
}

func (p *PaymentService) innerPayment(url string, payment *models.Payment) error {
	req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	payment.RequestedAt = time.Now().UTC()
	payload, err := sonic.Marshal(payment)
	if err != nil {
		return fmt.Errorf("failed to marshal payment: %w", err)
	}

	req.SetRequestURI(url)
	req.Header.SetMethod(http.MethodPost)
	req.Header.SetContentType("application/json")
	req.SetBody(payload)

	if err := p.client.DoTimeout(req, resp, 2*time.Second); err != nil {
		return fmt.Errorf("failed to make payment request in processor %s: %w", payment.ProcessingType, err)
	}

	statusCode := resp.StatusCode()
	if statusCode != http.StatusOK {
		if statusCode == http.StatusInternalServerError ||
			statusCode == http.StatusRequestTimeout ||
			statusCode == http.StatusTooManyRequests ||
			statusCode == http.StatusServiceUnavailable {
			return ErrPaymentProcessingFailed
		}

		return fmt.Errorf("payment request failed with status code: %d, in processor: %s", statusCode, payment.ProcessingType)
	}

	return nil
}
