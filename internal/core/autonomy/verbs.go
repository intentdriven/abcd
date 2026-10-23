package autonomy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// StoreName is the noun the checkout-root refusals are phrased with.
const StoreName = "the away store"

var (
	// ErrNoLocalTier is the set's refusal in a checkout abcd does not manage:
	// the mode store's own refusal, so a front door maps both alike.
	ErrNoLocalTier = mode.ErrNoLocalTier
	// ErrNotAway is a queued step with no away record holding.
	ErrNotAway = errors.New("autonomy: the person is not away")
)

// Options carry what the verbs read from the machine; the zero value reads
// the process's own home, clock and identity.
type Options struct {
	// Home is the home directory the away store lives under; "" means the
	// process's own.
	Home string
	// Now is the clock; nil means time.Now.
	Now func() time.Time
	// Who names the person acting in the checkout at root; nil means
	// resolveWho.
	Who func(root string) string
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

func (o Options) home() (string, error) {
	if o.Home != "" {
		return o.Home, nil
	}
	h, err := os.UserHomeDir()
	if err != nil || h == "" {
		return "", ErrNoHome
	}
	return h, nil
}

func (o Options) who(root, home string) string {
	if o.Who != nil {
		return o.Who(root)
	}
	return resolveWho(root, home)
}

// resolveWho names the person acting in the checkout at root: the holder
// name reservations resolve without a network read, else git's user.name,
// else "".
func resolveWho(root, home string) string {
	if h := claim.ReaderHolder(root, claim.Options{Home: home}); h != "" {
		return h
	}
	if id, err := identity.EffectiveIdentity(root); err == nil {
		return id.Name
	}
	return ""
}

// Resolve is the checkout root a caller standing in cwd acts from, the away
// store its repository keys, and the home it lives under.
func Resolve(cwd string, o Options) (root string, st Store, err error) {
	root, err = gitutil.CheckoutRoot(cwd, StoreName)
	if err != nil {
		return "", Store{}, err
	}
	home, err := o.home()
	if err != nil {
		return "", Store{}, err
	}
	st, err = openStore(home, root)
	return root, st, err
}

// Managed reports whether abcd manages the checkout at root: `.abcd/` is a
// directory at its root (a symbolic link is not), or root is a linked
// worktree of a repository whose main worktree has one. A worktree a session
// makes shares the repository's away store, and a branch cut before `.abcd/`
// was committed, or a `.abcd/` never committed, leaves it out of the
// worktree. The away mode's hooks and recorder act only in such a checkout,
// so a plugin installed for every repository reads nothing, writes nothing,
// runs no git past naming the roots, and says nothing in any other.
func Managed(root string) bool {
	if root == "" {
		return false
	}
	if hasAbcd(root) {
		return true
	}
	if fi, err := os.Lstat(filepath.Join(root, ".git")); err != nil || !fi.Mode().IsRegular() {
		return false // not a linked worktree: there is no main worktree to ask
	}
	common, err := gitutil.RevParseAbsPath(root, "--git-common-dir")
	if err != nil || filepath.Base(common) != ".git" {
		return false
	}
	return hasAbcd(filepath.Dir(common))
}

func hasAbcd(root string) bool {
	fi, err := os.Lstat(filepath.Join(root, ".abcd"))
	return err == nil && fi.IsDir()
}

// SetAway is `abcd mode autonomous [--until <time>] [--hold-pushes]` from
// cwd: until "" is open-ended. A past or unreadable --until, one past seven
// days, no checkout, and a checkout without the local tier are refused with
// nothing written; so is a store whose record or lock fails the checks.
func SetAway(cwd, until string, holdPushes bool, o Options) (*Record, error) {
	now := o.now()
	var end time.Time
	if until != "" {
		t, err := parseUntil(until, now)
		if err != nil {
			return nil, err
		}
		end = t
	}
	root, err := gitutil.CheckoutRoot(cwd, StoreName)
	if err != nil {
		return nil, err
	}
	if fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(mode.TierRelPath))); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("%w: %s/ is not a directory in this checkout, so abcd does not manage it and the away mode is not set from it",
			ErrNoLocalTier, mode.TierRelPath)
	}
	_, st, err := Resolve(root, o)
	if err != nil {
		return nil, err
	}
	who := Who{Name: o.who(root, st.home), Copy: root}
	// Judge the window before anything is created on disk.
	if _, err := begin(nil, who, now, end, holdPushes); err != nil {
		return nil, err
	}
	return st.update(st.carryRead, func(cur *Record) (*Record, error) {
		return begin(cur, who, now, end, holdPushes)
	})
}

// BackResult is what `abcd mode back` found and did.
type BackResult struct {
	// State is what the record was before: "away" (it held and is now
	// ended), "lapsed" (its time had passed and it is now closed), or "none"
	// (no record, or one already closed: nothing to end).
	State string
	// Record is the record after the verb, nil when there was none.
	Record *Record
	// Lines are the record's lines, printed one each.
	Lines []string
	// Notes say what could not be read or done as usual on the way to
	// ending the mode (a record past its bound, unparsable or not safely
	// this account's; a lock held past its timeout), one line each.
	Notes []string
}

// Back is `abcd mode back` from cwd: under the lock it reads the record's
// lines, records who ended it and from which copy, and closes it, on a
// record whose time has passed as on one that holds. It ends the mode
// whatever state the record is in (end.go): a record past its bound is
// printed whole and kept fitted; one it cannot read, or not safely this
// account's, is replaced by a closed record, printing nothing from it; and a
// lock not had within its timeout, or not safe to take, is gone around,
// closing without it. Each is named in the result's notes.
func Back(cwd string, o Options) (BackResult, error) {
	root, st, err := Resolve(cwd, o)
	if err != nil {
		return BackResult{}, err
	}
	now := o.now()
	who := Who{Name: o.who(root, st.home), Copy: root}
	res := BackResult{State: "none"}
	ran := false
	var rec *Record
	err = st.WithLock(func() error {
		ran = true
		var e error
		rec, e = st.endRecord(&res, who, now)
		return e
	})
	if err != nil && !ran {
		res = BackResult{State: "none", Notes: []string{fmt.Sprintf("the store's lock was not had (%s), so the record was closed without it", err)}}
		rec, err = st.endUnlocked(&res, who, now)
	}
	res.Record = rec
	return res, err
}

// QueueStep is `abcd mode queue "<step>"` from cwd: the step is kept for the
// person's return while the record holds, and refused otherwise.
func QueueStep(cwd, step string, o Options) (*Record, error) {
	root, st, err := Resolve(cwd, o)
	if err != nil {
		return nil, err
	}
	now := o.now()
	return st.Update(func(cur *Record) (*Record, error) {
		if !cur.Holds(now) {
			return nil, fmt.Errorf("%w: no away record holds, so there is no return to queue the step for; ask the person instead", ErrNotAway)
		}
		if err := cur.AddStep(step, root, now); err != nil {
			return nil, err
		}
		return cur, nil
	})
}

// AwayAt reads whether the person is away for the checkout at root, failing
// open: an unreadable record reads as not away, with the reason returned.
func AwayAt(root string, o Options) (*Record, bool, error) {
	home, err := o.home()
	if err != nil {
		return nil, false, err
	}
	st, err := openStore(home, root)
	if err != nil {
		return nil, false, err
	}
	return st.Away(o.now())
}
