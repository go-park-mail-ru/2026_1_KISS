package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
	CORS     CORSConfig
	Upload   UploadConfig
}

// UploadConfig holds file upload settings.
type UploadConfig struct {
	Dir     string
	MaxSize int64
}

type ServerConfig struct {
	Host string
	Port string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	URL      string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
}

func (d DatabaseConfig) DSN() string {
	if d.URL != "" {
		return d.URL
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode)
}

type AuthConfig struct {
	SessionTTL time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "", parseString),
			Port: getEnv("SERVER_PORT", "8080", parseString),
		},
		Database: DatabaseConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost", parseString),
			Port:     getEnv("POSTGRES_PORT", "5432", parseString),
			User:     getEnv("POSTGRES_USER", "postgres", parseString),
			Password: getEnv("POSTGRES_PASSWORD", "postgres", parseString),
			DBName:   getEnv("POSTGRES_DB", "colab", parseString),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable", parseString),
			URL:      getEnv("DATABASE_URL", "", parseString),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost", parseString),
			Port:     getEnv("REDIS_PORT", "6379", parseString),
			Password: getEnv("REDIS_PASSWORD", "", parseString),
		},
		Auth: AuthConfig{
			SessionTTL: getEnv("AUTH_SESSION_TTL", 24*time.Hour, time.ParseDuration),
		},
		CORS: CORSConfig{
			AllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000", parseString), ","),
		},
		Upload: UploadConfig{
			Dir:     getEnv("UPLOAD_DIR", "./uploads", parseString),
			MaxSize: getEnv("MAX_UPLOAD_SIZE", int64(2<<20), parseInt64),
		},
	}
}

func getEnv[T any](key string, defaultVal T, parse func(string) (T, error)) T {
	if val := os.Getenv(key); val != "" {
		if parsed, err := parse(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func parseString(s string) (string, error) { return s, nil }
func parseInt64(s string) (int64, error)   { return strconv.ParseInt(s, 10, 64) }
