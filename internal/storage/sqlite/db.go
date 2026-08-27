package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/vance1852/waste-dispatch/internal/config"
	_ "github.com/mattn/go-sqlite3"
)

// Open opens a SQLite database with production-ready settings.
func Open(cfg config.DatabaseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"%s?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000&_synchronous=NORMAL",
		cfg.Path,
	)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite3: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite3: %w", err)
	}

	return db, nil
}

// HealthCheck verifies the database is reachable.
func HealthCheck(db *sql.DB) error {
	if err := db.Ping(); err != nil {
		return fmt.Errorf("db ping: %w", err)
	}
	return nil
}
