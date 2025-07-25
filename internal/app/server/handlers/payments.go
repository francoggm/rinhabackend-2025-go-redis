package handlers

import (
	"encoding/json"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"net/http"
)

func (h *Handlers) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	var payment models.Payment
	err := json.NewDecoder(r.Body).Decode(&payment)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	select {
	case h.events <- &payment:
		w.WriteHeader(http.StatusAccepted)
	default:
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}
