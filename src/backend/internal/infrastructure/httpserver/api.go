// @title ЖКХ Контроль API
// @version 1.0
// @description Технический API приложения ЖКХ Контроль.
// @description Публичные методы доступны через префикс /api.
// @BasePath /api
// @schemes https http

package httpserver

// VersionResponse is the deployment version returned by the public API.
type VersionResponse struct {
	Version string `json:"version" example:"abc123"`
}
