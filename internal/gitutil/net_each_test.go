package gitutil_test

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestNetEachReadsAListingPastNetsBound: a listing longer than the megabyte
// Net returns is handed over line by line, every line, and a line longer than
// the line bound is handed over cut to it.
func TestNetEachReadsAListingPastNetsBound(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("seed")
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	r.Git("remote", "add", "origin", bare)
	r.Git("push", "-q", "origin", "HEAD:refs/heads/main")
	tip := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
	var b strings.Builder
	const n = 12000
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "create refs/heads/many/%05d-%s %s\n", i, strings.Repeat("m", 40), tip)
	}
	cmd := exec.Command("git", "-C", bare, "update-ref", "--stdin")
	cmd.Env, cmd.Stdin = gittest.Env(t), strings.NewReader(b.String())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("update-ref: %v\n%s", err, out)
	}
	if _, err := gitutil.Net(r.Root(), "ls-remote", "origin", "refs/heads/many/*"); err == nil {
		t.Fatal("the listing fits Net's bound, so it proves nothing")
	}
	lines := 0
	err := gitutil.NetEach(r.Root(), 60, func(line string) {
		lines++
		if len(line) > 60 || !strings.HasPrefix(line, tip) {
			t.Fatalf("line %q: want it cut to 60 bytes", line)
		}
	}, "ls-remote", "origin", "refs/heads/many/*")
	if err != nil || lines != n {
		t.Fatalf("NetEach = %v after %d lines; want all %d", err, lines, n)
	}
}
