package seeder

import (
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// Seeder defines the contract for database seed operations
type Seeder interface {
	Name() string
	Seed(db *gorm.DB) error
}

// Registry manages and executes registered database seeders
type Registry struct {
	seeders []Seeder
}

// NewRegistry creates a new seeder registry with default seeders
func NewRegistry(seeders ...Seeder) *Registry {
	return &Registry{
		seeders: seeders,
	}
}

// Register appends a seeder to the registry
func (r *Registry) Register(s Seeder) {
	r.seeders = append(r.seeders, s)
}

// Run executes all registered seeders in sequence
func (r *Registry) Run(db *gorm.DB) error {
	slog.Info("Starting database seeding process...")
	for _, s := range r.seeders {
		slog.Info("Executing seeder", "seeder", s.Name())
		if err := s.Seed(db); err != nil {
			return fmt.Errorf("seeder %s failed: %w", s.Name(), err)
		}
		slog.Info("Seeder completed successfully", "seeder", s.Name())
	}
	slog.Info("All database seeders completed successfully.")
	return nil
}

// RunAll creates a default registry with all built-in seeders and executes them
func RunAll(db *gorm.DB) error {
	reg := NewRegistry(
		NewUserSeeder(),
	)
	return reg.Run(db)
}
