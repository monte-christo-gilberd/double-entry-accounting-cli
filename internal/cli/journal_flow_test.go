package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/monte-christo-gilberd/double-entry-accounting-cli/internal/journal"
)

func TestCancelableEntriesExcludesReversals(t *testing.T) {
	originalID := int64(1)
	entries := []journal.JournalEntry{
		{ID: 1, Status: journal.StatusPosted},
		{ID: 2, Status: journal.StatusPosted, ReversalOf: &originalID},
		{ID: 3, Status: journal.StatusDraft},
		{ID: 4, Status: journal.StatusVoided},
	}

	got := cancelableEntries(entries)
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("expected only entry #1 cancelable, got %+v", got)
	}
}

func TestCancelableEntriesEmpty(t *testing.T) {
	if got := cancelableEntries(nil); len(got) != 0 {
		t.Fatalf("expected no cancelable entries, got %+v", got)
	}
}

func TestFormatBalance(t *testing.T) {
	cases := []struct {
		accountType string
		balance     float64
		want        string
	}{
		{"ASSET", 1000, "1000.00 Dr"},
		{"ASSET", -50, "50.00 Cr"},
		{"EXPENSE", 200, "200.00 Dr"},
		{"EXPENSE", 0, "0.00"},
		{"LIABILITY", -5000, "5000.00 Cr"},
		{"LIABILITY", 2000, "2000.00 Dr"},
		{"EQUITY", -3000, "3000.00 Cr"},
		{"REVENUE", -1500, "1500.00 Cr"},
		{"REVENUE", 100, "100.00 Dr"},
		{"ASSET", 0, "0.00"},
	}
	for _, tc := range cases {
		if got := formatBalance(tc.accountType, tc.balance); got != tc.want {
			t.Errorf("formatBalance(%q, %v) = %q, want %q",
				tc.accountType, tc.balance, got, tc.want)
		}
	}
}

func TestFormatJournalEntry(t *testing.T) {
	originalID := int64(7)
	entry := journal.JournalEntry{
		ID:          9,
		EntryDate:   time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC),
		Description: "Cash sale",
		Status:      journal.StatusPosted,
		ReversalOf:  &originalID,
		Lines: []journal.JournalLine{
			{AccountID: 1, Debit: 10000},
			{AccountID: 2, Credit: 10000},
		},
	}
	names := map[int64]string{1: "1000 - Cash", 2: "4000 - Sales"}

	got := formatJournalEntry(entry, names)
	for _, want := range []string{
		"#9 [13-09-2026] Cash sale - POSTED (reversal of #7)",
		"1000 - Cash",
		"4000 - Sales",
		"Dr",
		"Cr",
		"10000.00",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("formatted entry missing %q:\n%s", want, got)
		}
	}
}

func TestFormatJournalEntryUnknownAccount(t *testing.T) {
	entry := journal.JournalEntry{
		ID:          1,
		EntryDate:   time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC),
		Description: "Mystery",
		Status:      journal.StatusPosted,
		Lines:       []journal.JournalLine{{AccountID: 99, Debit: 5}},
	}
	got := formatJournalEntry(entry, map[int64]string{})
	if !strings.Contains(got, "account #99") {
		t.Errorf("expected fallback account label, got:\n%s", got)
	}
}
