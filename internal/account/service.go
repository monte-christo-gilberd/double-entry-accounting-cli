package account

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrInvalidAccount  = errors.New("invalid account")
	ErrAccountNotFound = errors.New("account not found")
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

	if err := s.repository.Create(ctx, account); err != nil {
		return fmt.Errorf("create account: %w", err)
	}

	return nil
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
		return fmt.Errorf(
			"%w: account %d does not belong to book %d",
			ErrAccountNotFound,
			accountID,
			bookID,
		)
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

	if account.Code == "" {
		return fmt.Errorf("%w: code is required", ErrInvalidAccount)
	}

	if account.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidAccount)
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
) error {
	if account != nil && account.ID <= 0 {
		return fmt.Errorf("%w: invalid account ID", ErrInvalidAccount)
	}

	if err := validateAccount(account); err != nil {
		return err
	}

	if err := s.repository.Update(ctx, account); err != nil {
		return fmt.Errorf("update account: %w", err)
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: invalid account ID", ErrInvalidAccount)
	}

	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}

	return nil
}
