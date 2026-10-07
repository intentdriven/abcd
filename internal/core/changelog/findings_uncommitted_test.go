package changelog

import (
	"slices"
	"strings"
	"testing"
)

// TestGuardFindingsRefusesUncommittedOpenRecords pins iss-2610050927169919. The
// gate reads open/ at HEAD, so a finding captured (or regraded) in the working
// tree and not yet committed was invisible to it, and with --allow-dirty past
// the ingest's dirty-tree gate the cut stepped over it. The gate now refuses an
// open-folder record that differs from HEAD, read through launch.DirtyTreeFiles
// exactly as the derivation's uncommitted-records refusal reads the terminal
// folders, while dirt the gate does not read leaves a clean verdict alone.
func TestGuardFindingsRefusesUncommittedOpenRecords(t *testing.T) {
	t.Run("an uncommitted capture is refused, naming it", func(t *testing.T) {
		r := findingsRepo(t)
		r.issue(openDir+"iss-2-found.md", "iss-2", "critical")

		g, err := GuardFindings(r.root, "v0.1.0")
		if err != nil {
			t.Fatalf("GuardFindings: %v", err)
		}
		if g.Status != FindingGuardFailed {
			t.Fatalf("status = %q, want failed — an uncommitted critical capture passed the gate", g.Status)
		}
		if !slices.Equal(g.Uncommitted, []string{openDir + "iss-2-found.md"}) {
			t.Errorf("Uncommitted = %v, want the uncaptured record", g.Uncommitted)
		}
		for _, want := range []string{openDir + "iss-2-found.md", "commit", "--allow-dirty"} {
			if !strings.Contains(g.Reason, want) || !strings.Contains(g.UncommittedReason(), want) {
				t.Errorf("reason does not name %q: %s", want, g.Reason)
			}
		}
	})

	t.Run("a staged regrade of a committed finding is refused", func(t *testing.T) {
		r := findingsRepo(t)
		r.issue(openDir+"iss-2-found.md", "iss-2", "minor")
		r.commit("capture a minor finding")
		r.issue(openDir+"iss-2-found.md", "iss-2", "critical")
		r.git("add", openDir+"iss-2-found.md")

		g, err := GuardFindings(r.root, "v0.1.0")
		if err != nil {
			t.Fatalf("GuardFindings: %v", err)
		}
		if g.Status != FindingGuardFailed || !slices.Contains(g.Uncommitted, openDir+"iss-2-found.md") {
			t.Fatalf("status = %q, Uncommitted = %v, want the regrade refused", g.Status, g.Uncommitted)
		}
		// The committed record is minor: nothing blocks on the HEAD half.
		if len(g.Unfixed) != 0 {
			t.Errorf("Unfixed = %v, want none — the committed grade is minor", findingIDs(g.Unfixed))
		}
	})

	t.Run("dirt the gate does not read is not refused", func(t *testing.T) {
		r := findingsRepo(t)
		r.write("README.md", "an edit in progress\n")
		r.write(openDir+"README.md", "notes beside the records\n")
		r.issue(resolvedDir+"iss-3-done.md", "iss-3", "critical")

		g, err := GuardFindings(r.root, "v0.1.0")
		if err != nil {
			t.Fatalf("GuardFindings: %v", err)
		}
		if g.Status != FindingGuardPassed || len(g.Uncommitted) != 0 {
			t.Fatalf("status = %q, Uncommitted = %v, want a pass on dirt outside open/ (%s)", g.Status, g.Uncommitted, g.Reason)
		}
	})
}
