#!/bin/bash
[ -f .env ] || cp .env.example .env
set -a; [ -f .env ] && source .env; set +a
PGPASSWORD="${DATABASE_PASSWORD}" createdb -h ${DATABASE_HOST:-localhost} -U ${DATABASE_USER:-postgres} ${DATABASE_NAME} 2>/dev/null || echo "DB exists or created, continuing..."
go mod tidy
go run ./cmd/accounting migrate
go run ./cmd/accounting