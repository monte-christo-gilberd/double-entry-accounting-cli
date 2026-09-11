package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/account"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/journal"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/prompt"
)

func transactionLogMenu(ctx context.Context, bookID int64, journalService *journal.Service) {
	for {
		fmt.Println("\n-- Transaction Log --")
		fmt.Println("1. View All Logs")
		fmt.Println("2. View Last N Logs")
		fmt.Println("0. Back")

		choice, ok, err := prompt.ReadInt("Select Menu: ")
		if err != nil {
			fmt.Println("failed to get transaction log: ", err)
		}
		if !ok {
			continue
		}

		switch choice {
		case 1:
			entries, err := journalService.ListByBookID(ctx, bookID)
			if err != nil {
				fmt.Println("failed to get transaction log:", err)
				continue
			}
			printJournalEntries(entries)

		case 2:
			n, err := prompt.ReadIntDefault("Last Log Total (empty = 1): ", 1)
			if err != nil {
				fmt.Println("failed to get transaction log: ", err)
				continue
			}

			entries, err := journalService.ListRecentByBookID(ctx, bookID, n)
			if err != nil {
				fmt.Println("failed to get transaction log:", err)
				continue
			}
			printJournalEntries(entries)

		case 0:
			return
		default:
			fmt.Println("Invalid Menu.")
		}
	}
}

func printJournalEntries(entries []journal.JournalEntry) {
	if len(entries) == 0 {
		fmt.Println("No transaction yet.")
		return
	}
	for _, e := range entries {
		fmt.Printf(
			"#%d [%s] %s - %s\n",
			e.ID,
			e.EntryDate.Format("02-01-2006"),
			e.Description,
			e.Status,
		)
	}
}

func viewAccountBalances(
	ctx context.Context,
	bookID int64,
	accountService *account.Service,
	journalService *journal.Service,
) {
	accounts, err := accountService.ListByBookID(ctx, bookID)
	if err != nil {
		fmt.Println("failed to get account list:", err)
		return
	}
	if len(accounts) == 0 {
		fmt.Println("There are no registered account yet.")
		return
	}

	balances, err := journalService.GetAccountBalances(ctx, bookID)
	if err != nil {
		fmt.Println("failed to calculate balance:", err)
		return
	}
	balanceByAccount := make(map[int64]float64, len(balances))
	for _, b := range balances {
		balanceByAccount[b.AccountID] = b.Balance
	}

	fmt.Println("\n-- Account's Balance --")
	for _, a := range accounts {
		fmt.Printf("%-8s %-20s %-10s %.2f\n", a.Code, a.Name, a.AccountType, balanceByAccount[a.ID])
	}
}

func doTransaction(
	ctx context.Context,
	bookID int64,
	accountService *account.Service,
	journalService *journal.Service,
) {
	accounts, err := accountService.ListByBookID(ctx, bookID)
	if err != nil {
		fmt.Println("failed to do transaction:", err)
		return
	}
	if len(accounts) < 2 {
		fmt.Println("Need a minimum of 2 account to do a transaction. Add more account first from menu 5.")
		return
	}

	fmt.Println("\nAvailable Account:")
	for _, a := range accounts {
		fmt.Printf("  id=%d  %s - %s (%s)\n", a.ID, a.Code, a.Name, a.AccountType)
	}

	description, err := prompt.ReadRequiredLine("Transaction Description: ")
	if err != nil {
		fmt.Println("failed to to transaction:", err)
		return
	}

	var lines []journal.JournalLine
	for {

		fmt.Printf("\nLine-%d (enter 0 as Account ID to finish)\n", len(lines)+1)
		accID, ok, err := prompt.ReadInt("  Account ID: ")
		if err != nil {
			fmt.Println("failed to to transaction:", err)
			return
		}

		if !ok {
			continue
		}
		if accID == 0 {
			if len(lines) < 2 {
				fmt.Println("  Need a minimum of 2 lines before finishing.")
				continue
			}
			break
		}

		side, err := prompt.ReadRequiredLine("  Debit or Credit? (d/c): ")
		if err != nil {
			fmt.Println("failed to to transaction:", err)
			return
		}

		side = strings.ToLower(side)

		amount, ok, err := prompt.ReadFloat("  Total: ")
		if err != nil {
			fmt.Println("failed to to transaction:", err)
			return
		}

		if !ok || amount <= 0 {
			fmt.Println("  Total must be more than 0, line skipped.")
			continue
		}

		line := journal.JournalLine{AccountID: int64(accID)}
		switch side {
		case "d":
			line.Debit = amount
		case "c":
			line.Credit = amount
		default:
			fmt.Println("  Select 'd' or 'c'. Line skipped.")
			continue
		}
		lines = append(lines, line)
	}

	entry := &journal.JournalEntry{
		BookID:      bookID,
		EntryDate:   time.Now(),
		Description: description,
		Lines:       lines,
	}

	if err := journalService.Transact(ctx, entry); err != nil {
		fmt.Println("Transaction failed:", err)
		return
	}
	fmt.Printf("Transaction logged: #%d %q\n", entry.ID, entry.Description)
}

func cancelTransaction(ctx context.Context, bookID int64, journalService *journal.Service) {
	entries, err := journalService.ListByBookID(ctx, bookID)
	if err != nil {
		fmt.Println("Failed to get transaction logs:", err)
		return
	}

	var postedEntries []journal.JournalEntry
	for _, e := range entries {
		if e.Status == journal.StatusPosted {
			postedEntries = append(postedEntries, e)
		}
	}
	if len(postedEntries) == 0 {
		fmt.Println("No canceled transaction.")
		return
	}

	fmt.Println("\nCancelable transactions:")
	for _, e := range postedEntries {
		fmt.Printf("  #%d [%s] %s\n", e.ID, e.EntryDate.Format("02-01-2006"), e.Description)
	}

	id, ok, err := prompt.ReadInt("Enter transaction id that you want to cancel: ")
	if err != nil {
		fmt.Println("failed to cancel transaction:", err)
		return
	}

	if !ok {
		return
	}

	input, err := prompt.ReadYesNo(fmt.Sprintf("Are you sure about cancelling transaction #%d? (y/n): ", id))
	if err != nil {
		fmt.Println("failed to cancel transaction:", err)
		return
	}

	if !input {
		fmt.Println("Cancelled.")
		return
	}

	if err := journalService.Void(ctx, int64(id)); err != nil {
		fmt.Println("failed to cancel transaction:", err)
		return
	}
	fmt.Println("Transaction cancelled (The balance has been restored to its original state).")
}
