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

func TestAuthMiddleware_HeaderAuth(t *testing.T) {
	jwtService := jwt.NewJWTService("super-secret-key", 24)
	token, err := jwtService.GenerateToken("user-123", "test@example.com")
	assert.NoError(t, err)

	router := setupTestRouter(jwtService)

	// Valid Bearer token
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user-123")
}

func TestAuthMiddleware_CookieAuth(t *testing.T) {
	jwtService := jwt.NewJWTService("super-secret-key", 24)
	token, err := jwtService.GenerateToken("user-456", "cookie@example.com")
	assert.NoError(t, err)

	router := setupTestRouter(jwtService)

	// Valid access_token cookie
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user-456")
}

func TestAuthMiddleware_MissingAuth(t *testing.T) {
	jwtService := jwt.NewJWTService("super-secret-key", 24)
	router := setupTestRouter(jwtService)

	// No cookie and no header
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	jwtService := jwt.NewJWTService("super-secret-key", 24)
	router := setupTestRouter(jwtService)

	// Invalid cookie value
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:     "access_token",
		Value:    "invalid.jwt.token",
		Path:     "/",
		HttpOnly: true,
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
