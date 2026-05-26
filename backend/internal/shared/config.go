package shared

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            int
	DatabaseURL     string
	ShutdownTimeout time.Duration
	LogLevel        string
	Version         string
	ReadTimeout		int
	WriteTimeout	int
	IdleTimeout		int
}

func LoadConfig() Config {
	return Config{
		Port:            getEnvInt("PORT", 8080),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://pudim:pudim_dev@localhost:5433/pudimproductivity?sslmode=disable"),
		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
		LogLevel:        getEnv("LOG_LEVEL", "debug"),
		Version:		 getEnv("VERSION", "0.0.1"),
		ReadTimeout:	 getEnvInt("READ_TIMEOUT", 10),
		WriteTimeout:	 getEnvInt("WRITE_TIMEOUT", 30),
		IdleTimeout:	 getEnvInt("IDLE_TIMEOUT", 60),
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
