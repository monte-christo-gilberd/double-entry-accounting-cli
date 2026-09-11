package cli

import (
	"context"
	"fmt"

	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/account"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/book"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/journal"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/prompt"
)

func createBookFlow(ctx context.Context, bookService *book.Service) {

	name, err := prompt.ReadRequiredLine("Book Name: ")
	if err != nil {
		fmt.Println("failed to create book:", err)
		return
	}

	description, err := prompt.ReadLine("Description (optional): ")
	if err != nil {
		fmt.Println("failed to create book:", err)
		return
	}

	book := &book.Book{Name: name}
	if description != "" {
		book.Description = &description
	}

	if err := bookService.Create(ctx, book); err != nil {
		fmt.Println("failed to create book:", err)
		return
	}

	fmt.Printf("Book created: #%d %q\n", book.ID, book.Name)
}

func listBooksNumbered(ctx context.Context, bookService *book.Service) []book.Book {
	books, err := bookService.List(ctx)
	if err != nil {
		fmt.Println("failed to get book lists:", err)
		return nil
	}
	if len(books) == 0 {
		fmt.Println("No available book. Create book first from menu 1.")
		return nil
	}
	for i, book := range books {
		fmt.Printf("%d. %s (id=%d)\n", i+1, book.Name, book.ID)
	}
	return books
}

func selectBookFlow(
	ctx context.Context,
	bookService *book.Service,
	accountService *account.Service,
	journalService *journal.Service,
) {
	books := listBooksNumbered(ctx, bookService)
	if books == nil {
		return
	}

	idx, ok, err := prompt.ReadInt("Select book (number): ")
	if err != nil {
		fmt.Println("failed to get book:", err)
		return
	}

	if !ok || idx < 1 || idx > len(books) {
		fmt.Println("Invalid choice.")
		return
	}

	selected := books[idx-1]
	bookMenu(ctx, &selected, bookService, accountService, journalService)
}

func deleteBookFlow(ctx context.Context, bookService *book.Service) {
	books := listBooksNumbered(ctx, bookService)
	if books == nil {
		return
	}

	idx, ok, err := prompt.ReadInt("Select the book you want to delete (number): ")
	if err != nil {
		fmt.Println("Failed to delete book:", err)
		return
	}

	if !ok || idx < 1 || idx > len(books) {
		fmt.Println("invalid choice.")
		return
	}
	selected := books[idx-1]

	fmt.Printf("Deleting book %q will also delete ALL accounts and transactions within it.\n", selected.Name)

	input, err := prompt.ReadYesNo("Are you sure? (y/n): ")
	if err != nil {
		fmt.Println("Failed to delete book:", err)
		return
	}

	if !input {
		fmt.Println("Cancelled.")
		return
	}

	if err := bookService.Delete(ctx, selected.ID); err != nil {
		fmt.Println("Failed to delete book:", err)
		return
	}

	fmt.Println("Book deleted.")
}

func bookMenu(
	ctx context.Context,
	b *book.Book,
	bookService *book.Service,
	accountService *account.Service,
	journalService *journal.Service,
) {
	for {
		fmt.Printf("\n=== Book: %s ===\n", b.Name)
		fmt.Println("1. View Transaction Log")
		fmt.Println("2. View the total value of all accounts")
		fmt.Println("3. Perform Transaction")
		fmt.Println("4. Cancel Transaction")
		fmt.Println("5. Add Account")
		fmt.Println("6. Edit Account")
		fmt.Println("7. Delete Account")
		fmt.Println("0. Back to Main Menu")

		choice, ok, err := prompt.ReadInt("Select menu: ")
		if err != nil {
			fmt.Println("failed to read input: ", err)
		}

		if !ok {
			continue
		}

		switch choice {
		case 1:
			transactionLogMenu(ctx, b.ID, journalService)
		case 2:
			viewAccountBalances(ctx, b.ID, accountService, journalService)
		case 3:
			doTransaction(ctx, b.ID, accountService, journalService)
		case 4:
			cancelTransaction(ctx, b.ID, journalService)
		case 5:
			addAccount(ctx, b.ID, accountService)
		case 6:
			editAccount(ctx, b.ID, accountService)
		case 7:
			deleteAccount(ctx, b.ID, accountService)
		case 0:
			return
		default:
			fmt.Println("Invalid Menu.")
		}
	}
}
