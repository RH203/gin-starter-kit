package handler

import (
	"context"
	"net/http"
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
// @Failure      503  {object}  response.APIResponse
// @Router       /health [get]
func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	healthy := true

	// Check Database
	dbStatus := "connected"
	if sqlDB, err := h.db.DB(); err != nil {
		dbStatus = "error: " + err.Error()
		healthy = false
	} else if err := sqlDB.PingContext(ctx); err != nil {
		dbStatus = "error: " + err.Error()
		healthy = false
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

	data := gin.H{
		"status":   "up",
		"database": dbStatus,
		"redis":    redisStatus,
		"time":     time.Now().UTC(),
	}

	if !healthy {
		c.JSON(http.StatusServiceUnavailable, response.APIResponse{
			Success: false,
			Message: "Service dependency degraded or unreachable",
			Data:    data,
		})
		return
	}

	response.OK(c, "Service is healthy", data)
}

// Liveness godoc
// @Summary      Kubernetes Liveness Probe
// @Description  Fast probe checking whether the HTTP server process is running and responsive
// @Tags         Health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health/live [get]
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
		"time":   time.Now().UTC(),
	})
}

// Readiness godoc
// @Summary      Kubernetes Readiness Probe
// @Description  Probe verifying primary database connectivity. Returns 503 if DB is unreachable.
// @Tags         Health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      503  {object}  map[string]string
// @Router       /health/ready [get]
func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil {
		errMsg := "database unavailable"
		if err != nil {
			errMsg = err.Error()
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unready",
			"reason": errMsg,
			"time":   time.Now().UTC(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"time":   time.Now().UTC(),
	})
}
