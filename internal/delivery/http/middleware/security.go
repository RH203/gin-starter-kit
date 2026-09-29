package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders applies OWASP-recommended HTTP security headers to protect against common web vulnerabilities
func SecurityHeaders(appEnv string) gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()

		// Prevent MIME-sniffing
		headers.Set("X-Content-Type-Options", "nosniff")

		// Prevent Clickjacking attacks
		headers.Set("X-Frame-Options", "DENY")

		// Cross-site scripting (XSS) filter protection for legacy browsers
		headers.Set("X-XSS-Protection", "1; mode=block")

		// Control referrer information leakage
		headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Enforce HTTPS HSTS only in production environments
		if appEnv == "production" {
			headers.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}
