package book

import (
	"context"
	"database/sql"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

func (r *PostgresRepository) Create(ctx context.Context, book *Book) error {
	const query = `
		INSERT INTO books (name, description)
		VALUES ($1, $2)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(
		ctx,
		query,
		book.Name,
		book.Description,
	).Scan(
		&book.ID,
		&book.CreatedAt,
	)
}

func (r *PostgresRepository) GetByID(
	ctx context.Context,
	id int64,
) (*Book, error) {
	const query = `
		SELECT id, name, description, created_at
		FROM books
		WHERE id = $1
	`
	var book Book

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&book.ID,
		&book.Name,
		&book.Description,
		&book.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *PostgresRepository) List(
	ctx context.Context,
) ([]Book, error) {
	const query = `
		SELECT id, name, description, created_at
		FROM books
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book

	for rows.Next() {
		var book Book

		if err := rows.Scan(
			&book.ID,
			&book.Name,
			&book.Description,
			&book.CreatedAt,
		); err != nil {
			return nil, err
		}
		books = append(books, book)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return books, nil
}

func (r *PostgresRepository) Delete(
	ctx context.Context,
	id int64,
) error {
	const query = `DELETE FROM books WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
