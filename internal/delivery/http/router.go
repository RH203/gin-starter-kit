package http

import (
	"gin-starter-pack/config"
	_ "gin-starter-pack/docs"
	"gin-starter-pack/internal/delivery/http/handler"
	"gin-starter-pack/internal/delivery/http/middleware"
	"gin-starter-pack/pkg/jwt"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type RouterConfig struct {
	AppEnv        string
	HealthHandler *handler.HealthHandler
	UserHandler   *handler.UserHandler
	JWTService    *jwt.Service
	CORS          config.CorsConfig
	RateLimit     config.RateLimitConfig
}

func SetupRouter(cfg RouterConfig) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	// Global Middlewares
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS(cfg.CORS))
	router.Use(middleware.NewIPRateLimiter(cfg.RateLimit).Handler())

	// Swagger API Docs
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health Check
	router.GET("/health", cfg.HealthHandler.Check)

	// API V1 Group
	v1 := router.Group("/api/v1")
	{
		// Authentication Routes (Public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", cfg.UserHandler.Create)
			auth.POST("/login", cfg.UserHandler.Login)
			auth.POST("/logout", cfg.UserHandler.Logout)

			// Protected Profile Route
			auth.GET("/me", middleware.AuthMiddleware(cfg.JWTService), cfg.UserHandler.GetProfile)
		}

		// User Management Routes (Protected)
		users := v1.Group("/users")
		users.Use(middleware.AuthMiddleware(cfg.JWTService))
		{
			users.POST("", cfg.UserHandler.Create)
			users.GET("", cfg.UserHandler.GetAll)
			users.GET("/:id", cfg.UserHandler.GetByID)
			users.PUT("/:id", cfg.UserHandler.Update)
			users.DELETE("/:id", cfg.UserHandler.Delete)
		}
	}

	return router
}
