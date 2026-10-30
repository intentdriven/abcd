package cli

import (
	"strings"
	"testing"
)

// A shipped intent whose Audit Notes carry a hand-written audit and no
// reviewer's marker is counted on its own in bare `abcd intent audit`'s header,
// so the four counts there add up to the shipped total (iss-2610241431221889).
func TestIntentAuditBareCountsAHandAudit(t *testing.T) {
	repo := owedRepo(t)
	writeRepoFile(t, repo, cliShipped+"/itd-16-s.md", cliShippedWithNotes("itd-16",
		"- ac-1: MET. The listing names every owed review."))

	out, err := runCLIErr(t, "intent", "audit")
	if err != nil {
		t.Fatalf("bare intent audit must exit 0: %v\n%s", err, out)
	}
	header, _, _ := strings.Cut(string(out), "\n")
	if !strings.Contains(header, "owed 3 · dead-lettered 1 · ingested 1 · hand-audited 1 (of 6 shipped)") {
		t.Errorf("the header does not count the hand-written audit apart:\n%s", header)
	}
}
