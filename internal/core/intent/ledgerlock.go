package intent

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// ledgerLock is the issue ledger's lock, registered by the package that owns
// it. This package cannot import that one (it imports this), yet a repoint
// rewrites ledger records that link to a moved intent or spec, and a ledger
// writer landing between the repoint's read and its write was erased
// (iss-2609262143209970). Nil in a binary that links no ledger: a repoint that
// finds a ledger then refuses rather than rewrite it unlocked.
var ledgerLock func(repoRoot string, fn func() error) error

// SetLedgerLock registers the issue ledger's lock, for the package that owns
// it to call once, from init.
func SetLedgerLock(lock func(repoRoot string, fn func() error) error) { ledgerLock = lock }

// errNoLedgerLock is the repoint's refusal on a ledger with no lock registered.
var errNoLedgerLock = errors.New("intent: the tree has an issue ledger but no ledger lock is registered, so ledger records are not rewritten")

// repoLedgerLock is the ledger lock of repoRoot's default ledger, as a pair
// acquisition takes it. A tree with no ledger has no ledger record to protect,
// and taking the lock would create the ledger, so none is taken there — the
// stance WithMintLock takes on a tree with no intent store.
func repoLedgerLock(repoRoot string) func(func() error) error {
	return func(fn func() error) error {
		if _, err := os.Lstat(filepath.Join(repoRoot, filepath.FromSlash(recordid.IssuesRelDir))); errors.Is(err, fs.ErrNotExist) {
			return fn()
		}
		if ledgerLock == nil {
			return errNoLedgerLock
		}
		return ledgerLock(repoRoot, fn)
	}
}

// pairIntentTry is how long one attempt of the three-lock acquisition waits
// for each of the intent store's lock and the spec store's with the earlier
// locks held, and pairLedgerRest how long it then leaves every lock free before
// the next attempt. The rest is twice the longest interval a ledger writer
// sleeps between its polls (fsutil.LockPollCeiling), so a ledger writer polling
// through a wait wakes inside the window at least once per attempt and finds
// the lock free; the intent and spec stores' writers poll every 10ms, far
// inside it. It is derived from the ceiling rather than restated beside it: a
// rest written as a number once sat below the poll's real ceiling
// (iss-2609262257227538).
var (
	pairIntentTry  = 25 * time.Millisecond
	pairLedgerRest = 2 * fsutil.LockPollCeiling
)

// WithLedgerThenMintLock runs fn holding the record stores' three locks in
// their one total order: the issue ledger's lock (taken through ledger), then
// the intent store's, then the spec store's (spec.WithStoreLock's). Every path
// holding more than one of them takes them in that order and none takes a
// later one and then an earlier one: this package never takes the ledger lock
// inside its own (it cannot import the ledger's package; capture registers the
// lock, SetLedgerLock), plan mints its spec inside the intent lock, and the
// spec package imports neither of the others. It is the one acquisition for a
// writer of records in more than one store — a link repoint (intent's
// repointUnderLock, capture's repointMovedIssue), capture's migration of the
// promote join's back-edge, a lifeboat embark — and a writer of specs outside
// the spec package takes the spec lock through it (iss-2609262218309668).
//
// It never holds an earlier lock while it WAITS for a later one
// (iss-2609262218059995). Waiting inside the hold chained two budgets: an
// intent hold of five seconds made a third process's ledger writer fail with
// contention while this caller waited on. So each attempt asks for the intent
// lock and then the spec lock only briefly (pairIntentTry each), and on
// contention for either lets every lock it holds go, rests, and tries again,
// until mintLockTimeout; past it the call returns an error naming the lock it
// last found busy and fn has not run. fn runs exactly once, with every lock
// held.
//
// A store with no directory contributes no lock, as WithMintLock and
// spec.WithStoreLock take none there: taking one would plant the store, and
// with no store there is no record in it to race. With neither the intent nor
// the spec store, fn runs under the ledger lock alone.
//
// None of the three is reentrant, so fn must not call a writer that takes any
// of them.
func WithLedgerThenMintLock(repoRoot string, ledger func(func() error) error, fn func() error) error {
	var inner []func(time.Duration, func() error) error
	if _, err := os.Lstat(filepath.Join(repoRoot, IntentsRelDir)); !errors.Is(err, fs.ErrNotExist) {
		inner = append(inner, func(d time.Duration, fn func() error) error { return withIntentMintLockWithin(repoRoot, d, fn) })
	}
	if _, err := os.Lstat(filepath.Join(repoRoot, spec.SpecsRelDir)); !errors.Is(err, fs.ErrNotExist) {
		inner = append(inner, func(d time.Duration, fn func() error) error { return spec.WithStoreLockWithin(repoRoot, d, fn) })
	}
	if len(inner) == 0 {
		return ledger(fn)
	}
	deadline := time.Now().Add(mintLockTimeout)
	for {
		var busy error
		err := ledger(func() error {
			entered := false
			err := holdInOrder(inner, func() error {
				entered = true
				return fn()
			})
			if !entered && (errors.Is(err, errIntentLockBusy) || errors.Is(err, spec.ErrStoreLockBusy)) {
				busy = err
				return nil
			}
			return err
		})
		if err != nil || busy == nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%w; tried for %s, with every lock released between attempts", busy, mintLockTimeout)
		}
		time.Sleep(pairLedgerRest)
	}
}

// holdInOrder takes each lock in turn, each with pairIntentTry to be granted,
// and runs fn with all of them held. A lock not granted unwinds the ones
// already taken, in reverse, before its error returns.
func holdInOrder(locks []func(time.Duration, func() error) error, fn func() error) error {
	if len(locks) == 0 {
		return fn()
	}
	return locks[0](pairIntentTry, func() error { return holdInOrder(locks[1:], fn) })
}
