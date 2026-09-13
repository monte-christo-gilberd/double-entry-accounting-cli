package journal

import "context"

type Repository interface {
	Create(ctx context.Context, entry *JournalEntry) error
	GetByID(ctx context.Context, id int64, bookID int64) (*JournalEntry, error)
	ListByBookID(ctx context.Context, bookID int64) ([]JournalEntry, error)
	ListDetailedByBookID(ctx context.Context, bookID int64) ([]JournalEntry, error)
	UpdateStatus(ctx context.Context, id int64, from Status, to Status, bookID int64) error
	DeleteDraft(ctx context.Context, id int64, bookID int64) error
	CreateAndVoid(ctx context.Context, originalID int64, reversal *JournalEntry, bookID int64) error
	ListRecentByBookID(ctx context.Context, bookID int64, limit int) ([]JournalEntry, error)
	GetAccountBalances(ctx context.Context, bookID int64) ([]AccountBalance, error)
}

type AccountBalance struct {
	AccountID int64
	Balance   float64 // positive = normal debit-side balance; negative = normal credit-side balance
}
