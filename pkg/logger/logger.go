package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"gin-starter-pack/config"

	"gopkg.in/natefinch/lumberjack.v2"
)

// HandlerBuilder builds an slog.Handler and an optional io.Closer from LogConfig.
// Third parties implementing slog.Handler (e.g., Sentry, Datadog, OpenTelemetry)
// can be registered as custom log drivers using RegisterDriver.
type HandlerBuilder func(cfg *config.LogConfig) (slog.Handler, io.Closer, error)

// WriterBuilder builds an io.Writer and an optional io.Closer from LogConfig.
// Third parties providing an io.Writer (e.g., Grafana Loki, Syslog, Kafka, Graylog)
// can be registered easily using RegisterWriter.
type WriterBuilder func(cfg *config.LogConfig) (io.Writer, io.Closer, error)

var (
	driverMu sync.RWMutex
	drivers  = make(map[string]HandlerBuilder)
)

type nopCloser struct{}

func (nopCloser) Close() error { return nil }

func init() {
	// Register built-in drivers

	// "stdout" or "console": writes directly to standard output
	RegisterWriter("stdout", func(cfg *config.LogConfig) (io.Writer, io.Closer, error) {
		return os.Stdout, nopCloser{}, nil
	})
	RegisterWriter("console", func(cfg *config.LogConfig) (io.Writer, io.Closer, error) {
		return os.Stdout, nopCloser{}, nil
	})

	// "file": daily/rotating file via Lumberjack
	RegisterWriter("file", func(cfg *config.LogConfig) (io.Writer, io.Closer, error) {
		fw := newRollingFileWriter(cfg)
		return fw, fw, nil
	})

	// "stack" or "multi": writes simultaneously to stdout and rolling file
	RegisterWriter("stack", func(cfg *config.LogConfig) (io.Writer, io.Closer, error) {
		fw := newRollingFileWriter(cfg)
		return io.MultiWriter(os.Stdout, fw), fw, nil
	})
	RegisterWriter("multi", func(cfg *config.LogConfig) (io.Writer, io.Closer, error) {
		fw := newRollingFileWriter(cfg)
		return io.MultiWriter(os.Stdout, fw), fw, nil
	})

	// "discard" or "null": discards all logs (useful for unit tests)
	RegisterWriter("discard", func(cfg *config.LogConfig) (io.Writer, io.Closer, error) {
		return io.Discard, nopCloser{}, nil
	})
	RegisterWriter("null", func(cfg *config.LogConfig) (io.Writer, io.Closer, error) {
		return io.Discard, nopCloser{}, nil
	})
}

// RegisterDriver registers a custom slog.Handler builder for third-party integrations
// (e.g. Sentry, Datadog, OpenTelemetry, Logstash).
func RegisterDriver(name string, builder HandlerBuilder) {
	driverMu.Lock()
	defer driverMu.Unlock()
	drivers[strings.ToLower(name)] = builder
}

// RegisterWriter is a convenience wrapper to register drivers that provide an io.Writer.
// The output format (JSON or Text) will be automatically handled according to cfg.Format.
func RegisterWriter(name string, builder WriterBuilder) {
	RegisterDriver(name, func(cfg *config.LogConfig) (slog.Handler, io.Closer, error) {
		w, closer, err := builder(cfg)
		if err != nil {
			return nil, nil, err
		}
		handler := NewHandlerForWriter(w, cfg)
		return handler, closer, nil
	})
}

// GetRegisteredDrivers returns a sorted list of registered driver names.
func GetRegisteredDrivers() []string {
	driverMu.RLock()
	defer driverMu.RUnlock()

	names := make([]string, 0, len(drivers))
	for name := range drivers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ParseLevel converts string level into slog.Level.
func ParseLevel(lvl string) slog.Level {
	switch strings.ToLower(lvl) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// NewHandlerForWriter creates an slog.Handler (JSON or Text) for the specified io.Writer.
func NewHandlerForWriter(w io.Writer, cfg *config.LogConfig) slog.Handler {
	opts := &slog.HandlerOptions{
		Level: ParseLevel(cfg.Level),
	}

	if strings.ToLower(cfg.Format) == "text" || strings.ToLower(cfg.Format) == "console" {
		return slog.NewTextHandler(w, opts)
	}
	return slog.NewJSONHandler(w, opts)
}

func newRollingFileWriter(cfg *config.LogConfig) *lumberjack.Logger {
	logDir := cfg.Directory
	if logDir == "" {
		logDir = "logs"
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		slog.Error("Failed to create log directory", "error", err)
	}

	filename := cfg.Filename
	if filename == "" {
		filename = "app.log"
	}

	return &lumberjack.Logger{
		Filename:   filepath.Join(logDir, filename),
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}
}

// InitLogger initializes an slog structured logger according to cfg.Driver and cfg.Format.
// If the driver is unrecognized, it logs a warning and falls back to stdout.
// The returned io.Closer is guaranteed to be non-nil and safe to call.
func InitLogger(cfg *config.LogConfig) (*slog.Logger, io.Closer) {
	driverName := strings.ToLower(cfg.Driver)
	if driverName == "" {
		driverName = "stack"
	}

	driverMu.RLock()
	builder, exists := drivers[driverName]
	driverMu.RUnlock()

	if !exists {
		fallbackHandler := NewHandlerForWriter(os.Stdout, cfg)
		fallbackLogger := slog.New(fallbackHandler)
		slog.SetDefault(fallbackLogger)

		fallbackLogger.Warn(
			fmt.Sprintf("Unsupported log driver '%s', falling back to stdout. Registered drivers: [%s]",
				cfg.Driver, strings.Join(GetRegisteredDrivers(), ", ")),
		)
		return fallbackLogger, nopCloser{}
	}

	handler, closer, err := builder(cfg)
	if err != nil {
		fallbackHandler := NewHandlerForWriter(os.Stdout, cfg)
		fallbackLogger := slog.New(fallbackHandler)
		slog.SetDefault(fallbackLogger)

		fallbackLogger.Error("Failed to initialize log driver", "driver", cfg.Driver, "error", err)
		return fallbackLogger, nopCloser{}
	}

	if closer == nil {
		closer = nopCloser{}
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	logger.Info("Logger initialized successfully",
		"driver", driverName,
		"format", cfg.Format,
		"level", cfg.Level,
	)

	return logger, closer
}
