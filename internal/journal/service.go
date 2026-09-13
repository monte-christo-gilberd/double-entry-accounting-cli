package journal

import (
	"context"
	"errors"
	"fmt"
	"math"
)

var (
	ErrInvalidJournal  = errors.New("invalid journal entry")
	ErrJournalNotFound = errors.New("journal entry not found")
	ErrJournalPosted   = errors.New("journal entry already posted")
	ErrJournalVoided   = errors.New("journal entry already voided")
	ErrCannotVoidDraft = errors.New("draft entries have no balance effect; delete them instead of voiding")
)

type AccountValidator interface {
	ValidateBelongsToBook(ctx context.Context, accountID int64, bookID int64) error
}

type Service struct {
	repository Repository
	accounts   AccountValidator
}

func NewService(repository Repository, account AccountValidator) *Service {
	return &Service{
		repository: repository,
		accounts:   account,
	}
}

func (s *Service) CreateDraft(
	ctx context.Context,
	entry *JournalEntry,
) error {
	if entry == nil {
		return fmt.Errorf("%w: entry is nil", ErrInvalidJournal)
	}
	normalizeMoney(entry)
	if err := validateJournalEntry(entry); err != nil {
		return err
	}

	for _, line := range entry.Lines {
		if err := s.accounts.ValidateBelongsToBook(
			ctx,
			line.AccountID,
			entry.BookID,
		); err != nil {
			return fmt.Errorf(
				"validate account %d: %w",
				line.AccountID,
				err,
			)
		}
	}

	entry.Status = StatusDraft

	if err := s.repository.Create(ctx, entry); err != nil {
		return fmt.Errorf("create journal entry: %w", err)
	}
	return nil
}

// moneyScale matches the NUMERIC(19,4) columns: amounts are exact in units
// of 1/10000, so balance checks use integer math instead of float epsilon.
const moneyScale = 10000

func toMoneyUnits(v float64) int64 {
	return int64(math.Round(v * moneyScale))
}

// normalizeMoney rounds every line to the database precision so the Go
// balance check and what Postgres stores can never disagree on dust like
// 0.30000000000000004 or a 5th decimal the column would silently round.
func normalizeMoney(entry *JournalEntry) {
	for i := range entry.Lines {
		entry.Lines[i].Debit = float64(toMoneyUnits(entry.Lines[i].Debit)) / moneyScale
		entry.Lines[i].Credit = float64(toMoneyUnits(entry.Lines[i].Credit)) / moneyScale
	}
}

func validateJournalEntry(entry *JournalEntry) error {
	if entry == nil {
		return fmt.Errorf("%w: entry is nil", ErrInvalidJournal)
	}

	if entry.BookID <= 0 {
		return fmt.Errorf("%w: invalid book ID", ErrInvalidJournal)
	}

	if entry.Description == "" {
		return fmt.Errorf("%w: description is required", ErrInvalidJournal)
	}

	if len(entry.Lines) < 2 {
		return fmt.Errorf("%w: journal must have at least 2 lines", ErrInvalidJournal)
	}

	if entry.EntryDate.IsZero() {
		return fmt.Errorf("%w: date is required", ErrInvalidJournal)
	}

	seen := make(map[int64]struct{}, len(entry.Lines))
	var totalDebitUnits int64
	var totalCreditUnits int64

	for i, line := range entry.Lines {
		if line.AccountID <= 0 {
			return fmt.Errorf(
				"%w: line %d has invalid account ID",
				ErrInvalidJournal,
				i+1,
			)
		}

		if _, dup := seen[line.AccountID]; dup {
			return fmt.Errorf(
				"%w: line %d reuses account %d (one line per account)",
				ErrInvalidJournal,
				i+1,
				line.AccountID,
			)
		}
		seen[line.AccountID] = struct{}{}

		if line.Debit < 0 || line.Credit < 0 {
			return fmt.Errorf(
				"%w: line %d has negative amount",
				ErrInvalidJournal,
				i+1,
			)
		}

		if line.Debit == 0 && line.Credit == 0 {
			return fmt.Errorf(
				"%w: line %d must have debit or credit",
				ErrInvalidJournal,
				i+1,
			)
		}

		if line.Debit > 0 && line.Credit > 0 {
			return fmt.Errorf(
				"%w: line %d cannot have both debit and credit",
				ErrInvalidJournal,
				i+1,
			)
		}

		totalDebitUnits += toMoneyUnits(line.Debit)
		totalCreditUnits += toMoneyUnits(line.Credit)
	}

	if totalDebitUnits <= 0 {
		return fmt.Errorf(
			"%w: total debit must be greater than zero",
			ErrInvalidJournal,
		)
	}

	if totalDebitUnits != totalCreditUnits {
		return fmt.Errorf(
			"%w: debit %.4f does not equal credit %.4f",
			ErrInvalidJournal,
			float64(totalDebitUnits)/moneyScale,
			float64(totalCreditUnits)/moneyScale,
		)
	}
	return nil
}

func (s *Service) Post(
	ctx context.Context,
	id int64,
	bookID int64,
) error {
	if id <= 0 {
		return fmt.Errorf("%w: invalid journal ID", ErrInvalidJournal)
	}
	if bookID <= 0 {
		return fmt.Errorf("%w: invalid book ID", ErrInvalidJournal)
	}

	entry, err := s.repository.GetByID(ctx, id, bookID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrJournalNotFound, err)
	}

	if entry.Status == StatusPosted {
		return ErrJournalPosted
	}

	if entry.Status == StatusVoided {
		return ErrJournalVoided
	}

	if err := validateJournalEntry(entry); err != nil {
		return err
	}

	for _, line := range entry.Lines {
		if err := s.accounts.ValidateBelongsToBook(
			ctx,
			line.AccountID,
			entry.BookID,
		); err != nil {
			return fmt.Errorf(
				"validate account %d: %w",
				line.AccountID,
				err,
			)
		}
	}

	// Guarded on from=DRAFT so a concurrent void/post cannot slip through.
	if err := s.repository.UpdateStatus(
		ctx,
		entry.ID,
		StatusDraft,
		StatusPosted,
		bookID,
	); err != nil {
		return fmt.Errorf("post journal entry: %w", err)
	}

	return nil
}

func (s *Service) Void(
	ctx context.Context,
	id int64,
	bookID int64,
) error {
	if id <= 0 {
		return fmt.Errorf("%w: invalid journal ID", ErrInvalidJournal)
	}
	if bookID <= 0 {
		return fmt.Errorf("%w: invalid book ID", ErrInvalidJournal)
	}

	entry, err := s.repository.GetByID(ctx, id, bookID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrJournalNotFound, err)
	}

	switch entry.Status {
	case StatusDraft:
		return ErrCannotVoidDraft
	case StatusVoided:
		return ErrJournalVoided
	case StatusPosted:
		reversal := entry.BuildReversal()
		if err := s.repository.CreateAndVoid(ctx, entry.ID, reversal, bookID); err != nil {
			return fmt.Errorf("void journal entry: %w", err)
		}
	default:
		return fmt.Errorf("%w: status %s", ErrInvalidJournal, entry.Status)
	}

	return nil
}

func (s *Service) Transact(
	ctx context.Context,
	entry *JournalEntry,
) error {
	if entry == nil {
		return fmt.Errorf("%w: entry is nil", ErrInvalidJournal)
	}
	normalizeMoney(entry)
	if err := validateJournalEntry(entry); err != nil {
		return err
	}

	for _, line := range entry.Lines {
		if err := s.accounts.ValidateBelongsToBook(
			ctx,
			line.AccountID,
			entry.BookID,
		); err != nil {
			return fmt.Errorf(
				"validate account %d: %w",
				line.AccountID,
				err,
			)
		}
	}

	entry.Status = StatusPosted

	if err := s.repository.Create(ctx, entry); err != nil {
		return fmt.Errorf("create journal entry: %w", err)
	}
	return nil
}

func (s *Service) ListByBookID(
	ctx context.Context,
	bookID int64,
) ([]JournalEntry, error) {
	return s.repository.ListByBookID(ctx, bookID)
}

func (s *Service) ListRecentByBookID(
	ctx context.Context,
	bookID int64,
	limit int,
) ([]JournalEntry, error) {

	if limit <= 0 {
		limit = 1
	}
	return s.repository.ListRecentByBookID(ctx, bookID, limit)
}

func (s *Service) GetAccountBalances(
	ctx context.Context,
	bookID int64,
) ([]AccountBalance, error) {
	return s.repository.GetAccountBalances(ctx, bookID)
}
