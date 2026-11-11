package implement

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The whole-run marker: the command-form waiter at the first gate records
// itself under abcd's home while it waits and while its command runs, so the
// away mode's stop check can tell an owed item whose preflight is queued or
// running from one nobody works.

func markerPath(home string, pid int) string {
	return abcdhome.Path(home, WholeRunsLeaf, strconv.Itoa(pid)+".json")
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

// markHead is the commit a fixture's checkout is at.
const markHead = "0123456789abcdef0123456789abcdef01234567"

// yes is a marker's Preflight field set.
func yes() *bool { b := true; return &b }

// wholeReq is the waiter at pid 60000 gating the preflight's stages in a
// checkout at markHead.
func wholeReq(home string, clock *fakeClock) WaitRequest {
	req := waitReq(home, 60000, clock, poll{load: 3})
	req.Command, req.Dir, req.Words = true, "/work/wt", wholeRunCommand
	req.ReadHead = func(string) (string, error) { return markHead, nil }
	return req
}

// goodMarker is a well-formed marker for pid, started at start.
func goodMarker(pid int, start int64, began time.Time) WholeRunMarker {
	return WholeRunMarker{SchemaVersion: wholeRunSchema, PID: pid, StartUnix: start, Dir: "/x", Head: markHead, Preflight: yes(), Kind: WholeRunKind, Began: began}
}

// TestWholeRunWaiterRecordsItself: a whole run's waiter opens with its marker
// written, naming its pid, its start time as the table read it, its checkout,
// the checkout's HEAD, that its command is the preflight's stages, and the
// whole kind; the release of its hold removes it.
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
		m.StartUnix < start-holdStartSlack || m.StartUnix > start+holdStartSlack || m.Began.IsZero() ||
		m.Head != markHead || m.Preflight == nil || !*m.Preflight {
		t.Fatalf("marker = %+v, want pid 60000 started about %d in /work/wt at %s, the preflight, whole", m, start, markHead)
	}
	got, err := ReadWholeRunMarkers(home)
	if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0], m) {
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
		req.Command, req.Dir, req.Timeout, req.Words = true, "/work/wt", time.Minute, wholeRunCommand
		req.ReadHead = func(string) (string, error) { return markHead, nil }
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
	if err := os.Symlink(t.TempDir(), abcdhome.Path(home, WholeRunsLeaf)); err != nil {
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
	dir := abcdhome.Path(home, WholeRunsLeaf)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	live := machineload.Proc{PID: 777, PPID: 1, PGID: 777, UID: 501, Age: time.Hour, Name: "abcd"}
	// Each planted marker was written an hour before the sweeper's reading.
	plant := func(pid int, start int64) {
		raw, _ := json.Marshal(goodMarker(pid, start, waitT0.Add(-time.Hour)))
		if err := os.WriteFile(markerPath(home, pid), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plant(777, waitT0.Add(-time.Hour).Unix())   // alive, its start matching
	plant(888, waitT0.Add(-time.Hour).Unix())   // gone
	plant(777+1, waitT0.Add(-time.Hour).Unix()) // gone
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, poll{load: 3, procs: []machineload.Proc{live}})
	req.Command, req.Dir, req.Words = true, "/work/wt", wholeRunCommand
	req.ReadHead = func(string) (string, error) { return markHead, nil }
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
// checkout, missing its commit or whether its command is the preflight, with
// a negative slack, or not a regular file counts for nothing; a symlinked
// folder, or one others can write, is refused.
func TestReadWholeRunMarkersIsStrict(t *testing.T) {
	home := t.TempDir()
	if got, err := ReadWholeRunMarkers(home); err != nil || got != nil {
		t.Fatalf("an absent folder read %+v, %v; want none", got, err)
	}
	dir := abcdhome.Path(home, WholeRunsLeaf)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	good := WholeRunMarker{SchemaVersion: wholeRunSchema, PID: 4242, StartUnix: 1700000000, Dir: "/work/wt", Head: markHead, Preflight: yes(), Kind: WholeRunKind, Began: waitT0}
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
	write("4248.json", bad(func(m *WholeRunMarker) { m.PID = 4248; m.SchemaVersion = 1 }))
	write("4249.json", bad(func(m *WholeRunMarker) { m.PID = 4249; m.StartUnix = 0 }))
	const tail = `"head":"` + markHead + `","preflight":true,"kind":"whole","began":"2026-10-11T18:00:00Z"`
	write("4250.json", `{"schema_version":2,"pid":4250,"start_unix":1700000000,"dir":"/w",`+tail+`,"extra":1}`)
	write("4251.json", `{"schema_version":2,"pid":4251,"pid":4251,"start_unix":1700000000,"dir":"/w",`+tail+`}`)
	write("4252.json", `{"schema_version":2,"pid":4252,"start_unix":1700000000,"dir":"/w",`+tail+`} {}`)
	write("4253.json", `{"schema_version":2,"pid":4253,"start_unix":1700000000,"dir":"/`+strings.Repeat("a", maxWholeRunMarkerBytes)+`",`+tail+`}`)
	// A marker missing the commit, or whether its command is the preflight.
	write("4255.json", bad(func(m *WholeRunMarker) { m.PID = 4255; m.Head = "" }))
	write("4256.json", bad(func(m *WholeRunMarker) { m.PID = 4256; m.Head = "abc1234" }))
	write("4257.json", `{"schema_version":2,"pid":4257,"start_unix":1700000000,"dir":"/w","head":"`+markHead+`","kind":"whole","began":"2026-10-11T18:00:00Z"}`)
	write("4258.json", bad(func(m *WholeRunMarker) { m.PID = 4258; m.StartSlack = -time.Second }))
	write("1.json", bad(func(m *WholeRunMarker) { m.PID = 1 }))
	write("notes.txt", "4242")
	if err := os.Symlink(filepath.Join(dir, "4242.json"), filepath.Join(dir, "4254.json")); err != nil {
		t.Fatal(err)
	}
	got, err := ReadWholeRunMarkers(home)
	if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0], good) {
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
	if err := os.Symlink(dir, abcdhome.Path(linked, WholeRunsLeaf)); err != nil {
		t.Fatal(err)
	}
	if got, err := ReadWholeRunMarkers(linked); err == nil {
		t.Fatalf("a symlinked folder was read: %+v", got)
	}
}

// TestOnlyThePreflightRecordsItself: a whole-form command that is not the
// preflight's stages writes no marker and says nothing, even where a marker
// could not be written.
func TestOnlyThePreflightRecordsItself(t *testing.T) {
	for _, words := range [][]string{
		{"sh", "-c", "make preflight-stages"},
		{"make", "-C", "/elsewhere", "preflight-stages"},
		{"go", "test", "./..."},
	} {
		home := t.TempDir()
		if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(t.TempDir(), abcdhome.Path(home, WholeRunsLeaf)); err != nil {
			t.Fatal(err)
		}
		req := wholeReq(home, &fakeClock{now: waitT0})
		req.Words = words
		said := 0
		req.OnMarkNote = func(string) { said++ }
		res, err := WaitLoad(req)
		if err != nil || res.Status != WaitOpen {
			t.Fatalf("%q: result %+v, err %v, want open", words, res, err)
		}
		if res.MarkNote != "" || said != 0 || res.EndMark != nil {
			t.Fatalf("%q: MarkNote %q, said %d times, EndMark set %v; want no marker and nothing said", words, res.MarkNote, said, res.EndMark != nil)
		}
	}
}

// TestWholeRunMarkerNoteIsSaidAtItsLook: a marker that cannot be written is
// said once, at the look that tried to write it, before the wait goes on, so
// a run queued for hours does not wait to say the stop check cannot see it.
func TestWholeRunMarkerNoteIsSaidAtItsLook(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), abcdhome.Path(home, WholeRunsLeaf)); err != nil {
		t.Fatal(err)
	}
	clock := &fakeClock{now: waitT0}
	req := waitReq(home, 60000, clock, poll{load: 40})
	req.Command, req.Dir, req.Timeout, req.Words = true, "/work/wt", time.Minute, wholeRunCommand
	req.ReadHead = func(string) (string, error) { return markHead, nil }
	waits := 0
	req.OnWait = func(WaitResult) { waits++ }
	var notes []string
	var waitsAtNote int
	req.OnMarkNote = func(n string) { notes = append(notes, n); waitsAtNote = waits }
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitTimedOut {
		t.Fatalf("result %+v, err %v, want timed out", res, err)
	}
	if len(notes) != 1 || notes[0] != res.MarkNote || !strings.Contains(notes[0], "could not be written") {
		t.Fatalf("notes said %q, MarkNote %q; want the one note said once", notes, res.MarkNote)
	}
	if waitsAtNote != 0 || waits == 0 {
		t.Fatalf("the note was said after %d of %d waits, want before the first", waitsAtNote, waits)
	}
}

// TestWholeRunMarkerNeedsTheHead: a checkout whose HEAD cannot be read
// writes no marker, says so in MarkNote, and the wait still opens.
func TestWholeRunMarkerNeedsTheHead(t *testing.T) {
	home := t.TempDir()
	req := wholeReq(home, &fakeClock{now: waitT0})
	req.ReadHead = func(string) (string, error) { return "", errors.New("not a git repository") }
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitOpen {
		t.Fatalf("result %+v, err %v, want open", res, err)
	}
	if _, ok := readMarkerFile(t, home, 60000); ok {
		t.Fatal("a marker was written with no commit")
	}
	if !strings.Contains(res.MarkNote, "HEAD") {
		t.Fatalf("MarkNote = %q, want it to say the HEAD was not read", res.MarkNote)
	}
}

// TestWholeRunMarkerStartIsTakenBeforeTheRead: the marker's start time is
// measured back from the moment the table read began, as sessionname.Table
// measures a host's, so it is early by up to the read's length and never
// late, and the read's length is recorded as its slack. Here the read takes
// five seconds and the table gives the age as at the read's start.
func TestWholeRunMarkerStartIsTakenBeforeTheRead(t *testing.T) {
	home := t.TempDir()
	clock := &fakeClock{now: waitT0}
	req := wholeReq(home, clock)
	inner := req.Read
	req.Read = func() (machineload.Snapshot, error) {
		snap, err := inner()
		clock.Sleep(5 * time.Second)
		return snap, err
	}
	if _, err := WaitLoad(req); err != nil {
		t.Fatal(err)
	}
	m, ok := readMarkerFile(t, home, 60000)
	if !ok {
		t.Fatal("no marker")
	}
	start := waitT0.Add(-time.Second).Unix() // waiterAt's abcd, a second old at waitT0
	if d := m.StartUnix - start; d < -machineload.StartTolerance || d > machineload.StartTolerance {
		t.Errorf("StartUnix = %d, want %d: the start is measured from the read's beginning", m.StartUnix, start)
	}
	if m.StartSlack != 5*time.Second {
		t.Errorf("StartSlack = %s, want the read's five seconds", m.StartSlack)
	}
}

// TestWholeRunMarkerSweepLeavesALaterMarker: a marker written after the
// sweeper's table was read is left, its pid absent from that table or not:
// its waiter may have started after the read. A marker dated later than the
// reading by more than wholeRunLaterAllowance is judged as an old one, so a
// date in the future cannot keep a file the sweep would remove.
func TestWholeRunMarkerSweepLeavesALaterMarker(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(home, WholeRunsLeaf), 0o700); err != nil {
		t.Fatal(err)
	}
	for pid, began := range map[int]time.Time{991: waitT0.Add(time.Minute), 992: waitT0.Add(-time.Minute),
		996: waitT0.Add(wholeRunLaterAllowance + time.Second), 997: waitT0.Add(24 * time.Hour)} {
		raw, _ := json.Marshal(goodMarker(pid, waitT0.Unix(), began))
		if err := os.WriteFile(markerPath(home, pid), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := WaitLoad(wholeReq(home, &fakeClock{now: waitT0})); err != nil {
		t.Fatal(err)
	}
	if _, ok := readMarkerFile(t, home, 991); !ok {
		t.Error("a marker written after the sweeper's reading was swept")
	}
	if _, ok := readMarkerFile(t, home, 992); ok {
		t.Error("a marker written before the sweeper's reading, its pid gone, was kept")
	}
	for _, pid := range []int{996, 997} {
		if _, ok := readMarkerFile(t, home, pid); ok {
			t.Errorf("marker %d, dated past the reading and its allowance, its pid gone, was kept", pid)
		}
	}
}

// TestWholeRunMarkerSweepRemovesWhatDoesNotParse: a file named as a marker
// that a reader would refuse is removed, so such files cannot fill the
// entries a reader looks at, unless it parses as a later build's marker (a
// higher schema version) or its pid is in the table; a file not named as a
// marker is left.
func TestWholeRunMarkerSweepRemovesWhatDoesNotParse(t *testing.T) {
	home := t.TempDir()
	dir := abcdhome.Path(home, WholeRunsLeaf)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	old, _ := json.Marshal(map[string]any{"schema_version": 1, "pid": 994, "start_unix": 1, "dir": "/x", "kind": "whole", "began": waitT0})
	// A later build's marker, which this build cannot read, is that build's to
	// sweep; so is any file whose pid the table holds.
	later, _ := json.Marshal(map[string]any{"schema_version": wholeRunSchema + 1, "pid": 995, "start_unix": 1, "added": true})
	liveOld, _ := json.Marshal(map[string]any{"schema_version": 1, "pid": 777, "start_unix": 1, "dir": "/x", "kind": "whole", "began": waitT0})
	for name, body := range map[string]string{"993.json": "not json", "994.json": string(old), "995.json": string(later),
		"777.json": string(liveOld), "notes.txt": "x"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	live := machineload.Proc{PID: 777, PPID: 1, PGID: 777, UID: 501, Age: time.Hour, Name: "abcd"}
	req := waitReq(home, 60000, &fakeClock{now: waitT0}, poll{load: 3, procs: []machineload.Proc{live}})
	req.Command, req.Dir, req.Words = true, "/work/wt", wholeRunCommand
	req.ReadHead = func(string) (string, error) { return markHead, nil }
	if _, err := WaitLoad(req); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]bool{"993.json": false, "994.json": false, "995.json": true, "777.json": true, "notes.txt": true} {
		if _, err := os.Stat(filepath.Join(dir, name)); (err == nil) != want {
			t.Errorf("%s present = %v, want %v", name, err == nil, want)
		}
	}
}

// TestWholeRunMarkerEndsWithItsCommand: above one slot a whole run takes the
// settle hold, whose release is not the command's end, so the result's
// EndMark removes the marker once the command exits.
func TestWholeRunMarkerEndsWithItsCommand(t *testing.T) {
	home := t.TempDir()
	req := wholeReq(home, &fakeClock{now: waitT0})
	req.Getenv = func(k string) string {
		if k == PreflightSlotsEnv {
			return "2"
		}
		return ""
	}
	res, err := WaitLoad(req)
	if err != nil || res.Status != WaitOpen || res.Hold == nil || res.Hold.Form != HoldSettle {
		t.Fatalf("result %+v, err %v, want open on the settle hold", res, err)
	}
	if _, ok := readMarkerFile(t, home, 60000); !ok {
		t.Fatal("an open whole run left no marker")
	}
	if res.EndMark == nil {
		t.Fatal("the result carries no EndMark")
	}
	res.EndMark()
	if _, ok := readMarkerFile(t, home, 60000); ok {
		t.Fatal("EndMark left the marker")
	}
}

// TestReadCheckoutHead pins the one git read the marker makes, against a real
// repository: a branch's commit and a detached HEAD's are read in full, and
// an unborn branch is an error, so no marker is written for it.
func TestReadCheckoutHead(t *testing.T) {
	repo := gittest.NewRepo(t)
	if _, err := readCheckoutHead(repo.Root()); err == nil {
		t.Fatal("an unborn branch read as a commit")
	}
	repo.Write("a.txt", "a")
	repo.Commit("first")
	first := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	if got, err := readCheckoutHead(repo.Root()); err != nil || got != first {
		t.Fatalf("on a branch: %q, %v; want %s", got, err, first)
	}
	repo.Write("b.txt", "b")
	repo.Commit("second")
	repo.Git("checkout", "--quiet", "--detach", first)
	if got, err := readCheckoutHead(repo.Root()); err != nil || got != first {
		t.Fatalf("detached: %q, %v; want %s", got, err, first)
	}
}
