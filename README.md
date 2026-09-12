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

- Go 1.25
- PostgreSQL 14+ (tested with `pgx/v5`)
- github.com/jackc/pgx/v5 (PostgreSQL driver)
- github.com/joho/godotenv (env loading)

## Requirements

Before running this project, make sure you have:

- Go 1.25+
- PostgreSQL 14+ (18+ recommended) running locally (`DATABASE_HOST=localhost`)
- Git
- `createdb`/`psql` in PATH (for one-click DB creation, optional)

## Installation

Clone the repository:

```bash
git clone https://github.com/monte-christo-gilberd/double-entry-accounting-cli.git
cd double-entry-accounting-cli
```

### Option A — One-click (Windows / Bash, no Docker)

**Windows:** double-click `init.bat` (no `cmd` needed, ends with `Press any key to continue . . .` via `pause`)
**Bash/macOS/Linux:** `./init.sh` (or `chmod +x init.sh && ./init.sh`, LF line endings)

This does: `copy .env.example -> .env` if missing → `createdb` via `PGPASSWORD` from `.env` (no `Password:` prompt, prints `DB check done.` / `DB exists or created`) → `go mod tidy` → `go run ./cmd/accounting migrate` (explicit) → `go run ./cmd/accounting` (auto-migrate + start). No `docker-compose.yml` needed.

### Option B — Manual

Configure environment:

```bash
cp .env.example .env
# edit .env — set DATABASE_HOST, DATABASE_PORT, DATABASE_USER, DATABASE_PASSWORD, DATABASE_NAME, DATABASE_SSLMODE
```

Create database (once):

```bash
createdb -h localhost -U postgres double_entry_accounting_db
# or: psql -h localhost -U postgres -c "CREATE DATABASE double_entry_accounting_db;"
```

Run the app (migrations run automatically):

```bash
go mod tidy
go run ./cmd/accounting migrate  # explicit: migrate and exit
go run ./cmd/accounting          # auto-migrate on every start, then launch CLI
# or build
go build -o bin/accounting ./cmd/accounting && ./bin/accounting
```

Migrations are handled in Go by `internal/database/migrate.go` (`os.ReadDir("migrations")` sorted + `db.ExecContext` per `*.sql`, `already exists` ignored, per-file transaction) — manual `psql -f` is no longer needed.

## Project Structure

```
double-entry-accounting-cli
├─ cmd
│  └─ accounting
│     └─ main.go          # + migrate/auto-migrate wiring
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
│  │  ├─ postgres.go
│  │  └─ migrate.go       # in-code migrator (no psql -f)
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
├─ init.bat               # one-click Windows
├─ init.sh                # one-click Bash
├─ .env.example
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
