package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/infrastructure/httpserver"
)

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()

	httpserver.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Body.String(); got != "ok\n" {
		t.Fatalf("body = %q, want %q", got, "ok\\n")
	}
}
