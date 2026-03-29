package config_test

import (
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := config.Load()
	if cfg.Server.Port != "8080" {
		t.Errorf("want port 8080, got %s", cfg.Server.Port)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("want localhost, got %s", cfg.Database.Host)
	}
	if cfg.Auth.SessionTTL != 24*time.Hour {
		t.Errorf("want 24h, got %v", cfg.Auth.SessionTTL)
	}
	if cfg.Runner.Image != "kiss-runner" {
		t.Errorf("want kiss-runner, got %s", cfg.Runner.Image)
	}
	if cfg.Runner.StartupTimeout != 20*time.Second {
		t.Errorf("want 20s startup timeout, got %v", cfg.Runner.StartupTimeout)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("POSTGRES_HOST", "myhost")
	t.Setenv("AUTH_SESSION_TTL", "48h")
	t.Setenv("RUNNER_IMAGE", "kiss-python-runner")
	t.Setenv("RUNNER_MEMORY_LIMIT_BYTES", "104857600")
	t.Setenv("RUNNER_STARTUP_TIMEOUT", "35s")

	cfg := config.Load()
	if cfg.Server.Port != "9090" {
		t.Errorf("want 9090, got %s", cfg.Server.Port)
	}
	if cfg.Database.Host != "myhost" {
		t.Errorf("want myhost, got %s", cfg.Database.Host)
	}
	if cfg.Auth.SessionTTL != 48*time.Hour {
		t.Errorf("want 48h, got %v", cfg.Auth.SessionTTL)
	}
	if cfg.Runner.Image != "kiss-python-runner" {
		t.Errorf("want kiss-python-runner, got %s", cfg.Runner.Image)
	}
	if cfg.Runner.MemoryLimitBytes != 104857600 {
		t.Errorf("want 104857600, got %d", cfg.Runner.MemoryLimitBytes)
	}
	if cfg.Runner.StartupTimeout != 35*time.Second {
		t.Errorf("want 35s, got %v", cfg.Runner.StartupTimeout)
	}
}

func TestDatabaseConfig_DSN_WithURL(t *testing.T) {
	cfg := &config.Config{}
	cfg.Database.URL = "postgres://user:pass@host/db"
	if got := cfg.Database.DSN(); got != "postgres://user:pass@host/db" {
		t.Errorf("want URL DSN, got %s", got)
	}
}

func TestDatabaseConfig_DSN_WithParts(t *testing.T) {
	cfg := &config.Config{}
	cfg.Database.User = "user"
	cfg.Database.Password = "pass"
	cfg.Database.Host = "localhost"
	cfg.Database.Port = "5432"
	cfg.Database.DBName = "colab"
	cfg.Database.SSLMode = "disable"
	dsn := cfg.Database.DSN()
	if dsn == "" {
		t.Error("DSN should not be empty")
	}
}
