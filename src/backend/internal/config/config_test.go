package config_test

import (
	"testing"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/config"
)

func TestLoadRequiresWebhookConfiguration(t *testing.T) {
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
