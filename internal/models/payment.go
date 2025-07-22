package models

import "time"

type Payment struct {
	CorrelationID  string    `json:"correlationId"`
	Amount         float64   `json:"amount"`
	RequestedAt    time.Time `json:"requestedAt,omitempty"`
	ProcessingType string    `json:"processingType,omitempty"`
}

type Summary struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}

type PaymentsSummary struct {
	DefaultSummary  Summary `json:"default"`
	FallbackSummary Summary `json:"fallback"`
}
