package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/account"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/journal"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/prompt"
)

func transactionLogMenu(ctx context.Context, bookID int64, accountService *account.Service, journalService *journal.Service) {
	for {
		fmt.Println("\n-- Transaction Log --")
		fmt.Println("1. View All Logs")
		fmt.Println("2. View Last N Logs")
		fmt.Println("0. Back")

		choice, ok, err := prompt.ReadInt("Select Menu: ")
		if err != nil {
			if errors.Is(err, prompt.ErrInputClosed) {
				return
			}
			fmt.Println("failed to get transaction log: ", err)
			continue
		}
		if !ok {
			continue
		}

		switch choice {
		case 1:
			entries, names, err := loadDetailedEntries(ctx, bookID, nil, accountService, journalService)
			if err != nil {
				fmt.Println("failed to get transaction log:", err)
				continue
			}
			printJournalEntries(entries, names)

		case 2:
			n, err := prompt.ReadIntDefault("Last Log Total (empty = 1): ", 1)
			if err != nil {
				fmt.Println("failed to get transaction log: ", err)
				continue
			}
			if n <= 0 {
				fmt.Println("Last Log Total must be more than 0.")
				continue
			}

			recent, err := journalService.ListRecentByBookID(ctx, bookID, n)
			if err != nil {
				fmt.Println("failed to get transaction log:", err)
				continue
			}
			entries, names, err := loadDetailedEntries(ctx, bookID, recent, accountService, journalService)
			if err != nil {
				fmt.Println("failed to get transaction log:", err)
				continue
			}
			printJournalEntries(entries, names)

		case 0:
			return
		default:
			fmt.Println("Invalid Menu.")
		}
	}
}

// loadDetailedEntries returns entries with their lines loaded plus an
// accountID -> "CODE - Name" map for rendering. When headers is nil the
// full detailed listing is used; otherwise each header is hydrated via
// GetByID (the recent-N path, where N is small).
func loadDetailedEntries(
	ctx context.Context,
	bookID int64,
	headers []journal.JournalEntry,
	accountService *account.Service,
	journalService *journal.Service,
) ([]journal.JournalEntry, map[int64]string, error) {
	accounts, err := accountService.ListByBookID(ctx, bookID)
	if err != nil {
		return nil, nil, err
	}
	names := make(map[int64]string, len(accounts))
	for _, a := range accounts {
		names[a.ID] = a.Code + " - " + a.Name
	}

	if headers == nil {
		entries, err := journalService.ListDetailedByBookID(ctx, bookID)
		if err != nil {
			return nil, nil, err
		}
		return entries, names, nil
	}

	entries := make([]journal.JournalEntry, 0, len(headers))
	for _, h := range headers {
		full, err := journalService.GetByID(ctx, h.ID, bookID)
		if err != nil {
			return nil, nil, err
		}
		entries = append(entries, *full)
	}
	return entries, names, nil
}

func printJournalEntries(entries []journal.JournalEntry, names map[int64]string) {
	if len(entries) == 0 {
		fmt.Println("No transaction yet.")
		return
	}
	for _, e := range entries {
		fmt.Print(formatJournalEntry(e, names))
	}
}

// formatJournalEntry renders one entry with its lines; pure for testability.
func formatJournalEntry(e journal.JournalEntry, names map[int64]string) string {
	var sb strings.Builder
	fmt.Fprintf(
		&sb,
		"#%d [%s] %s - %s",
		e.ID,
		e.EntryDate.Format("02-01-2006"),
		e.Description,
		e.Status,
	)
	if e.ReversalOf != nil {
		fmt.Fprintf(&sb, " (reversal of #%d)", *e.ReversalOf)
	}
	sb.WriteString("\n")
	for _, l := range e.Lines {
		name := names[l.AccountID]
		if name == "" {
			name = fmt.Sprintf("account #%d", l.AccountID)
		}
		if l.Debit > 0 {
			fmt.Fprintf(&sb, "    %-24s Dr %12.2f\n", name, l.Debit)
		} else {
			fmt.Fprintf(&sb, "    %-24s Cr %12.2f\n", name, l.Credit)
		}
	}
	return sb.String()
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

	fmt.Println("\n-- Account's Balance (Dr = debit-side, Cr = credit-side) --")
	for _, a := range accounts {
		fmt.Printf("%-8s %-20s %-10s %s\n", a.Code, a.Name, a.AccountType, formatBalance(a.AccountType, balanceByAccount[a.ID]))
	}
}

// formatBalance presents a raw debit-minus-credit balance in normal-balance
// form: absolute value with a Dr/Cr suffix based on the account type.
// ASSET and EXPENSE are normal-debit; LIABILITY, EQUITY and REVENUE are
// normal-credit. A contra-side balance flips the suffix. Zero prints plain.
func formatBalance(accountType string, balance float64) string {
	amount, side := balance, "Dr"
	if accountType != "ASSET" && accountType != "EXPENSE" {
		amount, side = -balance, "Cr"
	}
	if amount < 0 {
		amount = -amount
		if side == "Dr" {
			side = "Cr"
		} else {
			side = "Dr"
		}
	}
	if amount == 0 {
		return "0.00"
	}
	return fmt.Sprintf("%.2f %s", amount, side)
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
		fmt.Printf("  %s - %s (%s)\n", a.Code, a.Name, a.AccountType)
	}

	description, err := prompt.ReadLine("Transaction Description (empty to cancel): ")
	if err != nil {
		fmt.Println("failed to to transaction:", err)
		return
	}
	if description == "" {
		fmt.Println("Transaction cancelled.")
		return
	}

	lines, cancelled, err := collectJournalLines(ctx, bookID, accountService)
	if err != nil {
		fmt.Println("failed to to transaction:", err)
		return
	}
	if cancelled {
		fmt.Println("Transaction cancelled.")
		return
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

// collectJournalLines prompts for debit/credit lines identified by account
// code until the user finishes with 0 or cancels with q. It reports
// cancelled=true on user abort and err only on a hard input read error;
// anything else skips just that line.
func collectJournalLines(
	ctx context.Context,
	bookID int64,
	accountService *account.Service,
) (lines []journal.JournalLine, cancelled bool, err error) {
	for {

		fmt.Printf("\nLine-%d (0 to finish, q to cancel)\n", len(lines)+1)
		raw, err := prompt.ReadLine("  Account code: ")
		if err != nil {
			return nil, false, err
		}

		code, finish, cancel, valid := parseAccountCodeInput(raw)
		if cancel {
			return nil, true, nil
		}
		if !valid {
			continue
		}
		if finish {
			if len(lines) < 2 {
				fmt.Println("  Need a minimum of 2 lines before finishing.")
				continue
			}
			break
		}

		acc, err := accountService.GetByCodeAndBookID(ctx, code, bookID)
		if err != nil {
			fmt.Printf("  Unknown account code %q in this book. Line skipped.\n", code)
			continue
		}

		side, err := prompt.ReadRequiredLine("  Debit or Credit? (d/c): ")
		if err != nil {
			return nil, false, err
		}

		side = strings.ToLower(side)
		if side != "d" && side != "c" {
			fmt.Println("  Select 'd' or 'c'. Line skipped.")
			continue
		}

		amount, ok, err := prompt.ReadFloat("  Total: ")
		if err != nil {
			return nil, false, err
		}

		if !ok || amount <= 0 {
			fmt.Println("  Total must be more than 0, line skipped.")
			continue
		}

		line := journal.JournalLine{AccountID: acc.ID}
		if side == "d" {
			line.Debit = amount
		} else {
			line.Credit = amount
		}
		lines = append(lines, line)
	}
	return lines, false, nil
}

// parseAccountCodeInput interprets one raw account-code line: q cancels, 0
// finishes, empty input is invalid (caller reprompts), anything else is an
// account code to resolve.
func parseAccountCodeInput(input string) (code string, finish bool, cancelled bool, valid bool) {
	switch s := strings.TrimSpace(input); strings.ToLower(s) {
	case "q", "cancel":
		return "", false, true, false
	case "0":
		return "", true, false, true
	case "":
		return "", false, false, false
	default:
		return s, false, false, true
	}
}

func createDraftTransaction(
	ctx context.Context,
	bookID int64,
	accountService *account.Service,
	journalService *journal.Service,
) {
	accounts, err := accountService.ListByBookID(ctx, bookID)
	if err != nil {
		fmt.Println("failed to create draft:", err)
		return
	}
	if len(accounts) < 2 {
		fmt.Println("Need a minimum of 2 account to do a transaction. Add more account first from menu 5.")
		return
	}

	fmt.Println("\nAvailable Account:")
	for _, a := range accounts {
		fmt.Printf("  %s - %s (%s)\n", a.Code, a.Name, a.AccountType)
	}

	description, err := prompt.ReadLine("Draft Description (empty to cancel): ")
	if err != nil {
		fmt.Println("failed to create draft:", err)
		return
	}
	if description == "" {
		fmt.Println("Draft cancelled.")
		return
	}

	lines, cancelled, err := collectJournalLines(ctx, bookID, accountService)
	if err != nil {
		fmt.Println("failed to create draft:", err)
		return
	}
	if cancelled {
		fmt.Println("Draft cancelled.")
		return
	}

	entry := &journal.JournalEntry{
		BookID:      bookID,
		EntryDate:   time.Now(),
		Description: description,
		Lines:       lines,
	}

	if err := journalService.CreateDraft(ctx, entry); err != nil {
		fmt.Println("Failed to create draft:", err)
		return
	}
	fmt.Printf("Draft saved: #%d %q (post it from menu 9 to affect balances)\n", entry.ID, entry.Description)
}

// draftEntries returns DRAFT entries that may be posted.
func draftEntries(entries []journal.JournalEntry) []journal.JournalEntry {
	var drafts []journal.JournalEntry
	for _, e := range entries {
		if e.Status == journal.StatusDraft {
			drafts = append(drafts, e)
		}
	}
	return drafts
}

func postDraftTransaction(ctx context.Context, bookID int64, journalService *journal.Service) {
	entries, err := journalService.ListByBookID(ctx, bookID)
	if err != nil {
		fmt.Println("Failed to get transaction logs:", err)
		return
	}

	drafts := draftEntries(entries)
	if len(drafts) == 0 {
		fmt.Println("No draft transactions to post.")
		return
	}

	fmt.Println("\nDraft transactions:")
	for _, e := range drafts {
		fmt.Printf("  #%d [%s] %s\n", e.ID, e.EntryDate.Format("02-01-2006"), e.Description)
	}

	id, ok, err := prompt.ReadInt("Enter draft id to post: ")
	if err != nil {
		fmt.Println("failed to post draft:", err)
		return
	}
	if !ok {
		return
	}

	input, err := prompt.ReadYesNo(fmt.Sprintf("Post draft #%d? Balances will be affected. (y/n): ", id))
	if err != nil {
		fmt.Println("failed to post draft:", err)
		return
	}
	if !input {
		fmt.Println("Cancelled.")
		return
	}

	if err := journalService.Post(ctx, int64(id), bookID); err != nil {
		fmt.Println("failed to post draft:", err)
		return
	}
	fmt.Println("Draft posted.")
}

func deleteDraftTransaction(ctx context.Context, bookID int64, journalService *journal.Service) {
	entries, err := journalService.ListByBookID(ctx, bookID)
	if err != nil {
		fmt.Println("Failed to get transaction logs:", err)
		return
	}

	drafts := draftEntries(entries)
	if len(drafts) == 0 {
		fmt.Println("No draft transactions to delete.")
		return
	}

	fmt.Println("\nDraft transactions:")
	for _, e := range drafts {
		fmt.Printf("  #%d [%s] %s\n", e.ID, e.EntryDate.Format("02-01-2006"), e.Description)
	}

	id, ok, err := prompt.ReadInt("Enter draft id to delete: ")
	if err != nil {
		fmt.Println("failed to delete draft:", err)
		return
	}
	if !ok {
		return
	}

	input, err := prompt.ReadYesNo(fmt.Sprintf("Delete draft #%d permanently? (y/n): ", id))
	if err != nil {
		fmt.Println("failed to delete draft:", err)
		return
	}
	if !input {
		fmt.Println("Cancelled.")
		return
	}

	if err := journalService.DeleteDraft(ctx, int64(id), bookID); err != nil {
		fmt.Println("failed to delete draft:", err)
		return
	}
	fmt.Println("Draft deleted.")
}

// cancelableEntries returns POSTED entries that may be voided: drafts have
// no balance effect, voided entries are already dead, and reversal entries
// must not be voided (voiding a void would silently re-post the original).
func cancelableEntries(entries []journal.JournalEntry) []journal.JournalEntry {
	var postedEntries []journal.JournalEntry
	for _, e := range entries {
		if e.Status == journal.StatusPosted && e.ReversalOf == nil {
			postedEntries = append(postedEntries, e)
		}
	}
	return postedEntries
}

func cancelTransaction(ctx context.Context, bookID int64, journalService *journal.Service) {
	entries, err := journalService.ListByBookID(ctx, bookID)
	if err != nil {
		fmt.Println("Failed to get transaction logs:", err)
		return
	}

	postedEntries := cancelableEntries(entries)
	if len(postedEntries) == 0 {
		fmt.Println("No cancelable transactions (only POSTED, non-reversal entries can be cancelled).")
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

	if err := journalService.Void(ctx, int64(id), bookID); err != nil {
		fmt.Println("failed to cancel transaction:", err)
		return
	}
	fmt.Println("Transaction cancelled (The balance has been restored to its original state).")
}
