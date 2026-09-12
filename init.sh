#!/bin/bash
[ -f .env ] || cp .env.example .env
set -a; [ -f .env ] && source .env; set +a
createdb -h ${DATABASE_HOST:-localhost} -U ${DATABASE_USER:-postgres} ${DATABASE_NAME} 2>/dev/null || true
go mod tidy
go run ./cmd/accounting migrate
go run ./cmd/accounting