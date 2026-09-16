package journal

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
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

	detailedEntries []JournalEntry
	detailedErr     error

	deleteDraftCalled bool
	deletedDraftID    int64
	deleteDraftErr    error
}

func (m *mockRepository) CreateAndVoid(
	ctx context.Context,
	originalID int64,
	reversal *JournalEntry,
	bookID int64,
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
	bookID int64,
) (*JournalEntry, error) {
	if m.entry == nil {
		return nil, sql.ErrNoRows
	}
	return m.entry, nil
}

func (m *mockRepository) ListByBookID(
	ctx context.Context,
	bookID int64,
) ([]JournalEntry, error) {
	return nil, nil
}

func (m *mockRepository) DeleteDraft(
	ctx context.Context,
	id int64,
	bookID int64,
) error {
	m.deleteDraftCalled = true
	m.deletedDraftID = id
	return m.deleteDraftErr
}

func (m *mockRepository) ListDetailedByBookID(
	ctx context.Context,
	bookID int64,
) ([]JournalEntry, error) {
	return m.detailedEntries, m.detailedErr
}

func (m *mockRepository) UpdateStatus(
	ctx context.Context,
	id int64,
	from Status,
	to Status,
	bookID int64,
) error {
	m.updateStatusCalled = true
	m.updatedID = id
	m.updatedStatus = to

	return nil
}

func (m *mockRepository) ListRecentByBookID(
	ctx context.Context,
	bookID int64,
	limit int,
) ([]JournalEntry, error) {
	return nil, nil
}

func (m *mockRepository) GetAccountBalances(
	ctx context.Context,
	bookID int64,
) ([]AccountBalance, error) {
	return nil, nil
}

func TestCreateDraft(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})

	entry := &JournalEntry{
		BookID:      1,
		Description: "Cash sale",
		EntryDate:   time.Now(),
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
		EntryDate:   time.Now(),
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
		1,
	)

	if !errors.Is(err, ErrJournalPosted) {
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

	err := service.Void(context.Background(), 1, 1)
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
	// reversal must be VOIDED (audit only): the POSTED-only balance query
	// already drops the voided original, so a POSTED reversal would
	// subtract the entry a second time and double-count the void.
	if rev.Status != StatusVoided {
		t.Fatalf("expected reversal status %s, got %s", StatusVoided, rev.Status)
	}
}

func TestVoidReversalLeavesBalancesNetZero(t *testing.T) {
	// Simulates the POSTED-only balance query over: original (now VOIDED,
	// excluded), unrelated POSTED entry, reversal (VOIDED, excluded).
	// Only the unrelated entry may contribute.
	originalID := int64(1)
	entries := []JournalEntry{
		{ID: 1, Status: StatusVoided, Lines: []JournalLine{{AccountID: 1, Debit: 150}}},
		{ID: 2, Status: StatusPosted, Lines: []JournalLine{{AccountID: 1, Debit: 10}}},
	}
	entry := &JournalEntry{
		ID: 1, BookID: 1, Description: "Sale", Status: StatusPosted,
		Lines: []JournalLine{{AccountID: 1, Debit: 150}, {AccountID: 2, Credit: 150}},
	}
	repository := &mockRepository{entry: entry}
	service := NewService(repository, &mockAccountValidator{})

	if err := service.Void(context.Background(), 1, 1); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	rev := repository.voidedReversal
	entries = append(entries, JournalEntry{
		ID: 3, Status: rev.Status, ReversalOf: &originalID, Lines: rev.Lines,
	})

	var balance float64
	for _, e := range entries {
		if e.Status != StatusPosted {
			continue
		}
		for _, l := range e.Lines {
			if l.AccountID == 1 {
				balance += l.Debit - l.Credit
			}
		}
	}
	if balance != 10 {
		t.Fatalf("void double-counted: balance = %v, want 10", balance)
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

	err := service.Void(context.Background(), 1, 1)
	if !errors.Is(err, ErrJournalVoided) {
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

	err := service.Void(context.Background(), 1, 1)
	if !errors.Is(err, ErrCannotVoidDraft) {
		t.Fatalf("expected ErrCannotVoidDraft, got %v", err)
	}
	if repository.createAndVoidCalled {
		t.Fatal("CreateAndVoid should not be called")
	}
}

func TestPostRejectsForeignAccount(t *testing.T) {
	entry := &JournalEntry{
		ID:          1,
		BookID:      1,
		Description: "Cash sale",
		EntryDate:   time.Now(),
		Status:      StatusDraft,
		Lines: []JournalLine{
			{AccountID: 1, Debit: 10000},
			{AccountID: 2, Credit: 10000},
		},
	}
	repository := &mockRepository{entry: entry}
	service := NewService(repository, &mockAccountValidator{err: errors.New("no rows")})

	if err := service.Post(context.Background(), 1, 1); err == nil {
		t.Fatal("expected validation error for foreign account")
	}
	if repository.updateStatusCalled {
		t.Fatal("UpdateStatus should not be called")
	}
}

func TestCreateDraftRejectsDuplicateAccount(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})

	entry := &JournalEntry{
		BookID:      1,
		Description: "Same account both sides",
		EntryDate:   time.Now(),
		Lines: []JournalLine{
			{AccountID: 1, Debit: 100},
			{AccountID: 1, Credit: 100},
		},
	}

	if err := service.CreateDraft(context.Background(), entry); err == nil {
		t.Fatal("expected validation error for duplicate account")
	}
	if repository.createCalled {
		t.Fatal("repository Create should not be called")
	}
}

func TestTransactAcceptsFloatDust(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})

	// 0.1 + 0.2 in binary float is 0.30000000000000004; integer units
	// must still balance exactly against 0.3.
	entry := &JournalEntry{
		BookID:      1,
		Description: "Float dust",
		EntryDate:   time.Now(),
		Lines: []JournalLine{
			{AccountID: 1, Debit: 0.1},
			{AccountID: 2, Debit: 0.2},
			{AccountID: 3, Credit: 0.3},
		},
	}

	if err := service.Transact(context.Background(), entry); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repository.createCalled {
		t.Fatal("expected Create to be called")
	}
}

func TestTransactNormalizesFifthDecimal(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})

	entry := &JournalEntry{
		BookID:      1,
		Description: "Sub-precision dust",
		EntryDate:   time.Now(),
		Lines: []JournalLine{
			{AccountID: 1, Debit: 10.12345},
			{AccountID: 2, Credit: 10.12345},
		},
	}

	if err := service.Transact(context.Background(), entry); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repository.createdEntry.Lines[0].Debit != 10.1235 {
		t.Fatalf("expected debit normalized to 10.1235, got %v", repository.createdEntry.Lines[0].Debit)
	}
}

func TestGetByIDRejectsInvalidIDs(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})

	if _, err := service.GetByID(context.Background(), 0, 1); err == nil {
		t.Fatal("expected validation error for invalid journal ID")
	}
	if _, err := service.GetByID(context.Background(), 1, 0); err == nil {
		t.Fatal("expected validation error for invalid book ID")
	}
}

func TestGetByIDNotFoundIsClean(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})

	_, err := service.GetByID(context.Background(), 1, 1)
	if !errors.Is(err, ErrJournalNotFound) {
		t.Fatalf("expected ErrJournalNotFound, got %v", err)
	}
	if strings.Contains(err.Error(), "sql:") {
		t.Fatalf("leaked driver text in user-facing error: %v", err)
	}
}

func TestPostNotFoundIsClean(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})

	err := service.Post(context.Background(), 1, 1)
	if !errors.Is(err, ErrJournalNotFound) {
		t.Fatalf("expected ErrJournalNotFound, got %v", err)
	}
	if strings.Contains(err.Error(), "sql:") {
		t.Fatalf("leaked driver text in user-facing error: %v", err)
	}
}

func TestDeleteDraft(t *testing.T) {
	entry := &JournalEntry{ID: 1, BookID: 1, Status: StatusDraft}
	repository := &mockRepository{entry: entry}
	service := NewService(repository, &mockAccountValidator{})

	if err := service.DeleteDraft(context.Background(), 1, 1); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repository.deleteDraftCalled || repository.deletedDraftID != 1 {
		t.Fatal("expected DeleteDraft to be called with ID 1")
	}
}

func TestDeleteDraftRejectsPosted(t *testing.T) {
	entry := &JournalEntry{ID: 1, BookID: 1, Status: StatusPosted}
	repository := &mockRepository{entry: entry}
	service := NewService(repository, &mockAccountValidator{})

	if err := service.DeleteDraft(context.Background(), 1, 1); !errors.Is(err, ErrCannotDeletePosted) {
		t.Fatalf("expected ErrCannotDeletePosted, got %v", err)
	}
	if repository.deleteDraftCalled {
		t.Fatal("repository DeleteDraft should not be called")
	}
}

func TestDeleteDraftRejectsVoided(t *testing.T) {
	entry := &JournalEntry{ID: 1, BookID: 1, Status: StatusVoided}
	repository := &mockRepository{entry: entry}
	service := NewService(repository, &mockAccountValidator{})

	if err := service.DeleteDraft(context.Background(), 1, 1); !errors.Is(err, ErrCannotDeletePosted) {
		t.Fatalf("expected ErrCannotDeletePosted, got %v", err)
	}
	if repository.deleteDraftCalled {
		t.Fatal("repository DeleteDraft should not be called")
	}
}

func TestDeleteDraftNotFoundIsClean(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})

	err := service.DeleteDraft(context.Background(), 1, 1)
	if !errors.Is(err, ErrJournalNotFound) {
		t.Fatalf("expected ErrJournalNotFound, got %v", err)
	}
	if strings.Contains(err.Error(), "sql:") {
		t.Fatalf("leaked driver text in user-facing error: %v", err)
	}
}

func TestDeleteDraftRejectsInvalidIDs(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})

	if err := service.DeleteDraft(context.Background(), 0, 1); err == nil {
		t.Fatal("expected validation error for invalid journal ID")
	}
	if err := service.DeleteDraft(context.Background(), 1, 0); err == nil {
		t.Fatal("expected validation error for invalid book ID")
	}
	if repository.deleteDraftCalled {
		t.Fatal("repository DeleteDraft should not be called")
	}
}

func TestListDetailedByBookIDRejectsInvalidBookID(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})

	if _, err := service.ListDetailedByBookID(context.Background(), 0); err == nil {
		t.Fatal("expected validation error for invalid book ID")
	}
}

func TestPostRejectsInvalidBookID(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})

	if err := service.Post(context.Background(), 1, 0); err == nil {
		t.Fatal("expected validation error for invalid book ID")
	}
	if repository.updateStatusCalled {
		t.Fatal("UpdateStatus should not be called")
	}
}

func TestVoidRejectsInvalidBookID(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})

	if err := service.Void(context.Background(), 1, 0); err == nil {
		t.Fatal("expected validation error for invalid book ID")
	}
	if repository.createAndVoidCalled {
		t.Fatal("CreateAndVoid should not be called")
	}
}

func TestVoidRejectsReversal(t *testing.T) {
	orig := int64(1)
	entry := &JournalEntry{ID: 2, BookID: 1, Status: StatusPosted, ReversalOf: &orig}
	repository := &mockRepository{entry: entry}
	service := NewService(repository, &mockAccountValidator{})

	if err := service.Void(context.Background(), 2, 1); !errors.Is(err, ErrCannotVoidReversal) {
		t.Fatalf("expected ErrCannotVoidReversal, got %v", err)
	}
	if repository.createAndVoidCalled {
		t.Fatal("CreateAndVoid should not be called for a reversal")
	}
}

func TestCreateDraftRejectsReversalOf(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})
	orig := int64(1)
	entry := &JournalEntry{
		BookID: 1, Description: "x", EntryDate: time.Now(), ReversalOf: &orig,
		Lines: []JournalLine{{AccountID: 1, Debit: 10}, {AccountID: 2, Credit: 10}},
	}
	if err := service.CreateDraft(context.Background(), entry); !errors.Is(err, ErrInvalidJournal) {
		t.Fatalf("expected ErrInvalidJournal, got %v", err)
	}
}

func TestPostRejectsReversalEntry(t *testing.T) {
	orig := int64(1)
	entry := &JournalEntry{
		ID: 2, BookID: 1, Description: "x", EntryDate: time.Now(),
		Status: StatusDraft, ReversalOf: &orig,
		Lines: []JournalLine{{AccountID: 1, Debit: 10}, {AccountID: 2, Credit: 10}},
	}
	repository := &mockRepository{entry: entry}
	service := NewService(repository, &mockAccountValidator{})
	if err := service.Post(context.Background(), 2, 1); !errors.Is(err, ErrInvalidJournal) {
		t.Fatalf("expected ErrInvalidJournal, got %v", err)
	}
}

func TestListRecentRejectsInvalidLimit(t *testing.T) {
	repository := &mockRepository{}
	service := NewService(repository, &mockAccountValidator{})
	if _, err := service.ListRecentByBookID(context.Background(), 1, 0); !errors.Is(err, ErrInvalidJournal) {
		t.Fatalf("expected ErrInvalidJournal for limit<=0, got %v", err)
	}
}

func TestBuildReversalCopiesID(t *testing.T) {
	entry := &JournalEntry{ID: 7, BookID: 1, Description: "sale", Status: StatusPosted}
	rev := entry.BuildReversal()
	if rev.ReversalOf == nil || *rev.ReversalOf != 7 {
		t.Fatalf("expected reversal_of=7, got %+v", rev.ReversalOf)
	}
	entry.ID = 99
	if *rev.ReversalOf != 7 {
		t.Fatalf("reversal aliased original ID: got %d, want 7", *rev.ReversalOf)
	}
}