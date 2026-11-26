package landing

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// endingIn gives the bare repository at bare branches at sha named
// refs/heads/a/<n>/<padding>/<name>, which `git ls-remote` lists for the
// pattern name beside the ref name itself, until the listing runs past want
// bytes. They are made in one update-ref transaction.
func endingIn(t *testing.T, bare, sha, name string, want int) {
	t.Helper()
	pad := strings.Repeat("x", 200)
	var in strings.Builder
	for n, listed := 0, 0; listed <= want; n++ {
		ref := fmt.Sprintf("refs/heads/a/%d/%s/%s", n, pad, name)
		fmt.Fprintf(&in, "create %s %s\n", ref, sha)
		listed += len(sha) + 1 + len(ref) + 1
	}
	cmd := exec.Command("git", "-C", bare, "update-ref", "--stdin")
	cmd.Env, cmd.Stdin = gittest.Env(t), strings.NewReader(in.String())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("update-ref: %v\n%s", err, out)
	}
}

// TestOriginsTipsAreReadFromTheirOwnLines: an origin holding more branches
// whose names end in HEAD, and in refs/heads/feat, than a networked read
// keeps whole still answers the shell guard's read of the merge mode at its
// default branch, and the landing's read of feat's tip, through the runner
// the landing passes (gitutil.Net), which refuses the listing as past its
// bound and has it read again line by line. A branch origin does not hold,
// beside one ending in its name, reads as none.
func TestOriginsTipsAreReadFromTheirOwnLines(t *testing.T) {
	repo, root := mergingRepo(t, humanConfig)
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	repo.Git("remote", "add", Remote, bare)
	repo.Git("push", "-q", Remote, "main", "main:feat")
	tip := repo.Git("rev-parse", "HEAD")
	endingIn(t, bare, tip, "HEAD", gitutil.NetOutputCap)
	endingIn(t, bare, tip, "refs/heads/feat", gitutil.NetOutputCap)
	endingIn(t, bare, tip, "refs/heads/gone", 0)
	if _, err := gitutil.Net(root, "ls-remote", Remote, "refs/heads/feat"); err == nil {
		t.Fatal("origin's listing of feat fits Net's bound, so it proves nothing")
	}
	if f, err := ReadFetchedFacilitator(root); err != nil || f.Mode != ModeHuman {
		t.Errorf("the shell guard reads %+v, %v; want human", f, err)
	}
	if got, err := RemoteTip(gitutil.Net, root, "feat"); err != nil || got != tip {
		t.Errorf("RemoteTip(feat) = %q, %v; want %s", got, err, tip)
	}
	if got, err := RemoteTip(gitutil.Net, root, "gone"); err != nil || got != "" {
		t.Errorf("RemoteTip(gone) = %q, %v; want none", got, err)
	}
}
