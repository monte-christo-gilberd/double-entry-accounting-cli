package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidAccount  = errors.New("invalid account")
	ErrAccountNotFound = errors.New("account not found")
	ErrAccountExists   = errors.New("account code already exists")
	ErrAccountInUse    = errors.New("account has transactions and cannot be deleted")
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(
	ctx context.Context,
	account *Account,
) error {
	if err := validateAccount(account); err != nil {
		return err
	}
	account.Code = strings.TrimSpace(account.Code)
	account.Name = strings.TrimSpace(account.Name)

	if err := s.repository.Create(ctx, account); err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("create account %q: %w", account.Code, ErrAccountExists)
		}
		return fmt.Errorf("create account: %w", err)
	}

	return nil
}

// isUniqueViolation reports Postgres SQLSTATE 23505 (unique_violation).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// isForeignKeyViolation reports Postgres SQLSTATE 23503.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func (s *Service) ValidateBelongsToBook(
	ctx context.Context,
	accountID int64,
	bookID int64,
) error {
	if accountID <= 0 {
		return fmt.Errorf("%w: invalid account ID", ErrInvalidAccount)
	}

	if bookID <= 0 {
		return fmt.Errorf("%w: invalid book ID", ErrInvalidAccount)
	}

	_, err := s.repository.GetByIDAndBookID(ctx, accountID, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf(
				"%w: account %d does not belong to book %d",
				ErrAccountNotFound,
				accountID,
				bookID,
			)
		}
		return fmt.Errorf("validate account %d in book %d: %w", accountID, bookID, err)
	}
	return nil
}

func validateAccount(account *Account) error {
	if account == nil {
		return fmt.Errorf("%w: account is nil", ErrInvalidAccount)
	}

	if account.BookID <= 0 {
		return fmt.Errorf("%w: invalid book ID", ErrInvalidAccount)
	}

	code := strings.TrimSpace(account.Code)
	if code == "" {
		return fmt.Errorf("%w: code is required", ErrInvalidAccount)
	}
	if len(code) > 20 {
		return fmt.Errorf("%w: code exceeds 20 characters", ErrInvalidAccount)
	}

	// Codes double as the CLI selection keys (0 finishes, q cancels), so
	// these values can never address an account and are rejected outright.
	// Trimmed + case-folded so " 0 ", " Q " cannot slip through.
	switch strings.ToLower(code) {
	case "0", "q", "cancel":
		return fmt.Errorf("%w: code %q is reserved", ErrInvalidAccount, account.Code)
	}

	name := strings.TrimSpace(account.Name)
	if name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidAccount)
	}
	if len(name) > 100 {
		return fmt.Errorf("%w: name exceeds 100 characters", ErrInvalidAccount)
	}

	switch account.AccountType {
	case "ASSET", "LIABILITY", "EQUITY", "REVENUE", "EXPENSE":
	default:
		return fmt.Errorf(
			"%w: invalid account type %q",
			ErrInvalidAccount,
			account.AccountType,
		)
	}

	return nil
}

func (s *Service) Update(
	ctx context.Context,
	account *Account,
	bookID int64,
) error {
	if account != nil && account.ID <= 0 {
		return fmt.Errorf("%w: invalid account ID", ErrInvalidAccount)
	}

	if bookID <= 0 {
		return fmt.Errorf("%w: invalid book ID", ErrInvalidAccount)
	}

	if err := validateAccount(account); err != nil {
		return err
	}

	if account.BookID != bookID {
		return fmt.Errorf("%w: account belongs to book %d, not book %d", ErrInvalidAccount, account.BookID, bookID)
	}

	// AccountType feeds normal-balance display (Dr/Cr) for all history.
	// Changing it rewrites the meaning of past postings, so forbid it:
	// create a new account instead of retyping an in-use one.
	existing, err := s.repository.GetByIDAndBookID(ctx, account.ID, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: account %d in book %d", ErrAccountNotFound, account.ID, bookID)
		}
		return fmt.Errorf("update account: %w", err)
	}
	if existing.AccountType != account.AccountType {
		return fmt.Errorf("%w: cannot change account type from %q to %q; create a new account instead", ErrInvalidAccount, existing.AccountType, account.AccountType)
	}
	account.Code = strings.TrimSpace(account.Code)
	account.Name = strings.TrimSpace(account.Name)

	if err := s.repository.Update(ctx, account, bookID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: account %d in book %d", ErrAccountNotFound, account.ID, bookID)
		}
		if isUniqueViolation(err) {
			return fmt.Errorf("update account %q: %w", account.Code, ErrAccountExists)
		}
		return fmt.Errorf("update account: %w", err)
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, id int64, bookID int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: invalid account ID", ErrInvalidAccount)
	}

	if bookID <= 0 {
		return fmt.Errorf("%w: invalid book ID", ErrInvalidAccount)
	}

	if err := s.repository.Delete(ctx, id, bookID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: account %d in book %d", ErrAccountNotFound, id, bookID)
		}
		if isForeignKeyViolation(err) {
			return fmt.Errorf("delete account %d: %w", id, ErrAccountInUse)
		}
		return fmt.Errorf("delete account: %w", err)
	}

	return nil
}

func (s *Service) GetByIDAndBookID(
	ctx context.Context,
	id int64,
	bookID int64,
) (*Account, error) {
	if id <= 0 || bookID <= 0 {
		return nil, fmt.Errorf("%w: invalid ID", ErrInvalidAccount)
	}
	acc, err := s.repository.GetByIDAndBookID(ctx, id, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: account %d in book %d", ErrAccountNotFound, id, bookID)
		}
		return nil, fmt.Errorf("get account: %w", err)
	}
	return acc, nil
}

func (s *Service) GetByCodeAndBookID(
	ctx context.Context,
	code string,
	bookID int64,
) (*Account, error) {
	code = strings.TrimSpace(code)
	if code == "" || bookID <= 0 {
		return nil, fmt.Errorf("%w: invalid code or book ID", ErrInvalidAccount)
	}
	acc, err := s.repository.GetByCodeAndBookID(ctx, code, bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: account %q in book %d", ErrAccountNotFound, code, bookID)
		}
		return nil, fmt.Errorf("get account: %w", err)
	}
	return acc, nil
}

func (s *Service) ListByBookID(ctx context.Context, bookID int64) ([]Account, error) {
	if bookID <= 0 {
		return nil, fmt.Errorf("%w: invalid book ID", ErrInvalidAccount)
	}
	accounts, err := s.repository.ListByBookID(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("list accounts in book %d: %w", bookID, err)
	}
	return accounts, nil
}
