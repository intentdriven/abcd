package occupancy

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/sessionname"
)

// table is a process table read at now holding the given pids, each started
// an hour before it.
func table(now time.Time, pids ...int) sessionname.Table {
	t := sessionname.Table{ReadAt: now}
	for _, pid := range pids {
		t.Procs = append(t.Procs, machineload.Proc{PID: pid, PPID: 1, Name: "claude", Age: time.Hour})
	}
	return t
}

func read(t sessionname.Table) func() (sessionname.Table, error) {
	return func() (sessionname.Table, error) { return t, nil }
}

func hostOf(t sessionname.Table, pid int) sessionname.Host {
	for _, p := range t.Procs {
		if p.PID == pid {
			return sessionname.Host{PID: pid, StartUnix: t.ReadAt.Add(-p.Age).Unix()}
		}
	}
	panic("no such pid")
}

// TestSharersCountsLiveSessionsInOneCheckout: two running sessions that
// started in one checkout share it; a session elsewhere does not count, and a
// session whose host has ended stops counting and loses its marker.
func TestSharersCountsLiveSessionsInOneCheckout(t *testing.T) {
	home := t.TempDir()
	now := time.Now()
	tab := table(now, 101, 102, 103, 104)
	main := filepath.Join(home, "src", "repo")
	other := filepath.Join(home, "src", "other")
	for _, n := range []struct {
		id      string
		where   string
		hostPID int
	}{{"a", main, 101}, {"b", main, 102}, {"c", other, 103}, {"d", main, 104}} {
		if err := Note(home, n.id, n.where, hostOf(tab, n.hostPID), tab); err != nil {
			t.Fatal(err)
		}
	}

	later := table(now, 101, 102, 103) // d's host has ended
	got, err := sharers(home, main, read(later))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("sharers of the main checkout = %v, want [a b]", got)
	}
	if _, err := os.Stat(abcdhome.Path(home, dirLeaf, "d.json")); !os.IsNotExist(err) {
		t.Errorf("an ended session's marker must be removed: %v", err)
	}
}

// TestSharersIgnoresAReusedPID: a process given an ended host's pid later is
// a different process, and does not keep the marker alive.
func TestSharersIgnoresAReusedPID(t *testing.T) {
	home := t.TempDir()
	now := time.Now()
	tab := table(now, 201, 202)
	for id, pid := range map[string]int{"a": 201, "b": 202} {
		if err := Note(home, id, "/w", hostOf(tab, pid), tab); err != nil {
			t.Fatal(err)
		}
	}
	later := now.Add(2 * time.Hour)
	reused := sessionname.Table{ReadAt: later, Procs: []machineload.Proc{
		{PID: 201, PPID: 1, Name: "claude", Age: time.Minute},             // a new process on a's pid
		{PID: 202, PPID: 1, Name: "claude", Age: time.Hour + 2*time.Hour}, // b, still running
	}}
	got, err := sharers(home, "/w", read(reused))
	if err != nil || !slices.Equal(got, []string{"b"}) {
		t.Errorf("a reused pid must not count: %v %v", got, err)
	}
}

// TestSharersReadsNoTableForOneSession: a checkout one session works in is
// never shared, and costs no process-table read.
func TestSharersReadsNoTableForOneSession(t *testing.T) {
	home := t.TempDir()
	tab := table(time.Now(), 501)
	if err := Note(home, "a", "/w", hostOf(tab, 501), tab); err != nil {
		t.Fatal(err)
	}
	got, err := sharers(home, "/w", func() (sessionname.Table, error) {
		t.Fatal("the process table was read for one marker")
		return sessionname.Table{}, nil
	})
	if err != nil || len(got) != 0 {
		t.Errorf("one session shares nothing: %v %v", got, err)
	}
}

// TestNoteReplacesAndForgetRemoves: a session noted twice counts once, where
// it was noted last, and Forget takes it out.
func TestNoteReplacesAndForgetRemoves(t *testing.T) {
	home := t.TempDir()
	tab := table(time.Now(), 301, 302)
	if err := Note(home, "b", "/two", hostOf(tab, 302), tab); err != nil {
		t.Fatal(err)
	}
	if err := Note(home, "a", "/one", hostOf(tab, 301), tab); err != nil {
		t.Fatal(err)
	}
	if err := Note(home, "a", "/two", hostOf(tab, 301), tab); err != nil {
		t.Fatal(err)
	}
	if got, _ := sharers(home, "/two", read(tab)); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("the session must count where it was noted last: %v", got)
	}
	if err := Forget(home, "a"); err != nil {
		t.Fatal(err)
	}
	if got, _ := sharers(home, "/two", read(tab)); len(got) != 0 {
		t.Errorf("with a forgotten, one session is left and nothing is shared: %v", got)
	}
}

// TestNoteRefusesAnUnsafeSessionID: a session id is a file name here, so one
// that could leave the directory is refused.
func TestNoteRefusesAnUnsafeSessionID(t *testing.T) {
	home := t.TempDir()
	h := sessionname.Host{PID: 401, StartUnix: 1}
	for _, id := range []string{"", "..", "../x", "a/b"} {
		if err := Note(home, id, "/w", h, sessionname.Table{}); err == nil {
			t.Errorf("session id %q must be refused", id)
		}
	}
}

// TestMainWorkingTree tells a main working tree (a .git directory) from a
// linked worktree (a .git file).
func TestMainWorkingTree(t *testing.T) {
	main, linked := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(main, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linked, ".git"), []byte("gitdir: /x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !mainWorkingTree(main) || mainWorkingTree(linked) || mainWorkingTree(t.TempDir()) {
		t.Error("mainWorkingTree must hold only for a .git directory")
	}
}

// TestNoteReplacesAnotherMarkerUnderTheSameHost: one host runs one session at
// a time, so a /clear or a resume whose SessionEnd never ran leaves no second
// marker counting against a person working alone.
func TestNoteReplacesAnotherMarkerUnderTheSameHost(t *testing.T) {
	home := t.TempDir()
	tab := table(time.Now(), 601, 602)
	for _, n := range []struct {
		id  string
		pid int
	}{{"before-clear", 601}, {"other", 602}, {"after-clear", 601}} {
		if err := Note(home, n.id, "/w", hostOf(tab, n.pid), tab); err != nil {
			t.Fatal(err)
		}
	}
	got, err := sharers(home, "/w", read(tab))
	if err != nil || !slices.Equal(got, []string{"after-clear", "other"}) {
		t.Errorf("sharers = %v (%v), want [after-clear other]", got, err)
	}
	if _, err := os.Stat(abcdhome.Path(home, dirLeaf, "before-clear.json")); !os.IsNotExist(err) {
		t.Errorf("the replaced session's marker must be gone: %v", err)
	}
}

// TestNotePrunesEndedSessionsEverywhere: markers outside any shared checkout
// are pruned too, so the folder does not grow with every crashed session.
func TestNotePrunesEndedSessionsEverywhere(t *testing.T) {
	home := t.TempDir()
	now := time.Now()
	old := table(now, 701)
	if err := Note(home, "crashed", "/elsewhere", hostOf(old, 701), old); err != nil {
		t.Fatal(err)
	}
	cur := table(now, 702)
	if err := Note(home, "new", "/w", hostOf(cur, 702), cur); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(abcdhome.Path(home, dirLeaf, "crashed.json")); !os.IsNotExist(err) {
		t.Errorf("an ended session's marker must be pruned at the next note: %v", err)
	}
}

// TestSharersSkipsAMarkerAnotherAccountCanWrite: a marker writable by others
// is not necessarily this account's, and is not counted.
func TestSharersSkipsAMarkerAnotherAccountCanWrite(t *testing.T) {
	home := t.TempDir()
	tab := table(time.Now(), 801, 802)
	for id, pid := range map[string]int{"a": 801, "b": 802} {
		if err := Note(home, id, "/w", hostOf(tab, pid), tab); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(abcdhome.Path(home, dirLeaf, "b.json"), 0o666); err != nil {
		t.Fatal(err)
	}
	got, err := sharers(home, "/w", read(tab))
	if err != nil || len(got) != 0 {
		t.Errorf("a world-writable marker must not count: %v %v", got, err)
	}
}

// TestCheckRefusesOnlyASharedMainWorkingTree: the one judgement every writer
// makes. A main tree two live sessions share is refused; the same markers
// naming a linked worktree are not.
func TestCheckRefusesOnlyASharedMainWorkingTree(t *testing.T) {
	home := t.TempDir()
	main, linked := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(main, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linked, ".git"), []byte("gitdir: /x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tab4 := table(time.Now(), 901, 902, 903, 904)
	for _, n := range []struct {
		id    string
		where string
		pid   int
	}{{"m1", main, 901}, {"m2", main, 902}, {"l1", linked, 903}, {"l2", linked, 904}} {
		if err := Note(home, n.id, n.where, hostOf(tab4, n.pid), tab4); err != nil {
			t.Fatal(err)
		}
	}
	var shared *SharedError
	if err := Check(home, main, read(tab4)); !errors.As(err, &shared) || shared.Sessions != 2 {
		t.Errorf("a main tree two live sessions share must be refused; got %v", err)
	}
	if err := Check(home, linked, read(tab4)); err != nil {
		t.Errorf("a linked worktree must never be refused; got %v", err)
	}
}

// TestTreeOfWalksUpToTheNearestGit: a file's working tree is found by
// walking up to the nearest .git, directory or file, with no git
// subprocess, so the file tools' hook stays cheap. A file not yet written,
// in a folder not yet made, still names the tree it would land in; a path
// in no checkout names none.
func TestTreeOfWalksUpToTheNearestGit(t *testing.T) {
	base := t.TempDir()
	main := filepath.Join(base, "main")
	linked := filepath.Join(base, "linked")
	for _, d := range []string{filepath.Join(main, ".git"), filepath.Join(main, "a", "b"), linked, filepath.Join(base, "loose")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(linked, ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, want string
		ok         bool
	}{
		{filepath.Join(main, "a", "b", "file.go"), main, true},
		{filepath.Join(main, "new", "deeper", "file.md"), main, true},
		{filepath.Join(main, ".git", "config"), main, true},
		{filepath.Join(linked, "file.go"), linked, true},
		{filepath.Join(base, "loose", "file.go"), "", false},
	} {
		got, ok := TreeOf(tc.path)
		if ok != tc.ok || got != tc.want {
			t.Errorf("TreeOf(%s) = %q, %v; want %q, %v", tc.path, got, ok, tc.want, tc.ok)
		}
	}
}

// TestCheckPathJudgesTheTreeAFileLandsIn: CheckPath is Check on the working
// tree a file would be written into (iss-2610280051498465): refused in a
// main checkout two live sessions share, not in a linked worktree of it, and
// not outside any checkout.
func TestCheckPathJudgesTheTreeAFileLandsIn(t *testing.T) {
	home := t.TempDir()
	base := t.TempDir()
	main := filepath.Join(base, "main")
	linked := filepath.Join(base, "linked")
	for _, d := range []string{filepath.Join(main, ".git"), linked} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(linked, ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	tab := table(now, 101, 102)
	for id, pid := range map[string]int{"s1": 101, "s2": 102} {
		if err := Note(home, id, main, hostOf(tab, pid), tab); err != nil {
			t.Fatal(err)
		}
	}
	var shared *SharedError
	if err := CheckPath(home, filepath.Join(main, "x", "f.go"), read(tab)); !errors.As(err, &shared) || shared.Sessions != 2 {
		t.Errorf("a file in a shared main checkout: want a SharedError for 2 sessions, got %v", err)
	}
	if err := CheckPath(home, filepath.Join(linked, "f.go"), read(tab)); err != nil {
		t.Errorf("a file in a linked worktree must not be refused: %v", err)
	}
	if err := CheckPath(home, filepath.Join(base, "f.go"), read(tab)); err != nil {
		t.Errorf("a file in no checkout must not be refused: %v", err)
	}
}
