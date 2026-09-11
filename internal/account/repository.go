package account

import "context"

type Repository interface {
	Create(ctx context.Context, account *Account) error
	GetByID(ctx context.Context, id int64) (*Account, error)
	GetByIDAndBookID(ctx context.Context, id int64, bookID int64) (*Account, error)
	ListByBookID(ctx context.Context, bookID int64) ([]Account, error)
	Update(ctx context.Context, account *Account) error
	Delete(ctx context.Context, id int64) error
}
