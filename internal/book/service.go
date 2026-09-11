package book

import (
	"context"
	"errors"
	"fmt"
)

var ErrInvalidBook = errors.New("invalid book")

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(ctx context.Context, book *Book) error {
	if book == nil {
		return fmt.Errorf("%w: book is nil", ErrInvalidBook)
	}
	if book.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidBook)
	}

	if err := s.repository.Create(ctx, book); err != nil {
		return fmt.Errorf("create book: %w", err)
	}
	return nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Book, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: invalid book ID", ErrInvalidBook)
	}
	return s.repository.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Book, error) {
	return s.repository.List(ctx)
}

func (s *Service) Delete(ctx context.Context, id int64) error {

	if id <= 0 {
		return fmt.Errorf("%w: invalid book ID", ErrInvalidBook)
	}

	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete book: %w", err)
	}
	
	return nil
}
