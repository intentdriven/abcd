//go:build unix

package gitutil_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// homeFixture lays out a git working tree with a plain directory inside it,
// a plain directory outside every working tree, and a symlink in the tree
// pointing at that outside directory, all on their physical paths.
type homeFixture struct {
	repo, inside, outside, linkOut string
}

func newHomeFixture(t *testing.T) homeFixture {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := homeFixture{
		repo:    filepath.Join(base, "repo"),
		outside: filepath.Join(base, "outside"),
	}
	f.inside = filepath.Join(f.repo, "planted-home")
	f.linkOut = filepath.Join(f.repo, "link-out")
	for _, d := range []string{filepath.Join(f.repo, ".git"), f.inside, f.outside} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(f.outside, f.linkOut); err != nil {
		t.Fatal(err)
	}
	return f
}

func accountHomeIs(t *testing.T, home string, err error) {
	t.Helper()
	t.Cleanup(gitutil.SwapAccountHomeForTest(func() (string, error) { return home, err }))
}

// TestHomeInsideWorkingTree is iss-2609300012273350's one rule, shared by the
// layered machine layer and the credential store's read: a home whose
// physical path lies at or below any .git entry is refused, unless it is the
// account's own home as the account database names it, judged by file
// identity. A dotfiles-in-HOME account keeps its machine layer; a HOME planted
// into a checkout does not.
func TestHomeInsideWorkingTree(t *testing.T) {
	f := newHomeFixture(t)
	elsewhere := t.TempDir()
	linkToRepo := filepath.Join(t.TempDir(), "home-link")
	if err := os.Symlink(f.repo, linkToRepo); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, home, account string
		lookupErr           error
		refused             bool
	}{
		{"the account home that is itself a checkout", f.repo, f.repo, nil, false},
		{"the account home named by a link to it", f.repo, linkToRepo, nil, false},
		{"a planted home inside a checkout", f.inside, elsewhere, nil, true},
		{"a planted home at a checkout's root", f.repo, elsewhere, nil, true},
		{"a home spelled inside a checkout whose target is outside", f.linkOut, elsewhere, nil, false},
		{"a home spelled outside whose target is a checkout", linkToRepo, elsewhere, nil, true},
		{"a checkout home when the account lookup fails", f.repo, f.repo, errors.New("unknown userid"), true},
		{"a checkout home when the account names no home", f.repo, "", nil, true},
		{"a home outside every working tree", f.outside, elsewhere, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			accountHomeIs(t, c.account, c.lookupErr)
			got := gitutil.HomeInsideWorkingTree(c.home)
			if c.refused && got != f.repo {
				t.Errorf("HomeInsideWorkingTree(%s) = %q, want the working tree %q", c.home, got, f.repo)
			}
			if !c.refused && got != "" {
				t.Errorf("HomeInsideWorkingTree(%s) = %q, want no refusal", c.home, got)
			}
		})
	}
	if got := gitutil.HomeInsideWorkingTree(""); got != "" {
		t.Errorf("an unresolved home was judged inside %q", got)
	}
}
