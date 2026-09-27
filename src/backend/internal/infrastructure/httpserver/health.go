package httpserver

import (
	"encoding/json"
	"net/http"
)

// Handler exposes health checks used by Docker, Caddy and the status page.
func Handler(webhook ...http.Handler) http.Handler {
	var endpoint http.Handler
	if len(webhook) > 0 {
		endpoint = webhook[0]
	}
	return HandlerWithVersion(endpoint, "dev")
}

// HandlerWithVersion exposes health, deployment metadata and the MAX webhook.
func HandlerWithVersion(webhook http.Handler, version string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(struct {
			Version string `json:"version"`
		}{Version: version})
	})
	if webhook != nil {
		mux.Handle("/webhook", webhook)
	}
	return mux
}
