package handler

import (
	"context"
	"time"

	"gin-starter-pack/pkg/redis"
	"gin-starter-pack/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db    *gorm.DB
	cache *redis.Client
}

func NewHealthHandler(db *gorm.DB, cache *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, cache: cache}
}

// Check godoc
// @Summary      Health check
// @Description  Get system health status including database and redis connections
// @Tags         Health
// @Produce      json
// @Success      200  {object}  response.APIResponse
// @Router       /health [get]
func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	// Check Database
	dbStatus := "connected"
	if sqlDB, err := h.db.DB(); err != nil {
		dbStatus = "error: " + err.Error()
	} else if err := sqlDB.PingContext(ctx); err != nil {
		dbStatus = "error: " + err.Error()
	}

	// Check Redis
	redisStatus := "disabled"
	if h.cache.IsEnabled() {
		if err := h.cache.Ping(ctx); err != nil {
			redisStatus = "error: " + err.Error()
		} else {
			redisStatus = "connected"
		}
	}

	response.OK(c, "Service is healthy", gin.H{
		"status":   "up",
		"database": dbStatus,
		"redis":    redisStatus,
		"time":     time.Now().UTC(),
	})
}
