package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

func findMigrationsDir() string {
	candidates := []string{
		"migrations",
		filepath.Join("..", "migrations"),
		filepath.Join("..", "..", "migrations"),
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "migrations"))
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "..", "migrations"))
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "..", "..", "migrations"))
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	return "migrations"
}

func Migrate(ctx context.Context, db *sql.DB) error {
	migrationsDir := findMigrationsDir()
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		path := filepath.Join(migrationsDir, entry.Name())
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		// atomic per file
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx %s: %w", path, err)
		}
		if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback()
			if isDuplicateObjectError(err) {
				continue
			}
			return fmt.Errorf("migrate %s: %w", path, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit %s: %w", path, err)
		}
		fmt.Println("applied:", entry.Name())
	}
	return nil
}

// isDuplicateObjectError reports whether err only means "object already
// exists" so re-running migrations stays idempotent. It matches Postgres
// SQLSTATE codes instead of a raw substring, so genuine failures (bad
// syntax, FK violations, …) still abort the migration.
func isDuplicateObjectError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "42P07", // duplicate_table
			"42710", // duplicate_object (constraint, index, …)
			"42701", // duplicate_column
			"42723": // duplicate_function
			return true
		}
	}
	// Fallback for drivers that do not surface PgError (or messages like
	// `relation "books" already exists` without a code).
	return strings.Contains(err.Error(), "already exists")
}
