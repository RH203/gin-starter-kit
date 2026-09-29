package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gin-starter-pack/config"
	"gin-starter-pack/internal/delivery/http/handler"
	"gin-starter-pack/pkg/database"
	"gin-starter-pack/pkg/redis"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler_Liveness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.NewHealthHandler(nil, nil)

	r := gin.New()
	r.GET("/health/live", h.Liveness)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"alive"`)
}

func TestHealthHandler_Readiness_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := database.InitDB(&config.DBConfig{Driver: "sqlite", Name: ":memory:"}, "test")
	require.NoError(t, err)

	h := handler.NewHealthHandler(db, nil)

	r := gin.New()
	r.GET("/health/ready", h.Readiness)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"ready"`)
}

func TestHealthHandler_Readiness_FailureWhenDBClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := database.InitDB(&config.DBConfig{Driver: "sqlite", Name: ":memory:"}, "test")
	require.NoError(t, err)

	// Close database to simulate failure
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	h := handler.NewHealthHandler(db, nil)

	r := gin.New()
	r.GET("/health/ready", h.Readiness)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"unready"`)
}

func TestHealthHandler_Check(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := database.InitDB(&config.DBConfig{Driver: "sqlite", Name: ":memory:"}, "test")
	require.NoError(t, err)

	cache, _ := redis.InitRedis(&config.RedisConfig{Enabled: false})
	h := handler.NewHealthHandler(db, cache)

	r := gin.New()
	r.GET("/health", h.Check)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"up"`)
	assert.Contains(t, w.Body.String(), `"database":"connected"`)
	assert.Contains(t, w.Body.String(), `"redis":"disabled"`)
}
