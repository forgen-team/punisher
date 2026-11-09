package handlers

import (
	"context"
	"net/http"
	"time"
)

// Pinger is the subset of the store the health check needs.
type Pinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	Store Pinger
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.Store.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"error":  "db connection failed",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "ok",
		"service":   "punisher",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
