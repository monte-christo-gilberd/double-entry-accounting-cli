package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Migrate(ctx context.Context, db *sql.DB) error {

	entries, err := os.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		path := filepath.Join("migrations", entry.Name())
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		if _, err := db.ExecContext(ctx, string(sqlBytes)); err != nil {
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			return fmt.Errorf("migrate %s: %w", path, err)
		}
		fmt.Println("applied:", entry.Name())
	}
	return nil
}
