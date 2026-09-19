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

// AuthMiddleware creates a Gin middleware that validates JWT tokens from cookies or Authorization header
func AuthMiddleware(jwtService *jwt.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Attempt to extract token from HTTP-only cookie first
		if cookieToken, err := c.Cookie("access_token"); err == nil && cookieToken != "" {
			tokenString = cookieToken
		} else if cookieToken, err := c.Cookie("token"); err == nil && cookieToken != "" {
			tokenString = cookieToken
		} else {
			// Fallback to Authorization Header
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					tokenString = strings.TrimSpace(parts[1])
				}
			}
		}

		if tokenString == "" {
			response.Error(c, http.StatusUnauthorized, "Authentication required (missing auth cookie or Authorization header)", nil)
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
