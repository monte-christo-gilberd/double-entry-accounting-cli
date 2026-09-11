package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/config"
)

func NewPostgresDB(ctx context.Context, cfg *config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		url.QueryEscape(cfg.DatabaseUser),
		url.QueryEscape(cfg.DatabasePassword),
		cfg.DatabaseHost,
		cfg.DatabasePort,
		url.PathEscape(cfg.DatabaseName),
		url.QueryEscape(cfg.DatabaseSSLMode),
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}
