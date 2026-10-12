package gitutil

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
)

// HomeInsideWorkingTree is the one rule for whether a home ($HOME, as
// os.UserHomeDir returns it on Unix) may serve as this machine's abcd home
// (iss-2609300012273350). It returns the working tree the home lies in when
// the home's physical path is at or below a directory carrying a .git entry
// (RepoShapedRoot, the crude "could a commit carry it?" answer) and the home
// is not the account's own, and "" otherwise.
//
// An environment a repository can shape (a committed harness settings file
// whose env block sets HOME into the tree) would otherwise let a checkout
// supply the machine layer and the credential store. Any working tree is
// refused, not only the session's: the session's checkout is one place a
// planted HOME can point, a sibling checkout is another.
//
// The account's own home is exempt, because a dotfiles-in-HOME account keeps
// its home in a working tree and that home is the person's, not a checkout's.
// The account's home is the one the account database names for this uid
// (user.LookupId), never user.Current: without cgo, Current answers from $HOME
// itself when the uid is absent from /etc/passwd, which would let a planted
// HOME vouch for itself. A failed lookup grants no exemption.
//
// Containment is judged on the home's physical path (filepath.EvalSymlinks)
// and the exemption by file identity (os.SameFile), never by spelling, so a
// link spelled inside a checkout that leads outside every working tree is not
// refused, a link spelled outside that leads into one is, and a case variant
// or link to the account home is still the account home.
func HomeInsideWorkingTree(home string) string {
	if home == "" {
		return ""
	}
	phys, err := filepath.EvalSymlinks(home)
	if err != nil {
		phys = filepath.Clean(home)
	}
	tree := RepoShapedRoot(phys)
	if tree == "" || isAccountHome(phys) {
		return ""
	}
	return tree
}

// isAccountHome reports whether dir is the home the account database names
// for this process's uid, by file identity.
func isAccountHome(dir string) bool {
	acct, err := accountHome()
	if err != nil || acct == "" {
		return false
	}
	ai, err := os.Stat(acct)
	if err != nil {
		return false
	}
	di, err := os.Stat(dir)
	if err != nil {
		return false
	}
	return os.SameFile(ai, di)
}

// accountHome is the home the account database names for this process's uid.
var accountHome = func() (string, error) {
	u, err := user.LookupId(strconv.Itoa(os.Getuid()))
	if err != nil {
		return "", err
	}
	return u.HomeDir, nil
}

// SwapAccountHomeForTest substitutes the account-database home lookup
// HomeInsideWorkingTree exempts and returns the restore. It is exported
// because the tests of the rule's callers live in other packages. Tests only;
// never called in production code, and never safe to call from a parallel
// test.
func SwapAccountHomeForTest(fn func() (string, error)) (restore func()) {
	prev := accountHome
	accountHome = fn
	return func() { accountHome = prev }
}
