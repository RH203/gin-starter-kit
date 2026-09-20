package seeder

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gin-starter-pack/internal/domain"
	"gin-starter-pack/pkg/hash"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserSeeder seeds initial administrator and fake user records
type UserSeeder struct {
	Count int
}

// NewUserSeeder creates a UserSeeder with a specified or default fake user count
func NewUserSeeder(count ...int) *UserSeeder {
	c := 10
	if len(count) > 0 && count[0] > 0 {
		c = count[0]
	}
	return &UserSeeder{Count: c}
}

func (s *UserSeeder) Name() string {
	return "UserSeeder"
}

func (s *UserSeeder) Seed(db *gorm.DB) error {
	// Seed default administrator account if not already present
	adminEmail := "admin@example.com"
	var existingAdmin domain.User
	err := db.Where("email = ?", adminEmail).First(&existingAdmin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		adminPassword, err := hash.HashPassword("password123")
		if err != nil {
			return fmt.Errorf("failed to hash admin password: %w", err)
		}

		admin := domain.User{
			ID:        uuid.NewString(),
			Name:      "System Administrator",
			Email:     adminEmail,
			Password:  adminPassword,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		if err := db.Create(&admin).Error; err != nil {
			return fmt.Errorf("failed to create default admin: %w", err)
		}
		slog.Info("Default admin user created", "email", adminEmail)
	} else if err != nil {
		return fmt.Errorf("failed to query admin user: %w", err)
	} else {
		slog.Info("Default admin user already exists, skipping creation", "email", adminEmail)
	}

	// Seed fake users using gofakeit
	defaultPassword, err := hash.HashPassword("password123")
	if err != nil {
		return fmt.Errorf("failed to hash default fake password: %w", err)
	}

	createdCount := 0
	for i := 0; i < s.Count; i++ {
		email := gofakeit.Email()

		// Verify email does not already exist
		var count int64
		db.Model(&domain.User{}).Where("email = ?", email).Count(&count)
		if count > 0 {
			continue
		}

		fakeUser := domain.User{
			ID:        uuid.NewString(),
			Name:      gofakeit.Name(),
			Email:     email,
			Password:  defaultPassword,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		if err := db.Create(&fakeUser).Error; err != nil {
			slog.Warn("Failed to insert fake user", "email", email, "error", err)
			continue
		}
		createdCount++
	}

	slog.Info("Fake users seeded successfully", "created", createdCount, "target", s.Count)
	return nil
}
