package journal

import (
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
		EntryDate:   e.EntryDate,
		Description: "Void of entry #" + itoa(e.ID) + ": " + e.Description,
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

func itoa(id int64) string {
	if id == 0 {
		return "0"
	}

	neg := id < 0
	if neg {
		id = -id
	}
	var buf [20]byte
	i := len(buf)
	for id > 0 {
		i--
		buf[i] = byte('0' + id%10)
		id /= 10
	}
	if neg {
		id--
		buf[i] = '-'
	}
	return string(buf[i:])
}
