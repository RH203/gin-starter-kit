package logger_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gin-starter-pack/config"
	"gin-starter-pack/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitLogger_StdoutText(t *testing.T) {
	cfg := &config.LogConfig{
		Driver: "stdout",
		Format: "text",
		Level:  "debug",
	}

	l, closer := logger.InitLogger(cfg)
	require.NotNil(t, l)
	require.NotNil(t, closer)
	assert.NoError(t, closer.Close())

	assert.True(t, l.Enabled(context.Background(), slog.LevelDebug))
}

func TestInitLogger_Discard(t *testing.T) {
	cfg := &config.LogConfig{
		Driver: "discard",
		Format: "json",
		Level:  "info",
	}

	l, closer := logger.InitLogger(cfg)
	require.NotNil(t, l)
	require.NotNil(t, closer)
	assert.NoError(t, closer.Close())
}

func TestInitLogger_File(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.LogConfig{
		Driver:     "file",
		Format:     "json",
		Level:      "info",
		Directory:  tmpDir,
		Filename:   "test.log",
		MaxSizeMB:  1,
		MaxBackups: 1,
		MaxAgeDays: 1,
	}

	l, closer := logger.InitLogger(cfg)
	require.NotNil(t, l)
	require.NotNil(t, closer)

	l.Info("testing file logger output", "key", "val")
	_ = closer.Close()

	// Verify log file exists and contains message
	content, err := os.ReadFile(filepath.Join(tmpDir, "test.log"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "testing file logger output")
	assert.Contains(t, string(content), `"key":"val"`)
}

func TestInitLogger_FallbackOnUnknownDriver(t *testing.T) {
	cfg := &config.LogConfig{
		Driver: "nonexistent-driver",
		Format: "json",
		Level:  "info",
	}

	l, closer := logger.InitLogger(cfg)
	require.NotNil(t, l)
	require.NotNil(t, closer)
	assert.NoError(t, closer.Close())
}

func TestRegisterDriver_CustomHandler(t *testing.T) {
	var buf bytes.Buffer
	customCalled := false

	logger.RegisterDriver("custom-test-driver", func(cfg *config.LogConfig) (slog.Handler, io.Closer, error) {
		customCalled = true
		handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
		return handler, nil, nil
	})

	cfg := &config.LogConfig{
		Driver: "custom-test-driver",
		Level:  "info",
	}

	l, closer := logger.InitLogger(cfg)
	require.NotNil(t, l)
	require.NotNil(t, closer)
	assert.True(t, customCalled)

	l.Info("custom message", "source", "unit-test")
	assert.Contains(t, buf.String(), "custom message")
	assert.Contains(t, buf.String(), `"source":"unit-test"`)
	assert.NoError(t, closer.Close())
}

func TestRegisterWriter_CustomWriter(t *testing.T) {
	var buf bytes.Buffer

	logger.RegisterWriter("custom-writer-driver", func(cfg *config.LogConfig) (io.Writer, io.Closer, error) {
		return &buf, nil, nil
	})

	cfg := &config.LogConfig{
		Driver: "custom-writer-driver",
		Format: "text",
		Level:  "warn",
	}

	l, closer := logger.InitLogger(cfg)
	require.NotNil(t, l)
	require.NotNil(t, closer)

	l.Warn("warning from custom writer", "metric", 42)
	assert.Contains(t, buf.String(), "warning from custom writer")
	assert.Contains(t, buf.String(), "metric=42")
	assert.NoError(t, closer.Close())
}

func TestGetRegisteredDrivers(t *testing.T) {
	drivers := logger.GetRegisteredDrivers()
	assert.Contains(t, drivers, "stdout")
	assert.Contains(t, drivers, "file")
	assert.Contains(t, drivers, "stack")
	assert.Contains(t, drivers, "discard")
}

func TestParseLevel(t *testing.T) {
	assert.Equal(t, slog.LevelDebug, logger.ParseLevel("debug"))
	assert.Equal(t, slog.LevelWarn, logger.ParseLevel("warn"))
	assert.Equal(t, slog.LevelWarn, logger.ParseLevel("warning"))
	assert.Equal(t, slog.LevelError, logger.ParseLevel("error"))
	assert.Equal(t, slog.LevelInfo, logger.ParseLevel("info"))
	assert.Equal(t, slog.LevelInfo, logger.ParseLevel("unknown"))
	assert.Equal(t, slog.LevelInfo, logger.ParseLevel(strings.ToUpper("INFO")))
}
