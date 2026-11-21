package loop

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestTheAskOfOriginFromThePreflightCheckoutIsComparedFirst: a lane's
// preflight runs in a checkout the drive makes outside the repository's root
// (preflightcheckout.go), and the drive asks origin there for its default
// branch's tip (`git ls-remote --symref origin HEAD` through gitutil.Net) to
// hand the records-only check. With the drive's check set as its terminal
// sets it (heldBeforeGitIn), that ask runs only after the comparison of what
// the drive holds: origin's URL changed, or a URL rewrite added that points
// origin elsewhere, in the settings the lanes share, stops it naming the key,
// and with the settings as the drive read them origin answers with its tip.
func TestTheAskOfOriginFromThePreflightCheckoutIsComparedFirst(t *testing.T) {
	gone := func(t *testing.T) string { return filepath.Join(t.TempDir(), "elsewhere.git") }
	for _, c := range []struct {
		name   string
		change func(t *testing.T, repo *gittest.Repo, bare string)
		want   string
	}{
		{"unchanged", func(*testing.T, *gittest.Repo, string) {}, ""},
		{"origin's URL changed", func(t *testing.T, repo *gittest.Repo, _ string) {
			repo.Git("remote", "set-url", "origin", gone(t))
		}, "remote.origin.url"},
		{"a URL rewrite added", func(t *testing.T, repo *gittest.Repo, bare string) {
			repo.Git("config", "url."+gone(t)+".insteadOf", bare)
		}, ".insteadof"},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo, bare, tip := hookedOrigin(t)
			checkout := filepath.Join(t.TempDir(), preflightCheckoutPrefix+"x")
			repo.Git("worktree", "add", "-q", "--detach", checkout, "HEAD")
			id, err := readRemoteIdentity(repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			unset, err := gitutil.SetBeforeHookedGitAt(func(dir string, args []string) error { return heldBeforeGitIn(repo.Root(), id, dir, args) })
			if err != nil {
				t.Fatal(err)
			}
			defer unset()
			c.change(t, repo, bare)
			out, err := gitutil.Net(checkout, "ls-remote", "--symref", "origin", "HEAD")
			if c.want == "" {
				if err != nil || !strings.Contains(out, tip+"\tHEAD") {
					t.Fatalf("with the held settings standing, origin's answer = %q, %v; want its tip %s", out, err, tip)
				}
				return
			}
			var nr *gitutil.NotRunError
			if !errors.As(err, &nr) {
				t.Fatalf("the ask of origin ran after %s changed: %q, %v", c.want, out, err)
			}
			if r := mustRefusal(t, nr.Err); r.Stage != StageConfig || !strings.Contains(r.Reason, c.want) {
				t.Fatalf("the comparison does not name %s: %+v", c.want, r)
			}
		})
	}
}
