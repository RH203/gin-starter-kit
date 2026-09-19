package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gin-starter-pack/config"

	"gopkg.in/natefinch/lumberjack.v2"
)

// InitLogger initializes a Laravel-style daily/rotating structured slog logger
func InitLogger(cfg *config.LogConfig) (*slog.Logger, io.Closer) {
	// Ensure log directory exists
	logDir := cfg.Directory
	if logDir == "" {
		logDir = "logs"
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		slog.Error("Failed to create log directory", "error", err)
	}

	logFile := filepath.Join(logDir, cfg.Filename)
	if cfg.Filename == "" {
		logFile = filepath.Join(logDir, "app.log")
	}

	// Lumberjack rolling logger (rotates when exceeding MaxSizeMB or daily)
	fileWriter := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    cfg.MaxSizeMB,  // Megabytes
		MaxBackups: cfg.MaxBackups, // Max old log files retained
		MaxAge:     cfg.MaxAgeDays, // Max days to retain old files
		Compress:   cfg.Compress,   // Gzip compression
		LocalTime:  true,
	}

	// MultiWriter sends log to stdout (terminal) and file simultaneously
	multiWriter := io.MultiWriter(os.Stdout, fileWriter)

	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Logger initialized successfully",
		"level", cfg.Level,
		"file", logFile,
		"max_size_mb", cfg.MaxSizeMB,
		"max_backups", cfg.MaxBackups,
	)

	return logger, fileWriter
}
