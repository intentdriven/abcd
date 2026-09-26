package intent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// dirLockHeld reports whether some descriptor holds the flock a record store
// takes on its own directory, by asking for it without waiting and letting it
// go at once. A store with no directory has no lock to hold.
func dirLockHeld(t *testing.T, root, rel string) bool {
	t.Helper()
	fd, err := syscall.Open(filepath.Join(root, rel), syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, syscall.ENOENT) {
		return false
	}
	if err != nil {
		t.Fatalf("opening %s to probe its lock: %v", rel, err)
	}
	defer syscall.Close(fd)
	err = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return true
	}
	if err != nil {
		t.Fatalf("probing the lock on %s: %v", rel, err)
	}
	_ = syscall.Flock(fd, syscall.LOCK_UN)
	return false
}

// specLinkingTheDraft lays the draft itd-10 and an open spec, realising another
// intent, whose body links to that draft — the spec a plan of itd-10 repoints.
func specLinkingTheDraft(t *testing.T, root string) string {
	t.Helper()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	rel := specsOpen + "/spc-5-other.md"
	writeFile(t, root, rel, specNaming("spc-5", "other", "itd-99")+"\nFollows [itd-10](../../intents/drafts/itd-10-alpha.md).\n")
	return rel
}

// A repoint rewrites every spec linking a moved path, and spec close renames a
// spec open/ -> closed/. Unlocked, a close landing between the repoint's read
// of the spec at open/ and its write put the spec back in open/ beside its
// closed copy — one record in both status folders (iss-2609262218309668). The
// repoint holds the spec store's lock, which spec close takes too, so the
// close waits for the repoint's write and the spec ends in closed/ alone,
// carrying the repointed link.
func TestARepointRacingASpecCloseLeavesTheSpecInOneFolder(t *testing.T) {
	root := t.TempDir()
	specLinkingTheDraft(t, root)

	var landedEarly bool
	var closer chan error
	duringRepoint = func() {
		closer = make(chan error, 1)
		go func() {
			_, err := spec.Close(root, "spc-5")
			closer <- err
		}()
		select {
		case err := <-closer:
			landedEarly = true
			closer <- err
		case <-time.After(300 * time.Millisecond):
		}
	}
	t.Cleanup(func() { duringRepoint = nil })

	res, err := Plan(root, "itd-10", PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if closer == nil {
		t.Fatal("plan never reached its repoint")
	}
	if err := <-closer; err != nil {
		t.Fatal(err)
	}
	if landedEarly {
		t.Error("spec close landed while the repoint ran: the repoint does not hold the spec store's lock, or the close does not take it")
	}
	if _, err := os.Lstat(filepath.Join(root, specsOpen, "spc-5-other.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the closed spec is still in open/ (err %v): it sits in both status folders", err)
	}
	got := readRel(t, root, specsClosed+"/spc-5-other.md")
	if !strings.Contains(got, "(../../intents/planned/itd-10-alpha.md)") {
		t.Errorf("the closed spec must carry the repointed link:\n%s", got)
	}
	if res.RelinkError != "" || len(res.Relinked) != 1 {
		t.Errorf("plan must report the one rewrite: %+v %q", res.Relinked, res.RelinkError)
	}
}

// A spec writer — every one takes the spec store's lock, reads the record and
// writes it back — arriving while a repoint runs waits for the repoint's write
// instead of landing between its read and its write, where the repoint's
// write from stale bytes erased it (iss-2609262218309668). The spec ends up
// carrying both the repointed link and the edit.
func TestASpecEditRacingARepointIsNotLost(t *testing.T) {
	root := t.TempDir()
	specRel := specLinkingTheDraft(t, root)

	const edit = "\nA concurrent spec edit.\n"
	var landedEarly bool
	var writer chan error
	duringRepoint = func() {
		writer = make(chan error, 1)
		go func() {
			writer <- spec.WithStoreLock(root, func() error {
				abs := filepath.Join(root, specRel)
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

	if _, err := Plan(root, "itd-10", PlanOptions{}); err != nil {
		t.Fatal(err)
	}
	if writer == nil {
		t.Fatal("plan never reached its repoint")
	}
	if err := <-writer; err != nil {
		t.Fatal(err)
	}
	if landedEarly {
		t.Error("a spec writer landed while the repoint ran: the repoint does not hold the spec store's lock")
	}
	got := readRel(t, root, specRel)
	if !strings.Contains(got, "(../../intents/planned/itd-10-alpha.md)") || !strings.Contains(got, edit) {
		t.Errorf("the spec must carry both the repointed link and the concurrent edit:\n%s", got)
	}
}

// The three record-store locks are taken in one total order — the issue
// ledger's, then the intent store's, then the spec store's — and no path takes
// a later one and then an earlier one. The probes sit at the two acquisitions
// that could break the order from inside this package: the ledger lock is
// never requested with the intent or spec lock held, and the intent lock never
// with the spec lock held. (The spec package imports neither of the others, so
// nothing under the spec lock can request an earlier one; its own test pins
// that.) Every repoint runs with all three held. The paths driven are the
// lifecycle's: a plan (whose spec mint nests the spec lock inside the intent
// lock), a close that mints a remainder, and the close that ships.
func TestTheRecordLocksAreTakenLedgerThenIntentThenSpec(t *testing.T) {
	root := t.TempDir()
	issueLinkingTheDraft(t, root)
	specLinkingTheDraft(t, root)
	ledgerPath := filepath.Join(root, recordid.IssuesRelDir, ".iss-alloc.lock")

	var violations []string
	old := ledgerLock
	ledgerLock = func(repoRoot string, fn func() error) error {
		if dirLockHeld(t, root, IntentsRelDir) || dirLockHeld(t, root, spec.SpecsRelDir) {
			violations = append(violations, "the ledger lock was requested with a later lock held")
		}
		return fsutil.WithFileLock(ledgerPath, 5*time.Second, fn)
	}
	t.Cleanup(func() { ledgerLock = old })
	beforeIntentMintLock = func() {
		if dirLockHeld(t, root, spec.SpecsRelDir) {
			violations = append(violations, "the intent lock was requested with the spec lock held")
		}
	}
	t.Cleanup(func() { beforeIntentMintLock = nil })
	repoints := 0
	duringRepoint = func() {
		repoints++
		ledgerHeld := errors.Is(fsutil.WithFileLock(ledgerPath, 0, func() error { return nil }), fsutil.ErrLockContention)
		if !ledgerHeld || !dirLockHeld(t, root, IntentsRelDir) || !dirLockHeld(t, root, spec.SpecsRelDir) {
			violations = append(violations, "a repoint ran without all three locks held")
		}
	}
	t.Cleanup(func() { duringRepoint = nil })

	pr, err := Plan(root, "itd-10", PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	first, err := Reconcile(root, pr.Spec.ID, "", RemainderRequest{Slug: "rest"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Reconcile(root, first.Remainder.ID, "fix", RemainderRequest{}); err != nil {
		t.Fatal(err)
	}
	if repoints != 3 {
		t.Errorf("the three verbs must each repoint once, got %d repoints", repoints)
	}
	for _, v := range violations {
		t.Error(v)
	}
}

// The three-lock acquisition never holds an earlier lock while it waits for
// the spec store's (the rule iss-2609262218059995 set for the intent lock,
// extended to the third): with the spec lock held elsewhere, a ledger writer
// and an intent writer with budgets far below the acquisition's each land
// while it waits, fn runs once with all three held after the spec lock is
// released, and an acquisition that cannot get the spec lock within its budget
// returns an error naming it without running fn.
func TestTheAcquisitionLeavesEarlierLocksFreeWhileItWaitsForTheSpecLock(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, specsOpen+"/spc-5-other.md", specNaming("spc-5", "other", "itd-99"))
	lockPath := filepath.Join(root, "ledger.lock")
	ledger := func(timeout time.Duration) func(func() error) error {
		return func(fn func() error) error { return fsutil.WithFileLock(lockPath, timeout, fn) }
	}
	holdSpec := func() (release func(), holder chan error) {
		held, rel := make(chan struct{}), make(chan struct{})
		holder = make(chan error, 1)
		go func() {
			holder <- spec.WithStoreLock(root, func() error {
				close(held)
				<-rel
				return nil
			})
		}()
		<-held
		return func() { close(rel) }, holder
	}

	release, holder := holdSpec()
	var ranWithAll bool
	done := make(chan error, 1)
	go func() {
		done <- WithLedgerThenMintLock(root, ledger(5*time.Second), func() error {
			ranWithAll = errors.Is(fsutil.WithFileLock(lockPath, 0, func() error { return nil }), fsutil.ErrLockContention) &&
				dirLockHeld(t, root, IntentsRelDir) && dirLockHeld(t, root, spec.SpecsRelDir)
			return nil
		})
	}()
	time.Sleep(150 * time.Millisecond)
	ledgerErr := ledger(400 * time.Millisecond)(func() error { return nil })
	intentErr := withIntentMintLockWithin(root, 400*time.Millisecond, func() error { return nil })
	select {
	case err := <-done:
		release()
		t.Fatalf("the acquisition finished while the spec lock was held elsewhere (err %v): fn ran without it", err)
	default:
	}
	release()
	if err := <-holder; err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("the acquisition after the spec lock was released: %v", err)
	}
	if ledgerErr != nil {
		t.Errorf("a ledger writer arriving while the acquisition waited for the spec lock failed: %v", ledgerErr)
	}
	if intentErr != nil {
		t.Errorf("an intent writer arriving while the acquisition waited for the spec lock failed: %v", intentErr)
	}
	if !ranWithAll {
		t.Error("fn ran without all three locks held")
	}

	old := mintLockTimeout
	mintLockTimeout = 300 * time.Millisecond
	t.Cleanup(func() { mintLockTimeout = old })
	release, holder = holdSpec()
	ran := false
	err := WithLedgerThenMintLock(root, ledger(5*time.Second), func() error { ran = true; return nil })
	release()
	if err := <-holder; err != nil {
		t.Fatal(err)
	}
	if !errors.Is(err, spec.ErrStoreLockBusy) || ran {
		t.Errorf("an acquisition past its budget must refuse naming the spec lock without running fn: ran=%v err=%v", ran, err)
	}
}
