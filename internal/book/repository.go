package book

import "context"

type Repository interface {
	Create(ctx context.Context, book *Book) error
	GetByID(ctx context.Context, id int64) (*Book, error)
	List(ctx context.Context) ([]Book, error)
	Delete(ctx context.Context, id int64) error
}
