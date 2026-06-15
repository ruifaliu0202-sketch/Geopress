package systemconfig

import (
	"context"
	"testing"

	"geopress/backend/internal/ai"
)

func TestLoadAIConfigWithoutDatabaseUsesFallback(t *testing.T) {
	fallback := ai.Config{
		Provider:       ai.ProviderOpenAI,
		OpenAIAPIKey:   "sk-test",
		OpenAIBaseURL:  "https://example.test/v1",
		OpenAIModel:    "model-test",
		RequestTimeout: 12,
	}

	cfg, err := LoadAIConfig(context.Background(), nil, fallback)
	if err != nil {
		t.Fatalf("LoadAIConfig returned error: %v", err)
	}
	if cfg.Provider != fallback.Provider || cfg.OpenAIAPIKey != fallback.OpenAIAPIKey || cfg.OpenAIModel != fallback.OpenAIModel {
		t.Fatalf("config = %#v, want fallback %#v", cfg, fallback)
	}
}

func TestSaveAIConfigWithoutDatabaseIsNoop(t *testing.T) {
	err := SaveAIConfig(context.Background(), nil, ai.Config{Provider: ai.ProviderMock}, false, "")
	if err != nil {
		t.Fatalf("SaveAIConfig returned error: %v", err)
	}
}
