package main

import (
	"fmt"
	"log/slog"
	"os"

	"gin-starter-pack/config"
	"gin-starter-pack/database/seeder"
	"gin-starter-pack/internal/domain"
	"gin-starter-pack/pkg/database"
)

func main() {
	// Load application configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Connect to database
	db, err := database.InitDB(&cfg.Database, cfg.App.Env)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Ensure database schema exists before seeding
	slog.Info("Ensuring database schema exists before seeding...")
	if err := db.AutoMigrate(domain.Entities()...); err != nil {
		slog.Error("Database AutoMigrate failed before seeding", "error", err)
		os.Exit(1)
	}

	// Run all registered database seeders
	if err := seeder.RunAll(db); err != nil {
		slog.Error("Database seeding failed", "error", err)
		os.Exit(1)
	}

	fmt.Println("Database seeding completed successfully.")
}
