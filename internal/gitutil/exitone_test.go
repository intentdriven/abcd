package gitutil_test

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestRunCappedExitOneReadsExitOneAsAnAnswer: `merge-tree --write-tree` exits
// 1 on a conflict with the conflicted paths on stdout, which is an answer and
// keeps its output; exit 0 is the clean answer; any other failure is an error.
func TestRunCappedExitOneReadsExitOneAsAnAnswer(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("f.txt", "base\n")
	r.Commit("base")
	r.Git("checkout", "-q", "-b", "side")
	r.Write("f.txt", "side\n")
	r.Commit("side")
	r.Git("checkout", "-q", "main")
	r.Write("f.txt", "main\n")
	r.Commit("main")
	r.Write("g.txt", "g\n")
	r.Commit("g")

	out, one, err := gitutil.RunCappedExitOne(r.Root(), 1<<16, "merge-tree", "--write-tree", "--name-only", "--no-messages", "-z", "main", "side")
	if err != nil || !one || !strings.Contains(string(out), "\x00f.txt\x00") {
		t.Fatalf("a conflicted merge is exit 1 with its paths: %q %v %v", out, one, err)
	}
	out, one, err = gitutil.RunCappedExitOne(r.Root(), 1<<16, "merge-tree", "--write-tree", "--name-only", "--no-messages", "-z", "main", "main~1")
	if err != nil || one || len(out) == 0 {
		t.Fatalf("a clean merge is exit 0 with its tree: %q %v %v", out, one, err)
	}
	if _, _, err := gitutil.RunCappedExitOne(r.Root(), 1<<16, "merge-tree", "--write-tree", "--no-such-option", "main", "side"); err == nil {
		t.Fatal("a failure other than exit 1 is an error")
	}
	// git also exits 1 on some refusals, with nothing on stdout: the caller
	// checks the answer's shape.
	if out, one, err := gitutil.RunCappedExitOne(r.Root(), 1<<16, "merge-tree", "--write-tree", "main", "no-such-ref"); err != nil || !one || len(out) != 0 {
		t.Fatalf("an exit-1 refusal comes back as exit 1 with no answer: %q %v %v", out, one, err)
	}
	if _, _, err := gitutil.RunCappedExitOne(r.Root(), 8, "merge-tree", "--write-tree", "main", "side"); err == nil {
		t.Fatal("output past the cap is an error, never a truncated answer")
	}
}
