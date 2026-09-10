package journal

import (
	"context"
	"testing"
)

type mockAccountValidator struct {
	err error
}

func (m *mockAccountValidator) ValidateBelongsToBook(
	ctx context.Context,
	accountID int64,
	bookID int64,
) error {
	return m.err
}

type mockRepository struct {
	createCalled bool
	createdEntry *JournalEntry

	entry *JournalEntry

	updateStatusCalled bool
	updatedID          int64
	updatedStatus      Status

	createAndVoidCalled bool
	voidedOriginalID    int64
	voidedReversal      *JournalEntry
}

func (m *mockRepository) CreateAndVoid(
	ctx context.Context,
	originalID int64,
	reversal *JournalEntry,
) error {
	m.createAndVoidCalled = true
	m.voidedOriginalID = originalID
	m.voidedReversal = reversal
	return nil
}

func (m *mockRepository) Create(
	ctx context.Context,
	entry *JournalEntry,
) error {
	m.createCalled = true
	m.createdEntry = entry

	return nil
}

func (m *mockRepository) GetByID(
	ctx context.Context,
	id int64,
) (*JournalEntry, error) {
	return m.entry, nil
}

func (m *mockRepository) ListByBookID(
	ctx context.Context,
	bookID int64,
) ([]JournalEntry, error) {
	return nil, nil
}

func (m *mockRepository) UpdateStatus(
	ctx context.Context,
	id int64,
	status Status,
) error {
	m.updateStatusCalled = true
	m.updatedID = id
	m.updatedStatus = status

	return nil
}

func TestCreateDraft(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})

	entry := &JournalEntry{
		BookID:      1,
		Description: "Cash sale",
		Lines: []JournalLine{
			{
				AccountID: 1,
				Debit:     10000,
			},
			{
				AccountID: 2,
				Credit:    10000,
			},
		},
	}

	err := service.CreateDraft(
		context.Background(),
		entry,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repository.createCalled {
		t.Fatal("expected Create to be called")
	}

	if entry.Status != StatusDraft {
		t.Fatalf(
			"expected status %s, got %s",
			StatusDraft,
			entry.Status,
		)
	}
}

func TestCreateDraftRejectsUnbalancedJournal(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})

	entry := &JournalEntry{
		BookID:      1,
		Description: "Invalid transaction",
		Lines: []JournalLine{
			{
				AccountID: 1,
				Debit:     10000,
			},
			{
				AccountID: 2,
				Credit:    9000,
			},
		},
	}

	err := service.CreateDraft(
		context.Background(),
		entry,
	)

	if err == nil {
		t.Fatal("expected validation error")
	}

	if repository.createCalled {
		t.Fatal("repository Create should not be called")
	}
}

func TestPost(t *testing.T) {
	entry := &JournalEntry{
		ID:          1,
		BookID:      1,
		Description: "Cash sale",
		Status:      StatusDraft,
		Lines: []JournalLine{
			{
				AccountID: 1,
				Debit:     10000,
			},
			{
				AccountID: 2,
				Credit:    10000,
			},
		},
	}

	repository := &mockRepository{
		entry: entry,
	}

	service := NewService(repository, &mockAccountValidator{})

	err := service.Post(
		context.Background(),
		1,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repository.updateStatusCalled {
		t.Fatal("expected UpdateStatus to be called")
	}

	if repository.updatedID != 1 {
		t.Fatalf(
			"expected ID 1, got %d",
			repository.updatedID,
		)
	}

	if repository.updatedStatus != StatusPosted {
		t.Fatalf(
			"expected status %s, got %s",
			StatusPosted,
			repository.updatedStatus,
		)
	}
}

func TestPostRejectsAlreadyPostedJournal(t *testing.T) {
	entry := &JournalEntry{
		ID:          1,
		BookID:      1,
		Description: "Cash sale",
		Status:      StatusPosted,
		Lines: []JournalLine{
			{
				AccountID: 1,
				Debit:     10000,
			},
			{
				AccountID: 2,
				Credit:    10000,
			},
		},
	}

	repository := &mockRepository{
		entry: entry,
	}

	service := NewService(repository, &mockAccountValidator{})

	err := service.Post(
		context.Background(),
		1,
	)

	if err != ErrJournalPosted {
		t.Fatalf(
			"expected ErrJournalPosted, got %v",
			err,
		)
	}

	if repository.updateStatusCalled {
		t.Fatal("UpdateStatus should not be called")
	}
}

func TestVoidReversesAPostedEntry(t *testing.T) {
	entry := &JournalEntry{
		ID:          1,
		BookID:      1,
		Description: "Cash sale",
		Status:      StatusPosted,
		Lines: []JournalLine{
			{AccountID: 1, Debit: 10000},
			{AccountID: 2, Credit: 10000},
		},
	}

	repository := &mockRepository{entry: entry}
	service := NewService(repository, &mockAccountValidator{})

	err := service.Void(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repository.createAndVoidCalled {
		t.Fatal("expected CreateAndVoid to be called")
	}
	if repository.voidedOriginalID != 1 {
		t.Fatalf("expected original ID 1, got %d", repository.voidedOriginalID)
	}

	rev := repository.voidedReversal
	if rev == nil || len(rev.Lines) != 2 {
		t.Fatal("expected reversal entry with 2 lines")
	}
	// debit/credit must be swapped so balances net back to zero
	if rev.Lines[0].Credit != 10000 || rev.Lines[1].Debit != 10000 {
		t.Fatalf("expected debit/credit swapped in reversal, got %+v", rev.Lines)
	}
}

func TestVoidRejectsAlreadyVoidedEntry(t *testing.T) {
	entry := &JournalEntry{
		ID:     1,
		BookID: 1,
		Status: StatusVoided,
	}
	repository := &mockRepository{entry: entry}
	service := NewService(repository, &mockAccountValidator{})

	err := service.Void(context.Background(), 1)
	if err != ErrJournalVoided {
		t.Fatalf("expected ErrJournalVoided, got %v", err)
	}
	if repository.createAndVoidCalled {
		t.Fatal("CreateAndVoid should not be called")
	}
}

func TestVoidRejectsDraftEntry(t *testing.T) {
	entry := &JournalEntry{
		ID:     1,
		BookID: 1,
		Status: StatusDraft,
	}
	repository := &mockRepository{entry: entry}
	service := NewService(repository, &mockAccountValidator{})

	err := service.Void(context.Background(), 1)
	if err != ErrCannotVoidDraft {
		t.Fatalf("expected ErrCannotVoidDraft, got %v", err)
	}
	if repository.createAndVoidCalled {
		t.Fatal("CreateAndVoid should not be called")
	}
}
