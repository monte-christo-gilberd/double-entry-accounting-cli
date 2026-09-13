-- Fix reversal_of self-FK so deleting a book with voided entries works.
-- 05_journal_voiding.sql created it as plain REFERENCES (RESTRICT semantics):
-- wiping a book via ON DELETE CASCADE can then fail on FK ordering.
ALTER TABLE journal_entries DROP CONSTRAINT IF EXISTS journal_entries_reversal_of_fkey;

ALTER TABLE journal_entries
ADD CONSTRAINT journal_entries_reversal_of_fkey
FOREIGN KEY (reversal_of)
REFERENCES journal_entries(id)
ON DELETE CASCADE
DEFERRABLE INITIALLY DEFERRED;
