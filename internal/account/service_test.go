package account

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

type mockRepository struct {
	createCalled bool
	createdAcct  *Account
	createErr    error

	getByIDAndBookIDCalled bool
	getByIDAndBookIDResult *Account
	getByIDAndBookIDErr    error

	getByCodeAndBookIDCalled bool
	getByCodeAndBookIDResult *Account
	getByCodeAndBookIDErr    error

	updateCalled bool
	updatedAcct  *Account
	updateErr    error

	deleteCalled bool
	deletedID    int64
	deleteErr    error
}

func (m *mockRepository) Create(ctx context.Context, account *Account) error {
	m.createCalled = true
	m.createdAcct = account
	if m.createErr != nil {
		return m.createErr
	}
	account.ID = 1
	return nil
}

func (m *mockRepository) GetByID(ctx context.Context, id int64) (*Account, error) {
	return nil, sql.ErrNoRows
}

func (m *mockRepository) GetByIDAndBookID(
	ctx context.Context,
	id int64,
	bookID int64,
) (*Account, error) {
	m.getByIDAndBookIDCalled = true
	if m.getByIDAndBookIDErr != nil {
		return nil, m.getByIDAndBookIDErr
	}
	if m.getByIDAndBookIDResult == nil {
		return nil, sql.ErrNoRows
	}
	return m.getByIDAndBookIDResult, nil
}

func (m *mockRepository) GetByCodeAndBookID(
	ctx context.Context,
	code string,
	bookID int64,
) (*Account, error) {
	m.getByCodeAndBookIDCalled = true
	if m.getByCodeAndBookIDErr != nil {
		return nil, m.getByCodeAndBookIDErr
	}
	if m.getByCodeAndBookIDResult == nil {
		return nil, sql.ErrNoRows
	}
	return m.getByCodeAndBookIDResult, nil
}

func (m *mockRepository) ListByBookID(ctx context.Context, bookID int64) ([]Account, error) {
	return nil, nil
}

func (m *mockRepository) Update(ctx context.Context, account *Account, bookID int64) error {
	m.updateCalled = true
	m.updatedAcct = account
	return m.updateErr
}

func (m *mockRepository) Delete(ctx context.Context, id int64, bookID int64) error {
	m.deleteCalled = true
	m.deletedID = id
	return m.deleteErr
}

func TestCreate(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	account := &Account{
		BookID:      1,
		Code:        "1000",
		Name:        "Cash",
		AccountType: "ASSET",
	}

	err := service.Create(context.Background(), account)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repository.createCalled {
		t.Fatal("expected Create to be called")
	}

	if account.ID != 1 {
		t.Fatalf("expected account ID to be set by repository, got %d", account.ID)
	}
}

func TestCreateRejectsNilAccount(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	err := service.Create(context.Background(), nil)
	if !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("expected ErrInvalidAccount, got %v", err)
	}
	if repository.createCalled {
		t.Fatal("repository Create should not be called")
	}
}

func TestCreateRejectsInvalidBookID(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	account := &Account{
		BookID:      0,
		Code:        "1000",
		Name:        "Cash",
		AccountType: "ASSET",
	}

	err := service.Create(context.Background(), account)
	if !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("expected ErrInvalidAccount, got %v", err)
	}
	if repository.createCalled {
		t.Fatal("repository Create should not be called")
	}
}

func TestCreateRejectsEmptyCode(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	account := &Account{
		BookID:      1,
		Code:        "",
		Name:        "Cash",
		AccountType: "ASSET",
	}

	err := service.Create(context.Background(), account)
	if !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("expected ErrInvalidAccount, got %v", err)
	}
	if repository.createCalled {
		t.Fatal("repository Create should not be called")
	}
}

func TestCreateRejectsEmptyName(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	account := &Account{
		BookID:      1,
		Code:        "1000",
		Name:        "",
		AccountType: "ASSET",
	}

	err := service.Create(context.Background(), account)
	if !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("expected ErrInvalidAccount, got %v", err)
	}
	if repository.createCalled {
		t.Fatal("repository Create should not be called")
	}
}

func TestCreateRejectsInvalidAccountType(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	account := &Account{
		BookID:      1,
		Code:        "1000",
		Name:        "Cash",
		AccountType: "NOT_A_REAL_TYPE",
	}

	err := service.Create(context.Background(), account)
	if !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("expected ErrInvalidAccount, got %v", err)
	}
	if repository.createCalled {
		t.Fatal("repository Create should not be called")
	}
}

func TestCreateAcceptsAllValidAccountTypes(t *testing.T) {
	validTypes := []string{"ASSET", "LIABILITY", "EQUITY", "REVENUE", "EXPENSE"}

	for _, accType := range validTypes {
		repository := &mockRepository{}
		service := NewService(repository)

		account := &Account{
			BookID:      1,
			Code:        "1000",
			Name:        "Test Account",
			AccountType: accType,
		}

		if err := service.Create(context.Background(), account); err != nil {
			t.Errorf("account type %q: expected no error, got %v", accType, err)
		}
		if !repository.createCalled {
			t.Errorf("account type %q: expected Create to be called", accType)
		}
	}
}

func TestValidateBelongsToBook(t *testing.T) {
	repository := &mockRepository{
		getByIDAndBookIDResult: &Account{ID: 1, BookID: 1},
	}
	service := NewService(repository)

	err := service.ValidateBelongsToBook(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repository.getByIDAndBookIDCalled {
		t.Fatal("expected GetByIDAndBookID to be called")
	}
}

func TestValidateBelongsToBookRejectsInvalidAccountID(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	err := service.ValidateBelongsToBook(context.Background(), 0, 1)
	if !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("expected ErrInvalidAccount, got %v", err)
	}
	if repository.getByIDAndBookIDCalled {
		t.Fatal("repository should not be queried for an invalid account ID")
	}
}

func TestValidateBelongsToBookRejectsInvalidBookID(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	err := service.ValidateBelongsToBook(context.Background(), 1, 0)
	if !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("expected ErrInvalidAccount, got %v", err)
	}
	if repository.getByIDAndBookIDCalled {
		t.Fatal("repository should not be queried for an invalid book ID")
	}
}

func TestValidateBelongsToBookNotFound(t *testing.T) {
	repository := &mockRepository{
		getByIDAndBookIDErr: errors.New("sql: no rows in result set"),
	}
	service := NewService(repository)

	err := service.ValidateBelongsToBook(context.Background(), 1, 1)
	if !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("expected ErrAccountNotFound, got %v", err)
	}
}

func TestUpdate(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	account := &Account{
		ID:          1,
		BookID:      1,
		Code:        "1000",
		Name:        "Cash",
		AccountType: "ASSET",
	}

	if err := service.Update(context.Background(), account, account.BookID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repository.updateCalled {
		t.Fatal("expected Update to be called")
	}
}

func TestUpdateRejectsInvalidID(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	account := &Account{
		ID:          0,
		BookID:      1,
		Code:        "1000",
		Name:        "Cash",
		AccountType: "ASSET",
	}

	err := service.Update(context.Background(), account, account.BookID)
	if !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("expected ErrInvalidAccount, got %v", err)
	}
	if repository.updateCalled {
		t.Fatal("repository Update should not be called")
	}
}

func TestDelete(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	if err := service.Delete(context.Background(), 1, 1); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repository.deleteCalled || repository.deletedID != 1 {
		t.Fatalf("expected Delete to be called with ID 1, got called=%v id=%d", repository.deleteCalled, repository.deletedID)
	}
}

func TestDeleteRejectsInvalidID(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	err := service.Delete(context.Background(), 0, 0)
	if !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("expected ErrInvalidAccount, got %v", err)
	}
	if repository.deleteCalled {
		t.Fatal("repository Delete should not be called")
	}
}

func TestUpdateRejectsBookMismatch(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	account := &Account{
		ID:          1,
		BookID:      1,
		Code:        "1000",
		Name:        "Cash",
		AccountType: "ASSET",
	}

	err := service.Update(context.Background(), account, 2)
	if !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("expected ErrInvalidAccount, got %v", err)
	}
	if repository.updateCalled {
		t.Fatal("repository Update should not be called")
	}
}

func TestCreateMapsDuplicateCode(t *testing.T) {
	repository := &mockRepository{createErr: &pgconn.PgError{Code: "23505"}}
	service := NewService(repository)

	account := &Account{BookID: 1, Code: "1000", Name: "Cash", AccountType: "ASSET"}
	err := service.Create(context.Background(), account)
	if !errors.Is(err, ErrAccountExists) {
		t.Fatalf("expected ErrAccountExists, got %v", err)
	}
}

func TestUpdateMapsDuplicateCode(t *testing.T) {
	repository := &mockRepository{updateErr: &pgconn.PgError{Code: "23505"}}
	service := NewService(repository)

	account := &Account{ID: 1, BookID: 1, Code: "1000", Name: "Cash", AccountType: "ASSET"}
	err := service.Update(context.Background(), account, 1)
	if !errors.Is(err, ErrAccountExists) {
		t.Fatalf("expected ErrAccountExists, got %v", err)
	}
}

func TestDeleteMapsInUse(t *testing.T) {
	repository := &mockRepository{deleteErr: &pgconn.PgError{Code: "23503"}}
	service := NewService(repository)

	err := service.Delete(context.Background(), 1, 1)
	if !errors.Is(err, ErrAccountInUse) {
		t.Fatalf("expected ErrAccountInUse, got %v", err)
	}
}

func TestGetByIDAndBookIDNotFoundIsClean(t *testing.T) {
	repository := &mockRepository{getByIDAndBookIDErr: sql.ErrNoRows}
	service := NewService(repository)

	_, err := service.GetByIDAndBookID(context.Background(), 1, 1)
	if !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("expected ErrAccountNotFound, got %v", err)
	}
	if strings.Contains(err.Error(), "sql:") {
		t.Fatalf("leaked driver text in user-facing error: %v", err)
	}
}

func TestValidateBelongsToBookMissingIsNotFound(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	err := service.ValidateBelongsToBook(context.Background(), 1, 1)
	if !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("expected ErrAccountNotFound, got %v", err)
	}
}

func TestGetByCodeAndBookID(t *testing.T) {
	repository := &mockRepository{
		getByCodeAndBookIDResult: &Account{ID: 7, BookID: 1, Code: "1000", Name: "Cash", AccountType: "ASSET"},
	}
	service := NewService(repository)

	acc, err := service.GetByCodeAndBookID(context.Background(), "1000", 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if acc.ID != 7 || !repository.getByCodeAndBookIDCalled {
		t.Fatalf("expected account 7 via repository, got %+v", acc)
	}
}

func TestGetByCodeAndBookIDRejectsInvalid(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	if _, err := service.GetByCodeAndBookID(context.Background(), "", 1); !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("expected ErrInvalidAccount, got %v", err)
	}
	if _, err := service.GetByCodeAndBookID(context.Background(), "1000", 0); !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("expected ErrInvalidAccount, got %v", err)
	}
	if repository.getByCodeAndBookIDCalled {
		t.Fatal("repository should not be queried")
	}
}

func TestGetByCodeAndBookIDNotFoundIsClean(t *testing.T) {
	repository := &mockRepository{getByCodeAndBookIDErr: sql.ErrNoRows}
	service := NewService(repository)

	_, err := service.GetByCodeAndBookID(context.Background(), "9999", 1)
	if !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("expected ErrAccountNotFound, got %v", err)
	}
	if strings.Contains(err.Error(), "sql:") {
		t.Fatalf("leaked driver text in user-facing error: %v", err)
	}
}

func TestCreateRejectsReservedCodes(t *testing.T) {
	for _, code := range []string{"0", "q", "Q", "cancel", "Cancel"} {
		repository := &mockRepository{}
		service := NewService(repository)

		account := &Account{BookID: 1, Code: code, Name: "X", AccountType: "ASSET"}
		if err := service.Create(context.Background(), account); !errors.Is(err, ErrInvalidAccount) {
			t.Fatalf("code %q: expected ErrInvalidAccount, got %v", code, err)
		}
		if repository.createCalled {
			t.Fatalf("code %q: repository Create should not be called", code)
		}
	}
}

func TestDeleteRejectsInvalidBookID(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository)

	err := service.Delete(context.Background(), 1, 0)
	if !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("expected ErrInvalidAccount, got %v", err)
	}
	if repository.deleteCalled {
		t.Fatal("repository Delete should not be called")
	}
}
