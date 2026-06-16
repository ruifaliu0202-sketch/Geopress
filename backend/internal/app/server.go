package app

import (
	"context"
	"fmt"
	"log"

	"geopress/backend/internal/ai"
	"geopress/backend/internal/config"
	"geopress/backend/internal/database"
	"geopress/backend/internal/http/handler"
	"geopress/backend/internal/http/middleware"
	"geopress/backend/internal/systemconfig"
	"geopress/backend/internal/web"

	"github.com/gin-gonic/gin"
)

func NewServer(cfg config.Config) *gin.Engine {
	server, err := NewServerWithError(cfg)
	if err != nil {
		log.Fatalf("create api server: %v", err)
	}
	return server
}

func NewServerWithError(cfg config.Config) (*gin.Engine, error) {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}
	if db == nil {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(middleware.CORS(cfg.FrontendOrigin))

	api := router.Group("/api")
	handler.NewHealthHandler(db).Register(api)
	envAIConfig := ai.Config{
		Provider:       cfg.AIProvider,
		OpenAIAPIKey:   cfg.OpenAIAPIKey,
		OpenAIBaseURL:  cfg.OpenAIBaseURL,
		OpenAIModel:    cfg.OpenAIModel,
		RequestTimeout: cfg.AIRequestTimeout,
	}
	if err := systemconfig.SeedAIConfigIfMissing(context.Background(), db, envAIConfig); err != nil {
		return nil, fmt.Errorf("seed ai config: %w", err)
	}
	persistedAIConfig, err := systemconfig.LoadAIConfig(context.Background(), db, envAIConfig)
	if err != nil {
		return nil, fmt.Errorf("load ai config: %w", err)
	}
	aiConfig := ai.NewRuntimeConfig(persistedAIConfig)
	workspaceHandler, err := handler.NewWorkspaceHandlerWithError(db, aiConfig)
	if err != nil {
		return nil, err
	}
	workspaceHandler.Register(api, middleware.AuthWithDatabase(db))
	web.Register(router)

	return router, nil
}
