package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gin-starter-pack/config"
	"gin-starter-pack/internal/domain"
	"gin-starter-pack/internal/usecase"
	"gin-starter-pack/pkg/database"
	"gin-starter-pack/pkg/logger"
	"gin-starter-pack/pkg/mail"
	"gin-starter-pack/pkg/queue"
	"gin-starter-pack/pkg/redis"
)

func main() {
	// Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Worker failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize Logger
	_, logCloser := logger.InitLogger(&cfg.Log)
	defer logCloser.Close()

	slog.Info("Starting dedicated Queue Worker service...",
		"env", cfg.App.Env,
		"driver", cfg.Queue.Driver,
		"queue", cfg.Queue.QueueName,
		"concurrency", cfg.Queue.Concurrency,
	)

	// Connect to Database
	db, err := database.InitDB(&cfg.Database, cfg.App.Env)
	if err != nil {
		slog.Error("Worker failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Auto-migrate job table if enabled
	if cfg.Database.AutoMigrate {
		if err := db.AutoMigrate(domain.Entities()...); err != nil {
			slog.Warn("Worker auto-migration warning", "error", err)
		}
	}

	// Initialize Redis (Conditional)
	redisClient, err := redis.InitRedis(&cfg.Redis)
	if err != nil {
		slog.Error("Worker failed to initialize Redis", "error", err)
		os.Exit(1)
	}

	// Initialize Mailer
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

	// Setup Shared Queue Registry and register domain job handlers
	reg := queue.NewRegistry()
	usecase.RegisterUserQueueHandlers(reg, mailerService)

	// Initialize Queue Worker
	_, worker, err := queue.InitQueue(&cfg.Queue, reg, db, redisClient)
	if err != nil {
		slog.Error("Failed to initialize queue worker", "error", err)
		os.Exit(1)
	}

	// Start Worker Pool
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := worker.Start(ctx); err != nil {
		slog.Error("Worker failed to start", "error", err)
		os.Exit(1)
	}

	slog.Info("Queue Worker running. Press Ctrl+C to terminate.")

	// Listen for OS interrupt signals for Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutdown signal received, shutting down worker gracefully...")

	// Stop worker gracefully (drain active jobs)
	if err := worker.Stop(15 * time.Second); err != nil {
		slog.Error("Error during worker shutdown", "error", err)
	}

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

	slog.Info("Queue Worker stopped cleanly.")
}
