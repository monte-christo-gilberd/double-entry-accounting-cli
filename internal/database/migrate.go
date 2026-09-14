package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/migrations"
)

// Migrate applies pending *.sql files embedded in the binary. It is silent
// on success (errors still propagate to the caller) so opening the app goes
// straight from the DB check to the program interface.
func Migrate(ctx context.Context, db *sql.DB) error {
	entries, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	sort.Strings(entries)

	for _, path := range entries {
		if !strings.HasSuffix(path, ".sql") {
			continue
		}

		sqlBytes, err := migrations.FS.ReadFile(path)
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
