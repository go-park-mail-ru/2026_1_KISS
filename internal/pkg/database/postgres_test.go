package database

import (
	"testing"
)

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
