package app

import (
	"log"

	"geopress/backend/internal/ai"
	"geopress/backend/internal/config"
	"geopress/backend/internal/database"
	"geopress/backend/internal/http/handler"
	"geopress/backend/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

func NewServer(cfg config.Config) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Printf("database unavailable, continuing in memory mode: %v", err)
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
	handler.NewWorkspaceHandler(db, aiConfig).Register(api, middleware.Auth())

	return router
}
