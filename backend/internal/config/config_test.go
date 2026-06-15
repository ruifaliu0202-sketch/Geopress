package config

import "testing"

func TestLoadDefaultsAIProviderToOpenAIWhenKeyConfigured(t *testing.T) {
	t.Setenv("AI_PROVIDER", "")
	t.Setenv("OPENAI_API_KEY", "sk-test")

	cfg := Load()
	if cfg.AIProvider != "openai" {
		t.Fatalf("AIProvider = %q, want openai", cfg.AIProvider)
	}
}

func TestLoadDefaultsAIProviderToMockWithoutKey(t *testing.T) {
	t.Setenv("AI_PROVIDER", "")
	t.Setenv("OPENAI_API_KEY", "")

	cfg := Load()
	if cfg.AIProvider != "mock" {
		t.Fatalf("AIProvider = %q, want mock", cfg.AIProvider)
	}
}
