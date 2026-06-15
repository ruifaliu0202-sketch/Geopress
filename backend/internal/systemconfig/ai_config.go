package systemconfig

import (
	"context"

	"geopress/backend/internal/ai"
	"geopress/backend/internal/database"
)

const (
	AISettingKey       = "ai.runtime"
	OpenAIAPISecretKey = "ai.openai_api_key"
)

type AISetting struct {
	Provider           string                        `json:"provider"`
	OpenAIBaseURL      string                        `json:"openAIBaseUrl"`
	OpenAIModel        string                        `json:"openAIModel"`
	RequestTimeout     int                           `json:"requestTimeoutSeconds"`
	GenerationPipeline ai.GenerationPipelineSettings `json:"generationPipeline"`
	EnableMockFallback bool                          `json:"enableMockFallback"`
}

func LoadAIConfig(ctx context.Context, db *database.DB, fallback ai.Config) (ai.Config, error) {
	if db == nil || db.SQL() == nil {
		return fallback, nil
	}

	cfg := fallback
	var setting AISetting
	found, err := db.SystemSetting(ctx, AISettingKey, &setting)
	if err != nil {
		return ai.Config{}, err
	}
	if found {
		cfg.Provider = setting.Provider
		cfg.OpenAIBaseURL = setting.OpenAIBaseURL
		cfg.OpenAIModel = setting.OpenAIModel
		cfg.RequestTimeout = setting.RequestTimeout
		cfg.GenerationPipeline = setting.GenerationPipeline
	}

	if secret, ok, err := db.SystemSecret(ctx, OpenAIAPISecretKey); err != nil {
		return ai.Config{}, err
	} else if ok {
		cfg.OpenAIAPIKey = secret
	}
	return cfg, nil
}

func SaveAIConfig(ctx context.Context, db *database.DB, cfg ai.Config, clearAPIKey bool, updatedBy string) error {
	if db == nil || db.SQL() == nil {
		return nil
	}

	setting := AISetting{
		Provider:           cfg.Provider,
		OpenAIBaseURL:      cfg.OpenAIBaseURL,
		OpenAIModel:        cfg.OpenAIModel,
		RequestTimeout:     cfg.RequestTimeout,
		GenerationPipeline: cfg.GenerationPipeline,
		EnableMockFallback: true,
	}
	if err := db.UpsertSystemSetting(ctx, AISettingKey, setting, "json", "AI provider, model and generation pipeline runtime settings.", updatedBy); err != nil {
		return err
	}
	if clearAPIKey {
		return db.DeleteSystemSecret(ctx, OpenAIAPISecretKey)
	}
	if cfg.OpenAIAPIKey != "" {
		return db.UpsertSystemSecret(ctx, OpenAIAPISecretKey, cfg.OpenAIAPIKey, "db", "OpenAI-compatible provider API key.", updatedBy)
	}
	return nil
}

func SeedAIConfigIfMissing(ctx context.Context, db *database.DB, cfg ai.Config) error {
	if db == nil || db.SQL() == nil {
		return nil
	}

	var setting AISetting
	found, err := db.SystemSetting(ctx, AISettingKey, &setting)
	if err != nil {
		return err
	}
	if found {
		if cfg.OpenAIAPIKey != "" {
			if _, ok, err := db.SystemSecret(ctx, OpenAIAPISecretKey); err != nil {
				return err
			} else if !ok {
				return db.UpsertSystemSecret(ctx, OpenAIAPISecretKey, cfg.OpenAIAPIKey, "db", "OpenAI-compatible provider API key.", "")
			}
		}
		return nil
	}

	return SaveAIConfig(ctx, db, cfg, false, "")
}
