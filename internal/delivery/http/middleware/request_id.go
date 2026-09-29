package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the HTTP header key for Request ID
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the context key for storing the Request ID
	RequestIDKey = "RequestID"
)

// RequestID generates a unique Request ID for each incoming HTTP request
// or preserves an existing one from incoming request headers if valid and sanitized
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(RequestIDHeader)
		if !isValidRequestID(reqID) {
			reqID = uuid.NewString()
		}

		c.Set(RequestIDKey, reqID)
		c.Writer.Header().Set(RequestIDHeader, reqID)

		c.Next()
	}
}

// isValidRequestID ensures the incoming Request ID contains only safe alphanumeric and dash/underscore characters
// and is between 1 and 64 characters long to prevent header splitting and log injection
func isValidRequestID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for i := 0; i < len(id); i++ {
		b := id[i]
		if !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '-' || b == '_' || b == '.') {
			return false
		}
	}
	return true
}

// GetRequestID retrieves the Request ID from the gin context
func GetRequestID(c *gin.Context) string {
	if val, ok := c.Get(RequestIDKey); ok {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}
