package gitutil_test

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// headEnding gives the bare repository at bare branches at sha named
// refs/heads/HEAD/<n>/<padding>/HEAD, which `git ls-remote --symref origin
// HEAD` lists beside HEAD itself, until the listing runs past want bytes.
func headEnding(t *testing.T, bare, sha string, want int) {
	t.Helper()
	pad := strings.Repeat("x", 200)
	var in strings.Builder
	for n, listed := 0, 0; listed <= want; n++ {
		ref := fmt.Sprintf("refs/heads/HEAD/%d/%s/HEAD", n, pad)
		fmt.Fprintf(&in, "create %s %s\n", ref, sha)
		listed += len(sha) + 1 + len(ref) + 1
	}
	cmd := exec.Command("git", "-C", bare, "update-ref", "--stdin")
	cmd.Env, cmd.Stdin = gittest.Env(t), strings.NewReader(in.String())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("update-ref: %v\n%s", err, out)
	}
}

// TestTheDefaultBranchIsReadFromHEADsOwnLines: an origin holding more
// branches whose names end in /HEAD than a networked read keeps whole still
// names its default branch and that branch's tip, both through the
// line-wise default runner and through the whole-answer runners a caller
// passes (Net, NetContext), which refuse the listing as past their bound and
// have it read again line by line.
func TestTheDefaultBranchIsReadFromHEADsOwnLines(t *testing.T) {
	r, bare := remoteWorld(t, "develop")
	want, err := gitutil.Run(bare, "rev-parse", "refs/heads/develop")
	if err != nil {
		t.Fatal(err)
	}
	headEnding(t, bare, want, gitutil.NetOutputCap)
	if _, err := gitutil.Net(r.Root(), "ls-remote", "--symref", "origin", "HEAD"); err == nil {
		t.Fatal("origin's listing of HEAD fits Net's bound, so it proves nothing")
	}
	for name, git := range map[string]func(string, ...string) (string, error){"the default runner": nil, "Net": gitutil.Net} {
		if got, err := gitutil.RemoteHeadBranch(git, r.Root(), "origin"); err != nil || got != "develop" {
			t.Errorf("RemoteHeadBranch through %s = %q, %v; want develop", name, got, err)
		}
	}
	for name, git := range map[string]func(context.Context, string, ...string) (string, error){"the default runner": nil, "NetContext": gitutil.NetContext} {
		if branch, tip, err := gitutil.RemoteHeadTip(context.Background(), git, r.Root(), "origin"); err != nil || branch != "develop" || tip != want {
			t.Errorf("RemoteHeadTip through %s = %q %q, %v; want develop at %s", name, branch, tip, err, want)
		}
	}
}
