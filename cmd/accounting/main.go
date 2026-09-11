package main

import (
	"context"
	"fmt"
	"os"

	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/account"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/book"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/cli"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/config"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/database"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/journal"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Failed to load config:", err)
		os.Exit(1)
	}

	db, err := database.NewPostgresDB(ctx, cfg)
	if err != nil {
		fmt.Println("Failed to connect database:", err)
		os.Exit(1)
	}
	defer db.Close()

	bookService := book.NewService(book.NewPostgresRepository(db))
	accountService := account.NewService(account.NewPostgresRepository(db))
	journalService := journal.NewService(journal.NewPostgresRepository(db), accountService)

	cli.Run(ctx, bookService, accountService, journalService)
}