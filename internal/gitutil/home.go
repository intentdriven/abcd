package gitutil

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
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
// (accountHome), and never anything the environment supplies: a planted HOME
// must not vouch for itself. A failed lookup grants no exemption.
//
// A home whose directory does not exist is judged on its cleaned spelling and
// gets no exemption, because os.Stat fails and there is no file to compare by
// identity. That is the intended outcome: a missing home inside a checkout is
// refused, and nothing is read from or written to it.
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

// WorkingTreesAround is every working tree dir lies in, nearest first: each
// directory at or above dir that carries a .git entry (RepoShapedRoot's
// marker, judged at every level rather than the nearest alone), walked both as
// dir is spelled and at its physical path. A submodule's superproject, and a
// checkout that holds another checkout, are named with the tree dir is in.
// The account's own home is left out, as HomeInsideWorkingTree leaves it out:
// a dotfiles-in-HOME account keeps its home in a working tree, and that home
// is the person's.
//
// It is the bound a program abcd runs is held outside of (tools.GuardForms):
// git's own toplevel for dir is one of these directories, since git names a
// toplevel only where its .git sits (ToplevelContext). It reads directory
// entries alone, starts no git and reads no repository configuration, so it
// answers at once whatever a repository's configuration holds, and it still
// names a tree git would refuse to answer for (another account's checkout, an
// unsafe directory).
func WorkingTreesAround(dir string) []string {
	if dir == "" {
		return nil
	}
	starts := []string{filepath.Clean(dir)}
	if phys, err := filepath.EvalSymlinks(dir); err == nil && phys != starts[0] {
		starts = append(starts, phys)
	}
	var trees []string
	for _, start := range starts {
		for d := start; ; {
			if _, err := os.Lstat(filepath.Join(d, ".git")); err == nil && !slices.Contains(trees, d) {
				trees = append(trees, d)
			}
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
			d = parent
		}
	}
	return slices.DeleteFunc(trees, isAccountHome)
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

// accountSource is where accountHome reads the account database. osUser is
// true only where os/user answers from the system's account database and
// never from the environment (accountFromOSUser); everywhere else the passwd
// file is parsed directly. A test points passwd at a fixture.
var accountSource = struct {
	osUser bool
	passwd string
}{accountFromOSUser, "/etc/passwd"}

// accountHome is the home the account database names for this process's uid.
var accountHome = func() (string, error) {
	return lookupAccountHome(accountSource.osUser, os.Getuid(), accountSource.passwd)
}

// lookupAccountHome names uid's home from the account database, never from
// $HOME.
//
// os/user is not trusted for this off darwin: LookupId answers the process's
// own uid through user.Current, and the pure-Go Current (cgo off, as in the
// cross-compiled linux release binaries, or the osusergo tag anywhere) falls
// back to os.UserHomeDir, which is $HOME, when the uid has no /etc/passwd
// entry. That would let a HOME planted in a checkout vouch for itself. So off
// darwin the passwd file is read directly: the first entry whose uid field
// matches names the home. An account the file does not list (one served only
// by NSS, LDAP or sssd) gets no exemption, which fails closed. On darwin
// without osusergo, os/user asks the directory service through libSystem
// (getpwuid_r), which never consults the environment.
func lookupAccountHome(osUser bool, uid int, passwd string) (string, error) {
	if osUser {
		u, err := user.LookupId(strconv.Itoa(uid))
		if err != nil {
			return "", err
		}
		return u.HomeDir, nil
	}
	return passwdHome(passwd, uid)
}

// passwdHome returns the home field of the first entry in the passwd-format
// file at path whose uid field is uid. A missing entry, an empty or relative
// home, or any read error is an error, and an error grants no exemption.
func passwdHome(path string, uid int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	want := strconv.Itoa(uid)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		// name:password:uid:gid:gecos:home:shell
		fields := strings.Split(line, ":")
		if len(fields) < 7 || fields[2] != want {
			continue
		}
		home := fields[5]
		if !filepath.IsAbs(home) {
			return "", fmt.Errorf("%s: uid %d names no absolute home", path, uid)
		}
		return home, nil
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("%s: no entry for uid %d", path, uid)
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
