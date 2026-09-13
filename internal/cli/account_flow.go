package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/account"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/prompt"
)

func addAccount(ctx context.Context, bookID int64, accountService *account.Service) {
	code, err := prompt.ReadRequiredLine("Account code (eg. 1000): ")
	if err != nil {
		fmt.Println("Failed to create account:", err)
		return
	}
	name, err := prompt.ReadRequiredLine("Account name: ")
	if err != nil {
		fmt.Println("Failed to create account:", err)
		return
	}
	accType := readAccountType()

	a := &account.Account{
		BookID:      bookID,
		Code:        code,
		Name:        name,
		AccountType: accType,
	}

	if err := accountService.Create(ctx, a); err != nil {
		fmt.Println("Failed to create account:", err)
		return
	}
	fmt.Printf("Account created: id=%d %s - %s\n", a.ID, a.Code, a.Name)
}

func readAccountType() string {
	for {
		fmt.Println("Account Type: ASSET, LIABILITY, EQUITY, REVENUE, EXPENSE")
		accType, err := prompt.ReadRequiredLine("Select type: ")
		if err != nil {
			fmt.Println("Failed to read account type:", err)
			return ""
		}

		accType = strings.ToUpper(accType)
		switch accType {
		case "ASSET", "LIABILITY", "EQUITY", "REVENUE", "EXPENSE":
			return accType
		default:
			fmt.Println("  Invalid Type.")
		}
	}
}

func listAccountsNumbered(ctx context.Context, bookID int64, accountService *account.Service) []account.Account {
	accounts, err := accountService.ListByBookID(ctx, bookID)
	if err != nil {
		fmt.Println("Failed to get account list:", err)
		return nil
	}
	if len(accounts) == 0 {
		fmt.Println("No account in this book yet.")
		return nil
	}
	for _, a := range accounts {
		fmt.Printf("  id=%d  %s - %s (%s)\n", a.ID, a.Code, a.Name, a.AccountType)
	}
	return accounts
}

func editAccount(ctx context.Context, bookID int64, accountService *account.Service) {
	if listAccountsNumbered(ctx, bookID, accountService) == nil {
		return
	}

	id, ok, err := prompt.ReadInt("Account ID that you want to edit: ")
	if err != nil {
		fmt.Println("Failed to edit account:", err)
		return
	}

	if !ok {
		return
	}

	a, err := accountService.GetByIDAndBookID(ctx, int64(id), bookID)
	if err != nil {
		fmt.Println("Account not found in this book:", err)
		return
	}

	fmt.Println("Empty input to use previous name.")
	a.Code, err = prompt.ReadLineDefault(fmt.Sprintf("Code [%s]: ", a.Code), a.Code)
	if err != nil {
		fmt.Println("Failed to edit account:", err)
		return
	}

	a.Name, err = prompt.ReadLineDefault(fmt.Sprintf("Name [%s]: ", a.Name), a.Name)
	if err != nil {
		fmt.Println("Failed to edit account:", err)
		return
	}

	fmt.Printf("Account Type Currently: %s\n", a.AccountType)
	input, err := prompt.ReadYesNo("Change type? (y/n): ")
	if err != nil {
		fmt.Println("Failed to edit account:", err)
		return
	}

	if input {
		a.AccountType = readAccountType()
	}

	if err := accountService.Update(ctx, a, bookID); err != nil {
		fmt.Println("Failed to account:", err)
		return
	}
	fmt.Println("Account successfully updated.")
}

func deleteAccount(ctx context.Context, bookID int64, accountService *account.Service) {
	if listAccountsNumbered(ctx, bookID, accountService) == nil {
		return
	}

	id, ok, err := prompt.ReadInt("Account ID that you want to delete: ")
	if err != nil {
		fmt.Println("Failed to delete account:", err)
		return
	}

	if !ok {
		return
	}

	if _, err := accountService.GetByIDAndBookID(ctx, int64(id), bookID); err != nil {
		fmt.Println("Account not found in this book:", err)
		return
	}

	input, err := prompt.ReadYesNo("Are you sure to delete this account? (y/n): ")
	if err != nil {
		fmt.Println("Failed to delete account:", err)
		return
	}

	if !input {
		fmt.Println("Cancelled.")
		return
	}

	if err := accountService.Delete(ctx, int64(id), bookID); err != nil {
		fmt.Println("Failed to delete Account (This account might still have transactions):", err)
		return
	}
	fmt.Println("Account deleted.")
}
