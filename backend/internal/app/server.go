package app

import (
	"fmt"
	"log"

	"geopress/backend/internal/ai"
	"geopress/backend/internal/config"
	"geopress/backend/internal/database"
	"geopress/backend/internal/http/handler"
	"geopress/backend/internal/http/middleware"

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
	aiConfig := ai.NewRuntimeConfig(ai.Config{
		Provider:       cfg.AIProvider,
		OpenAIAPIKey:   cfg.OpenAIAPIKey,
		OpenAIBaseURL:  cfg.OpenAIBaseURL,
		OpenAIModel:    cfg.OpenAIModel,
		RequestTimeout: cfg.AIRequestTimeout,
	})
	workspaceHandler, err := handler.NewWorkspaceHandlerWithError(db, aiConfig)
	if err != nil {
		return nil, err
	}
	workspaceHandler.Register(api, middleware.AuthWithDatabase(db))

	return router, nil
}
