package capture

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/intent"
)

// ledgerLockHeld reports whether some descriptor holds the ledger lock right
// now, by asking for it without waiting and letting it go at once.
func ledgerLockHeld(t *testing.T, ir string) bool {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(ir, lockFilename), os.O_RDWR, 0)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	if err != nil {
		t.Fatalf("opening the ledger lock: %v", err)
	}
	defer f.Close()
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return true
	}
	if err != nil {
		t.Fatalf("probing the ledger lock: %v", err)
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return false
}

// lockedIntentAppend is an intent writer as every writer in the intent store
// is one: it takes the store's lock, reads the record, and writes it back with
// one line more.
func lockedIntentAppend(repo, rel, line string) error {
	return intent.WithMintLock(repo, func() error {
		abs := filepath.Join(repo, filepath.FromSlash(rel))
		data, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		return os.WriteFile(abs, append(data, []byte(line)...), 0o644)
	})
}

// landsWithin starts fn and reports whether it finished within d, and a
// channel its result arrives on.
func landsWithin(d time.Duration, fn func() error) (bool, chan error) {
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		done <- err
		return true, done
	case <-time.After(d):
		return false, done
	}
}

// issueLinkedFromAnIntent captures one issue and lays a draft intent linking to
// it in open/, returning the issue id, its filename and the intent's path.
func issueLinkedFromAnIntent(t *testing.T) (repo, ir, id, name, itdRel string) {
	t.Helper()
	repo, ir = ledger(t)
	res, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	name = filepath.Base(res.Path)
	itdRel = ".abcd/development/intents/drafts/itd-7-seven.md"
	writeTree(t, repo, itdRel, "---\nid: itd-7\nslug: seven\nspec_id: null\nkind: null\n---\n\n# Seven\n\n"+
		"Occasioned by [alpha](../../../work/issues/open/"+name+").\n")
	return repo, ir, res.ID, name, itdRel
}

// A verb that needs the ledger lock and the intent store's lock never holds
// the first while it waits for the second (iss-2609262218059995). Waiting on
// the intent lock inside the ledger lock chained two five-second budgets: an
// intent hold of five seconds made a third process's ledger writer fail with
// ErrAllocatorContention while the verb itself waited on. So, with the intent
// lock held elsewhere for longer than a ledger writer's whole budget, a resolve
// and a migrate apply each wait, a capture arriving meanwhile lands, and each
// verb finishes once the intent lock is released.
func TestAWaitingVerbLeavesTheLedgerToOtherWriters(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(t *testing.T) (repo, ir string, verb func() error)
	}{
		{"resolve", func(t *testing.T) (string, string, func() error) {
			repo, ir, id, _, _ := issueLinkedFromAnIntent(t)
			return repo, ir, func() error {
				res, err := Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: id, Resolution: "fixed", Impact: "fix"})
				if err == nil && (res.RelinkError != "" || len(res.Relinked) != 1) {
					err = fmt.Errorf("the resolve must repoint its one link once the intent lock is free: %+v %q", res.Relinked, res.RelinkError)
				}
				return err
			}
		}},
		{"migrate --apply", func(t *testing.T) (string, string, func() error) {
			repo, ir, _ := migrateFixture(t)
			return repo, ir, func() error {
				_, err := Migrate(MigrateRequest{RepoRoot: repo, IssuesRoot: ir, Apply: true})
				return err
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, ir, verb := tc.run(t)
			old := lockTimeout
			lockTimeout = 400 * time.Millisecond
			t.Cleanup(func() { lockTimeout = old })

			held, release := make(chan struct{}), make(chan struct{})
			holder := make(chan error, 1)
			go func() {
				holder <- intent.WithMintLock(repo, func() error {
					close(held)
					<-release
					return nil
				})
			}()
			<-held
			done := make(chan error, 1)
			go func() { done <- verb() }()
			time.Sleep(150 * time.Millisecond)

			_, capErr := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "a third writer", Severity: SeverityMinor,
				Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "third"})
			select {
			case err := <-done:
				close(release)
				t.Fatalf("%s finished while another holder had the intent lock (err %v): it wrote intent records without taking it", tc.name, err)
			default:
			}
			close(release)
			if err := <-holder; err != nil {
				t.Fatal(err)
			}
			if err := <-done; err != nil {
				t.Fatalf("%s after the intent lock was released: %v", tc.name, err)
			}
			if capErr != nil {
				t.Fatalf("a ledger writer arriving while %s waited for the intent lock failed: %v", tc.name, capErr)
			}
		})
	}
}

// An intent verb's repoint rewrites ledger records that link to the record it
// moved, so it takes the ledger lock — registered with the intent package by
// this one, which owns it — before the intent store's lock
// (iss-2609262143209970). With the ledger lock held elsewhere, a plan whose
// draft an issue links to waits for it, and repoints the issue's link once it
// is released.
func TestAnIntentVerbRepointTakesTheLedgerLock(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
		Category: "bug", Source: "user-observation", FoundDuring: "t", Slug: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	issueRel := filepath.ToSlash(res.Path)
	writeTree(t, repo, ".abcd/development/intents/drafts/itd-7-seven.md", "---\nid: itd-7\nslug: seven\nspec_id: null\nkind: null\n---\n# seven\n\n"+
		"## Acceptance Criteria\n\n- **Given** a user, **when** they act, **then** it works.\n")
	abs := filepath.Join(repo, filepath.FromSlash(issueRel))
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, append(data, []byte("\nOccasioned [itd-7](../../../development/intents/drafts/itd-7-seven.md).\n")...), 0o644); err != nil {
		t.Fatal(err)
	}

	held, release := make(chan struct{}), make(chan struct{})
	holder := make(chan error, 1)
	go func() {
		holder <- WithLedgerLock(repo, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	landed, done := landsWithin(300*time.Millisecond, func() error {
		pr, err := intent.Plan(repo, "itd-7", intent.PlanOptions{})
		if err == nil && pr.RelinkError != "" {
			err = fmt.Errorf("plan's repoint: %s", pr.RelinkError)
		}
		return err
	})
	close(release)
	if err := <-holder; err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if landed {
		t.Error("plan finished while another holder had the ledger lock: its repoint rewrote a ledger record without taking it")
	}
	if got := readTree(t, repo, issueRel); !strings.Contains(got, "(../../../development/intents/planned/itd-7-seven.md)") {
		t.Errorf("the issue's link must be repointed to the planned intent:\n%s", got)
	}
}

// A resolve's repoint rewrites every intent linking the issue, and it does so
// with the intent store's lock held inside the ledger lock: an intent writer
// arriving while the repoint runs waits for its write instead of racing it,
// and the record ends up carrying both edits (iss-2609261254247117).
func TestIssueRepointHoldsTheIntentLock(t *testing.T) {
	repo, ir, id, name, itdRel := issueLinkedFromAnIntent(t)
	const edit = "\nA concurrent intent edit.\n"
	var landedEarly, ledgerHeld bool
	var writer chan error
	duringIssueRepoint = func() {
		ledgerHeld = ledgerLockHeld(t, ir)
		landedEarly, writer = landsWithin(300*time.Millisecond, func() error {
			return lockedIntentAppend(repo, itdRel, edit)
		})
	}
	t.Cleanup(func() { duringIssueRepoint = nil })

	res, err := Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: id, Resolution: "fixed", Impact: "fix"})
	if err != nil {
		t.Fatal(err)
	}
	if writer == nil {
		t.Fatal("the resolve never reached its repoint")
	}
	if err := <-writer; err != nil {
		t.Fatal(err)
	}
	if !ledgerHeld {
		t.Error("the repoint ran without the ledger lock")
	}
	if landedEarly {
		t.Error("an intent writer landed while the repoint ran: the repoint does not hold the intent store's lock")
	}
	got := readTree(t, repo, itdRel)
	if !strings.Contains(got, "(../../../work/issues/resolved/"+name+")") || !strings.Contains(got, edit) {
		t.Errorf("the intent must carry both the repointed link and the concurrent edit:\n%s", got)
	}
	if res.RelinkError != "" || len(res.Relinked) != 1 {
		t.Errorf("the resolve must report the one rewrite: %+v %q", res.Relinked, res.RelinkError)
	}
}

// A migrate apply rewrites intent records from what its scan read, so an
// intent writer landing between the scan and the write was erased by it. Under
// the intent store's lock that writer waits, and the record keeps both the
// migrated back-edge and its edit (iss-2609261941039204).
func TestMigrateApplyKeepsAConcurrentIntentEdit(t *testing.T) {
	repo, ir, _ := migrateFixture(t)
	const rel = ".abcd/development/intents/planned/itd-3-three.md"
	const edit = "\nA concurrent intent edit.\n"
	var landedEarly, ledgerHeld bool
	var writer chan error
	afterMigrateScan = func() {
		ledgerHeld = ledgerLockHeld(t, ir)
		landedEarly, writer = landsWithin(300*time.Millisecond, func() error {
			return lockedIntentAppend(repo, rel, edit)
		})
	}
	t.Cleanup(func() { afterMigrateScan = nil })

	if _, err := Migrate(MigrateRequest{RepoRoot: repo, IssuesRoot: ir, Apply: true}); err != nil {
		t.Fatal(err)
	}
	if writer == nil {
		t.Fatal("the apply never reached its writes")
	}
	if err := <-writer; err != nil {
		t.Fatal(err)
	}
	if !ledgerHeld {
		t.Error("the migration wrote without the ledger lock")
	}
	if landedEarly {
		t.Error("an intent writer landed between the migration's scan and its write")
	}
	got := readTree(t, repo, rel)
	if !strings.Contains(got, "\nrelated_issues: [iss-3]\n") || !strings.Contains(got, edit) {
		t.Errorf("the intent must carry both the migrated back-edge and the concurrent edit:\n%s", got)
	}
}
