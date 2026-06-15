package config

import (
	"os"
	"strings"
)

type Config struct {
	AppEnv           string
	HTTPAddr         string
	FrontendOrigin   string
	DatabaseURL      string
	AIProvider       string
	OpenAIAPIKey     string
	OpenAIBaseURL    string
	OpenAIModel      string
	AIRequestTimeout int
}

func Load() Config {
	return Config{
		AppEnv:           getEnv("APP_ENV", "development"),
		HTTPAddr:         getEnv("HTTP_ADDR", ":18080"),
		FrontendOrigin:   getEnv("FRONTEND_ORIGIN", "http://localhost:5173"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://geopress:geopress@localhost:5432/geopress?sslmode=disable"),
		AIProvider:       getEnv("AI_PROVIDER", defaultAIProvider()),
		OpenAIAPIKey:     getEnv("OPENAI_API_KEY", ""),
		OpenAIBaseURL:    getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAIModel:      getEnv("OPENAI_MODEL", "gpt-5.5"),
		AIRequestTimeout: getEnvInt("AI_REQUEST_TIMEOUT_SECONDS", 45),
	}
}

func defaultAIProvider() string {
	if strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) != "" {
		return "openai"
	}
	return "mock"
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	var result int
	for _, char := range value {
		if char < '0' || char > '9' {
			return fallback
		}
		result = result*10 + int(char-'0')
	}
	if result <= 0 {
		return fallback
	}
	return result
}
