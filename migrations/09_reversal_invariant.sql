-- Enforce reversal invariant at the database level: a row that references
-- another entry via reversal_of must be VOIDED (audit only, excluded from
-- balances). Without this, a POSTED reversal double-counts the void because
-- the balance query sums POSTED entries only. 08 repaired legacy rows; this
-- prevents regressions even if the app layer slips.
ALTER TABLE journal_entries DROP CONSTRAINT IF EXISTS journal_entries_reversal_status_check;
ALTER TABLE journal_entries
ADD CONSTRAINT journal_entries_reversal_status_check
CHECK (reversal_of IS NULL OR status = 'VOIDED');

-- Same-book enforcement for reversals: the original composite UNIQUE
-- (id, book_id) from 03 lets us bind reversal_of to the same book.
-- Drop the single-column self-FK (now composite) if present.
ALTER TABLE journal_entries DROP CONSTRAINT IF EXISTS journal_entries_reversal_of_fkey;
ALTER TABLE journal_entries
ADD CONSTRAINT journal_entries_reversal_of_book_fk
FOREIGN KEY (reversal_of, book_id)
REFERENCES journal_entries (id, book_id)
ON DELETE CASCADE
DEFERRABLE INITIALLY DEFERRED;
