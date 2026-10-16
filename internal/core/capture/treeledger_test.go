package capture

import (
	"slices"
	"testing"
)

// TestTreeLedgerTerminalCountsOnlyResolvedAndWontfix: a commit's ledger names
// an id finished only from resolved/ or wontfix/. A record in open/, or in a
// folder that is no terminal status (a stray one, or a status added later), is
// not finished, so a drain never reports it as work a stale checkout repeats.
func TestTreeLedgerTerminalCountsOnlyResolvedAndWontfix(t *testing.T) {
	dir := ".abcd/work/issues/"
	l := &TreeLedger{Source: "origin/main at 0123abcd4567", Files: []LedgerFile{
		{Folder: "open", Path: dir + "open/iss-2609300000000001-a.md"},
		{Folder: "resolved", Path: dir + "resolved/iss-2609300000000002-b.md"},
		{Folder: "wontfix", Path: dir + "wontfix/iss-2609300000000003-c.md"},
		{Folder: "drafts", Path: dir + "drafts/iss-2609300000000004-d.md"},
		{Folder: "deferred", Path: dir + "deferred/iss-2609300000000005-e.md"},
	}}
	want := []string{"iss-2609300000000002", "iss-2609300000000003"}
	if got := l.Terminal(); !slices.Equal(got, want) {
		t.Fatalf("Terminal() = %v; want %v", got, want)
	}
}
