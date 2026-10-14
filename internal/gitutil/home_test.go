//go:build unix

package gitutil_test

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

// writePasswd writes a passwd-format account database holding lines.
func writePasswd(t *testing.T, lines ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "passwd")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestAccountHomeNeverComesFromHOME is the review finding against the first
// exemption: os/user's LookupId answers this process's own uid through
// Current, and the pure-Go Current (no cgo, off darwin: the shipped
// cross-compiled linux binaries) returns $HOME as the home when the uid has no
// /etc/passwd entry, so a HOME planted in a checkout vouched for itself. The
// account home is read from the passwd file alone; $HOME pointing at the
// checkout changes nothing, and only an entry for this uid naming the
// checkout exempts it.
func TestAccountHomeNeverComesFromHOME(t *testing.T) {
	f := newHomeFixture(t)
	uid := strconv.Itoa(os.Getuid())
	other := strconv.Itoa(os.Getuid() + 1)
	elsewhere := t.TempDir()
	cases := []struct {
		name    string
		passwd  []string // nil: no passwd file at all
		refused bool
	}{
		{"no entry for this uid", []string{"root:*:0:0:root:/root:/bin/sh"}, true},
		{"no passwd file", nil, true},
		{"this uid's entry names another home", []string{"me:*:" + uid + ":20::" + elsewhere + ":/bin/sh"}, true},
		{"another uid's entry names the checkout", []string{"them:*:" + other + ":20::" + f.repo + ":/bin/sh"}, true},
		{"this uid's entry names a relative home", []string{"me:*:" + uid + ":20::repo:/bin/sh"}, true},
		{"this uid's entry names no home", []string{"me:*:" + uid + ":20:::/bin/sh"}, true},
		{"a short line for this uid", []string{"me:*:" + uid + ":20:" + f.repo}, true},
		{"a commented entry naming the checkout", []string{"#me:*:" + uid + ":20::" + f.repo + ":/bin/sh"}, true},
		{"this uid's entry names the checkout", []string{"# local accounts", "", "me:*:" + uid + ":20:Me:" + f.repo + ":/bin/sh"}, false},
		{"the first of two entries for this uid wins", []string{"me:*:" + uid + ":20::" + elsewhere + ":/bin/sh", "me2:*:" + uid + ":20::" + f.repo + ":/bin/sh"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("HOME", f.repo)
			passwd := filepath.Join(t.TempDir(), "absent")
			if c.passwd != nil {
				passwd = writePasswd(t, c.passwd...)
			}
			gitutil.SetAccountPasswdForTest(t, passwd)
			got := gitutil.HomeInsideWorkingTree(f.repo)
			if c.refused && got != f.repo {
				t.Errorf("HomeInsideWorkingTree(%s) with HOME there = %q, want the working tree %q", f.repo, got, f.repo)
			}
			if !c.refused && got != "" {
				t.Errorf("HomeInsideWorkingTree(%s) = %q, want the exemption the passwd entry grants", f.repo, got)
			}
		})
	}
}

// TestPlantedHOMEIsRefusedByTheRealLookup drives the production lookup, not a
// substitute: with HOME pointed into a checkout the account database does not
// name, the checkout is refused. Built with -tags osusergo it covers the build
// in which os/user would fall back to $HOME, which abcd then does not call.
// os/user caches Current for the process, so against a lookup that went
// through it this test is only decisive when it is the first to call it.
func TestPlantedHOMEIsRefusedByTheRealLookup(t *testing.T) {
	f := newHomeFixture(t)
	t.Setenv("HOME", f.repo)
	if got := gitutil.HomeInsideWorkingTree(f.repo); got != f.repo {
		t.Errorf("HomeInsideWorkingTree(%s) with HOME there = %q, want the working tree %q", f.repo, got, f.repo)
	}
	if got := gitutil.HomeInsideWorkingTree(f.inside); got != f.repo {
		t.Errorf("HomeInsideWorkingTree(%s) with HOME at the checkout = %q, want the working tree %q", f.inside, got, f.repo)
	}
}

// TestAMissingHomeInsideACheckoutIsRefused: a home whose directory does not
// exist is judged on its spelling, and the exemption cannot apply because
// there is nothing to compare by identity, so it is refused.
func TestAMissingHomeInsideACheckoutIsRefused(t *testing.T) {
	f := newHomeFixture(t)
	missing := filepath.Join(f.repo, "no-such-home")
	accountHomeIs(t, missing, nil)
	if got := gitutil.HomeInsideWorkingTree(missing); got != f.repo {
		t.Errorf("HomeInsideWorkingTree(%s) = %q, want the working tree %q", missing, got, f.repo)
	}
}
