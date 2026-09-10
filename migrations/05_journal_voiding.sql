ALTER TABLE journal_entries
ADD COLUMN reversal_of BIGINT
    REFERENCES journal_entries(id);

CREATE INDEX idx_journal_entries_reversal_of
    ON journal_entries(reversal_of)
    WHERE reversal_of IS NOT NULL;

ALTER TABLE journal_entries
DROP CONSTRAINT journal_entries_status_check;

ALTER TABLE journal_entries
ADD CONSTRAINT journal_entries_status_check
CHECK (
    status IN ('DRAFT', 'POSTED', 'VOIDED')
);