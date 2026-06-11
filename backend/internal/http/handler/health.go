package handler

import (
	"context"
	"net/http"
	"time"

	"geopress/backend/internal/database"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	startedAt time.Time
	db        *database.DB
}

func NewHealthHandler(db *database.DB) *HealthHandler {
	return &HealthHandler{
		startedAt: time.Now().UTC(),
		db:        db,
	}
}

func (h *HealthHandler) Register(router gin.IRouter) {
	router.GET("/healthz", h.Healthz)
}

func (h *HealthHandler) Healthz(c *gin.Context) {
	dbStatus := "not_configured"
	if h.db != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		if err := h.db.Ping(ctx); err != nil {
			dbStatus = "down"
		} else {
			dbStatus = "ok"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"startedAt": h.startedAt,
		"database":  dbStatus,
	})
}
