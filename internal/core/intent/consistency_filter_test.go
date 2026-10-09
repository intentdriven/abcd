package intent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestConsistencyDirtyCheckRunsNoRepoFilter is iss-2610090821548169: the
// consistency pass's dirty check diffs the working tree against the pinned
// commit, and over a corpus whose index stat no longer matches git re-hashes
// each document through the repository's clean filter. The check must compare
// bytes without running it, still read a byte-identical corpus clean, and still
// name a real edit.
func TestConsistencyDirtyCheckRunsNoRepoFilter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	r := consistencyRepo(t)
	root := r.Root()
	mark := filepath.Join(t.TempDir(), "filter-ran")
	script := filepath.Join(t.TempDir(), "evil-filter.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ran >> "+mark+"\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "filter.evil.clean", script)
	if err := os.WriteFile(filepath.Join(root, ".git", "info", "attributes"), []byte("* filter=evil\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale := func(age time.Duration) {
		old := time.Now().Add(-age)
		if err := os.Chtimes(filepath.Join(root, filepath.FromSlash(cxBrief)), old, old); err != nil {
			t.Fatal(err)
		}
	}
	stale(48 * time.Hour)

	// The fixture is live: a plain diff under the same environment runs it.
	cmd := exec.Command("git", "-C", root, "diff", "--name-only", "HEAD")
	cmd.Env = r.Env()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture diff: %v: %s", err, out)
	}
	if _, err := os.Stat(mark); err != nil {
		t.Fatal("fixture: a plain git diff did not run the clean filter, so this test proves nothing")
	}
	if err := os.Remove(mark); err != nil {
		t.Fatal(err)
	}
	stale(24 * time.Hour)

	em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatalf("EmitConsistency: %v", err)
	}
	if _, err := os.Stat(mark); err == nil {
		t.Fatal("the consistency dirty check ran the repository's clean filter")
	}
	if m := emitJSON(t, em); m["dirty"] != false {
		t.Fatalf("a byte-identical corpus must read clean, got %v", m)
	}

	r.Write(cxBrief, "# Review queue\n\nAn uncommitted edit.\n\n"+cxQuoteBrief+"\n")
	em, err = EmitConsistency(root, "", ConsistencyEmitOptions{})
	if err != nil {
		t.Fatalf("EmitConsistency after an edit: %v", err)
	}
	if _, err := os.Stat(mark); err == nil {
		t.Fatal("the consistency dirty check ran the repository's clean filter over an edited page")
	}
	if got := fmt.Sprint(emitJSON(t, em)["dirty_paths"]); got != "["+cxBrief+"]" {
		t.Fatalf("a real edit must still be named, got %s", got)
	}
}
