package journal

import (
	"context"
	"database/sql"
	"fmt"
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
	if entry.ReversalOf != nil {
		return fmt.Errorf("create journal entry: reversal_of must be nil on create")
	}
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
	bookID int64,
) (*JournalEntry, error) {
	const entryQuery = `
		SELECT id, book_id, entry_date, description, status, reversal_of, created_at
		FROM journal_entries
		WHERE id = $1 AND book_id = $2
	`

	var entry JournalEntry
	var reversalOf sql.NullInt64
	err := r.db.QueryRowContext(ctx, entryQuery, id, bookID).Scan(
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
		v := reversalOf.Int64
		entry.ReversalOf = &v
	}

	lines, err := loadJournalLines(ctx, r.db, id)
	if err != nil {
		return nil, err
	}
	entry.Lines = lines

	return &entry, nil
}

// lineQuerier is satisfied by *sql.DB and *sql.Tx so line loading can be
// shared between single-entry and bulk listing queries.
type lineQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func loadJournalLines(ctx context.Context, q lineQuerier, entryID int64) ([]JournalLine, error) {
	const lineQuery = `
		SELECT id, journal_entry_id, book_id, account_id, debit, credit
		FROM journal_lines
		WHERE journal_entry_id = $1
		ORDER BY id
	`

	rows, err := q.QueryContext(ctx, lineQuery, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []JournalLine
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
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func (r *PostgresRepository) CreateAndVoid(
	ctx context.Context,
	originalID int64,
	reversal *JournalEntry,
	bookID int64,
) error {
	if reversal == nil {
		return fmt.Errorf("create reversal: reversal is nil")
	}
	if reversal.ReversalOf == nil || *reversal.ReversalOf != originalID {
		return fmt.Errorf("create reversal: reversal_of must reference original %d", originalID)
	}
	if reversal.BookID != bookID {
		return fmt.Errorf("create reversal: cross-book reversal rejected")
	}
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

	// Atomic guard: only a POSTED entry in this book can be voided.
	// Parentheses matter: AND binds tighter than OR, so the status
	// predicate must be grouped with the id/book predicates.
	const voidQuery = `UPDATE journal_entries SET status = $1 WHERE id = $2 AND book_id = $3 AND status = 'POSTED'`
	result, err := tx.ExecContext(ctx, voidQuery, StatusVoided, originalID, bookID)
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
			reversal_of,
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
		var reversalOf sql.NullInt64

		if err := rows.Scan(
			&entry.ID,
			&entry.BookID,
			&entry.EntryDate,
			&entry.Description,
			&entry.Status,
			&reversalOf,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		if reversalOf.Valid {
			v := reversalOf.Int64
			entry.ReversalOf = &v
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *PostgresRepository) ListDetailedByBookID(
	ctx context.Context,
	bookID int64,
) ([]JournalEntry, error) {
	// Snapshot the headers + lines in one transaction so a concurrent
	// insert cannot produce phantom/missing lines between queries.
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	const query = `
		SELECT id, book_id, entry_date, description, status, reversal_of, created_at
		FROM journal_entries
		WHERE book_id = $1
		ORDER BY entry_date, id
	`
	rows, err := tx.QueryContext(ctx, query, bookID)
	if err != nil {
		return nil, err
	}
	var entries []JournalEntry
	for rows.Next() {
		var entry JournalEntry
		var reversalOf sql.NullInt64
		if err := rows.Scan(
			&entry.ID,
			&entry.BookID,
			&entry.EntryDate,
			&entry.Description,
			&entry.Status,
			&reversalOf,
			&entry.CreatedAt,
		); err != nil {
			rows.Close()
			return nil, err
		}
		if reversalOf.Valid {
			v := reversalOf.Int64
			entry.ReversalOf = &v
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for i := range entries {
		lines, err := loadJournalLines(ctx, tx, entries[i].ID)
		if err != nil {
			return nil, err
		}
		entries[i].Lines = lines
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *PostgresRepository) DeleteDraft(
	ctx context.Context,
	id int64,
	bookID int64,
) error {
	// Guarded to DRAFT: posted/voided history is immutable and can only be
	// offset by a reversal, never removed. Lines cascade per the
	// journal_lines FK (ON DELETE CASCADE).
	const query = `
		DELETE FROM journal_entries
		WHERE id = $1 AND book_id = $2 AND status = 'DRAFT'
	`

	result, err := r.db.ExecContext(ctx, query, id, bookID)
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

func (r *PostgresRepository) UpdateStatus(
	ctx context.Context,
	id int64,
	from Status,
	to Status,
	bookID int64,
) error {
	// Atomic guard on the expected previous status closes the
	// check-then-act race between GetByID and the status update.
	const query = `
		UPDATE journal_entries
		SET status = $1
		WHERE id = $2 AND book_id = $3 AND status = $4
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		to,
		id,
		bookID,
		from,
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
			reversal_of,
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
		var reversalOf sql.NullInt64
		if err := rows.Scan(
			&entry.ID,
			&entry.BookID,
			&entry.EntryDate,
			&entry.Description,
			&entry.Status,
			&reversalOf,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		if reversalOf.Valid {
			v := reversalOf.Int64
			entry.ReversalOf = &v
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
	bookID int64,
) ([]AccountBalance, error) {
	const query = `
		SELECT jl.account_id,
		       COALESCE(SUM(jl.debit), 0) - COALESCE(SUM(jl.credit), 0) AS balance
		FROM journal_lines jl
		JOIN journal_entries je ON je.id = jl.journal_entry_id
		WHERE je.book_id = $1
		  AND jl.book_id = $1
		  AND je.status = 'POSTED'
		GROUP BY jl.account_id
	`

	rows, err := r.db.QueryContext(ctx, query, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []AccountBalance
	for rows.Next() {
		var b AccountBalance
		if err := rows.Scan(&b.AccountID, &b.Balance); err != nil {
			return nil, err
		}
		balances = append(balances, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return balances, nil
}
