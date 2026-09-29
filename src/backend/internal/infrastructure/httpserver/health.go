package httpserver

import (
	"encoding/json"
	"net/http"

	swaggerdocs "github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/infrastructure/httpserver/swagger"
	httpSwagger "github.com/swaggo/http-swagger/v2"
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
	swaggerdocs.SwaggerInfo.Version = version
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/api/version", versionHandler(version))
	mux.Handle("/api/swagger/", httpSwagger.Handler(httpSwagger.URL("/api/swagger/doc.json")))
	if webhook != nil {
		mux.Handle("/webhook", webhook)
	}
	return mux
}

// versionHandler returns the deployed application revision.
//
// @Summary Получить версию приложения
// @Description Возвращает ревизию backend, запущенную в текущем окружении.
// @Tags system
// @Produce json
// @Success 200 {object} VersionResponse
// @Router /version [get]
func versionHandler(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(VersionResponse{Version: version})
	}
}
