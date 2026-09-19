package middleware

import (
	"net/http"
	"strings"

	"gin-starter-pack/config"

	"github.com/gin-gonic/gin"
)

// CORS provides configurable Cross-Origin Resource Sharing middleware
func CORS(cfg config.CorsConfig) gin.HandlerFunc {
	methods := "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	if len(cfg.AllowedMethods) > 0 {
		methods = strings.Join(cfg.AllowedMethods, ", ")
	}

	headers := "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With"
	if len(cfg.AllowedHeaders) > 0 {
		headers = strings.Join(cfg.AllowedHeaders, ", ")
	}

	allowedOriginsMap := make(map[string]bool)
	allowAll := false
	for _, origin := range cfg.AllowedOrigins {
		if origin == "*" {
			allowAll = true
		}
		allowedOriginsMap[origin] = true
	}

	return func(c *gin.Context) {
		reqOrigin := c.Request.Header.Get("Origin")

		// Determine allowed origin header
		if reqOrigin != "" {
			if allowAll {
				if cfg.AllowCredentials {
					// Browsers require explicit origin when credentials are included
					c.Writer.Header().Set("Access-Control-Allow-Origin", reqOrigin)
				} else {
					c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
				}
			} else if allowedOriginsMap[reqOrigin] {
				c.Writer.Header().Set("Access-Control-Allow-Origin", reqOrigin)
			}
		} else if allowAll && !cfg.AllowCredentials {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		if cfg.AllowCredentials {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", headers)
		c.Writer.Header().Set("Access-Control-Allow-Methods", methods)
		c.Writer.Header().Set("Vary", "Origin")

		// Handle preflight requests
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
