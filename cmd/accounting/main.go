package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/account"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/book"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/cli"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/config"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/database"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/journal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

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

	for _, arg := range os.Args[1:] {
		if arg != "migrate" {
			fmt.Printf("Unknown argument %q (usage: %s [migrate])\n", arg, os.Args[0])
			os.Exit(2)
		}
	}
	if len(os.Args) > 2 {
		fmt.Printf("Too many arguments (usage: %s [migrate])\n", os.Args[0])
		os.Exit(2)
	}
	// migrate subcommand: go run ./cmd/accounting migrate (silent on
	// success; failures print and exit non-zero)
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := database.Migrate(ctx, db); err != nil {
			fmt.Println("Migration failed:", err)
			os.Exit(1)
		}
		return
	}
	// auto-migrate on normal run (quiet: straight to the interface)
	if err := database.Migrate(ctx, db); err != nil {
		fmt.Println("Auto-migrate failed:", err)
		os.Exit(1)
	}

	bookService := book.NewService(book.NewPostgresRepository(db))
	accountService := account.NewService(account.NewPostgresRepository(db))
	journalService := journal.NewService(journal.NewPostgresRepository(db), accountService)

	cli.Run(ctx, bookService, accountService, journalService)
}
