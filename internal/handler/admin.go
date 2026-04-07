package handler

import (
	"crypto/subtle"
	"net/http"
	"time"

	"jhol.dev/internal/analytics"
)

// Admin handles the admin stats endpoint.
type Admin struct {
	analytics *analytics.Store
	token     string
}

// NewAdmin creates an Admin handler.
func NewAdmin(store *analytics.Store, token string) *Admin {
	return &Admin{analytics: store, token: token}
}

// RegisterRoutes registers the admin API route on the given mux.
func (a *Admin) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/admin/stats", a.getStats)
}

func (a *Admin) getStats(w http.ResponseWriter, r *http.Request) {
	if a.token == "" {
		http.Error(w, "admin disabled", http.StatusServiceUnavailable)
		return
	}

	provided := r.Header.Get("X-Admin-Token")
	if subtle.ConstantTimeCompare([]byte(provided), []byte(a.token)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, a.analytics.Stats(24*time.Hour))
}
