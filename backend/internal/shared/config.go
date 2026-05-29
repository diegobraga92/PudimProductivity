package shared

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	LogLevel string
	Version  string
}

type ServerConfig struct {
	Port            int
	ShutdownTimeout time.Duration
	ReadTimeout     int
	WriteTimeout    int
	IdleTimeout     int
}

type DatabaseConfig struct {
	URL             string
	ConnectTimeout  time.Duration
	MaxConns        int
	MinConns        int
	MaxConnLifetime int
	MaxConnIdleTime int
}

func LoadConfig() Config {
	return Config{
		Server: ServerConfig{
			Port:            getEnvInt("PORT", 8080),
			ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
			ReadTimeout:     getEnvInt("READ_TIMEOUT", 10),
			WriteTimeout:    getEnvInt("WRITE_TIMEOUT", 30),
			IdleTimeout:     getEnvInt("IDLE_TIMEOUT", 60),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://pudim:change_me_in_production@localhost:5433/pudimproductivity?sslmode=disable"),
			ConnectTimeout:  getEnvDuration("DATABASE_CONNECT_TIMEOUT", 30*time.Second),
			MaxConns:        getEnvInt("DATABASE_MAX_CONNS", 20),
			MinConns:        getEnvInt("DATABASE_MIN_CONNS", 2),
			MaxConnLifetime: getEnvInt("DATABASE_MAX_CONN_LIFETIME", 30),
			MaxConnIdleTime: getEnvInt("DATABASE_MAX_CONN_IDLETIME", 5),
		},
		LogLevel: getEnv("LOG_LEVEL", "debug"),
		Version:  getEnv("VERSION", "0.0.1"),
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}
