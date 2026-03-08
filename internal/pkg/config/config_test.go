package config_test

import (
	"os"
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
}

func TestLoad_FromEnv(t *testing.T) {
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("POSTGRES_HOST", "myhost")
	os.Setenv("AUTH_SESSION_TTL", "48h")
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("AUTH_SESSION_TTL")
	}()

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
