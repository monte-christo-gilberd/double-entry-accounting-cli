@echo off
if not exist .env copy .env.example .env
for /f "usebackq tokens=1,* delims==" %%a in (".env") do if not "%%a"=="" if not "%%a:~0,1%"=="#" set %%a=%%b
createdb -h %DATABASE_HOST% -U %DATABASE_USER% %DATABASE_NAME% 2>nul || echo DB exists, continuing...
go mod tidy
go run ./cmd/accounting migrate
go run ./cmd/accounting
pause