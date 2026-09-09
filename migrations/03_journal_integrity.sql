ALTER TABLE accounts
ADD CONSTRAINT accounts_id_book_id_unique
UNIQUE (id, book_id);

ALTER TABLE journal_entries
ADD CONSTRAINT journal_entries_id_book_id_unique
UNIQUE (id, book_id);

ALTER TABLE journal_lines
ADD COLUMN book_id BIGINT NOT NULL;

ALTER TABLE journal_lines
ADD CONSTRAINT journal_lines_journal_entry_fk
FOREIGN KEY (journal_entry_id, book_id)
REFERENCES journal_entries (id, book_id)
ON DELETE CASCADE;

ALTER TABLE journal_lines
ADD CONSTRAINT journal_lines_account_fk
FOREIGN KEY (account_id, book_id)
REFERENCES accounts (id, book_id);