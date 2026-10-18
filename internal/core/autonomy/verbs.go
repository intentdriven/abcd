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
	// ResolveWho.
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
	return ResolveWho(root, home)
}

// ResolveWho names the person acting in the checkout at root: the holder
// name reservations resolve without a network read, else git's user.name,
// else "".
func ResolveWho(root, home string) string {
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
	st, err = OpenStore(home, root)
	return root, st, err
}

// Managed reports whether abcd manages the checkout at root: `.abcd/` is a
// directory at its root (a symbolic link is not). The away mode's hooks act
// only in such a checkout, so a plugin installed for every repository reads
// nothing, runs no git past naming the root, and says nothing in any other.
func Managed(root string) bool {
	if root == "" {
		return false
	}
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
		t, err := ParseUntil(until, now)
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
	if _, err := Begin(nil, who, now, end, holdPushes); err != nil {
		return nil, err
	}
	return st.Update(func(cur *Record) (*Record, error) {
		return Begin(cur, who, now, end, holdPushes)
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
}

// Back is `abcd mode back` from cwd: under the lock it reads the record's
// lines, records who ended it and from which copy, and closes it, on a
// record whose time has passed as on one that holds.
func Back(cwd string, o Options) (BackResult, error) {
	root, st, err := Resolve(cwd, o)
	if err != nil {
		return BackResult{}, err
	}
	now := o.now()
	res := BackResult{State: "none"}
	rec, err := st.Update(func(cur *Record) (*Record, error) {
		if cur == nil || cur.Closed != nil {
			return nil, nil
		}
		res.State = "lapsed"
		if cur.Holds(now) {
			res.State = "away"
		}
		res.Lines = cur.Lines()
		cur.Close(Who{Name: o.who(root, st.home), Copy: root}, now)
		return cur, nil
	})
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
	st, err := OpenStore(home, root)
	if err != nil {
		return nil, false, err
	}
	return st.Away(o.now())
}
