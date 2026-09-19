package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gin-starter-pack/config"
	httpDelivery "gin-starter-pack/internal/delivery/http"
	"gin-starter-pack/internal/delivery/http/handler"
	"gin-starter-pack/internal/domain"
	gormRepo "gin-starter-pack/internal/repository/gorm"
	"gin-starter-pack/internal/usecase"
	"gin-starter-pack/pkg/database"
	"gin-starter-pack/pkg/jwt"
	"gin-starter-pack/pkg/logger"
	"gin-starter-pack/pkg/mail"
	"gin-starter-pack/pkg/redis"
	"gin-starter-pack/pkg/worker"
)

// @title           Gin Clean Architecture Starter Pack API
// @version         1.0
// @description     Enterprise REST API starter pack in Go with Clean Architecture, multi-database support, Redis, JWT, rate limiting, and background worker.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	// Load Configuration with Viper
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize Laravel-style Rotating Structured Logger
	_, logCloser := logger.InitLogger(&cfg.Log)
	defer logCloser.Close()

	slog.Info("Starting Gin Starter Pack application...", "env", cfg.App.Env, "app", cfg.App.Name)

	// Initialize Database (Postgres / MySQL / SQLite)
	db, err := database.InitDB(&cfg.Database, cfg.App.Env)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	// Run Auto Migrations
	slog.Info("Running database auto-migrations...")
	if err := db.AutoMigrate(domain.Entities()...); err != nil {
		slog.Error("Failed to auto-migrate database", "error", err)
		os.Exit(1)
	}

	// Initialize Redis (Conditional)
	redisClient, err := redis.InitRedis(&cfg.Redis)
	if err != nil {
		slog.Error("Failed to initialize Redis", "error", err)
		os.Exit(1)
	}

	// Initialize JWT Service
	jwtService := jwt.NewJWTService(cfg.JWT.Secret, cfg.JWT.ExpiryHours)

	// Initialize Background Worker Pool
	workerPool := worker.NewPool(cfg.Worker.Concurrency, cfg.Worker.QueueSize)
	workerPool.Start()

	// Initialize Mailer (SMTP or Log mode)
	mailerService := mail.NewMailer(&mail.Config{
		Driver:      cfg.Mail.Driver,
		Host:        cfg.Mail.Host,
		Port:        cfg.Mail.Port,
		Username:    cfg.Mail.Username,
		Password:    cfg.Mail.Password,
		FromAddress: cfg.Mail.FromAddress,
		FromName:    cfg.Mail.FromName,
		Encryption:  cfg.Mail.Encryption,
	})

	// Dependency Injection
	userRepo := gormRepo.NewUserRepository(db)
	userUsecase := usecase.NewUserUsecase(userRepo, redisClient, jwtService, workerPool, mailerService)

	userHandler := handler.NewUserHandler(userUsecase, cfg.JWT.ExpiryHours*3600)
	healthHandler := handler.NewHealthHandler(db, redisClient)

	router := httpDelivery.SetupRouter(httpDelivery.RouterConfig{
		AppEnv:        cfg.App.Env,
		HealthHandler: healthHandler,
		UserHandler:   userHandler,
		JWTService:    jwtService,
		CORS:          cfg.CORS,
		RateLimit:     cfg.RateLimit,
	})

	// Setup HTTP Server
	serverAddr := fmt.Sprintf(":%s", cfg.App.Port)
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("HTTP server listening", "address", serverAddr, "swagger", fmt.Sprintf("http://localhost:%s/swagger/index.html", cfg.App.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed to listen", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful Shutdown listener
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutdown signal received, shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	// Stop worker pool
	workerPool.Stop(5 * time.Second)

	// Close Redis
	if err := redisClient.Close(); err != nil {
		slog.Error("Error closing Redis client", "error", err)
	}

	// Close Database
	if sqlDB, err := db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			slog.Error("Error closing database connection", "error", err)
		}
	}

	slog.Info("Server stopped cleanly")
}
