package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
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
	MessageBrokerURL       string
	InviteCodeTTL          time.Duration
	LLMEnabled             bool
	PolzaAIBaseURL         string
	PolzaAIAPIKey          string
	PolzaAIModel           string
	PolzaAITimeout         time.Duration
	PolzaAIMaxTokens       int
	PolzaAIMaxImages       int
	PolzaAIMaxImageBytes   int64
	PolzaAIImageDetail     string
	PolzaAIAllowFallbacks  bool
	PolzaAIMaxPriceRUB     float64
	PolzaAIProviderOnly    []string
	PolzaAIPromptVersion   string
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
		MessageBrokerURL:       envOrDefault("MESSAGE_BROKER_URL", "nats://nats:4222"),
		InviteCodeTTL:          24 * time.Hour,
		PolzaAIBaseURL:         envOrDefault("POLZA_AI_BASE_URL", "https://polza.ai/api/v1"),
		PolzaAIAPIKey:          os.Getenv("POLZA_AI_API_KEY"),
		PolzaAIModel:           envOrDefault("POLZA_AI_MODEL", "openai/gpt-4o"),
		PolzaAITimeout:         45 * time.Second,
		PolzaAIMaxTokens:       800,
		PolzaAIMaxImages:       4,
		PolzaAIMaxImageBytes:   8 << 20,
		PolzaAIImageDetail:     envOrDefault("POLZA_AI_IMAGE_DETAIL", "high"),
		PolzaAIAllowFallbacks:  true,
		PolzaAIMaxPriceRUB:     10,
		PolzaAIProviderOnly:    splitCSV(os.Getenv("POLZA_AI_PROVIDER_ONLY")),
		PolzaAIPromptVersion:   envOrDefault("POLZA_AI_PROMPT_VERSION", "v1"),
	}
	if rawEnabled := envOrDefault("LLM_ENABLED", "false"); rawEnabled != "" {
		enabled, parseErr := strconv.ParseBool(rawEnabled)
		if parseErr != nil {
			return Config{}, fmt.Errorf("LLM_ENABLED must be a boolean")
		}
		cfg.LLMEnabled = enabled
	}
	if rawFallbacks := envOrDefault("POLZA_AI_ALLOW_FALLBACKS", "true"); rawFallbacks != "" {
		allowFallbacks, parseErr := strconv.ParseBool(rawFallbacks)
		if parseErr != nil {
			return Config{}, fmt.Errorf("POLZA_AI_ALLOW_FALLBACKS must be a boolean")
		}
		cfg.PolzaAIAllowFallbacks = allowFallbacks
	}
	if rawTimeout := envOrDefault("POLZA_AI_TIMEOUT", "45s"); rawTimeout != "" {
		timeout, parseErr := time.ParseDuration(rawTimeout)
		if parseErr != nil || timeout <= 0 {
			return Config{}, fmt.Errorf("POLZA_AI_TIMEOUT must be a positive duration, for example 45s")
		}
		cfg.PolzaAITimeout = timeout
	}
	if rawMaxPrice := envOrDefault("POLZA_AI_MAX_PRICE_RUB", "10"); rawMaxPrice != "" {
		maxPrice, parseErr := strconv.ParseFloat(rawMaxPrice, 64)
		if parseErr != nil || maxPrice <= 0 {
			return Config{}, fmt.Errorf("POLZA_AI_MAX_PRICE_RUB must be a positive number")
		}
		cfg.PolzaAIMaxPriceRUB = maxPrice
	}
	var parseErr error
	if cfg.PolzaAIMaxTokens, parseErr = positiveIntEnv("POLZA_AI_MAX_TOKENS", 800); parseErr != nil {
		return Config{}, parseErr
	}
	if cfg.PolzaAIMaxImages, parseErr = positiveIntEnv("POLZA_AI_MAX_IMAGES", 4); parseErr != nil {
		return Config{}, parseErr
	}
	maxImageBytes, parseErr := positiveIntEnv("POLZA_AI_MAX_IMAGE_BYTES", 8<<20)
	if parseErr != nil {
		return Config{}, parseErr
	}
	cfg.PolzaAIMaxImageBytes = int64(maxImageBytes)
	if rawTTL := envOrDefault("INVITE_CODE_TTL", "24h"); rawTTL != "" {
		ttl, parseErr := time.ParseDuration(rawTTL)
		if parseErr != nil || ttl <= 0 {
			return Config{}, fmt.Errorf("INVITE_CODE_TTL must be a positive duration, for example 24h")
		}
		cfg.InviteCodeTTL = ttl
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
	if cfg.LLMEnabled {
		if cfg.PolzaAIAPIKey == "" {
			return Config{}, fmt.Errorf("POLZA_AI_API_KEY is required when LLM_ENABLED=true")
		}
		parsedBaseURL, parseErr := url.Parse(cfg.PolzaAIBaseURL)
		if parseErr != nil || parsedBaseURL.Scheme != "https" || parsedBaseURL.Host == "" {
			return Config{}, fmt.Errorf("POLZA_AI_BASE_URL must be an https URL")
		}
		if cfg.PolzaAIModel == "" {
			return Config{}, fmt.Errorf("POLZA_AI_MODEL is required when LLM_ENABLED=true")
		}
		if cfg.PolzaAIImageDetail != "auto" && cfg.PolzaAIImageDetail != "low" && cfg.PolzaAIImageDetail != "high" {
			return Config{}, fmt.Errorf("POLZA_AI_IMAGE_DETAIL must be auto, low or high")
		}
	}

	return cfg, nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func positiveIntEnv(name string, fallback int) (int, error) {
	raw := envOrDefault(name, strconv.Itoa(fallback))
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}

func splitCSV(raw string) []string {
	var values []string
	for _, value := range strings.Split(raw, ",") {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	return values
}
