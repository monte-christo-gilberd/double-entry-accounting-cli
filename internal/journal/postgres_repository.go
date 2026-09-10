package journal

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
	entry *JournalEntry,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const entryQuery = `
		INSERT INTO journal_entries (
			book_id,
			entry_date,
			description,
			status	
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	err = tx.QueryRowContext(
		ctx,
		entryQuery,
		entry.BookID,
		entry.EntryDate,
		entry.Description,
		entry.Status,
	).Scan(
		&entry.ID,
		&entry.CreatedAt,
	)
	if err != nil {
		return err
	}

	const lineQuery = `
		INSERT INTO journal_lines (
			journal_entry_id,
			book_id,
			account_id,
			debit,
			credit
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	for i := range entry.Lines {
		line := &entry.Lines[i]

		err := tx.QueryRowContext(
			ctx,
			lineQuery,
			entry.ID,
			entry.BookID,
			line.AccountID,
			line.Debit,
			line.Credit,
		).Scan(&line.ID)

		if err != nil {
			return err
		}

		line.JournalEntryID = entry.ID
		line.BookID = entry.BookID
	}

	return tx.Commit()
}

func (r *PostgresRepository) ListByBookID(
	ctx context.Context,
	bookID int64,
) ([]JournalEntry, error) {
	const query = `
		SELECT
			id,
			book_id,
			entry_date,
			description,
			status,
			created_at
		FROM journal_entries
		WHERE book_id = $1
		ORDER BY entry_date, id
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		bookID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []JournalEntry

	for rows.Next() {
		var entry JournalEntry

		if err := rows.Scan(
			&entry.ID,
			&entry.BookID,
			&entry.EntryDate,
			*&entry.Description,
			&entry.Status,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}
