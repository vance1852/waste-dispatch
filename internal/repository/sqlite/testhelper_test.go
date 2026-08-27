package sqlite_test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

// openTestDB opens a temp-file SQLite database and applies migrations.
// We use a file-based DB because golang-migrate opens a separate connection
// that cannot share an :memory: database.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "waste-dispatch-test-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	migrationsPath := migrationsDir(t)

	db, err := sql.Open("sqlite3", tmpFile.Name()+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		t.Fatalf("create migrate driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+migrationsPath, "sqlite3", driver)
	if err != nil {
		t.Fatalf("create migrate: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("apply migrations: %v", err)
	}

	return db
}

// migrationsDir finds the migrations directory by walking up from the current
// working directory until it locates a "migrations" folder.
func migrationsDir(t *testing.T) string {
	t.Helper()
	candidates := []string{
		"../../../migrations",
		"../../migrations",
		"../migrations",
		"migrations",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	t.Fatalf("could not find migrations directory")
	return ""
}
