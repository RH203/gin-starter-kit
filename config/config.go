package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App       AppConfig
	Database  DBConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Mail      MailConfig
	Log       LogConfig
	CORS      CorsConfig
	RateLimit RateLimitConfig
	Worker    WorkerConfig
}

type AppConfig struct {
	Name string
	Port string
	Env  string // "development", "production", "test"
}

type DBConfig struct {
	Driver          string // "postgres", "mysql", "sqlite"
	DSN             string // Direct connection string (optional)
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int // in minutes
}

type RedisConfig struct {
	Enabled  bool
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret      string
	ExpiryHours int
}

type MailConfig struct {
	Driver      string // "smtp", "log"
	Host        string
	Port        int
	Username    string
	Password    string
	FromAddress string
	FromName    string
	Encryption  string // "tls", "ssl", "none"
}

type LogConfig struct {
	Driver     string // "stdout", "file", "stack", "discard", or custom registered driver
	Format     string // "json", "text"
	Level      string // "debug", "info", "warn", "error"
	Directory  string // e.g. "logs"
	Filename   string // e.g. "app.log"
	MaxSizeMB  int    // max size in megabytes before rotation
	MaxBackups int    // max number of old log files retained
	MaxAgeDays int    // max days to retain old log files
	Compress   bool   // compress rotated files with gzip
}

type CorsConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

type RateLimitConfig struct {
	Enabled           bool
	RequestsPerSecond float64
	Burst             int
}

type WorkerConfig struct {
	Concurrency int
	QueueSize   int
}

func LoadConfig() (*Config, error) {
	v := viper.New()

	// Default configurations
	v.SetDefault("APP_NAME", "gin-starter-pack")
	v.SetDefault("APP_PORT", "8080")
	v.SetDefault("APP_ENV", "development")

	v.SetDefault("DB_DRIVER", "sqlite")
	v.SetDefault("DB_DSN", "")
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", "5432")
	v.SetDefault("DB_USER", "postgres")
	v.SetDefault("DB_PASSWORD", "postgres")
	v.SetDefault("DB_NAME", "app.db")
	v.SetDefault("DB_SSLMODE", "disable")
	v.SetDefault("DB_MAX_OPEN_CONNS", 25)
	v.SetDefault("DB_MAX_IDLE_CONNS", 10)
	v.SetDefault("DB_CONN_MAX_LIFETIME", 15)

	v.SetDefault("REDIS_ENABLED", false)
	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", "6379")
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", 0)

	v.SetDefault("JWT_SECRET", "super-secret-key-change-me-in-production")
	v.SetDefault("JWT_EXPIRY_HOURS", 24)

	v.SetDefault("MAIL_DRIVER", "log")
	v.SetDefault("MAIL_HOST", "")
	v.SetDefault("MAIL_PORT", 587)
	v.SetDefault("MAIL_USERNAME", "")
	v.SetDefault("MAIL_PASSWORD", "")
	v.SetDefault("MAIL_FROM_ADDRESS", "noreply@example.com")
	v.SetDefault("MAIL_FROM_NAME", "Gin Starter Pack")
	v.SetDefault("MAIL_ENCRYPTION", "tls")

	v.SetDefault("LOG_DRIVER", "stack")
	v.SetDefault("LOG_FORMAT", "json")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_DIR", "logs")
	v.SetDefault("LOG_FILENAME", "app.log")
	v.SetDefault("LOG_MAX_SIZE_MB", 100)
	v.SetDefault("LOG_MAX_BACKUPS", 30)
	v.SetDefault("LOG_MAX_AGE_DAYS", 30)
	v.SetDefault("LOG_COMPRESS", true)

	v.SetDefault("CORS_ALLOWED_ORIGINS", "*")
	v.SetDefault("CORS_ALLOWED_METHODS", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	v.SetDefault("CORS_ALLOWED_HEADERS", "Content-Type,Authorization,X-Requested-With,Accept")
	v.SetDefault("CORS_ALLOW_CREDENTIALS", true)

	v.SetDefault("RATE_LIMIT_ENABLED", true)
	v.SetDefault("RATE_LIMIT_RPS", 20.0)
	v.SetDefault("RATE_LIMIT_BURST", 40)

	v.SetDefault("WORKER_CONCURRENCY", 5)
	v.SetDefault("WORKER_QUEUE_SIZE", 100)

	// Read .env file
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	if err := v.ReadInConfig(); err != nil {
		// Ignore ConfigFileNotFoundError, environment variables may be supplied directly
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// File exists but parsing error
			_ = err
		}
	}

	// Read environment variables override
	v.AutomaticEnv()

	// Parse CORS slice values
	originsStr := v.GetString("CORS_ALLOWED_ORIGINS")
	methodsStr := v.GetString("CORS_ALLOWED_METHODS")
	headersStr := v.GetString("CORS_ALLOWED_HEADERS")

	cfg := &Config{
		App: AppConfig{
			Name: v.GetString("APP_NAME"),
			Port: v.GetString("APP_PORT"),
			Env:  v.GetString("APP_ENV"),
		},
		Database: DBConfig{
			Driver:          v.GetString("DB_DRIVER"),
			DSN:             v.GetString("DB_DSN"),
			Host:            v.GetString("DB_HOST"),
			Port:            v.GetString("DB_PORT"),
			User:            v.GetString("DB_USER"),
			Password:        v.GetString("DB_PASSWORD"),
			Name:            v.GetString("DB_NAME"),
			SSLMode:         v.GetString("DB_SSLMODE"),
			MaxOpenConns:    v.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    v.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: v.GetInt("DB_CONN_MAX_LIFETIME"),
		},
		Redis: RedisConfig{
			Enabled:  v.GetBool("REDIS_ENABLED"),
			Host:     v.GetString("REDIS_HOST"),
			Port:     v.GetString("REDIS_PORT"),
			Password: v.GetString("REDIS_PASSWORD"),
			DB:       v.GetInt("REDIS_DB"),
		},
		JWT: JWTConfig{
			Secret:      v.GetString("JWT_SECRET"),
			ExpiryHours: v.GetInt("JWT_EXPIRY_HOURS"),
		},
		Mail: MailConfig{
			Driver:      v.GetString("MAIL_DRIVER"),
			Host:        v.GetString("MAIL_HOST"),
			Port:        v.GetInt("MAIL_PORT"),
			Username:    v.GetString("MAIL_USERNAME"),
			Password:    v.GetString("MAIL_PASSWORD"),
			FromAddress: v.GetString("MAIL_FROM_ADDRESS"),
			FromName:    v.GetString("MAIL_FROM_NAME"),
			Encryption:  v.GetString("MAIL_ENCRYPTION"),
		},
		Log: LogConfig{
			Driver:     v.GetString("LOG_DRIVER"),
			Format:     v.GetString("LOG_FORMAT"),
			Level:      v.GetString("LOG_LEVEL"),
			Directory:  v.GetString("LOG_DIR"),
			Filename:   v.GetString("LOG_FILENAME"),
			MaxSizeMB:  v.GetInt("LOG_MAX_SIZE_MB"),
			MaxBackups: v.GetInt("LOG_MAX_BACKUPS"),
			MaxAgeDays: v.GetInt("LOG_MAX_AGE_DAYS"),
			Compress:   v.GetBool("LOG_COMPRESS"),
		},
		CORS: CorsConfig{
			AllowedOrigins:   splitTrim(originsStr),
			AllowedMethods:   splitTrim(methodsStr),
			AllowedHeaders:   splitTrim(headersStr),
			AllowCredentials: v.GetBool("CORS_ALLOW_CREDENTIALS"),
		},
		RateLimit: RateLimitConfig{
			Enabled:           v.GetBool("RATE_LIMIT_ENABLED"),
			RequestsPerSecond: v.GetFloat64("RATE_LIMIT_RPS"),
			Burst:             v.GetInt("RATE_LIMIT_BURST"),
		},
		Worker: WorkerConfig{
			Concurrency: v.GetInt("WORKER_CONCURRENCY"),
			QueueSize:   v.GetInt("WORKER_QUEUE_SIZE"),
		},
	}

	if cfg.App.Port == "" {
		return nil, fmt.Errorf("app port is required")
	}

	return cfg, nil
}

func splitTrim(s string) []string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return []string{}
	}

	// Support JSON array format e.g. ["http://localhost:3000", "http://localhost:5173"]
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		var arr []string
		if err := json.Unmarshal([]byte(trimmed), &arr); err == nil {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if clean := strings.TrimSpace(item); clean != "" {
					result = append(result, clean)
				}
			}
			return result
		}
		// Fallback for unquoted bracketed lists e.g. [http://localhost:3000, http://localhost:5173]
		trimmed = strings.TrimPrefix(trimmed, "[")
		trimmed = strings.TrimSuffix(trimmed, "]")
	}

	// Comma-separated list parsing
	parts := strings.Split(trimmed, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		clean := strings.Trim(strings.TrimSpace(p), `"'`)
		if clean != "" {
			res = append(res, clean)
		}
	}
	return res
}
