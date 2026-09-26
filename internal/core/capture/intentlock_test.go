package capture

import (
	"errors"
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

// Every path that holds both the ledger lock and the intent store's lock takes
// the ledger lock FIRST (iss-2609261254247117, iss-2609261941039204): with the
// intent lock held elsewhere, a resolve and a migrate apply each hold the
// ledger lock while they wait for it, and finish once it is released. The
// reverse order is unreachable in the core — the intent package cannot import
// this one — so the one order is the order.
func TestLedgerLockIsTakenBeforeTheIntentLock(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(t *testing.T) (ir string, verb func() error)
	}{
		{"resolve", func(t *testing.T) (string, func() error) {
			repo, ir, id, _, _ := issueLinkedFromAnIntent(t)
			return ir, func() error {
				_, err := Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: id, Resolution: "fixed", Impact: "fix"})
				return err
			}
		}},
		{"migrate --apply", func(t *testing.T) (string, func() error) {
			repo, ir, _ := migrateFixture(t)
			return ir, func() error {
				_, err := Migrate(MigrateRequest{RepoRoot: repo, IssuesRoot: ir, Apply: true})
				return err
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ir, verb := tc.run(t)
			repo := filepath.Dir(filepath.Dir(filepath.Dir(ir)))
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

			// The verb's own brief ledger holds before the transition (the
			// orphan sweep) come and go; the hold that WAITS on the intent lock
			// is the one that persists.
			deadline := time.Now().Add(3 * time.Second)
			steady := 0
			for steady < 3 {
				select {
				case err := <-done:
					close(release)
					t.Fatalf("%s finished while another holder had the intent lock (err %v): it wrote intent records without taking it", tc.name, err)
				default:
				}
				if time.Now().After(deadline) {
					close(release)
					<-done
					t.Fatalf("%s never held the ledger lock while waiting for the intent lock", tc.name)
				}
				if ledgerLockHeld(t, ir) {
					steady++
				} else {
					steady = 0
				}
				time.Sleep(100 * time.Millisecond)
			}
			close(release)
			if err := <-holder; err != nil {
				t.Fatal(err)
			}
			if err := <-done; err != nil {
				t.Fatalf("%s after the intent lock was released: %v", tc.name, err)
			}
		})
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
	var landedEarly bool
	var writer chan error
	afterMigrateScan = func() {
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
	if landedEarly {
		t.Error("an intent writer landed between the migration's scan and its write")
	}
	got := readTree(t, repo, rel)
	if !strings.Contains(got, "\nrelated_issues: [iss-3]\n") || !strings.Contains(got, edit) {
		t.Errorf("the intent must carry both the migrated back-edge and the concurrent edit:\n%s", got)
	}
}
