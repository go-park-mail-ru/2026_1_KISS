package database

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestConnect_Success(t *testing.T) {
	_, _, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
}

func TestConnect_InvalidDSN(t *testing.T) {
	_, err := Connect("")
	if err == nil {
		t.Fatal("expected error for empty DSN")
	}
}

func TestConnect_InvalidDriver(t *testing.T) {
	_, err := Connect("invalid://invalid")
	if err == nil {
		t.Fatal("expected error for invalid driver")
	}
}
