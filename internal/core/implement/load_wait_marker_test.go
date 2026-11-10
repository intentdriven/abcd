package implement

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
)

// The whole-run marker: the command-form waiter at the first gate records
// itself under abcd's home while it waits and while its command runs, so the
// away mode's stop check can tell an owed item whose preflight is queued or
// running from one nobody works.

func markerPath(home string, pid int) string {
	return abcdhome.Path(home, wholeRunsLeaf, strconv.Itoa(pid)+".json")
}

func readMarkerFile(t *testing.T, home string, pid int) (WholeRunMarker, bool) {
	t.Helper()
	raw, err := os.ReadFile(markerPath(home, pid))
	if os.IsNotExist(err) {
		return WholeRunMarker{}, false
	}
	if err != nil {
		t.Fatal(err)
	}
	var m WholeRunMarker
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m, true
}

func wholeReq(home string, clock *fakeClock) WaitRequest {
	req := waitReq(home, 60000, clock, poll{load: 3})
	req.Command, req.Dir = true, "/work/wt"
	return req
}

// TestWholeRunWaiterRecordsItself: a whole run's waiter opens with its marker
// written, naming its pid, its start time as the table read it, its checkout
// and the whole kind; the release of its hold removes it.
func TestWholeRunWaiterRecordsItself(t *testing.T) {
	home := t.TempDir()
	clock := &fakeClock{now: waitT0}
	res, err := WaitLoad(wholeReq(home, clock))
	if err != nil || res.Status != WaitOpen || res.MarkNote != "" {
		t.Fatalf("result %+v, err %v, want open with no note", res, err)
	}
	m, ok := readMarkerFile(t, home, 60000)
	if !ok {
		t.Fatal("an open whole run left no marker")
	}
	start := waitT0.Add(-time.Second).Unix()
	if m.PID != 60000 || m.Dir != "/work/wt" || m.Kind != WholeRunKind || m.SchemaVersion != wholeRunSchema ||
		m.StartUnix < start-holdStartSlack || m.StartUnix > start+holdStartSlack || m.Began.IsZero() {
		t.Fatalf("marker = %+v, want pid 60000 started about %d in /work/wt, whole", m, start)
	}
	got, err := ReadWholeRunMarkers(home)
	if err != nil || len(got) != 1 || got[0] != m {
		t.Fatalf("read back %+v, err %v, want the one marker written", got, err)
	}
	if res.Hold == nil {
		t.Fatal("the whole run took no hold")
	}
	if err := ReleaseWaitHold(home, *res.Hold); err != nil {
		t.Fatal(err)
	}
	if _, ok := readMarkerFile(t, home, 60000); ok {
		t.Fatal("the hold's release left the marker")
	}
}

// TestOnlyAWholeRunRecordsItself: the plain form, a narrow run and the race
// gate write no marker; nor does a whole run's wait that timed out leave one.
func TestOnlyAWholeRunRecordsItself(t *testing.T) {
	cases := map[string]func(*WaitRequest){
		"the plain form": func(r *WaitRequest) { r.Command = false },
		"a narrow run":   func(r *WaitRequest) { r.Narrow = true },
		"the race gate":  func(r *WaitRequest) { r.Race = true },
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			req := wholeReq(home, &fakeClock{now: waitT0})
			edit(&req)
			if _, err := WaitLoad(req); err != nil {
				t.Fatal(err)
			}
			if _, ok := readMarkerFile(t, home, 60000); ok {
				t.Fatalf("%s wrote a whole run's marker", name)
			}
		})
	}
	t.Run("a timed-out whole run", func(t *testing.T) {
		home := t.TempDir()
		clock := &fakeClock{now: waitT0}
		req := waitReq(home, 60000, clock, poll{load: 40})
		req.Command, req.Dir, req.Timeout = true, "/work/wt", time.Minute
		seen := false
		req.OnWait = func(WaitResult) { _, seen = readMarkerFile(t, home, 60000) }
		res, err := WaitLoad(req)
		if err != nil || res.Status != WaitTimedOut {
			t.Fatalf("result %+v, err %v", res, err)
		}
		if !seen {
			t.Fatal("a whole run waiting at the gate had no marker")
		}
		if _, ok := readMarkerFile(t, home, 60000); ok {
			t.Fatal("a whole run's wait that timed out left its marker")
		}
	})
}

// TestWholeRunMarkerWriteFailureIsANote: a marker folder that cannot be
// written (a symlink stands at its name) is said in MarkNote, and the wait
// still opens and takes its hold.
func TestWholeRunMarkerWriteFailureIsANote(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), abcdhome.Path(home, wholeRunsLeaf)); err != nil {
		t.Fatal(err)
	}
	res, err := WaitLoad(wholeReq(home, &fakeClock{now: waitT0}))
	if err != nil || res.Status != WaitOpen || res.Hold == nil {
		t.Fatalf("result %+v, err %v, want the wait open and holding despite the marker", res, err)
	}
	if !strings.Contains(res.MarkNote, "could not be written") || strings.Contains(res.MarkNote, home) {
		t.Fatalf("MarkNote = %q, want it to say the marker could not be written, home redacted", res.MarkNote)
	}
}

// TestWholeRunMarkerSweepsTheDead: writing a marker removes each marker whose
// pid the table does not hold with its start time, and keeps a live one.
func TestWholeRunMarkerSweepsTheDead(t *testing.T) {
	home := t.TempDir()
	dir := abcdhome.Path(home, wholeRunsLeaf)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	live := machineload.Proc{PID: 777, PPID: 1, PGID: 777, UID: 501, Age: time.Hour, Name: "abcd"}
	plant := func(pid int, start int64) {
		raw, _ := json.Marshal(WholeRunMarker{SchemaVersion: wholeRunSchema, PID: pid, StartUnix: start, Dir: "/x", Kind: WholeRunKind, Began: waitT0})
		if err := os.WriteFile(markerPath(home, pid), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plant(777, waitT0.Add(-time.Hour).Unix())   // alive, its start matching
	plant(888, waitT0.Add(-time.Hour).Unix())   // gone
	plant(777+1, waitT0.Add(-time.Hour).Unix()) // gone
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, poll{load: 3, procs: []machineload.Proc{live}})
	req.Command, req.Dir = true, "/work/wt"
	if _, err := WaitLoad(req); err != nil {
		t.Fatal(err)
	}
	for pid, want := range map[int]bool{777: true, 888: false, 778: false, 60000: true} {
		if _, ok := readMarkerFile(t, home, pid); ok != want {
			t.Errorf("marker %d present = %v, want %v", pid, ok, want)
		}
	}
	// A marker whose pid is held by a later process (another start) is swept.
	plant(777, waitT0.Add(-3*time.Hour).Unix())
	if _, err := WaitLoad(req); err != nil {
		t.Fatal(err)
	}
	if _, ok := readMarkerFile(t, home, 777); ok {
		t.Error("a marker whose pid now names another process was kept")
	}
}

// TestReadWholeRunMarkersIsStrict: the markers are untrusted on read. Absent
// is none; a file that is too large, not one JSON document of the marker's
// shape, named for another pid, of another kind, with a relative or unclean
// checkout, or not a regular file counts for nothing; a symlinked folder, or
// one others can write, is refused.
func TestReadWholeRunMarkersIsStrict(t *testing.T) {
	home := t.TempDir()
	if got, err := ReadWholeRunMarkers(home); err != nil || got != nil {
		t.Fatalf("an absent folder read %+v, %v; want none", got, err)
	}
	dir := abcdhome.Path(home, wholeRunsLeaf)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	good := WholeRunMarker{SchemaVersion: wholeRunSchema, PID: 4242, StartUnix: 1700000000, Dir: "/work/wt", Kind: WholeRunKind, Began: waitT0}
	write := func(name string, v any) {
		var raw []byte
		switch x := v.(type) {
		case string:
			raw = []byte(x)
		default:
			raw, _ = json.Marshal(x)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("4242.json", good)
	bad := func(edit func(*WholeRunMarker)) WholeRunMarker { m := good; edit(&m); return m }
	write("4243.json", bad(func(m *WholeRunMarker) { m.PID = 4244 })) // named for another pid
	write("4245.json", bad(func(m *WholeRunMarker) { m.PID = 4245; m.Kind = "narrow" }))
	write("4246.json", bad(func(m *WholeRunMarker) { m.PID = 4246; m.Dir = "work/wt" }))
	write("4247.json", bad(func(m *WholeRunMarker) { m.PID = 4247; m.Dir = "/work/../etc" }))
	write("4248.json", bad(func(m *WholeRunMarker) { m.PID = 4248; m.SchemaVersion = 2 }))
	write("4249.json", bad(func(m *WholeRunMarker) { m.PID = 4249; m.StartUnix = 0 }))
	write("4250.json", `{"schema_version":1,"pid":4250,"start_unix":1700000000,"dir":"/w","kind":"whole","began":"2026-10-11T18:00:00Z","extra":1}`)
	write("4251.json", `{"schema_version":1,"pid":4251,"pid":4251,"start_unix":1700000000,"dir":"/w","kind":"whole","began":"2026-10-11T18:00:00Z"}`)
	write("4252.json", `{"schema_version":1,"pid":4252,"start_unix":1700000000,"dir":"/w","kind":"whole","began":"2026-10-11T18:00:00Z"} {}`)
	write("4253.json", `{"schema_version":1,"pid":4253,"start_unix":1700000000,"dir":"/`+strings.Repeat("a", maxWholeRunMarkerBytes)+`","kind":"whole","began":"2026-10-11T18:00:00Z"}`)
	write("1.json", bad(func(m *WholeRunMarker) { m.PID = 1 }))
	write("notes.txt", "4242")
	if err := os.Symlink(filepath.Join(dir, "4242.json"), filepath.Join(dir, "4254.json")); err != nil {
		t.Fatal(err)
	}
	got, err := ReadWholeRunMarkers(home)
	if err != nil || len(got) != 1 || got[0] != good {
		t.Fatalf("read %+v, err %v, want only the one well-formed marker", got, err)
	}

	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	if got, err := ReadWholeRunMarkers(home); err == nil {
		t.Fatalf("a folder others can write was read: %+v", got)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	linked := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(linked), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dir, abcdhome.Path(linked, wholeRunsLeaf)); err != nil {
		t.Fatal(err)
	}
	if got, err := ReadWholeRunMarkers(linked); err == nil {
		t.Fatalf("a symlinked folder was read: %+v", got)
	}
}
