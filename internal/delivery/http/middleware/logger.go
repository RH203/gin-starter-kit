package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		attrs := []any{
			"status", statusCode,
			"method", method,
			"path", path,
			"query", query,
			"ip", clientIP,
			"latency_ms", latency.Milliseconds(),
		}

		switch {
		case statusCode >= 500:
			slog.Error("HTTP Server Error", attrs...)
		case statusCode >= 400:
			slog.Warn("HTTP Client Error", attrs...)
		default:
			slog.Info("HTTP Request", attrs...)
		}
	}
}
