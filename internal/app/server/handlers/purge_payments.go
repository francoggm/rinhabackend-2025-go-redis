package handlers

import (
	"fmt"
	"net/http"
)

func (h *Handlers) PurgePayments(w http.ResponseWriter, r *http.Request) {
	if err := purgeProcessor(h.cfg.PaymentProcessorConfig.DefaultURL + "/admin/purge-payments"); err != nil {
		http.Error(w, fmt.Sprintf("failed to prune default payments: %v", err), http.StatusBadGateway)
		return
	}

	if err := purgeProcessor(h.cfg.PaymentProcessorConfig.FallbackURL + "/admin/purge-payments"); err != nil {
		http.Error(w, fmt.Sprintf("failed to prune fallback payments: %v", err), http.StatusBadGateway)
		return
	}

	if err := h.storageService.PurgePayments(r.Context()); err != nil {
		http.Error(w, fmt.Sprintf("failed to prune storage payments: %v", err), http.StatusBadGateway)
		return
	}
}

func purgeProcessor(url string) error {
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Rinha-Token", "123")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to prune payments, status code: %d", res.StatusCode)
	}

	return nil
}
