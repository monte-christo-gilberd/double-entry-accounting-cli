package journal

import "context"

type Repository interface {
	Create(ctx context.Context, entry *JournalEntry) error
	GetByID(ctx context.Context, id int64) (*JournalEntry, error)
	ListByBookID(ctx context.Context, bookID int64) ([]JournalEntry, error)
	UpdateStatus(ctx context.Context, id int64, status Status) error
	CreateAndVoid(ctx context.Context, originalID int64, reversal *JournalEntry) error
}
