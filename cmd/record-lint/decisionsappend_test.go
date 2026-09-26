package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/gittest"
)

// daFrontDoorRepo is a repository whose work branch inserts an entry mid-file
// (a DA001 refusal) and whose clean branch appends at the tail.
func daFrontDoorRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	ledger := "# DECISIONS\n\nAppend-only, newest last.\n\n- 2026-01-01 — The first decision.\n- 2026-01-02 — The second decision.\n"
	r.Write(lint.DecisionsLedger, ledger)
	r.Commit("baseline: the ledger")
	r.Git("checkout", "-q", "-b", "clean")
	r.Write(lint.DecisionsLedger, ledger+"- 2026-01-03 — Appended at the tail.\n")
	r.Commit("append an entry")
	r.Git("checkout", "-q", "-b", "work", "main")
	r.Write(lint.DecisionsLedger, strings.Replace(ledger, "- 2026-01-02", "- 2026-01-09 — Inserted.\n- 2026-01-02", 1))
	r.Commit("insert an entry mid-file")
	return r
}

func runDA(root string, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := runDecisionsAppend(args, root, &out, &errb)
	return code, out.String(), errb.String()
}

// TestDecisionsAppendFrontDoorExitCodes pins the three polarities the gate's
// callers (make lint-decisions, CI's decisions step) branch on: 0 clean or an
// announced skip, 1 a rule violation, 2 the gate could not answer. A malformed
// invocation is the last of these — it must refuse, not default to scanning
// nothing and reporting OK (the shell suite's "a missing head ref refuses").
func TestDecisionsAppendFrontDoorExitCodes(t *testing.T) {
	r := daFrontDoorRepo(t)

	if code, out, errs := runDA(r.Root(), "main", "clean"); code != 0 || !strings.Contains(out, "checked 1 commit(s) in main..clean") || !strings.Contains(out, "OK") {
		t.Errorf("clean append: exit %d, stdout %q, stderr %q; want 0 with the range named and OK", code, out, errs)
	}
	code, out, errs := runDA(r.Root(), "main", "work")
	if code != 1 || !strings.Contains(out, "[BLOCKER DA001]") || !strings.Contains(errs, "FAILED") {
		t.Errorf("insertion: exit %d, stdout %q, stderr %q; want 1 with a DA001 finding line", code, out, errs)
	}
	if !strings.Contains(out, lint.DecisionsLedger+":") {
		t.Errorf("finding line does not name the ledger: %q", out)
	}
	for _, base := range []string{"", strings.Repeat("0", 40)} {
		if code, out, errs := runDA(r.Root(), base, "work"); code != 0 || !strings.Contains(out, "skipped") {
			t.Errorf("base %q: exit %d, stdout %q, stderr %q; want 0 with the skip announced", base, code, out, errs)
		}
	}
	if code, _, errs := runDA(r.Root(), "main"); code != 2 || !strings.Contains(errs, "usage") {
		t.Errorf("a missing head ref: exit %d, stderr %q; want 2 with the usage", code, errs)
	}
	if code, _, _ := runDA(r.Root(), "main", "work", "extra"); code != 2 {
		t.Errorf("an extra argument: exit %d; want 2", code)
	}
	if code, _, errs := runDA(r.Root(), "no-such-base", "work"); code != 2 || errs == "" {
		t.Errorf("an unresolvable base: exit %d, stderr %q; want 2 with the reason", code, errs)
	}
	if code, _, errs := runDA(t.TempDir(), "main", "work"); code != 2 || errs == "" {
		t.Errorf("outside a repository: exit %d, stderr %q; want 2", code, errs)
	}
}
