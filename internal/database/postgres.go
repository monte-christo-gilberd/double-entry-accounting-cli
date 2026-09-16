package database

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/config"
)

func NewPostgresDB(ctx context.Context, cfg *config.Config) (*sql.DB, error) {
	userInfo := url.UserPassword(cfg.DatabaseUser, cfg.DatabasePassword)
	hostPort := net.JoinHostPort(cfg.DatabaseHost, cfg.DatabasePort)
	u := url.URL{
		Scheme:   "postgres",
		User:     userInfo,
		Host:     hostPort,
		Path:     "/" + cfg.DatabaseName,
		RawQuery: url.Values{"sslmode": {cfg.DatabaseSSLMode}}.Encode(),
	}
	dsn := u.String()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}
