package config_test

import (
	"testing"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/config"
)

func TestLoadRequiresWebhookConfiguration(t *testing.T) {
	t.Setenv("LLM_ENABLED", "false")
	for _, name := range []string{
		"APP_NAME", "MAX_BOT_TOKEN", "PUBLIC_BASE_URL", "MAX_WEBHOOK_SECRET",
		"TARANTOOL_PASSWORD", "OBJECT_STORAGE_ACCESS_KEY", "OBJECT_STORAGE_SECRET_KEY",
	} {
		t.Setenv(name, "")
	}

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() error = nil, want missing token error")
	}
}

func TestLoad(t *testing.T) {
	t.Setenv("LLM_ENABLED", "false")
	t.Setenv("MAX_BOT_TOKEN", "token")
	t.Setenv("INITIAL_ADMIN_MAX_USER_ID", "123456")
	t.Setenv("PUBLIC_BASE_URL", "https://max.conspiracy-team.ru/")
	t.Setenv("MAX_WEBHOOK_SECRET", "secret-123")
	t.Setenv("TARANTOOL_PASSWORD", "tarantool-password")
	t.Setenv("OBJECT_STORAGE_ACCESS_KEY", "seaweedfs")
	t.Setenv("OBJECT_STORAGE_SECRET_KEY", "seaweedfs-password")
	t.Setenv("INVITE_CODE_TTL", "72h")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.PublicBaseURL != "https://max.conspiracy-team.ru" {
		t.Fatalf("PublicBaseURL = %q, want trimmed URL", cfg.PublicBaseURL)
	}
	if cfg.ApplicationName != "\u0416\u041a\u0425 \u043a\u043e\u043d\u0442\u0440\u043e\u043b\u044c" {
		t.Fatalf("ApplicationName = %q, want default", cfg.ApplicationName)
	}
	if cfg.InviteCodeTTL != 72*time.Hour {
		t.Fatalf("InviteCodeTTL = %s, want 72h", cfg.InviteCodeTTL)
	}
}

func TestLoadPolzaAI(t *testing.T) {
	t.Setenv("MAX_BOT_TOKEN", "token")
	t.Setenv("INITIAL_ADMIN_MAX_USER_ID", "123456")
	t.Setenv("PUBLIC_BASE_URL", "https://max.conspiracy-team.ru")
	t.Setenv("MAX_WEBHOOK_SECRET", "secret-123")
	t.Setenv("TARANTOOL_PASSWORD", "tarantool-password")
	t.Setenv("OBJECT_STORAGE_ACCESS_KEY", "seaweedfs")
	t.Setenv("OBJECT_STORAGE_SECRET_KEY", "seaweedfs-password")
	t.Setenv("LLM_ENABLED", "true")
	t.Setenv("POLZA_AI_API_KEY", "polza-key")
	t.Setenv("POLZA_AI_MODEL", "openai/gpt-4o")
	t.Setenv("POLZA_AI_TIMEOUT", "30s")
	t.Setenv("POLZA_AI_MAX_IMAGES", "3")
	t.Setenv("POLZA_AI_MAX_IMAGE_BYTES", "1024")
	t.Setenv("POLZA_AI_MAX_PRICE_RUB", "2.5")
	t.Setenv("POLZA_AI_PROVIDER_ONLY", "openai, azure")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.LLMEnabled || cfg.PolzaAIAPIKey != "polza-key" || cfg.PolzaAIModel != "openai/gpt-4o" {
		t.Fatalf("unexpected Polza.ai config: %#v", cfg)
	}
	if cfg.PolzaAITimeout != 30*time.Second || cfg.PolzaAIMaxImages != 3 || cfg.PolzaAIMaxImageBytes != 1024 {
		t.Fatalf("unexpected Polza.ai limits: %#v", cfg)
	}
	if cfg.PolzaAIMaxPriceRUB != 2.5 || len(cfg.PolzaAIProviderOnly) != 2 {
		t.Fatalf("unexpected Polza.ai routing: %#v", cfg)
	}
}

func TestLoadRequiresPolzaAIKeyOnlyWhenEnabled(t *testing.T) {
	t.Setenv("MAX_BOT_TOKEN", "token")
	t.Setenv("INITIAL_ADMIN_MAX_USER_ID", "123456")
	t.Setenv("PUBLIC_BASE_URL", "https://max.conspiracy-team.ru")
	t.Setenv("MAX_WEBHOOK_SECRET", "secret-123")
	t.Setenv("TARANTOOL_PASSWORD", "tarantool-password")
	t.Setenv("OBJECT_STORAGE_ACCESS_KEY", "seaweedfs")
	t.Setenv("OBJECT_STORAGE_SECRET_KEY", "seaweedfs-password")
	t.Setenv("LLM_ENABLED", "true")
	t.Setenv("POLZA_AI_API_KEY", "")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() error = nil, want missing Polza.ai key error")
	}
}
