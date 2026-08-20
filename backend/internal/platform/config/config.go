package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	RedisURL string
	LogLevel string
	Version  string
}

type ServerConfig struct {
	Port               int
	ShutdownTimeout    time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	RequestTimeout     time.Duration
	CORSAllowedOrigins map[string]bool
}

type DatabaseConfig struct {
	URL             string
	ConnectTimeout  time.Duration
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// TODO: Check if this makes sense after migrating Providers to DB
type ScoreProviderConfig struct {
	Movie    string
	Series   string
	Game     string
	Book     string
	Keys     map[string]string
	BaseURLs map[string]string
}

func LoadConfig() Config {
	return Config{
		Server: ServerConfig{
			Port:               getEnvInt("PORT", 8080),
			ShutdownTimeout:    getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
			ReadTimeout:        getEnvDuration("READ_TIMEOUT", 10*time.Second),
			WriteTimeout:       getEnvDuration("WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:        getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
			RequestTimeout:     getEnvDuration("REQUEST_TIMEOUT", 15*time.Second),
			CORSAllowedOrigins: ParseAllowedOrigins(getEnv("CORS_ALLOWED_ORIGINS", "")),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://pudim:change_me_in_production@localhost:5433/pudimproductivity?sslmode=disable"),
			ConnectTimeout:  getEnvDuration("DATABASE_CONNECT_TIMEOUT", 30*time.Second),
			MaxConns:        getEnvInt("DATABASE_MAX_CONNS", 20),
			MinConns:        getEnvInt("DATABASE_MIN_CONNS", 2),
			MaxConnLifetime: getEnvDuration("DATABASE_MAX_CONN_LIFETIME", 30*time.Minute),
			MaxConnIdleTime: getEnvDuration("DATABASE_MAX_CONN_IDLETIME", 5*time.Minute),
		},
		RedisURL: getEnv("REDIS_URL", ""),
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

// ParseAllowedOrigins splits a comma-separated origin list into a lookup set.
// Whitespace is trimmed and empty entries are ignored.
func ParseAllowedOrigins(raw string) map[string]bool {
	set := make(map[string]bool)
	for _, entry := range strings.Split(raw, ",") {
		if entry = strings.TrimSpace(entry); entry != "" {
			set[entry] = true
		}
	}
	return set
}

// LoadScoreProviderConfig reads the score-provider selection from the
// environment. Keys are optional: an empty key means the provider is configured
// but unusable, which the scoring registry surfaces as a startup warning.
func LoadScoreProviderConfig() ScoreProviderConfig {
	return ScoreProviderConfig{
		Movie:  getEnv("SCORE_PROVIDER_MOVIE", ""),
		Series: getEnv("SCORE_PROVIDER_SERIES", ""),
		Game:   getEnv("SCORE_PROVIDER_GAME", ""),
		Book:   getEnv("SCORE_PROVIDER_BOOK", ""),
		Keys: map[string]string{
			"omdb": getEnv("OMDB_API_KEY", ""),
			"rawg": getEnv("RAWG_API_KEY", ""),
		},
		BaseURLs: map[string]string{
			"omdb": getEnv("OMDB_BASE_URL", ""),
			"rawg": getEnv("RAWG_BASE_URL", ""),
		},
	}
}
