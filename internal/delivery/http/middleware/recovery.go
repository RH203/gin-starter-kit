package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"gin-starter-pack/pkg/response"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Panic recovered",
					"error", r,
					"stack", string(debug.Stack()),
				)
				response.Error(c, http.StatusInternalServerError, "Internal server error occurred", nil)
				c.Abort()
			}
		}()
		c.Next()
	}
}
