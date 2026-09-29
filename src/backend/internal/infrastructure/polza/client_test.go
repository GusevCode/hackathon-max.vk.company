package polza

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/usecase/inspection"
)

func TestAnalyzeSendsVisionRequestAndParsesStructuredResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected Authorization header")
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "openai/gpt-4o" {
			t.Fatalf("model = %#v", body["model"])
		}
		format := body["response_format"].(map[string]any)
		schema := format["json_schema"].(map[string]any)
		if schema["strict"] != true {
			t.Fatalf("strict schema is not enabled: %#v", schema)
		}
		messages := body["messages"].([]any)
		system := messages[0].(map[string]any)
		if !strings.Contains(system["content"].(string), "только на русском") {
			t.Fatalf("system prompt does not require Russian report text: %q", system["content"])
		}
		user := messages[1].(map[string]any)
		parts := user["content"].([]any)
		image := parts[2].(map[string]any)["image_url"].(map[string]any)
		if image["url"] != "data:image/jpeg;base64,aW1hZ2U=" {
			t.Fatalf("unexpected image URL: %#v", image["url"])
		}
		provider := body["provider"].(map[string]any)
		maxPrice := provider["max_price"].(map[string]any)
		if maxPrice["request"] != 3.5 {
			t.Fatalf("unexpected max request price: %#v", maxPrice)
		}
		only := provider["only"].([]any)
		if len(only) != 1 || only[0] != "openai" {
			t.Fatalf("unexpected provider allowlist: %#v", only)
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"model": "openai/gpt-4o", "provider": "test-provider",
			"choices": []any{map[string]any{"message": map[string]any{"content": `{"relevant":true,"quality":"usable","observations":["clean elevator floor"],"missing_requirements":[],"comment_summary":"work appears complete","recommendation":"approve","confidence":0.82,"questions":[]}`}}},
			"usage":   map[string]any{"prompt_tokens": 100, "completion_tokens": 50, "total_tokens": 150, "cost_rub": 1.25},
		})
	}))
	defer server.Close()

	client, err := New(Config{BaseURL: server.URL, APIKey: "test-key", Model: "openai/gpt-4o", Timeout: time.Second, MaxTokens: 800, ImageDetail: "high", AllowFallbacks: true, MaxPriceRUB: 3.5, ProviderOnly: []string{"openai"}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Analyze(context.Background(), inspection.EvidenceInput{
		Title: "Elevator cleaning", Images: []inspection.EvidenceImage{{Kind: "after", ContentType: "image/jpeg", Data: []byte("image")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Recommendation != domain.RecommendationApprove || result.Confidence != 0.82 || result.CostRUB != 1.25 || result.Provider != "test-provider" {
		t.Fatalf("unexpected analysis: %#v", result)
	}
}

func TestAnalyzeMarksRateLimitAsTemporary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusTooManyRequests)
		_, _ = writer.Write([]byte(`{"error":{"code":"RATE_LIMITED"}}`))
	}))
	defer server.Close()
	client, err := New(Config{BaseURL: server.URL, APIKey: "test-key", Model: "openai/gpt-4o", Timeout: time.Second, MaxTokens: 800, ImageDetail: "low", MaxPriceRUB: 3.5})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Analyze(context.Background(), inspection.EvidenceInput{})
	polzaErr, ok := err.(*Error)
	if !ok || !polzaErr.Temporary() || polzaErr.Code() != "rate_limited" {
		t.Fatalf("unexpected error: %#v", err)
	}
}
