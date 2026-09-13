package cli

import (
	"testing"

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
