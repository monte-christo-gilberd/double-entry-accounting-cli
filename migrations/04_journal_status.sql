ALTER TABLE journal_entries
ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'DRAFT';

ALTER TABLE journal_entries
ADD CONSTRAINT journal_entries_status_check
CHECK (
    status IN ('DRAFT', 'POSTED')
);