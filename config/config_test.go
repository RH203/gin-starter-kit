package config_test

import (
	"os"
	"testing"

	"gin-starter-pack/config"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_JSONArrayCORS(t *testing.T) {
	// Set environment variables with JSON array CORS origins
	_ = os.Setenv("CORS_ALLOWED_ORIGINS", `["http://localhost:3000","http://localhost:5173","https://example.com"]`)
	_ = os.Setenv("CORS_ALLOW_CREDENTIALS", "true")
	defer func() {
		_ = os.Unsetenv("CORS_ALLOWED_ORIGINS")
		_ = os.Unsetenv("CORS_ALLOW_CREDENTIALS")
	}()

	cfg, err := config.LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	expectedOrigins := []string{"http://localhost:3000", "http://localhost:5173", "https://example.com"}
	assert.Equal(t, expectedOrigins, cfg.CORS.AllowedOrigins)
	assert.True(t, cfg.CORS.AllowCredentials)
}

func TestLoadConfig_CommaSeparatedCORS(t *testing.T) {
	// Set environment variables with comma-separated CORS origins
	_ = os.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:8000, http://localhost:9000")
	defer func() {
		_ = os.Unsetenv("CORS_ALLOWED_ORIGINS")
	}()

	cfg, err := config.LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	expectedOrigins := []string{"http://localhost:8000", "http://localhost:9000"}
	assert.Equal(t, expectedOrigins, cfg.CORS.AllowedOrigins)
}
