package models

import "time"

type Payment struct {
	CorrelationID  string    `json:"correlationId"`
	Amount         float32   `json:"amount"`
	RequestedAt    time.Time `json:"requestedAt"`
	ProcessingType string    `json:"processingType,omitempty"`
}

type ProcessorSummary struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float32 `json:"totalAmount"`
}
