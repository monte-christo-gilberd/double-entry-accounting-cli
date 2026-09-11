ALTER TABLE journal_lines DROP CONSTRAINT IF EXISTS journal_lines_account_id_fkey;
ALTER TABLE journal_lines DROP CONSTRAINT IF EXISTS journal_lines_account_fk;

-- single-column FK (from 01_initial_schema.sql) — keep RESTRICT semantics but deferrable
ALTER TABLE journal_lines
ADD CONSTRAINT journal_lines_account_id_fkey
FOREIGN KEY (account_id)
REFERENCES accounts(id)
DEFERRABLE INITIALLY DEFERRED;

-- composite FK (from 03_journal_integrity.sql) — deferrable so book CASCADE passes
ALTER TABLE journal_lines
ADD CONSTRAINT journal_lines_account_fk
FOREIGN KEY (account_id, book_id)
REFERENCES accounts (id, book_id)
DEFERRABLE INITIALLY DEFERRED;
