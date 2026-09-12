@echo off
setlocal
if not exist .env copy .env.example .env
for /f "usebackq eol=# tokens=1,* delims==" %%a in (".env") do if not "%%a"=="" set %%a=%%b
set PGPASSWORD=%DATABASE_PASSWORD%
createdb -h "%DATABASE_HOST%" -U "%DATABASE_USER%" "%DATABASE_NAME%" 2>nul
if %errorlevel% neq 0 echo DB exists or created, continuing...
set PGPASSWORD=
echo DB check done.
go mod tidy
go run ./cmd/accounting migrate
go run ./cmd/accounting
pause