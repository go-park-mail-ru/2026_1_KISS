package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Auth      AuthConfig
	CORS      CORSConfig
	Runner    RunnerConfig
	Upload    UploadConfig
	RateLimit RateLimitConfig
}

type RateLimitConfig struct {
	MaxRequests int
	Window      time.Duration
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
	SessionTTL   time.Duration
	CookieSecure bool
}

type CORSConfig struct {
	AllowedOrigins []string
}

type RunnerConfig struct {
	Images              map[string]string // language -> image name, e.g. "python" -> "kiss-python-runner"
	NamePrefix          string
	AgentPort           string
	MemoryLimitBytes    int64
	NanoCPUs            int64
	StartupTimeout      time.Duration
	HealthCheckInterval time.Duration
	NetworkName         string
	TmpfsSize           string
	PidsLimit           int64
	IdleTimeout         time.Duration
	ExecutionTimeout    time.Duration
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", ""),
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "postgres"),
			Password: getEnv("POSTGRES_PASSWORD", "postgres"),
			DBName:   getEnv("POSTGRES_DB", "colab"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
			URL:      getEnv("DATABASE_URL", ""),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
		},
		Auth: AuthConfig{
			SessionTTL:   getEnvDuration("AUTH_SESSION_TTL", 24*time.Hour),
			CookieSecure: getEnvBool("COOKIE_SECURE", true),
		},
		CORS: CORSConfig{
			AllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"), ","),
		},
		Runner: RunnerConfig{
			Images: map[string]string{
				"python": getEnv("RUNNER_IMAGE_PYTHON", "kiss-python-runner"),
			},
			NamePrefix:          getEnv("RUNNER_NAME_PREFIX", "runner-"),
			AgentPort:           getEnv("RUNNER_AGENT_PORT", "8080"),
			MemoryLimitBytes:    getEnvInt64("RUNNER_MEMORY_LIMIT_BYTES", 512*1024*1024),
			NanoCPUs:            getEnvInt64("RUNNER_NANO_CPUS", 1_000_000_000),
			StartupTimeout:      getEnvDuration("RUNNER_STARTUP_TIMEOUT", 20*time.Second),
			HealthCheckInterval: getEnvDuration("RUNNER_HEALTHCHECK_INTERVAL", 300*time.Millisecond),
			NetworkName:         getEnv("NETWORK_NAME", "bridge"), // 2026_1_kiss_app-network
			TmpfsSize:           getEnv("RUNNER_TMPFS_SIZE", "100m"),
			PidsLimit:           getEnvInt64("RUNNER_PIDS_LIMIT", 64),
			IdleTimeout:         getEnvDuration("RUNNER_IDLE_TIMEOUT", 15*time.Minute),
			ExecutionTimeout:    getEnvDuration("RUNNER_EXECUTION_TIMEOUT", 120*time.Second),
		},
		Upload: UploadConfig{
			Dir:     getEnv("UPLOAD_DIR", "/app/uploads"),
			MaxSize: getEnvInt64("UPLOAD_MAX_SIZE", 2*1024*1024),
		},
		RateLimit: RateLimitConfig{
			MaxRequests: int(getEnvInt64("RATE_LIMIT_MAX_REQUESTS", 300)),
			Window:      getEnvDuration("RATE_LIMIT_WINDOW", time.Minute),
		},
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.ParseBool(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}
