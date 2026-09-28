package spec

import (
	"errors"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// holdStoreLock takes the spec store's lock on another goroutine and returns
// once it is held, with the function that releases it and the channel the
// holder's result arrives on.
func holdStoreLock(t *testing.T, root string) (release func(), holder chan error) {
	t.Helper()
	held, rel := make(chan struct{}), make(chan struct{})
	holder = make(chan error, 1)
	go func() {
		holder <- WithStoreLock(root, func() error {
			close(held)
			<-rel
			return nil
		})
	}()
	<-held
	return func() { close(rel) }, holder
}

// landsWithin starts fn and reports whether it finished within d, and the
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

// Every writer of the spec store takes its one lock (iss-2609262218309668):
// with the lock held elsewhere, a close and a discard each wait for it rather
// than moving or removing a spec another writer is between reading and
// writing, and each completes once it is released.
func TestEverySpecWriterWaitsForTheStoreLock(t *testing.T) {
	for _, tc := range []struct {
		name  string
		write func(root string, sp Spec) error
		gone  string // the bucket the spec must have left
	}{
		{"close", func(root string, sp Spec) error { _, err := Close(root, sp.ID); return err }, StatusOpen},
		{"discard", func(root string, sp Spec) error { return Discard(root, sp) }, StatusOpen},
		{"mint", func(root string, _ Spec) error { _, err := Create(root, "itd-8", "another", ""); return err }, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			sp, err := Create(root, "itd-9", "my-feature", "")
			if err != nil {
				t.Fatal(err)
			}
			release, holder := holdStoreLock(t, root)
			landed, done := landsWithin(300*time.Millisecond, func() error { return tc.write(root, sp) })
			release()
			if err := <-holder; err != nil {
				t.Fatal(err)
			}
			if err := <-done; err != nil {
				t.Fatalf("%s after the store lock was released: %v", tc.name, err)
			}
			if landed {
				t.Errorf("%s finished while another holder had the spec store's lock: it wrote without taking it", tc.name)
			}
			if tc.gone != "" {
				if _, err := os.Lstat(filepath.Join(root, SpecsRelDir, tc.gone, filepath.Base(sp.Path))); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("%s left the spec in %s/ (err %v)", tc.name, tc.gone, err)
				}
			}
		})
	}
}

// Discard takes back only a spec in open/ under the store's own directory: a
// path it did not mint is refused before anything is removed.
func TestDiscardRefusesAPathOutsideOpen(t *testing.T) {
	root := t.TempDir()
	sp, err := Create(root, "itd-9", "my-feature", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		filepath.Join(SpecsRelDir, StatusClosed, filepath.Base(sp.Path)),
		filepath.Join(SpecsRelDir, StatusOpen, "..", "..", "intents", "x.md"),
		"README.md",
	} {
		bad := sp
		bad.Path = p
		if err := Discard(root, bad); err == nil {
			t.Errorf("Discard must refuse %q", p)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, sp.Path)); err != nil {
		t.Errorf("a refused discard removed the minted spec: %v", err)
	}
}

// WithStoreLock on a tree with no spec store runs fn without the lock and
// plants no store: taking the lock would create one, and with no store there is
// no spec record for fn to race.
func TestWithStoreLockPlantsNoStore(t *testing.T) {
	root := t.TempDir()
	ran := false
	if err := WithStoreLock(root, func() error { ran = true; return nil }); err != nil || !ran {
		t.Fatalf("fn must run: ran=%v err=%v", ran, err)
	}
	if _, err := os.Lstat(filepath.Join(root, SpecsRelDir)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("WithStoreLock planted a spec store (err %v)", err)
	}
}

// A holder past its budget is ErrStoreLockBusy, the sentinel the three-lock
// acquisition in the intent package retries on, and fn has not run.
func TestWithStoreLockWithinNamesItsBusyLock(t *testing.T) {
	root := t.TempDir()
	if _, err := Create(root, "itd-9", "my-feature", ""); err != nil {
		t.Fatal(err)
	}
	release, holder := holdStoreLock(t, root)
	ran := false
	err := WithStoreLockWithin(root, 30*time.Millisecond, func() error { ran = true; return nil })
	release()
	if herr := <-holder; herr != nil {
		t.Fatal(herr)
	}
	if !errors.Is(err, ErrStoreLockBusy) || ran {
		t.Errorf("a busy store lock must be ErrStoreLockBusy with fn not run: ran=%v err=%v", ran, err)
	}
}

// The spec store's lock is the innermost of the record-store locks (ledger,
// then intent, then spec). This package imports neither the ledger's nor the
// intent store's package, so nothing it runs under its own lock can request an
// earlier one: the order cannot be inverted from here. Were it to import one,
// the order would rest on every such call site instead of on the import graph.
func TestTheSpecStoreImportsNoEarlierLock(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	earlier := []string{"internal/core/capture", "internal/core/intent", "internal/core/lifeboat"}
	fset := token.NewFileSet()
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, e.Name(), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			for _, bad := range earlier {
				if strings.HasSuffix(p, bad) {
					t.Errorf("%s imports %s, which holds an earlier lock in the order", e.Name(), p)
				}
			}
		}
	}
}

// Close and Discard on a tree with no spec store have nothing to move or
// remove, so they must not create the store to lock it: Close refuses the
// id as not found and Discard succeeds, and neither plants
// .abcd/development/specs/ (iss-2609262342345159).
func TestAWriterOnATreeWithNoSpecStorePlantsNone(t *testing.T) {
	t.Run("close", func(t *testing.T) {
		root := t.TempDir()
		if _, err := Close(root, "spc-1"); err == nil || !strings.Contains(err.Error(), "not found") {
			t.Errorf("Close with no store must refuse the id as not found, got %v", err)
		}
		if _, err := os.Lstat(filepath.Join(root, SpecsRelDir)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Close planted a spec store (err %v)", err)
		}
	})
	t.Run("discard", func(t *testing.T) {
		root := t.TempDir()
		sp := Spec{Path: filepath.Join(SpecsRelDir, StatusOpen, "spc-1-my-feature.md")}
		if err := Discard(root, sp); err != nil {
			t.Errorf("Discard with no store has nothing to remove and must succeed, got %v", err)
		}
		if _, err := os.Lstat(filepath.Join(root, SpecsRelDir)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Discard planted a spec store (err %v)", err)
		}
	})
}

// A store removed after Close or Discard decided to lock it, and before the
// lock is taken, must not be re-planted empty by the lock: Close refuses the
// id as not found and Discard succeeds, and neither leaves
// .abcd/development/specs/ behind (iss-2609262342345159, the review's
// remove-between-check-and-lock note). beforeStoreLock stands in for the
// concurrent deletion.
func TestAStoreRemovedBeforeTheLockIsNotReplanted(t *testing.T) {
	removeStoreAtTheLock := func(t *testing.T, root string) {
		t.Helper()
		beforeStoreLock = func() {
			if err := os.RemoveAll(filepath.Join(root, SpecsRelDir)); err != nil {
				t.Fatalf("removing the store at the lock: %v", err)
			}
		}
		t.Cleanup(func() { beforeStoreLock = nil })
	}
	t.Run("close", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, SpecsRelDir, StatusOpen), 0o755); err != nil {
			t.Fatal(err)
		}
		removeStoreAtTheLock(t, root)
		if _, err := Close(root, "spc-1"); err == nil || !strings.Contains(err.Error(), "not found") {
			t.Errorf("Close on a store removed before the lock must refuse the id as not found, got %v", err)
		}
		if _, err := os.Lstat(filepath.Join(root, SpecsRelDir)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Close re-planted the removed spec store (err %v)", err)
		}
	})
	t.Run("discard", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, SpecsRelDir, StatusOpen), 0o755); err != nil {
			t.Fatal(err)
		}
		removeStoreAtTheLock(t, root)
		sp := Spec{Path: filepath.Join(SpecsRelDir, StatusOpen, "spc-1-my-feature.md")}
		if err := Discard(root, sp); err != nil {
			t.Errorf("Discard on a store removed before the lock has nothing to remove and must succeed, got %v", err)
		}
		if _, err := os.Lstat(filepath.Join(root, SpecsRelDir)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Discard re-planted the removed spec store (err %v)", err)
		}
	})
}
