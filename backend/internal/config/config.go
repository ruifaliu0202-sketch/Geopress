package config

import "os"

type Config struct {
	AppEnv         string
	HTTPAddr       string
	FrontendOrigin string
	DatabaseURL    string
}

func Load() Config {
	return Config{
		AppEnv:         getEnv("APP_ENV", "development"),
		HTTPAddr:       getEnv("HTTP_ADDR", ":8080"),
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:5173"),
		DatabaseURL:    getEnv("DATABASE_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
