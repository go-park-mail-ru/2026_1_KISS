package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/database"
)

func main() {
	cfg := config.Load()
	db, err := database.Connect(cfg.Database.DSN())
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer db.Close()

	if err := ensureMigrationsTable(db); err != nil {
		log.Fatalf("ensure migrations table: %v", err)
	}

	entries, err := os.ReadDir("migrations")
	if err != nil {
		log.Fatalf("read migrations dir: %v", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version, err := parseVersion(entry.Name())
		if err != nil {
			log.Fatalf("parse version from %s: %v", entry.Name(), err)
		}

		applied, err := isMigrationApplied(db, version)
		if err != nil {
			log.Fatalf("check migration %s: %v", entry.Name(), err)
		}
		if applied {
			continue
		}

		if err := applyMigration(db, version, entry.Name()); err != nil {
			log.Fatalf("apply migration %s: %v", entry.Name(), err)
		}
		log.Printf("applied migration %s", entry.Name())
	}
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	return err
}

func parseVersion(filename string) (int, error) {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid migration filename format: %s", filename)
	}
	return strconv.Atoi(parts[0])
}

func isMigrationApplied(db *sql.DB, version int) (bool, error) {
	var v int
	err := db.QueryRow(`SELECT version FROM schema_migrations WHERE version = $1`, version).Scan(&v)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func applyMigration(db *sql.DB, version int, filename string) error {
	content, err := os.ReadFile("migrations/" + filename)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if _, err := tx.Exec(string(content)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("exec sql: %w", err)
	}

	if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record migration: %w", err)
	}

	return tx.Commit()
}
