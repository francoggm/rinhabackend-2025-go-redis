package paymentclient

import (
	"context"
	"fmt"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/valyala/fasthttp"
)

type PaymentClient struct {
	url     string
	timeout time.Duration
	client  *fasthttp.Client
}

func NewPaymentClient(url string, timeout time.Duration) *PaymentClient {
	return &PaymentClient{
		url:     url,
		timeout: timeout,
		client:  &fasthttp.Client{},
	}
}

func (c *PaymentClient) MakePayment(ctx context.Context, payment *models.Payment) error {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	payload, err := sonic.Marshal(payment)
	if err != nil {
		return fmt.Errorf("failed to marshal payment: %w", err)
	}

	req.SetRequestURI(c.url + "/payments")
	req.Header.SetMethod(http.MethodPost)
	req.Header.SetContentType("application/json")
	req.SetBody(payload)

	if err := c.client.DoTimeout(req, resp, c.timeout); err != nil {
		return fmt.Errorf("failed to make payment request: %w", err)
	}

	statusCode := resp.StatusCode()
	if statusCode != http.StatusOK {
		return fmt.Errorf("payment request failed with status code: %d", statusCode)
	}

	return err
}
