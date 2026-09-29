package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gin-starter-pack/internal/delivery/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRequestIDMiddleware_GeneratesNewID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID())

	var capturedID string
	router.GET("/test-id", func(c *gin.Context) {
		capturedID = middleware.GetRequestID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test-id", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, capturedID)
	assert.Equal(t, capturedID, w.Header().Get(middleware.RequestIDHeader))
}

func TestRequestIDMiddleware_PreservesExistingID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID())

	existingID := "custom-trace-uuid-12345"
	var capturedID string
	router.GET("/test-id", func(c *gin.Context) {
		capturedID = middleware.GetRequestID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test-id", nil)
	req.Header.Set(middleware.RequestIDHeader, existingID)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, existingID, capturedID)
	assert.Equal(t, existingID, w.Header().Get(middleware.RequestIDHeader))
}

func TestRequestIDMiddleware_SanitizesInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID())

	maliciousID := "malicious-id\r\nSet-Cookie: evil=true"
	var capturedID string
	router.GET("/test-id", func(c *gin.Context) {
		capturedID = middleware.GetRequestID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test-id", nil)
	req.Header.Set(middleware.RequestIDHeader, maliciousID)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEqual(t, maliciousID, capturedID, "Malicious CRLF request ID must be rejected")
	assert.NotEmpty(t, capturedID)
	assert.NotContains(t, capturedID, "\r")
	assert.NotContains(t, capturedID, "\n")
}
