package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gin-starter-pack/internal/delivery/http/middleware"
	"gin-starter-pack/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter(jwtService *jwt.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/protected", middleware.AuthMiddleware(jwtService), func(c *gin.Context) {
		userID, _ := middleware.GetUserIDFromContext(c)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	return r
}

func TestAuthMiddleware_ValidBearerToken(t *testing.T) {
	jwtService := jwt.NewJWTService("super-secret-key", 24)
	token, err := jwtService.GenerateToken("user-123", "test@example.com")
	assert.NoError(t, err)

	router := setupTestRouter(jwtService)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user-123")
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	jwtService := jwt.NewJWTService("super-secret-key", 24)
	router := setupTestRouter(jwtService)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authorization header is required")
}

func TestAuthMiddleware_InvalidHeaderFormat(t *testing.T) {
	jwtService := jwt.NewJWTService("super-secret-key", 24)
	router := setupTestRouter(jwtService)

	testCases := []struct {
		name       string
		authHeader string
	}{
		{"Not Bearer", "Basic some-token"},
		{"Missing Token", "Bearer"},
		{"Missing Token with space", "Bearer "},
		{"Raw Token without Bearer", "raw-token-string"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", tc.authHeader)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestAuthMiddleware_InvalidOrExpiredToken(t *testing.T) {
	jwtService := jwt.NewJWTService("super-secret-key", 24)
	router := setupTestRouter(jwtService)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid or expired token")
}
