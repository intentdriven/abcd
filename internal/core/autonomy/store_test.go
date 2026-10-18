package autonomy

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

// managedCopy is a checkout with one commit and the local tier, under a HOME
// the test owns, so the away store is the test's own.
func managedCopy(t *testing.T) (root, home string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	repo := gittest.NewRepo(t)
	repo.Commit("seed")
	root, err := filepath.EvalSymlinks(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".abcd", ".work.local"), 0o700); err != nil {
		t.Fatal(err)
	}
	return root, home
}

func fixedOptions(home string, now time.Time) Options {
	return Options{
		Home: home,
		Now:  func() time.Time { return now },
		Who:  func(string) string { return "alice" },
	}
}

// TestConcurrentWritesUnderLock: queued steps racing keep every step, and a
// stop recorded after `mode back` re-reads inside the lock and does not turn
// the mode back on.
func TestConcurrentWritesUnderLock(t *testing.T) {
	root, home := managedCopy(t)
	now := time.Now().Truncate(time.Second)
	o := fixedOptions(home, now)
	if _, err := SetAway(root, "", false, o); err != nil {
		t.Fatal(err)
	}
	const n = 8
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := QueueStep(root, "step "+string(rune('a'+i)), o)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	st, err := OpenStore(home, root)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := st.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Queue) != n {
		t.Fatalf("queue holds %d steps after %d racing writes", len(rec.Queue), n)
	}

	// A stop check that read away before `mode back` must not write it back.
	if _, holds, _ := st.Away(now); !holds {
		t.Fatal("the record does not hold before `mode back`")
	}
	if _, err := Back(root, o); err != nil {
		t.Fatal(err)
	}
	wrote, err := st.RecordStop(Stop{Kind: StopWorkOwed, Detail: "feat/x"}, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if wrote {
		t.Error("a stop was recorded on a record `mode back` had ended")
	}
	after, err := st.Read()
	if err != nil {
		t.Fatal(err)
	}
	if after.Holds(now.Add(time.Minute)) || len(after.Stops) != 0 {
		t.Errorf("the late stop turned the mode back on or was kept: %+v", after)
	}
	if after.Closed == nil || after.Closed.By != "alice" || after.Closed.From != root {
		t.Errorf("`mode back` did not record who ended it and from which copy: %+v", after.Closed)
	}
}

// TestRecordOwnershipFailsOpen: a record or lock that is a symlink, not this
// account's, writable by others, or over 64 KiB reads as not away, with the
// reason naming the file, and a verb asked to change it refuses naming it.
func TestRecordOwnershipFailsOpen(t *testing.T) {
	for _, c := range []struct {
		name  string
		file  string // RecordFile or LockFile
		spoil func(t *testing.T, path string)
	}{
		{"record symlink", RecordFile, func(t *testing.T, p string) {
			real := p + ".real"
			mustRename(t, p, real)
			if err := os.Symlink(real, p); err != nil {
				t.Fatal(err)
			}
		}},
		{"record writable by others", RecordFile, func(t *testing.T, p string) { mustChmod(t, p, 0o666) }},
		{"record over 64 KiB", RecordFile, func(t *testing.T, p string) {
			raw, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			pad := strings.Repeat(" ", MaxFileBytes)
			if err := os.WriteFile(p, append(raw, pad...), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"record not this account's", RecordFile, func(t *testing.T, p string) {
			restore := fsutil.SwapOwnerUIDForTest(func(q string) (uint32, error) {
				if q == p {
					return uint32(os.Getuid()) + 1, nil
				}
				return fsutil.OwnerUID(q)
			})
			t.Cleanup(restore)
		}},
		{"record does not parse", RecordFile, func(t *testing.T, p string) {
			if err := os.WriteFile(p, []byte("{not json"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"lock symlink", LockFile, func(t *testing.T, p string) {
			other := filepath.Join(t.TempDir(), "elsewhere")
			if err := os.WriteFile(other, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(other, p); err != nil {
				t.Fatal(err)
			}
		}},
		{"lock writable by others", LockFile, func(t *testing.T, p string) { mustChmod(t, p, 0o666) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			root, home := managedCopy(t)
			now := time.Now().Truncate(time.Second)
			o := fixedOptions(home, now)
			if _, err := SetAway(root, "", false, o); err != nil {
				t.Fatal(err)
			}
			st, err := OpenStore(home, root)
			if err != nil {
				t.Fatal(err)
			}
			path := st.RecordPath()
			if c.file == LockFile {
				path = st.LockPath()
			}
			c.spoil(t, path)

			rec, holds, err := st.Away(now)
			if holds || rec != nil {
				t.Errorf("a spoilt %s reads as away", c.file)
			}
			if err == nil || !strings.Contains(err.Error(), c.file) {
				t.Errorf("Away err = %v, want one naming %s", err, c.file)
			}
			if _, err := QueueStep(root, "push feat/x", o); err == nil || !errors.Is(err, ErrUnsafe) || !strings.Contains(err.Error(), c.file) {
				t.Errorf("queue on a spoilt %s: err = %v, want ErrUnsafe naming it", c.file, err)
			}
			// Ending the mode never depends on the record (the diff review's
			// major 3): back succeeds, naming the spoilt file in a note, and
			// the mode no longer holds.
			res, err := Back(root, o)
			if err != nil || !strings.Contains(strings.Join(res.Notes, " "), c.file) {
				t.Errorf("back on a spoilt %s: %+v (%v), want it ended with a note naming the file", c.file, res, err)
			}
			if _, holds, _ := st.Away(now); holds {
				t.Errorf("back on a spoilt %s left the mode holding", c.file)
			}
		})
	}
}

// TestSetAwayRefusesWithNothingWritten: a past or over-cap --until, a checkout
// without the local tier, and no checkout at all are refused before the store
// is touched.
func TestSetAwayRefusesWithNothingWritten(t *testing.T) {
	root, home := managedCopy(t)
	now := time.Now().Truncate(time.Second)
	o := fixedOptions(home, now)
	past := now.Add(-time.Hour).Format("2006-01-02T15:04:05Z07:00")
	far := now.Add(MaxWindow + time.Hour).Format("2006-01-02T15:04:05Z07:00")
	if _, err := SetAway(root, past, false, o); !errors.Is(err, ErrUntilPast) {
		t.Errorf("past --until: err = %v", err)
	}
	if _, err := SetAway(root, far, false, o); !errors.Is(err, ErrUntilBeyondCap) {
		t.Errorf("--until past seven days: err = %v", err)
	}
	if err := os.RemoveAll(filepath.Join(root, ".abcd")); err != nil {
		t.Fatal(err)
	}
	if _, err := SetAway(root, "", false, o); !errors.Is(err, ErrNoLocalTier) {
		t.Errorf("no tier: err = %v", err)
	}
	if _, err := SetAway(t.TempDir(), "", false, o); err == nil {
		t.Error("outside a checkout the set was not refused")
	}
	if entries, _ := os.ReadDir(abcdhome.Path(home)); len(entries) != 0 {
		t.Errorf("a refused set left %d entries in the home store", len(entries))
	}
}

func mustChmod(t *testing.T, p string, m os.FileMode) {
	t.Helper()
	if err := os.Chmod(p, m); err != nil {
		t.Fatal(err)
	}
}

func mustRename(t *testing.T, a, b string) {
	t.Helper()
	if err := os.Rename(a, b); err != nil {
		t.Fatal(err)
	}
}
