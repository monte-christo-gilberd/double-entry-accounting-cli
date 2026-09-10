package main

import (
	"context"
	"fmt"
	"log"

	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/config"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/database"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.NewPostgresDB(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("Database connection successful!")
}
