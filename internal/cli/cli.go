package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/account"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/book"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/journal"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/prompt"
)

func Run(
	ctx context.Context,
	bookService *book.Service,
	accountService *account.Service,
	journalService *journal.Service,
) {
	for {
		fmt.Println("\n=== Double-Entry Accounting CLI ===")
		fmt.Println("1. Create New Book")
		fmt.Println("2. Select Existing Book")
		fmt.Println("3. Delete Book")
		fmt.Println("0. Close")

		choice, ok, err := prompt.ReadInt("Select Menu: ")
		if err != nil {
			if errors.Is(err, prompt.ErrInputClosed) {
				fmt.Println("Good Bye!")
				return
			}
			fmt.Println("Failed to read input: ", err)
			continue
		}
		if !ok {
			continue
		}

		switch choice {
		case 1:
			createBookFlow(ctx, bookService)
		case 2:
			selectBookFlow(ctx, bookService, accountService, journalService)
		case 3:
			deleteBookFlow(ctx, bookService)
		case 0:
			fmt.Println("Good Bye!")
			return
		default:
			fmt.Println("Invalid Choice.")
		}
	}
}
