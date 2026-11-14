package ahoy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// failHooksPathRead puts a git first on PATH, for the rest of the test, that
// fails the read of core.hooksPath as a git that cannot read its settings
// does, and otherwise runs the real git.
func failHooksPathRead(t *testing.T) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	script := "#!/bin/sh\ncase \"$*\" in *--get-all\\ core.hooksPath*) echo 'fatal: bad config line 1 in file /x' >&2; exit 128 ;; esac\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestTheOutboundShimSaysItCouldNotTellWhereTheHooksPathReadFails: where git
// cannot answer the read of core.hooksPath, the opt-in writes no hook git
// runs and its note says abcd could not tell whether git runs the check,
// naming the read; it never says the check is or is not wired
// (iss-2611140432160343). Watched failing before: the note said the check is
// not wired.
func TestTheOutboundShimSaysItCouldNotTellWhereTheHooksPathReadFails(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t)
	failHooksPathRead(t)
	plan := planOutboundShim(repo.Root())
	if plan.rel == shimHookRel {
		t.Fatalf("a hook git runs is not written where the read failed: %+v", plan)
	}
	if !strings.Contains(plan.note, "could not tell whether git runs the commit-message check") ||
		!strings.Contains(plan.note, "git config --get-all core.hooksPath") ||
		strings.Contains(plan.note, "is not wired") {
		t.Fatalf("the note says abcd could not tell, naming the read: %q", plan.note)
	}
	if _, err := os.Lstat(shimPath(repo)); err == nil {
		t.Fatal("the plan wrote")
	}
}
