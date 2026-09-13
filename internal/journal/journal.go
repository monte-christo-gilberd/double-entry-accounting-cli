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

func (e *JournalEntry) BuildReversal() *JournalEntry {
	reversal := &JournalEntry{
		BookID:      e.BookID,
		EntryDate:   time.Now(),
		Description: "Void of entry #" + strconv.FormatInt(e.ID, 10) + ": " + e.Description,
		Status:      StatusPosted,
		ReversalOf:  &e.ID,
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
