package database

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"gin-starter-pack/config"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DialectorBuilder function to build a gorm.Dialector from DBConfig
type DialectorBuilder func(cfg *config.DBConfig) (gorm.Dialector, error)

var (
	driverMu sync.RWMutex
	drivers  = make(map[string]DialectorBuilder)
)

func init() {
	// Register default built-in drivers
	RegisterDriver("postgres", func(cfg *config.DBConfig) (gorm.Dialector, error) {
		dsn := cfg.DSN
		if dsn == "" {
			dsn = fmt.Sprintf(
				"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
				cfg.Host, cfg.User, cfg.Password, cfg.Name, cfg.Port, cfg.SSLMode,
			)
		}
		return postgres.Open(dsn), nil
	})

	RegisterDriver("mysql", func(cfg *config.DBConfig) (gorm.Dialector, error) {
		dsn := cfg.DSN
		if dsn == "" {
			dsn = fmt.Sprintf(
				"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
			)
		}
		return mysql.Open(dsn), nil
	})

	RegisterDriver("sqlite", func(cfg *config.DBConfig) (gorm.Dialector, error) {
		dbPath := cfg.DSN
		if dbPath == "" {
			dbPath = cfg.Name
		}
		if dbPath == "" {
			dbPath = "app.db"
		}
		return sqlite.Open(dbPath), nil
	})
}

// RegisterDriver allows users/plugins to register custom or third-party GORM drivers
// (e.g. sqlserver, clickhouse, oracle) without modifying core database logic.
func RegisterDriver(name string, builder DialectorBuilder) {
	driverMu.Lock()
	defer driverMu.Unlock()
	drivers[strings.ToLower(name)] = builder
}

// GetRegisteredDrivers returns a list of all currently registered database driver names
func GetRegisteredDrivers() []string {
	driverMu.RLock()
	defer driverMu.RUnlock()
	list := make([]string, 0, len(drivers))
	for name := range drivers {
		list = append(list, name)
	}
	return list
}

// InitDB initializes database connection using registered drivers and connection pooling
func InitDB(cfg *config.DBConfig, appEnv string) (*gorm.DB, error) {
	driverName := strings.ToLower(cfg.Driver)

	driverMu.RLock()
	builder, exists := drivers[driverName]
	driverMu.RUnlock()

	if !exists {
		available := strings.Join(GetRegisteredDrivers(), ", ")
		return nil, fmt.Errorf(
			"unsupported or unregistered database driver: '%s'. Registered drivers: [%s]. To add a driver, call database.RegisterDriver()",
			cfg.Driver, available,
		)
	}

	dialector, err := builder(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to configure driver [%s]: %w", cfg.Driver, err)
	}

	gormLogLevel := logger.Warn
	if appEnv == "development" {
		gormLogLevel = logger.Info
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database [%s]: %w", cfg.Driver, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get generic database object: %w", err)
	}

	// Tune connection pool based on driver characteristics
	if driverName == "sqlite" {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	} else {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)
	}

	slog.Info("Database connected successfully", "driver", cfg.Driver)
	return db, nil
}
