package autonomy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// record_full_test.go — the diff review's major 3 (spc-2610180914469489,
// "Diff review round 1"): a record that fills must never stop the person
// ending the mode, and normal use must keep it within its 64 KiB.

func recordBytes(t *testing.T, r *Record) int {
	t.Helper()
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return len(raw) + 1
}

// TestAddStopCollapsesAnywhere: two copies blocked on different items,
// alternating, keep two entries, each counted, not one entry per stop.
func TestAddStopCollapsesAnywhere(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	r, _ := begin(nil, alice, now, time.Time{}, false)
	a := Stop{Kind: StopWorkOwed, Detail: "pull request #12 (feat/a) is behind its base", PossiblyForcedByCap: true}
	b := Stop{Kind: StopWorkOwed, Detail: "pull request #13 (feat/b) is failing its checks", PossiblyForcedByCap: true}
	for i := 0; i < 200; i++ {
		s := a
		if i%2 == 1 {
			s = b
		}
		r.AddStop(s, now.Add(time.Duration(i)*5*time.Minute))
	}
	if len(r.Stops) != 2 || r.Stops[0].Count != 100 || r.Stops[1].Count != 100 {
		t.Fatalf("alternating stops = %d entries %+v, want two of 100 each", len(r.Stops), r.Stops)
	}
}

// TestRecordStaysWithinBoundInUse: hundreds of distinct stops and hand-backs
// rotate out the oldest, counted and named on `mode back`; queued steps stop
// at their own bound with ErrQueueFull, nothing dropped; the record stays
// within 64 KiB throughout.
func TestRecordStaysWithinBoundInUse(t *testing.T) {
	now := at(t, "2026-10-18 21:00")
	r, _ := begin(nil, alice, now, time.Time{}, false)
	long := strings.Repeat("<", 240)
	for i := 0; i < 600; i++ {
		r.AddStop(Stop{Kind: StopWorkOwed, Detail: fmt.Sprintf("%d %s", i, long), PossiblyForcedByCap: true}, now.Add(time.Duration(i)*time.Minute))
		r.AddHandBack(fmt.Sprintf("feat/%d-%s", i, strings.Repeat("x", 100)), strings.Repeat("a", 40), now)
	}
	full := false
	for i := 0; i < 600 && !full; i++ {
		err := r.AddStep(fmt.Sprintf("%d %s", i, strings.Repeat("<", MaxStepLen-8)), "/work/a", now)
		if errors.Is(err, ErrQueueFull) {
			full = true
		} else if err != nil {
			t.Fatal(err)
		}
	}
	if !full {
		t.Fatalf("the queue took 600 steps of %d characters", MaxStepLen)
	}
	if n := recordBytes(t, r); n > MaxFileBytes {
		t.Fatalf("the record holds %d bytes, past %d", n, MaxFileBytes)
	}
	if r.DroppedStops == 0 || r.DroppedHandBacks == 0 {
		t.Fatalf("dropped stops %d, hand-backs %d: want the oldest of each rotated out", r.DroppedStops, r.DroppedHandBacks)
	}
	lines := strings.Join(r.Lines(), "\n")
	for _, want := range []string{fmt.Sprintf("%d earlier stops", r.DroppedStops), fmt.Sprintf("%d earlier items handed back", r.DroppedHandBacks)} {
		if !strings.Contains(lines, want) {
			t.Errorf("mode back's lines lack %q", want)
		}
	}
}

// fillRecord writes the stored record as one too full to write back: past
// the store's bound, as a record from before the bounds, or a harness, can
// leave it.
func fillRecord(t *testing.T, st Store, now time.Time) {
	t.Helper()
	r, _ := begin(nil, alice, now.Add(-time.Hour), time.Time{}, false)
	r.Queue = append(r.Queue, QueuedStep{Step: "approve the embargo close", At: now, From: "/work/a"})
	for i := 0; i < 400; i++ {
		r.Stops = append(r.Stops, Stop{Kind: StopWorkOwed, Detail: fmt.Sprintf("%d %s", i, strings.Repeat("x", 200)), Count: 1, First: now, Last: now})
	}
	raw, _ := json.MarshalIndent(r, "", "  ")
	if len(raw) <= MaxFileBytes {
		t.Fatalf("the fixture holds %d bytes, not past the bound", len(raw))
	}
	if err := st.WithLock(func() error { return os.WriteFile(st.RecordPath(), raw, 0o600) }); err != nil {
		t.Fatal(err)
	}
}

// TestBackEndsAnyRecord: `abcd mode back` ends the mode whatever state the
// record is in: past its bound, partly or wholly unparsable, not safely this
// account's, or with its lock held past the timeout. It prints what could be
// read, says what could not, and afterwards the record reads closed.
func TestBackEndsAnyRecord(t *testing.T) {
	type spoil func(t *testing.T, st Store, now time.Time)
	for _, c := range []struct {
		name      string
		spoil     spoil
		wantLine  string // a line Back prints
		wantNote  string // a note Back prints
		readAfter bool   // the store's own read sees the closed record after
	}{
		{"past its bound", fillRecord, "399 xxxxx", "past its 64 KiB", true},
		{"partly unparsable", func(t *testing.T, st Store, now time.Time) {
			raw := `{"schema_version":1,"set_by":"alice","start":"` + now.Add(-time.Hour).Format(time.RFC3339) + `","until":"` + now.Add(time.Hour).Format(time.RFC3339) + `","queue":[{"step":"approve the embargo close"}],"stops":"garbled"}`
			if err := os.WriteFile(st.RecordPath(), []byte(raw), 0o600); err != nil {
				t.Fatal(err)
			}
		}, "approve the embargo close", "could not be read", true},
		{"unparsable", func(t *testing.T, st Store, now time.Time) {
			if err := os.WriteFile(st.RecordPath(), []byte("{not json"), 0o600); err != nil {
				t.Fatal(err)
			}
		}, "", "could not be read", true},
		{"writable by others", func(t *testing.T, st Store, now time.Time) { mustChmod(t, st.RecordPath(), 0o666) }, "", "not safely this account's", true},
		{"lock writable by others", func(t *testing.T, st Store, now time.Time) { mustChmod(t, st.LockPath(), 0o666) }, "approve the embargo close", "lock", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			root, home := managedCopy(t)
			now := time.Now().Truncate(time.Second)
			o := fixedOptions(home, now)
			if _, err := SetAway(root, "", false, o); err != nil {
				t.Fatal(err)
			}
			if _, err := QueueStep(root, "approve the embargo close", o); err != nil {
				t.Fatal(err)
			}
			st, _ := openStore(home, root)
			c.spoil(t, st, now)
			res, err := Back(root, o)
			if err != nil {
				t.Fatalf("back: %v", err)
			}
			all := strings.Join(append(res.Lines, res.Notes...), "\n")
			if c.wantLine != "" && !strings.Contains(strings.Join(res.Lines, "\n"), c.wantLine) {
				t.Errorf("back printed %q, want a line with %q", all, c.wantLine)
			}
			if !strings.Contains(strings.Join(res.Notes, "\n"), c.wantNote) {
				t.Errorf("back's notes %q lack %q", res.Notes, c.wantNote)
			}
			if c.readAfter {
				rec, err := st.Read()
				if err != nil || rec == nil || rec.Closed == nil {
					t.Fatalf("after back the record reads %+v (%v), want closed", rec, err)
				}
				raw, _ := os.ReadFile(st.RecordPath())
				if len(raw) > MaxFileBytes {
					t.Errorf("the closed record holds %d bytes", len(raw))
				}
			}
			if _, holds, _ := st.Away(now); holds {
				t.Errorf("the mode still holds after back")
			}
		})
	}
}

// TestBackWithLockHeld: a lock held past the timeout does not keep the mode
// on: back closes the record without it and says so.
func TestBackWithLockHeld(t *testing.T) {
	root, home := managedCopy(t)
	now := time.Now().Truncate(time.Second)
	o := fixedOptions(home, now)
	if _, err := SetAway(root, "", false, o); err != nil {
		t.Fatal(err)
	}
	st, _ := openStore(home, root)
	var res BackResult
	var berr error
	if err := st.WithLock(func() error {
		res, berr = Back(root, o)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if berr != nil || res.State != "away" || !strings.Contains(strings.Join(res.Notes, " "), "lock") {
		t.Fatalf("back with the lock held: %+v (%v)", res, berr)
	}
	if rec, err := st.Read(); err != nil || rec.Closed == nil {
		t.Fatalf("after back the record reads %+v (%v), want closed", rec, err)
	}
}

// TestBeginCarriesAFullRecord: setting the mode again over an unclosed record
// past its bound carries what fits, counting what it dropped.
func TestBeginCarriesAFullRecord(t *testing.T) {
	root, home := managedCopy(t)
	now := time.Now().Truncate(time.Second)
	o := fixedOptions(home, now)
	st, _ := openStore(home, root)
	fillRecord(t, st, now)
	rec, err := SetAway(root, "", false, o)
	if err != nil {
		t.Fatalf("set over a full record: %v", err)
	}
	if rec.DroppedStops == 0 || len(rec.Queue) != 1 {
		t.Errorf("carried %d steps, dropped %d stops", len(rec.Queue), rec.DroppedStops)
	}
}
