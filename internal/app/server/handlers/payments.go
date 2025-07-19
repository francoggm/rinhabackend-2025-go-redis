package handlers

import (
	"encoding/json"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"net/http"
	"time"
)

func (h *Handlers) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	var payment models.Payment
	err := json.NewDecoder(r.Body).Decode(&payment)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	payment.RequestedAt = time.Now().UTC()
	h.paymentEventsCh <- &payment

	w.WriteHeader(http.StatusAccepted)
}
