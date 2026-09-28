package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ApplicationVersion     string
	ApplicationName        string
	MaxBotToken            string
	InitialAdminMaxUserID  uint64
	PublicBaseURL          string
	MaxWebhookSecret       string
	TarantoolAddress       string
	TarantoolUser          string
	TarantoolPassword      string
	ObjectStorageEndpoint  string
	ObjectStorageAccessKey string
	ObjectStorageSecretKey string
	ObjectStorageBucket    string
}

func Load() (Config, error) {
	cfg := Config{
		ApplicationVersion:     envOrDefault("APP_VERSION", "dev"),
		ApplicationName:        envOrDefault("APP_NAME", "\u0416\u041a\u0425 \u043a\u043e\u043d\u0442\u0440\u043e\u043b\u044c"),
		MaxBotToken:            os.Getenv("MAX_BOT_TOKEN"),
		InitialAdminMaxUserID:  0,
		PublicBaseURL:          strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/"),
		MaxWebhookSecret:       os.Getenv("MAX_WEBHOOK_SECRET"),
		TarantoolAddress:       envOrDefault("TARANTOOL_ADDRESS", "tarantool:3301"),
		TarantoolUser:          envOrDefault("TARANTOOL_USER", "app"),
		TarantoolPassword:      os.Getenv("TARANTOOL_PASSWORD"),
		ObjectStorageEndpoint:  envOrDefault("OBJECT_STORAGE_ENDPOINT", "seaweedfs:8333"),
		ObjectStorageAccessKey: os.Getenv("OBJECT_STORAGE_ACCESS_KEY"),
		ObjectStorageSecretKey: os.Getenv("OBJECT_STORAGE_SECRET_KEY"),
		ObjectStorageBucket:    envOrDefault("OBJECT_STORAGE_BUCKET", "work-evidence"),
	}
	if rawAdminID := os.Getenv("INITIAL_ADMIN_MAX_USER_ID"); rawAdminID != "" {
		adminID, parseErr := strconv.ParseUint(rawAdminID, 10, 64)
		if parseErr != nil || adminID == 0 {
			return Config{}, fmt.Errorf("INITIAL_ADMIN_MAX_USER_ID must be a positive integer")
		}
		cfg.InitialAdminMaxUserID = adminID
	}

	if cfg.MaxBotToken == "" {
		return Config{}, fmt.Errorf("MAX_BOT_TOKEN is required")
	}
	if cfg.InitialAdminMaxUserID == 0 {
		return Config{}, fmt.Errorf("INITIAL_ADMIN_MAX_USER_ID is required")
	}
	if cfg.PublicBaseURL == "" {
		return Config{}, fmt.Errorf("PUBLIC_BASE_URL is required for webhook mode")
	}
	parsedURL, err := url.Parse(cfg.PublicBaseURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return Config{}, fmt.Errorf("PUBLIC_BASE_URL must be an https URL")
	}
	if len(cfg.MaxWebhookSecret) < 5 {
		return Config{}, fmt.Errorf("MAX_WEBHOOK_SECRET must contain at least 5 characters")
	}
	if cfg.TarantoolPassword == "" {
		return Config{}, fmt.Errorf("TARANTOOL_PASSWORD is required")
	}
	if cfg.ObjectStorageAccessKey == "" || cfg.ObjectStorageSecretKey == "" {
		return Config{}, fmt.Errorf("OBJECT_STORAGE_ACCESS_KEY and OBJECT_STORAGE_SECRET_KEY are required")
	}

	return cfg, nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
