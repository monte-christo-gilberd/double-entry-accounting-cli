package book

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidBook = errors.New("invalid book")
	ErrBookNotFound = errors.New("book not found")
)

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
	book.Name = strings.TrimSpace(book.Name)
	if book.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidBook)
	}
	if len(book.Name) > 100 {
		return fmt.Errorf("%w: name exceeds 100 characters", ErrInvalidBook)
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
	b, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: book %d", ErrBookNotFound, id)
		}
		return nil, fmt.Errorf("get book %d: %w", id, err)
	}
	return b, nil
}

func (s *Service) List(ctx context.Context) ([]Book, error) {
	books, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list books: %w", err)
	}
	return books, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {

	if id <= 0 {
		return fmt.Errorf("%w: invalid book ID", ErrInvalidBook)
	}

	if err := s.repository.Delete(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: book %d", ErrBookNotFound, id)
		}
		return fmt.Errorf("delete book %d: %w", id, err)
	}
	
	return nil
}
