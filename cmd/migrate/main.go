package main

import (
	"fmt"
	"log/slog"
	"os"

	"gin-starter-pack/config"
	"gin-starter-pack/internal/domain"
	"gin-starter-pack/pkg/database"
)

func main() {
	// Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Connect to Database
	db, err := database.InitDB(&cfg.Database, cfg.App.Env)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Execute GORM AutoMigrate for registered domain entities
	slog.Info("Running GORM AutoMigrate...")
	if err := db.AutoMigrate(domain.Entities()...); err != nil {
		slog.Error("GORM AutoMigrate failed", "error", err)
		os.Exit(1)
	}

	fmt.Println("GORM AutoMigrate executed successfully.")
}
