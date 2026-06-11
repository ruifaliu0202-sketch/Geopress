package main

import (
	"log"

	"geopress/backend/internal/app"
	"geopress/backend/internal/config"
)

func main() {
	cfg := config.Load()

	server := app.NewServer(cfg)
	if err := server.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("api server stopped: %v", err)
	}
}
