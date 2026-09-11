# double-entry-accounting-cli

A standalone CLI application for offline double-entry accounting, built with Go and PostgreSQL. Manage multiple books, chart of accounts, and balanced journal transactions with posting/voiding and real-time account balances.

## Features

- Book Management — Create, select, and delete accounting books (cascade deletes handled)
- Chart of Accounts — Add, edit, list, and delete accounts with types `ASSET`, `LIABILITY`, `EQUITY`, `REVENUE`, `EXPENSE` and unique code per book
- Double-Entry Transactions — Perform validated transactions (≥2 lines, debit = credit, account belongs to book)
- Transaction Log — View all logs or last N logs ordered by entry date / created time
- Account Balances — View total value per account calculated from `POSTED` journals (`SUM(debit)-SUM(credit)`)
- Posting & Voiding — Draft → Posted flow and void via reversal entry (balances restored)
- Offline CLI — Interactive prompt menus with input validation

## Tech Stack

- Go 1.23
- PostgreSQL 14+ (tested with `pgx/v5`)
- github.com/spf13/cobra (CLI)
- github.com/jackc/pgx/v5 (PostgreSQL driver)
- github.com/joho/godotenv (env loading)

## Requirements

Before running this project, make sure you have:

- Go 1.23+
- PostgreSQL 14+ (18+ recommended)
- Git

## Installation

Clone the repository:

```bash
git clone https://github.com/monte-christo-gilberd/double-entry-accounting-cli.git
cd double-entry-accounting-cli
```

Configure environment:

```bash
cp .env.example .env
# edit .env — set DATABASE_HOST, DATABASE_PORT, DATABASE_USER, DATABASE_PASSWORD, DATABASE_NAME, DATABASE_SSLMODE
```

Create database and run migrations in order:

```bash
createdb double_entry_accounting
psql $DATABASE_URL -f migrations/01_initial_schema.sql
psql $DATABASE_URL -f migrations/02_account_constraints.sql
psql $DATABASE_URL -f migrations/03_journal_integrity.sql
psql $DATABASE_URL -f migrations/04_journal_status.sql
psql $DATABASE_URL -f migrations/05_journal_voiding.sql
psql $DATABASE_URL -f migrations/06_fix_book_cascade.sql
```

Run the app:

```bash
go mod tidy
go run ./cmd/accounting
# or build
go build -o bin/accounting ./cmd/accounting && ./bin/accounting
```

## Project Structure

```
double-entry-accounting-cli
├─ cmd
│  └─ accounting
│     └─ main.go
├─ internal
│  ├─ account
│  │  ├─ account.go
│  │  ├─ postgres_repository.go
│  │  ├─ repository.go
│  │  ├─ service.go
│  │  └─ service_test.go
│  ├─ book
│  │  ├─ book.go
│  │  ├─ postgres_repository.go
│  │  └─ repository.go
│  ├─ cli
│  │  ├─ cli.go
│  │  ├─ book_flow.go
│  │  ├─ account_flow.go
│  │  └─ journal_flow.go
│  ├─ config
│  │  └─ config.go
│  ├─ database
│  │  └─ postgres.go
│  ├─ journal
│  │  ├─ journal.go
│  │  ├─ postgres_repository.go
│  │  ├─ repository.go
│  │  ├─ service.go
│  │  └─ service_test.go
│  └─ prompt
│     └─ prompt.go
├─ migrations
│  ├─ 01_initial_schema.sql
│  ├─ 02_account_constraints.sql
│  ├─ 03_journal_integrity.sql
│  ├─ 04_journal_status.sql
│  ├─ 05_journal_voiding.sql
│  └─ 06_fix_book_cascade.sql
├─ go.mod
└─ README.md
```

## Usage

1. Main Menu: Create New Book → Select Existing Book → Delete Book
2. Inside Book: View Transaction Log → View Account Balances → Perform Transaction (enter `0` as Account ID to finish, `d`/`c` for debit/credit) → Cancel Transaction (voids posted entry via reversal) → Add/Edit/Delete Account

## Testing

```bash
go vet ./...
go test ./...
```
