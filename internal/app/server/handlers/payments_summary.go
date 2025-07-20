package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
)

func (h *Handlers) GetPaymentsSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()

	var from, to *time.Time
	layout := time.RFC3339

	if fromStr := query.Get("from"); fromStr != "" {
		if t, err := time.Parse(layout, fromStr); err == nil {
			from = &t
		}
	}

	if toStr := query.Get("to"); toStr != "" {
		if t, err := time.Parse(layout, toStr); err == nil {
			to = &t
		}
	}

	summary, err := h.storageService.GetPaymentsSummary(ctx, from, to)
	if err != nil {
		fmt.Println("Error getting payment sumarry:", err)
		http.Error(w, "failed to get payments summary", http.StatusInternalServerError)
		return
	}

	data, err := sonic.Marshal(summary)
	if err != nil {
		fmt.Println("Error encoding response:", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
