package journal

import "time"

type Status string

const (
	StatusDraft  Status = "DRAFT"
	StatusPosted Status = "POSTED"
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
	CreatedAt   time.Time
	Lines       []JournalLine
}
