package journal

import (
	"strconv"
	"time"
)

type Status string

const (
	StatusDraft  Status = "DRAFT"
	StatusPosted Status = "POSTED"
	StatusVoided Status = "VOIDED"
)

type JournalLine struct {
	ID             int64
	JournalEntryID int64
	BookID         int64
	AccountID      int64
	Debit          float64
	Credit         float64
}

type JournalEntry struct {
	ID          int64
	BookID      int64
	EntryDate   time.Time
	Description string
	Status      Status
	ReversalOf  *int64
	CreatedAt   time.Time
	Lines       []JournalLine
}

// BuildReversal builds the offsetting entry for Void. The reversal is
// VOIDED, not POSTED: VOIDED uniformly means "excluded from balances,
// audit trail only". The balance query sums POSTED entries only, so the
// voided original (now VOIDED, excluded) is removed exactly once — a POSTED
// reversal would subtract it a second time and double-count the void.
// Returns nil if e is nil so callers get a clean failure instead of a panic.
func (e *JournalEntry) BuildReversal() *JournalEntry {
	if e == nil {
		return nil
	}
	id := e.ID
	reversal := &JournalEntry{
		BookID:      e.BookID,
		EntryDate:   time.Now(),
		Description: "Void of entry #" + strconv.FormatInt(id, 10) + ": " + e.Description,
		Status:      StatusVoided,
		ReversalOf:  &id,
	}
	for _, line := range e.Lines {
		reversal.Lines = append(reversal.Lines, JournalLine{
			BookID:    e.BookID,
			AccountID: line.AccountID,
			Debit:     line.Credit,
			Credit:    line.Debit,
		})
	}
	return reversal
}
