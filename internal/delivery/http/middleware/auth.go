package middleware

import (
	"net/http"
	"strings"

	"gin-starter-pack/pkg/jwt"
	"gin-starter-pack/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserIDKey    = "userID"
	ContextUserEmailKey = "userEmail"
)

// AuthMiddleware creates a Gin middleware that validates JWT tokens from Authorization header
func AuthMiddleware(jwtService *jwt.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, "Authorization header is required", nil)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(c, http.StatusUnauthorized, "Authorization header must be in Bearer token format", nil)
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			response.Error(c, http.StatusUnauthorized, "Bearer token is empty", nil)
			c.Abort()
			return
		}

		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "Invalid or expired token", err.Error())
			c.Abort()
			return
		}

		// Store user details in gin context
		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextUserEmailKey, claims.Email)

		c.Next()
	}
}

// GetUserIDFromContext retrieves the authenticated user ID from context
func GetUserIDFromContext(c *gin.Context) (string, bool) {
	val, exists := c.Get(ContextUserIDKey)
	if !exists {
		return "", false
	}
	id, ok := val.(string)
	return id, ok
}
