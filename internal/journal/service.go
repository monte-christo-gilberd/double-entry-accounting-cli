package journal

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrInvalidJournal  = errors.New("invalid journal entry")
	ErrJournalNotFound = errors.New("journal entry not found")
	ErrJournalPosted   = errors.New("journal entry already posted")
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) CreateDraft(
	ctx context.Context,
	entry *JournalEntry,
) error {
	// if err := validate

	entry.Status = StatusDraft

	if err := s.repository.Create(ctx, entry); err != nil {
		return fmt.Errorf("create journal entry: %w", err)
	}
	return nil
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
		return fmt.Errorf(
			"%w: journal must have at least 2 lines",
			ErrInvalidJournal,
		)
	}

	var totalDebit float64
	var totalCredit float64

	for i, line := range entry.Lines {
		if line.AccountID <= 0 {
			return fmt.Errorf(
				"%w: line %d has invalid account ID",
				ErrInvalidJournal,
				i+1,
			)
		}

		if line.Debit < 0 || line.Credit < 0 {
			return fmt.Errorf(
				"%w: line %d has negative amount",
				ErrInvalidJournal,
				i+1,
			)
		}

		if line.Debit == 0 || line.Credit == 0 {
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

		totalDebit += line.Debit
		totalCredit += line.Credit
	}

	if totalDebit <= 0 {
		return fmt.Errorf(
			"%w: total debit must be greater than zero",
			ErrInvalidJournal,
		)
	}

	if totalDebit != totalCredit {
		return fmt.Errorf(
			"%w: debit %.2f does not equal credit %.2f",
			ErrInvalidJournal,
			totalDebit,
			totalCredit,
		)
	}
	return nil
}

func (s *Service) Post(
	ctx context.Context,
	id int64,
) error {
	entry, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get journal entry: %w", err)
	}

	if entry.Status == StatusPosted {
		return ErrJournalPosted
	}

	if err := validateJournalEntry(entry); err != nil {
		return err
	}

	if err := s.repository.UpdateStatus(
		ctx,
		entry.ID,
		StatusPosted,
	); err != nil {
		return fmt.Errorf("post journal entry: %w", err)
	}

	return nil
}
