package sqlite

import (
	"strings"
)

// isUniqueConstraintError returns true when the error is a SQLite UNIQUE constraint violation.
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "UNIQUE constraint failed") || strings.Contains(s, "unique constraint")
}

// nullString converts an empty string to a SQL NULL.
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
