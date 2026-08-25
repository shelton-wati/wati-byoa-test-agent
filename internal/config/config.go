package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const StateDir = ".wati-byoa-state"

type Config struct {
	HTTPAddr string

	WebhookToken string

	WatiBaseURL  string
	WatiTenantID string
	WatiAPIToken string
	SourceType   string

	LLMBaseURL string
	LLMAPIKey  string
	LLMModel   string
	LLMTimeout time.Duration

	MaxSteps         int
	MaxOutputTokens  int
	MaxContextTokens int
}

func Default() Config {
	return Config{
		HTTPAddr:         ":8090",
		WebhookToken:     "change-me-webhook-token",
		WatiBaseURL:      "https://live-server.wati.io",
		WatiTenantID:     "",
		WatiAPIToken:     "",
		SourceType:       "API",
		LLMBaseURL:       "https://api.openai.com/v1",
		LLMAPIKey:        "",
		LLMModel:         "gpt-4.1-mini",
		LLMTimeout:       2 * time.Minute,
		MaxSteps:         6,
		MaxOutputTokens:  2048,
		MaxContextTokens: 8000,
	}
}

func Load() Config {
	cfg := Default()
	if v := strings.TrimSpace(os.Getenv("HTTP_ADDR")); v != "" {
		cfg.HTTPAddr = v
	}
	if v := strings.TrimSpace(os.Getenv("WEBHOOK_TOKEN")); v != "" {
		cfg.WebhookToken = v
	}
	if v := strings.TrimSpace(os.Getenv("WATI_BASE_URL")); v != "" {
		cfg.WatiBaseURL = strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv("WATI_TENANT_ID")); v != "" {
		cfg.WatiTenantID = v
	}
	if v := strings.TrimSpace(os.Getenv("WATI_API_TOKEN")); v != "" {
		cfg.WatiAPIToken = v
	}
	if v := strings.TrimSpace(os.Getenv("WATI_SOURCE_TYPE")); v != "" {
		cfg.SourceType = v
	}
	if v := strings.TrimSpace(os.Getenv("LLM_BASE_URL")); v != "" {
		cfg.LLMBaseURL = strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv("LLM_API_KEY")); v != "" {
		cfg.LLMAPIKey = v
	}
	if v := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); v != "" && cfg.LLMAPIKey == "" {
		cfg.LLMAPIKey = v
	}
	if v := strings.TrimSpace(os.Getenv("LLM_MODEL")); v != "" {
		cfg.LLMModel = v
	}
	if v := strings.TrimSpace(os.Getenv("LLM_TIMEOUT")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.LLMTimeout = d
		}
	}
	if v := strings.TrimSpace(os.Getenv("MAX_STEPS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxSteps = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("MAX_OUTPUT_TOKENS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxOutputTokens = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("MAX_CONTEXT_TOKENS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxContextTokens = n
		}
	}
	return cfg
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.WebhookToken) == "" {
		return fmt.Errorf("WEBHOOK_TOKEN is required")
	}
	if strings.TrimSpace(c.WatiBaseURL) == "" {
		return fmt.Errorf("WATI_BASE_URL is required")
	}
	if strings.TrimSpace(c.WatiAPIToken) == "" {
		return fmt.Errorf("WATI_API_TOKEN is required")
	}
	if strings.TrimSpace(c.LLMAPIKey) == "" {
		return fmt.Errorf("LLM_API_KEY or OPENAI_API_KEY is required")
	}
	if strings.TrimSpace(c.LLMModel) == "" {
		return fmt.Errorf("LLM_MODEL is required")
	}
	return nil
}
