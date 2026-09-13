package account

import (
	"context"
	"errors"
	"testing"
)

type mockRepository struct {
	createCalled bool
	createdAcct  *Account
	createErr    error

	getByIDAndBookIDCalled bool
	getByIDAndBookIDResult *Account
	getByIDAndBookIDErr    error

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
	return nil, nil
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
	return m.getByIDAndBookIDResult, nil
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
