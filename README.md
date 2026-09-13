# double-entry-accounting-cli

A standalone CLI application for offline double-entry accounting, built with Go and PostgreSQL. Manage multiple books, chart of accounts, and balanced journal transactions with posting/voiding and real-time account balances.

## Features

- Book Management — Create, select, and delete accounting books (cascade deletes handled)
- Chart of Accounts — Add, edit, list, and delete accounts with types `ASSET`, `LIABILITY`, `EQUITY`, `REVENUE`, `EXPENSE` and unique code per book
- Double-Entry Transactions — Perform validated transactions (≥2 lines, debit = credit, one line per account, account belongs to book, amounts rounded to 4 decimals)
- Transaction Log — View all logs or last N logs with per-line debit/credit detail, account names, and reversal markers, ordered by entry date / created time
- Account Balances — View totals per account from `POSTED` journals in normal-balance form (`150.75 Dr` / `5000.00 Cr`)
- Posting & Voiding — Draft → Posted flow; voiding flips the original to `VOIDED` and records a `VOIDED` reversal (audit-only), so balances are restored exactly once
- Offline CLI — Interactive prompt menus with validation, reprompts on bad input, and clean exit on closed stdin (EOF)

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

Migrations are handled in Go by `internal/database/migrate.go` (`os.ReadDir("migrations")` sorted + `db.ExecContext` per `*.sql`, duplicate objects skipped via Postgres SQLSTATE codes, per-file transaction) — manual `psql -f` is no longer needed. Re-running the app re-applies `migrations/` idempotently, including data repairs such as `08_fix_reversal_status.sql`.

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
│  │  ├─ journal_flow.go
│  │  └─ journal_flow_test.go
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
│     ├─ prompt.go
│     └─ prompt_test.go
├─ migrations
│  ├─ 01_initial_schema.sql
│  ├─ 02_account_constraints.sql
│  ├─ 03_journal_integrity.sql
│  ├─ 04_journal_status.sql
│  ├─ 05_journal_voiding.sql
│  ├─ 06_fix_book_cascade.sql
│  ├─ 07_fix_reversal_fk.sql
│  └─ 08_fix_reversal_status.sql
├─ init.bat               # one-click Windows
├─ init.sh                # one-click Bash
├─ .env.example
├─ go.mod
└─ README.md
```

## Usage

1. Main Menu: Create New Book → Select Existing Book → Delete Book (cascade deletes everything inside, with confirmation) → Close
2. Inside Book:

```
1. View Transaction Log      # all logs or last N, with per-line Dr/Cr detail
2. View Account Balances      # normal-balance Dr/Cr per account
3. Perform Transaction        # posts immediately (0 as Account ID finishes, d/c per line)
4. Cancel Transaction         # voids a POSTED entry (reversals can't be voided)
5. Add Account
6. Edit Account               # empty input keeps the current value
7. Delete Account             # blocked while the account has transactions
8. Create Draft Transaction   # saved as DRAFT, no balance effect
9. Post Draft Transaction     # DRAFT → POSTED, balances move
0. Back to Main Menu
```

## Testing

```bash
go vet ./...
go test ./...
```
