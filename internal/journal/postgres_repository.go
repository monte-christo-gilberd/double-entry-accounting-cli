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

func (r *PostgresRepository) GetByID(
	ctx context.Context,
	id int64,
) (*JournalEntry, error) {
	const entryQuery = `
		SELECT id, book_id, entry_date, description, status, reversal_of, created_at
		FROM journal_entries
		WHERE id = $1
	`

	var entry JournalEntry
	var reversalOf sql.NullInt64
	err := r.db.QueryRowContext(ctx, entryQuery, id).Scan(
		&entry.ID,
		&entry.BookID,
		&entry.EntryDate,
		&entry.Description,
		&entry.Status,
		&reversalOf,
		&entry.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if reversalOf.Valid {
		entry.ReversalOf = &reversalOf.Int64
	}

	const lineQuery = `
		SELECT id, journal_entry_id, book_id, account_id, debit, credit
		FROM journal_lines
		WHERE journal_entry_id = $1
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, lineQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var line JournalLine
		if err := rows.Scan(
			&line.ID,
			&line.JournalEntryID,
			&line.BookID,
			&line.AccountID,
			&line.Debit,
			&line.Credit,
		); err != nil {
			return nil, err
		}
		entry.Lines = append(entry.Lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &entry, nil
}

func (r *PostgresRepository) CreateAndVoid(
	ctx context.Context,
	originalID int64,
	reversal *JournalEntry,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const entryQuery = `
		INSERT INTO journal_entries (book_id, entry_date, description, status, reversal_of)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	if err := tx.QueryRowContext(
		ctx, entryQuery,
		reversal.BookID, reversal.EntryDate, reversal.Description, reversal.Status, reversal.ReversalOf,
	).Scan(&reversal.ID, &reversal.CreatedAt); err != nil {
		return err
	}

	const lineQuery = `
		INSERT INTO journal_lines (journal_entry_id, book_id, account_id, debit, credit)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	for i := range reversal.Lines {
		line := &reversal.Lines[i]
		if err := tx.QueryRowContext(
			ctx, lineQuery, reversal.ID, reversal.BookID, line.AccountID, line.Debit, line.Credit,
		).Scan(&line.ID); err != nil {
			return err
		}
		line.JournalEntryID = reversal.ID
		line.BookID = reversal.BookID
	}

	const voidQuery = `UPDATE journal_entries SET status = $1 WHERE id = $2`
	if _, err := tx.ExecContext(ctx, voidQuery, StatusVoided, originalID); err != nil {
		return err
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
			&entry.Description,
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

func (r *PostgresRepository) UpdateStatus(
	ctx context.Context,
	id int64,
	status Status,
) error {
	const query = `
		UPDATE journal_entries
		SET status = $1
		WHERE id = $2
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		status,
		id,
	)
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

func (r *PostgresRepository) ListRecentByBookID(
	ctx context.Context,
	bookID int64,
	limit int,
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
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, bookID, limit)
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
			&entry.Description,
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

func (r *PostgresRepository) GetAccountBalances(
	ctx context.Context,
	id int64,
	status Status,
) error {
	const query = `
		UPDATE journal_entries
		SET status = $1
		WHERE id = $2
	`
	result, err := r.db.ExecContext(
		ctx,
		query,
		status,
		id,
	)
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