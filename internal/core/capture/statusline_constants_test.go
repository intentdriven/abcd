package capture

import (
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// statusline counts open issues by reading recordid.IssuesRelDir and
// issueschema.StatusDirs[0], not this package's constants, because importing
// capture from statusline is a cycle. This test pins the two spellings together,
// so the folder the status line counts is the folder capture writes an open
// record into. The other status folders are pinned beside it.
func TestStatuslineReadsTheFolderCaptureWrites(t *testing.T) {
	if LedgerRelPath != recordid.IssuesRelDir {
		t.Errorf("LedgerRelPath %q != recordid.IssuesRelDir %q", LedgerRelPath, recordid.IssuesRelDir)
	}
	want := []State{StateOpen, StateResolved, StateWontfix}
	if len(issueschema.StatusDirs) != len(want) {
		t.Fatalf("issueschema.StatusDirs = %q, want the %d states %q in order", issueschema.StatusDirs, len(want), want)
	}
	for i, s := range want {
		if string(s) != issueschema.StatusDirs[i] {
			t.Errorf("issueschema.StatusDirs[%d] = %q, want %q", i, issueschema.StatusDirs[i], s)
		}
	}
}
