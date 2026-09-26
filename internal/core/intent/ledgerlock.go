package intent

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/intentdriven/abcd/internal/core/recordid"
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

// pairIntentTry is how long one attempt of a pair acquisition waits for the
// intent store's lock with the ledger lock held, and pairLedgerRest how long it
// then leaves the ledger lock free before the next attempt. The rest is longer
// than the ledger's longest poll interval (fsutil's backoff tops out at 100ms),
// so a ledger writer polling through a wait finds the lock free at least once
// per attempt.
var (
	pairIntentTry  = 25 * time.Millisecond
	pairLedgerRest = 150 * time.Millisecond
)

// WithLedgerThenMintLock runs fn holding the ledger lock (taken through
// ledger) and then the intent store's lock — the one order every path holding
// both takes; the intent package never takes the ledger lock inside its own.
//
// It never holds the ledger lock while it WAITS for the intent lock
// (iss-2609262218059995). Waiting inside the hold chained two budgets: an
// intent hold of five seconds made a third process's ledger writer fail with
// contention while this caller waited on. So each attempt asks for the intent
// lock only briefly, and on contention lets the ledger go, rests, and tries
// again, until mintLockTimeout; past it the pair returns an error naming the
// intent lock and fn has not run. fn runs exactly once, with both held.
//
// With no intent store, fn runs under the ledger lock alone, as WithMintLock
// runs it with none.
func WithLedgerThenMintLock(repoRoot string, ledger func(func() error) error, fn func() error) error {
	if _, err := os.Lstat(filepath.Join(repoRoot, IntentsRelDir)); errors.Is(err, fs.ErrNotExist) {
		return ledger(fn)
	}
	deadline := time.Now().Add(mintLockTimeout)
	for {
		busy := false
		err := ledger(func() error {
			entered := false
			err := withIntentMintLockWithin(repoRoot, pairIntentTry, func() error {
				entered = true
				return fn()
			})
			if !entered && errors.Is(err, errIntentLockBusy) {
				busy = true
				return nil
			}
			return err
		})
		if err != nil || !busy {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%w within %s, waiting with the ledger lock released between attempts", errIntentLockBusy, mintLockTimeout)
		}
		time.Sleep(pairLedgerRest)
	}
}
