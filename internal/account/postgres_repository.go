package account

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

func (r *PostgresRepository) Create(
	ctx context.Context,
	account *Account,
) error {
	const query = `
		INSERT INTO account (
			book_id,
			code,
			name,
			account_type
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(
		ctx,
		query,
		account.BookID,
		account.Code,
		account.Name,
		account.AccountType,
	).Scan(
		&account.ID,
		&account.CreatedAt,
	)
}

func (r *PostgresRepository) GetByID(
	ctx context.Context,
	id int64,
) (*Account, error) {
	const query = `
		SELECT
			id,
			book_id,
			code,
			name,
			account_type,
			created_at
		FROM accounts
		WHERE id = $1
	`

	var account Account

	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&account.ID,
		&account.BookID,
		&account.Code,
		&account.Name,
		&account.AccountType,
		&account.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *PostgresRepository) ListByBookID(
	ctx context.Context,
	bookID int64,
) ([]Account, error) {
	const query = `
		SELECT
			id,
			book_id,
			code,
			name,
			account_type,
			created_at
		FROM accounts
		WHERE book_id = $1
		ORDER BY code
	`
	rows, err := r.db.QueryContext(ctx, query, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account

	for rows.Next() {
		var account Account

		if err := rows.Scan(
			&account.ID,
			&account.BookID,
			&account.Code,
			&account.Name,
			&account.AccountType,
			&account.CreatedAt,
		); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return accounts, nil
}
