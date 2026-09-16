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
// Each file runs in one transaction with an advisory lock so concurrent
// starts serialize, and each statement inside the file is applied
// individually: a duplicate_object on one statement skips just that
// statement instead of abandoning the rest of the file.
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
		// Serialize concurrent migrators (e.g. two `accounting` starts).
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('double-entry-accounting-migrate'))`); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("lock %s: %w", path, err)
		}
		for i, stmt := range splitStatements(string(sqlBytes)) {
			// Savepoint per statement: a failed statement poisons the
			// transaction in Postgres (25P02 on all following commands),
			// so a skippable duplicate_object must roll back to just
			// before that statement instead of `continue`-ing in a
			// poisoned tx.
			sp := fmt.Sprintf("mig_sp_%d", i)
			if _, err := tx.ExecContext(ctx, "SAVEPOINT "+sp); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("migrate %s: %w", path, err)
			}
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				if isDuplicateObjectError(err) {
					if _, rbErr := tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT "+sp); rbErr != nil {
						_ = tx.Rollback()
						return fmt.Errorf("migrate %s: %w", path, rbErr)
					}
					if _, relErr := tx.ExecContext(ctx, "RELEASE SAVEPOINT "+sp); relErr != nil {
						_ = tx.Rollback()
						return fmt.Errorf("migrate %s: %w", path, relErr)
					}
					continue
				}
				_ = tx.Rollback()
				return fmt.Errorf("migrate %s: %w", path, err)
			}
			if _, err := tx.ExecContext(ctx, "RELEASE SAVEPOINT "+sp); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("migrate %s: %w", path, err)
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit %s: %w", path, err)
		}
	}
	return nil
}

// splitStatements splits a migration file on semicolons and drops empty
// statements and pure comments, so one multi-statement file degrades
// gracefully when re-run after a partial apply. Semicolons inside
// single-quoted strings, double-quoted identifiers, dollar-quoted bodies
// ($$ ... $$, $tag$ ... $tag$), line comments (--) and block comments
// (/* ... */) do not split.
func splitStatements(src string) []string {
	var out []string
	var cur strings.Builder
	i, n := 0, len(src)
	var dollarTag string
	var inSingle, inDouble, inLineComment, inBlockComment bool

	flush := func() {
		stmt := strings.TrimSpace(cur.String())
		cur.Reset()
		if stmt == "" {
			return
		}
		lines := strings.Split(stmt, "\n")
		hasCode := false
		for _, l := range lines {
			t := strings.TrimSpace(l)
			if t == "" || strings.HasPrefix(t, "--") {
				continue
			}
			hasCode = true
			break
		}
		if !hasCode {
			return
		}
		out = append(out, stmt)
	}

	for i < n {
		c := src[i]
		// Inside dollar-quoted body: only the matching closing tag ends it.
		if dollarTag != "" {
			if c == '$' {
				j := i + 1
				for j < n && (src[j] == '_' || src[j] >= 'a' && src[j] <= 'z' || src[j] >= 'A' && src[j] <= 'Z' || src[j] >= '0' && src[j] <= '9') {
					j++
				}
				if j < n && src[j] == '$' {
					tag := src[i : j+1]
					cur.WriteString(tag)
					i = j + 1
					if tag == dollarTag {
						dollarTag = ""
					}
					continue
				}
			}
			cur.WriteByte(c)
			i++
			continue
		}
		if inLineComment {
			cur.WriteByte(c)
			if c == '\n' {
				inLineComment = false
			}
			i++
			continue
		}
		if inBlockComment {
			if c == '*' && i+1 < n && src[i+1] == '/' {
				cur.WriteString("*/")
				i += 2
				inBlockComment = false
				continue
			}
			cur.WriteByte(c)
			i++
			continue
		}
		if inSingle {
			cur.WriteByte(c)
			if c == '\'' {
				if i+1 < n && src[i+1] == '\'' {
					cur.WriteByte('\'')
					i += 2
					continue
				}
				inSingle = false
			}
			i++
			continue
		}
		if inDouble {
			cur.WriteByte(c)
			if c == '"' {
				if i+1 < n && src[i+1] == '"' {
					cur.WriteByte('"')
					i += 2
					continue
				}
				inDouble = false
			}
			i++
			continue
		}
		// Not inside anything: look for comment/string/dollar-tag opens.
		if c == '-' && i+1 < n && src[i+1] == '-' {
			inLineComment = true
			cur.WriteString("--")
			i += 2
			continue
		}
		if c == '/' && i+1 < n && src[i+1] == '*' {
			inBlockComment = true
			cur.WriteString("/*")
			i += 2
			continue
		}
		if c == '\'' {
			inSingle = true
			cur.WriteByte(c)
			i++
			continue
		}
		if c == '"' {
			inDouble = true
			cur.WriteByte(c)
			i++
			continue
		}
		if c == '$' {
			j := i + 1
			for j < n && (src[j] == '_' || src[j] >= 'a' && src[j] <= 'z' || src[j] >= 'A' && src[j] <= 'Z' || src[j] >= '0' && src[j] <= '9') {
				j++
			}
			if j < n && src[j] == '$' {
				dollarTag = src[i : j+1]
				cur.WriteString(dollarTag)
				i = j + 1
				continue
			}
			cur.WriteByte(c)
			i++
			continue
		}
		if c == ';' {
			flush()
			i++
			continue
		}
		cur.WriteByte(c)
		i++
	}
	flush()
	return out
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
	return false
}
