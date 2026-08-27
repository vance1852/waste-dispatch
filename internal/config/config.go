package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Log      LogConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host            string
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds SQLite database configuration.
type DatabaseConfig struct {
	Path            string
	MigrationsPath  string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	TokenSecret     string
	TokenExpiry     time.Duration
	SessionExpiry   time.Duration
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level  string
	Pretty bool
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnv("SERVER_PORT", "8080"),
			ReadTimeout:     getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			ShutdownTimeout: getDurationEnv("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Path:            getEnv("DB_PATH", "./data/waste_dispatch.db"),
			MigrationsPath:  getEnv("DB_MIGRATIONS_PATH", "file://migrations"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 10),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		},
		Auth: AuthConfig{
			TokenSecret:   getEnv("AUTH_TOKEN_SECRET", "change-me-in-production-secret-key"),
			TokenExpiry:   getDurationEnv("AUTH_TOKEN_EXPIRY", 24*time.Hour),
			SessionExpiry: getDurationEnv("AUTH_SESSION_EXPIRY", 7*24*time.Hour),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Pretty: getBoolEnv("LOG_PRETTY", false),
		},
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getIntEnv(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func getBoolEnv(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultVal
}

func getDurationEnv(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}
