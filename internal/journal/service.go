package journal

import (
	"context"
	"database/sql"
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
	ErrCannotDeletePosted = errors.New("only draft entries can be deleted; void posted entries instead")
	ErrCannotVoidReversal = errors.New("reversal entries cannot be voided")
)

// maxMoneyValue is the largest absolute amount accepted in a single line.
// NUMERIC(19,4) tops out near 1e15 and float64 is exact only to 2^53, so
// 9e14 keeps units (v*10000 = 9e18) inside int64 (max ~9.22e18).
const maxMoneyValue = 900_000_000_000_000.0

// recentListMax caps ListRecentByBookID so a huge limit cannot OOM.
const recentListMax = 1000

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
	if entry.ReversalOf != nil {
		return fmt.Errorf("%w: reversal_of must be nil on create", ErrInvalidJournal)
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
// It works on a copy-friendly basis: callers that need the original
// preserved on validation failure should normalize a clone first. Here the
// service normalizes in place only after the reversal_of guard, and
// validation runs immediately, so a failure leaves a rounded entry —
// documented so callers clone if they must preserve raw input.
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

	if entry.ReversalOf != nil {
		return fmt.Errorf("%w: reversal_of must be nil on a normal entry", ErrInvalidJournal)
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

		if line.Debit > maxMoneyValue || line.Credit > maxMoneyValue {
			return fmt.Errorf(
				"%w: line %d exceeds maximum amount %.4f",
				ErrInvalidJournal,
				i+1,
				maxMoneyValue,
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
		return mapNotFound(id, bookID, err)
	}

	if entry.Status == StatusPosted {
		return fmt.Errorf("%w: entry %d in book %d", ErrJournalPosted, entry.ID, bookID)
	}

	if entry.Status == StatusVoided {
		return fmt.Errorf("%w: entry %d in book %d", ErrJournalVoided, entry.ID, bookID)
	}

	if entry.ReversalOf != nil {
		return fmt.Errorf("%w: entry %d is a reversal and cannot be posted", ErrInvalidJournal, entry.ID)
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
		return mapStatusConflict(ctx, s.repository, entry.ID, bookID, "post journal entry", err)
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
		return mapNotFound(id, bookID, err)
	}

	switch entry.Status {
	case StatusDraft:
		return ErrCannotVoidDraft
	case StatusVoided:
		return fmt.Errorf("%w: entry %d in book %d", ErrJournalVoided, entry.ID, bookID)
	case StatusPosted:
		if entry.ReversalOf != nil {
			return fmt.Errorf("%w: entry %d is a reversal", ErrCannotVoidReversal, entry.ID)
		}
		reversal := entry.BuildReversal()
		if reversal == nil {
			return fmt.Errorf("%w: cannot build reversal for entry %d", ErrInvalidJournal, entry.ID)
		}
		if err := s.repository.CreateAndVoid(ctx, entry.ID, reversal, bookID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return mapStatusConflict(ctx, s.repository, entry.ID, bookID, "void journal entry", err)
			}
			return fmt.Errorf("void journal entry: %w", err)
		}
	default:
		return fmt.Errorf("%w: status %s", ErrInvalidJournal, entry.Status)
	}

	return nil
}

func (s *Service) DeleteDraft(
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
		return mapNotFound(id, bookID, err)
	}

	if entry.Status != StatusDraft {
		return ErrCannotDeletePosted
	}

	if err := s.repository.DeleteDraft(ctx, entry.ID, bookID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mapStatusConflict(ctx, s.repository, entry.ID, bookID, "delete draft entry", err)
		}
		return fmt.Errorf("delete draft entry: %w", err)
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
	if entry.ReversalOf != nil {
		return fmt.Errorf("%w: reversal_of must be nil on create", ErrInvalidJournal)
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

func (s *Service) GetByID(
	ctx context.Context,
	id int64,
	bookID int64,
) (*JournalEntry, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: invalid journal ID", ErrInvalidJournal)
	}
	if bookID <= 0 {
		return nil, fmt.Errorf("%w: invalid book ID", ErrInvalidJournal)
	}
	entry, err := s.repository.GetByID(ctx, id, bookID)
	if err != nil {
		return nil, mapNotFound(id, bookID, err)
	}
	return entry, nil
}

// mapNotFound turns a missing row into a clean ErrJournalNotFound without
// leaking driver text like "sql: no rows in result set" to CLI users.
func mapNotFound(id, bookID int64, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: entry %d in book %d", ErrJournalNotFound, id, bookID)
	}
	return fmt.Errorf("get journal entry: %w", err)
}

// mapStatusConflict turns a rowsAffected==0 (sql.ErrNoRows) from a guarded
// status change into a precise user-facing error by re-reading the entry.
// A concurrent Post/Void/Delete can change DRAFT->POSTED/VOIDED between
// GetByID and the guarded UPDATE/DELETE; reporting "not found" would hide it.
func mapStatusConflict(ctx context.Context, repo Repository, id, bookID int64, op string, err error) error {
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s: %w", op, err)
	}
	current, getErr := repo.GetByID(ctx, id, bookID)
	if getErr != nil {
		if errors.Is(getErr, sql.ErrNoRows) {
			return fmt.Errorf("%w: entry %d in book %d", ErrJournalNotFound, id, bookID)
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	switch current.Status {
	case StatusPosted:
		return fmt.Errorf("%s: %w: entry %d in book %d", op, ErrJournalPosted, id, bookID)
	case StatusVoided:
		return fmt.Errorf("%s: %w: entry %d in book %d", op, ErrJournalVoided, id, bookID)
	default:
		return fmt.Errorf("%s: %w: entry %d in book %d", op, ErrJournalNotFound, id, bookID)
	}
}

func (s *Service) ListByBookID(
	ctx context.Context,
	bookID int64,
) ([]JournalEntry, error) {
	if bookID <= 0 {
		return nil, fmt.Errorf("%w: invalid book ID", ErrInvalidJournal)
	}
	return s.repository.ListByBookID(ctx, bookID)
}

func (s *Service) ListDetailedByBookID(
	ctx context.Context,
	bookID int64,
) ([]JournalEntry, error) {
	if bookID <= 0 {
		return nil, fmt.Errorf("%w: invalid book ID", ErrInvalidJournal)
	}
	return s.repository.ListDetailedByBookID(ctx, bookID)
}

func (s *Service) ListRecentByBookID(
	ctx context.Context,
	bookID int64,
	limit int,
) ([]JournalEntry, error) {

	if bookID <= 0 {
		return nil, fmt.Errorf("%w: invalid book ID", ErrInvalidJournal)
	}
	if limit <= 0 {
		return nil, fmt.Errorf("%w: limit must be more than 0", ErrInvalidJournal)
	}
	if limit > recentListMax {
		limit = recentListMax
	}
	return s.repository.ListRecentByBookID(ctx, bookID, limit)
}

func (s *Service) GetAccountBalances(
	ctx context.Context,
	bookID int64,
) ([]AccountBalance, error) {
	if bookID <= 0 {
		return nil, fmt.Errorf("%w: invalid book ID", ErrInvalidJournal)
	}
	return s.repository.GetAccountBalances(ctx, bookID)
}
