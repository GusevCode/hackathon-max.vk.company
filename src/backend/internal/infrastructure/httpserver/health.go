package httpserver

import "net/http"

// Handler exposes health checks used by Docker, Caddy and the status page.
func Handler(webhook ...http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	if len(webhook) > 0 && webhook[0] != nil {
		mux.Handle("/webhook", webhook[0])
	}
	return mux
}
