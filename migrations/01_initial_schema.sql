CREATE TABLE books (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    book_id BIGINT NOT NULL
        REFERENCES books(id)
        ON DELETE CASCADE,
    code VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    account_type VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (book_id, code)
);

CREATE TABLE journal_entries (
    id BIGSERIAL PRIMARY KEY,
    book_id BIGINT NOT NULL
        REFERENCES books(id)
        ON DELETE CASCADE,
    entry_date DATE NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE journal_lines (
    id BIGSERIAL PRIMARY KEY,
    journal_entry_id BIGINT NOT NULL 
        REFERENCES journal_entries(id) 
        ON DELETE CASCADE,
    account_id BIGINT NOT NULL 
        REFERENCES accounts(id),
    debit NUMERIC(19,4) NOT NULL DEFAULT 0,
    credit NUMERIC(19, 4) NOT NULL DEFAULT 0,

    CHECK (debit >= 0),
    CHECK (credit >= 0),
    CHECK (
        (debit > 0 AND credit = 0)
        OR
        (debit = 0 AND credit > 0)
    )
);

