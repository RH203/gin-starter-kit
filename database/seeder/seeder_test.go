package seeder_test

import (
	"testing"

	"gin-starter-pack/config"
	"gin-starter-pack/database/seeder"
	"gin-starter-pack/internal/domain"
	"gin-starter-pack/pkg/database"

	"github.com/stretchr/testify/assert"
)

func TestSeeder_RunAll(t *testing.T) {
	cfg := &config.DBConfig{
		Driver: "sqlite",
		Name:   ":memory:",
	}

	db, err := database.InitDB(cfg, "test")
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// Auto-migrate schema
	err = db.AutoMigrate(domain.Entities()...)
	assert.NoError(t, err)

	// Run seeders first time
	err = seeder.RunAll(db)
	assert.NoError(t, err)

	// Verify admin exists
	var admin domain.User
	err = db.Where("email = ?", "admin@example.com").First(&admin).Error
	assert.NoError(t, err)
	assert.Equal(t, "System Administrator", admin.Name)

	// Verify total count (1 admin + 10 fake users = 11)
	var count int64
	err = db.Model(&domain.User{}).Count(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(11), count)

	// Re-run seeders to test idempotency
	err = seeder.RunAll(db)
	assert.NoError(t, err)
}
