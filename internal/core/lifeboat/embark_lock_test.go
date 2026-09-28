package lifeboat

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/intent"
)

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

// A record created at a planned target between embark's classification and
// its write is judged again under the locks and refuses the whole write, for
// the intent store (iss-2609262143265180) and the issue ledger
// (iss-2609262218306589) alike: it is never replaced without a conflict.
func TestEmbarkRejudgesARecordThatLandedAfterThePlan(t *testing.T) {
	for _, rel := range []string{
		".abcd/development/intents/drafts/itd-1-alpha.md",
		".abcd/work/issues/open/iss-1-open-thing.md",
	} {
		t.Run(filepath.Base(filepath.Dir(filepath.Dir(rel))), func(t *testing.T) {
			source := embarkableSourceFixture(t)
			dest := packSource(t, source)
			target := t.TempDir()
			// The store exists before the embark, as a target's usually does.
			for _, dir := range []string{".abcd/development/intents/drafts", ".abcd/work/issues/open"} {
				if err := os.MkdirAll(filepath.Join(target, dir), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			const planted = "A record written in the window.\n"
			afterEmbarkPlan = func() { mustWrite(t, filepath.Join(target, rel), []byte(planted)) }
			t.Cleanup(func() { afterEmbarkPlan = nil })

			res, err := EmbarkFrom(dest, target)
			if !errors.Is(err, ErrEmbarkConflicts) {
				t.Fatalf("want ErrEmbarkConflicts, got %v (written %d)", err, res.Written)
			}
			if res.Written != 0 || len(res.Conflicts) != 1 || res.Conflicts[0].Path != rel {
				t.Errorf("the refusal must name the one record and write nothing: written=%d conflicts=%+v", res.Written, res.Conflicts)
			}
			if got, _ := os.ReadFile(filepath.Join(target, rel)); string(got) != planted {
				t.Errorf("the record written in the window was replaced: %q", got)
			}
			if _, err := os.Stat(filepath.Join(target, ".abcd/development/decisions/adrs/0002-single-binary.md")); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("a refused embark wrote another record (err %v)", err)
			}
		})
	}
}

// Embark writes under the ledger lock and the intent store's lock: a ledger
// writer and an intent writer arriving while it writes wait for it.
func TestEmbarkWritesUnderTheLedgerAndIntentLocks(t *testing.T) {
	source := embarkableSourceFixture(t)
	dest := packSource(t, source)
	target := t.TempDir()
	if err := os.MkdirAll(filepath.Join(target, ".abcd/development/intents"), 0o755); err != nil {
		t.Fatal(err)
	}
	var ledgerEarly, intentEarly bool
	var ledgerW, intentW chan error
	duringEmbarkWrite = func() {
		ledgerEarly, ledgerW = landsWithin(300*time.Millisecond, func() error {
			return capture.WithLedgerLock(target, func() error { return nil })
		})
		intentEarly, intentW = landsWithin(300*time.Millisecond, func() error {
			return intent.WithMintLock(target, func() error { return nil })
		})
	}
	t.Cleanup(func() { duringEmbarkWrite = nil })

	res, err := EmbarkFrom(dest, target)
	if err != nil {
		t.Fatal(err)
	}
	if ledgerW == nil {
		t.Fatal("embark never reached its write")
	}
	if err := <-ledgerW; err != nil {
		t.Fatal(err)
	}
	if err := <-intentW; err != nil {
		t.Fatal(err)
	}
	if ledgerEarly {
		t.Error("a ledger writer landed while embark wrote: it does not hold the ledger lock")
	}
	if intentEarly {
		t.Error("an intent writer landed while embark wrote: it does not hold the intent store's lock")
	}
	if res.Written == 0 {
		t.Error("embark wrote nothing")
	}
}
