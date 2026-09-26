package intent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// registerTestLedgerLock stands in for the ledger lock the capture package
// registers in production, which this package cannot import: an flock on a
// file in the ledger, taken with the same five-second budget.
func registerTestLedgerLock(t *testing.T) {
	t.Helper()
	old := ledgerLock
	ledgerLock = func(repoRoot string, fn func() error) error {
		return fsutil.WithFileLock(filepath.Join(repoRoot, recordid.IssuesRelDir, ".iss-alloc.lock"), 5*time.Second, fn)
	}
	t.Cleanup(func() { ledgerLock = old })
}

// unregisterLedgerLock runs a test as a binary that links no ledger would.
func unregisterLedgerLock(t *testing.T) {
	t.Helper()
	old := ledgerLock
	ledgerLock = nil
	t.Cleanup(func() { ledgerLock = old })
}

// issueLinkingTheDraft lays a draft and an open issue whose body links to it.
func issueLinkingTheDraft(t *testing.T, root string) string {
	t.Helper()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	const rel = recordid.IssuesRelDir + "/open/iss-1-one.md"
	writeFile(t, root, rel, "---\nid: iss-1\n---\n\nOccasioned [itd-10](../../../development/intents/drafts/itd-10-alpha.md).\n")
	return rel
}

// A record-moving verb's repoint rewrites every ledger record linking the
// moved path, so it takes the ledger lock, before the intent store's lock: a
// ledger writer arriving while the repoint runs waits for its write instead of
// racing it, and the issue ends up carrying both the repointed link and the
// writer's edit (iss-2609262143209970). The verb is plan; every record-moving
// verb repoints through the same helper.
func TestRepointHoldsTheLedgerLockAgainstALedgerWriter(t *testing.T) {
	root := t.TempDir()
	registerTestLedgerLock(t)
	issueRel := issueLinkingTheDraft(t, root)

	const edit = "\nA concurrent ledger edit.\n"
	var landedEarly bool
	var writer chan error
	duringRepoint = func() {
		writer = make(chan error, 1)
		go func() {
			writer <- ledgerLock(root, func() error {
				abs := filepath.Join(root, issueRel)
				data, err := os.ReadFile(abs)
				if err != nil {
					return err
				}
				return os.WriteFile(abs, append(data, []byte(edit)...), 0o644)
			})
		}()
		select {
		case err := <-writer:
			landedEarly = true
			writer <- err
		case <-time.After(300 * time.Millisecond):
		}
	}
	t.Cleanup(func() { duringRepoint = nil })

	res, err := Plan(root, "itd-10", PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if writer == nil {
		t.Fatal("plan never reached its repoint")
	}
	if err := <-writer; err != nil {
		t.Fatal(err)
	}
	if landedEarly {
		t.Error("a ledger writer landed while the repoint ran: the repoint does not hold the ledger lock")
	}
	got := readRel(t, root, issueRel)
	if !strings.Contains(got, "(../../../development/intents/planned/itd-10-alpha.md)") || !strings.Contains(got, edit) {
		t.Errorf("the issue must carry both the repointed link and the concurrent edit:\n%s", got)
	}
	if res.RelinkError != "" || len(res.Relinked) != 1 {
		t.Errorf("plan must report the one rewrite: %+v %q", res.Relinked, res.RelinkError)
	}
}

// With a ledger present and no ledger lock registered, the repoint refuses
// rather than rewrite ledger records unlocked: the verb stands (its record has
// moved), the refusal is reported as the repoint's error, and nothing is
// repointed.
func TestRepointRefusesALedgerWithNoLedgerLockRegistered(t *testing.T) {
	root := t.TempDir()
	unregisterLedgerLock(t)
	issueRel := issueLinkingTheDraft(t, root)

	res, err := Plan(root, "itd-10", PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.RelinkError, "no ledger lock is registered") || len(res.Relinked) != 0 {
		t.Errorf("the repoint must refuse and repoint nothing: %+v %q", res.Relinked, res.RelinkError)
	}
	if got := readRel(t, root, issueRel); !strings.Contains(got, "(../../../development/intents/drafts/itd-10-alpha.md)") {
		t.Errorf("the issue must be left as written:\n%s", got)
	}
}

// A tree with no ledger has no ledger record to protect, so the repoint takes
// no ledger lock — registered or not — and never creates the ledger that
// taking it would.
func TestRepointWithNoLedgerCreatesNone(t *testing.T) {
	for _, registered := range []bool{false, true} {
		root := t.TempDir()
		if registered {
			registerTestLedgerLock(t)
		} else {
			unregisterLedgerLock(t)
		}
		writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
		writeFile(t, root, draftsDir+"/itd-11-beta.md",
			"---\nid: itd-11\nslug: beta\nspec_id: null\nkind: null\n---\n# beta\n\nAfter [itd-10](itd-10-alpha.md).\n")

		res, err := Plan(root, "itd-10", PlanOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if res.RelinkError != "" || len(res.Relinked) != 1 {
			t.Errorf("registered=%v: plan must report the one rewrite: %+v %q", registered, res.Relinked, res.RelinkError)
		}
		if _, err := os.Lstat(filepath.Join(root, ".abcd", "work")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("registered=%v: the repoint created a ledger (err %v)", registered, err)
		}
	}
}

// The pair acquisition never holds the ledger lock while it waits for the
// intent store's lock (iss-2609262218059995): with the intent lock held
// elsewhere, a third writer takes the ledger lock with a budget far below the
// pair's, fn runs with both held once the intent lock is released, and a pair
// that cannot get the intent lock within its budget returns an error naming it
// without running fn.
func TestThePairLeavesTheLedgerFreeWhileItWaitsForTheIntentLock(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	lockPath := filepath.Join(root, "ledger.lock")
	ledger := func(timeout time.Duration) func(func() error) error {
		return func(fn func() error) error { return fsutil.WithFileLock(lockPath, timeout, fn) }
	}

	holdIntent := func() (release func(), holder chan error) {
		held, rel := make(chan struct{}), make(chan struct{})
		holder = make(chan error, 1)
		go func() {
			holder <- withIntentMintLock(root, func() error {
				close(held)
				<-rel
				return nil
			})
		}()
		<-held
		return func() { close(rel) }, holder
	}

	release, holder := holdIntent()
	var ranWithBoth bool
	done := make(chan error, 1)
	go func() {
		done <- WithLedgerThenMintLock(root, ledger(5*time.Second), func() error {
			ranWithBoth = errors.Is(fsutil.WithFileLock(lockPath, 0, func() error { return nil }), fsutil.ErrLockContention)
			return nil
		})
	}()
	time.Sleep(150 * time.Millisecond)
	thirdErr := ledger(300 * time.Millisecond)(func() error { return nil })
	release()
	if err := <-holder; err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("the pair after the intent lock was released: %v", err)
	}
	if thirdErr != nil {
		t.Errorf("a ledger writer arriving while the pair waited for the intent lock failed: %v", thirdErr)
	}
	if !ranWithBoth {
		t.Error("fn ran without the ledger lock held")
	}

	old := mintLockTimeout
	mintLockTimeout = 300 * time.Millisecond
	t.Cleanup(func() { mintLockTimeout = old })
	release, holder = holdIntent()
	ran := false
	err := WithLedgerThenMintLock(root, ledger(5*time.Second), func() error { ran = true; return nil })
	release()
	if err := <-holder; err != nil {
		t.Fatal(err)
	}
	if err == nil || ran || !strings.Contains(err.Error(), "mint lock") {
		t.Errorf("a pair past its budget must refuse without running fn: ran=%v err=%v", ran, err)
	}
}
