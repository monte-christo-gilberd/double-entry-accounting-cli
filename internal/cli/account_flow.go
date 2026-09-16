package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/account"
	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/prompt"
)

func addAccount(ctx context.Context, bookID int64, accountService *account.Service) {
	code, err := prompt.ReadRequiredLine("Account code (eg. 1000): ")
	if err != nil {
		if errors.Is(err, prompt.ErrInputClosed) {
			return
		}
		fmt.Println("Failed to create account:", err)
		return
	}
	name, err := prompt.ReadRequiredLine("Account name: ")
	if err != nil {
		if errors.Is(err, prompt.ErrInputClosed) {
			return
		}
		fmt.Println("Failed to create account:", err)
		return
	}
	accType, ok := readAccountType()
	if !ok {
		fmt.Println("Cancelled.")
		return
	}

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
	fmt.Printf("Account created: %s - %s\n", a.Code, a.Name)
}

func readAccountType() (string, bool) {
	for {
		fmt.Println("Account Type: ASSET, LIABILITY, EQUITY, REVENUE, EXPENSE")
		accType, err := prompt.ReadLine("Select type (empty to cancel): ")
		if err != nil {
			return "", false
		}
		if accType == "" {
			return "", false
		}

		accType = strings.ToUpper(accType)
		switch accType {
		case "ASSET", "LIABILITY", "EQUITY", "REVENUE", "EXPENSE":
			return accType, true
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
		fmt.Printf("  %s - %s (%s)\n", a.Code, a.Name, a.AccountType)
	}
	return accounts
}

func editAccount(ctx context.Context, bookID int64, accountService *account.Service) {
	if listAccountsNumbered(ctx, bookID, accountService) == nil {
		return
	}

	code, err := prompt.ReadLine("Account code that you want to edit (empty to cancel): ")
	if err != nil {
		if errors.Is(err, prompt.ErrInputClosed) {
			return
		}
		fmt.Println("Failed to edit account:", err)
		return
	}
	if code == "" {
		fmt.Println("Cancelled.")
		return
	}

	a, err := accountService.GetByCodeAndBookID(ctx, code, bookID)
	if err != nil {
		fmt.Println("Account not found in this book:", err)
		return
	}

	fmt.Println("Empty input to use previous name.")
	a.Code, err = prompt.ReadLineDefault(fmt.Sprintf("Code [%s]: ", a.Code), a.Code)
	if err != nil {
		if errors.Is(err, prompt.ErrInputClosed) {
			return
		}
		fmt.Println("Failed to edit account:", err)
		return
	}

	a.Name, err = prompt.ReadLineDefault(fmt.Sprintf("Name [%s]: ", a.Name), a.Name)
	if err != nil {
		if errors.Is(err, prompt.ErrInputClosed) {
			return
		}
		fmt.Println("Failed to edit account:", err)
		return
	}

	fmt.Printf("Account Type: %s (type cannot be changed; create a new account for a different type)\n", a.AccountType)

	if err := accountService.Update(ctx, a, bookID); err != nil {
		fmt.Println("Failed to update account:", err)
		return
	}
	fmt.Println("Account successfully updated.")
}

func deleteAccount(ctx context.Context, bookID int64, accountService *account.Service) {
	if listAccountsNumbered(ctx, bookID, accountService) == nil {
		return
	}

	code, err := prompt.ReadLine("Account code that you want to delete (empty to cancel): ")
	if err != nil {
		if errors.Is(err, prompt.ErrInputClosed) {
			return
		}
		fmt.Println("Failed to delete account:", err)
		return
	}
	if code == "" {
		fmt.Println("Cancelled.")
		return
	}

	del, err := accountService.GetByCodeAndBookID(ctx, code, bookID)
	if err != nil {
		fmt.Println("Account not found in this book:", err)
		return
	}

	input, err := prompt.ReadYesNo(fmt.Sprintf("Are you sure to delete %q? (y/n): ", del.Code+" - "+del.Name))
	if err != nil {
		if errors.Is(err, prompt.ErrInputClosed) {
			return
		}
		fmt.Println("Failed to delete account:", err)
		return
	}

	if !input {
		fmt.Println("Cancelled.")
		return
	}

	if err := accountService.Delete(ctx, del.ID, bookID); err != nil {
		fmt.Println("Failed to delete account:", err)
		return
	}
	fmt.Println("Account deleted.")
}
