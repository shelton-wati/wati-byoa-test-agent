package config

import "testing"

func TestLoadLLMFromEnv(t *testing.T) {
	t.Setenv("LLM_BASE_URL", "https://opencode.ai/zen/go/v1")
	t.Setenv("LLM_API_KEY", "test-key")
	t.Setenv("LLM_MODEL", "glm-5.2")
	t.Setenv("OPENAI_API_KEY", "")

	cfg := Load()
	if cfg.LLMBaseURL != "https://opencode.ai/zen/go/v1" {
		t.Fatalf("LLMBaseURL = %q", cfg.LLMBaseURL)
	}
	if cfg.LLMAPIKey != "test-key" {
		t.Fatalf("LLMAPIKey = %q", cfg.LLMAPIKey)
	}
	if cfg.LLMModel != "glm-5.2" {
		t.Fatalf("LLMModel = %q", cfg.LLMModel)
	}
}

func TestLoadOpenAIAPIKeyFallback(t *testing.T) {
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "sk-test")

	cfg := Load()
	if cfg.LLMAPIKey != "sk-test" {
		t.Fatalf("LLMAPIKey = %q", cfg.LLMAPIKey)
	}
}
