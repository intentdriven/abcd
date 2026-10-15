package sessionname

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// machine is a fixture computer: a temporary home and a process table the
// test starts and ends host processes in. Every session it starts runs its
// hook under a shell whose parent is the session's host, the live shape.
type machine struct {
	t        *testing.T
	home     string
	procs    []machineload.Proc
	tableErr error
	timeout  time.Duration
}

func newMachine(t *testing.T) *machine {
	t.Helper()
	return &machine{t: t, home: t.TempDir(), timeout: 2 * time.Second}
}

// at is the Machine a hook whose parent is from sees.
func (m *machine) at(from int) Machine {
	return Machine{
		Home: m.home,
		From: from,
		ReadTable: func() (Table, error) {
			if m.tableErr != nil {
				return Table{}, m.tableErr
			}
			return Table{Procs: slices.Clone(m.procs), ReadAt: readAt}, nil
		},
		LockTimeout: m.timeout,
	}
}

// launch starts a host process at pid, an hour old, and the hook shell under
// it at pid+1, and returns the shell's pid: the process a hook's walk starts
// from.
func (m *machine) launch(pid int) int {
	m.procs = append(m.procs,
		proc(pid, 1, "claude", time.Hour),
		proc(pid+1, pid, "sh", time.Second))
	return pid + 1
}

// end removes the host process at pid and its shell: an exit, or a crash that
// ran no hook, which the register cannot tell apart and need not.
func (m *machine) end(pid int) {
	m.procs = slices.DeleteFunc(m.procs, func(p machineload.Proc) bool { return p.PID == pid || p.PID == pid+1 })
}

// start runs a new session's start hook at host pid in root, with session id
// sid, and returns the name it was given ("" for none).
func (m *machine) start(pid int, root, sid string) Outcome {
	m.t.Helper()
	return Start(m.at(m.launch(pid)), Request{Root: root, SessionID: sid, Source: "startup"})
}

// register reads the register file back, or nil when there is none.
func (m *machine) register() []byte {
	m.t.Helper()
	raw, err := os.ReadFile(abcdhome.Path(m.home, "sessions", "names.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		m.t.Fatal(err)
	}
	return raw
}

func mustName(t *testing.T, got Outcome, want string) {
	t.Helper()
	if got.Name != want {
		t.Fatalf("named %q (notes %q), want %q", got.Name, got.Notes, want)
	}
	if want != "" && len(got.Notes) != 0 {
		t.Fatalf("named %q with notes %q", got.Name, got.Notes)
	}
}

// TestStartNamesSessionsInPersonaOrder is C1 and C2: the first session is
// Alice, the next Bob, then Carol, and after the fourteenth persona the next
// is Alice Two.
func TestStartNamesSessionsInPersonaOrder(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	want := append(Bundled(), "Alice Two", "Bob Two")
	for i, name := range want {
		mustName(t, m.start(100*(i+1), root, "s"+name), name)
	}
}

// TestStartNamesAreUniqueAcrossRepositories is C6 and decision 7: a session in
// a second repository is not given a name a session in the first holds, even
// when its own list starts with that name.
func TestStartNamesAreUniqueAcrossRepositories(t *testing.T) {
	m := newMachine(t)
	mustName(t, m.start(100, t.TempDir(), "a"), "Alice")
	mustName(t, m.start(200, t.TempDir(), "b"), "Bob")
	mustName(t, m.start(300, writePersonas(t, `{"personas":[{"name":"Alice"},{"name":"Zed"}]}`), "c"), "Zed")
}

// TestStartDoesNotHandANameOutAgainWhileAnyRuns is C7: Bob ending while Alice
// and Carol run does not free Bob, so a peer that knew Bob is never handed a
// new session under the old name.
func TestStartDoesNotHandANameOutAgainWhileAnyRuns(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "a"), "Alice")
	mustName(t, m.start(200, root, "b"), "Bob")
	mustName(t, m.start(300, root, "c"), "Carol")
	m.end(200)
	mustName(t, m.start(400, root, "d"), "Dave")
}

// TestStartRestartsAtAliceOnceEveryNamedSessionHasEnded is C8 and the
// 2026-10-11 ruling: when every named session has ended, by exit or by a crash
// that ran no hook, naming starts again at Alice.
func TestStartRestartsAtAliceOnceEveryNamedSessionHasEnded(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "a"), "Alice")
	mustName(t, m.start(200, root, "b"), "Bob")
	m.end(100)
	m.end(200)
	mustName(t, m.start(300, root, "c"), "Alice")
	mustName(t, m.start(400, root, "d"), "Bob")
}

// TestStartCountsAReusedProcessIDAsEnded is C9: Alice's host gone and its
// process id now held by a process started later is an ended session, not a
// running one, so with nothing else running naming restarts at Alice.
func TestStartCountsAReusedProcessIDAsEnded(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "a"), "Alice")
	m.end(100)
	m.procs = append(m.procs, proc(100, 1, "vim", 10*time.Minute))
	mustName(t, m.start(200, root, "b"), "Alice")
	if bytes.Count(m.register(), []byte(`"session_id"`)) != 1 {
		t.Fatalf("the register keeps the ended entry:\n%s", m.register())
	}
}

// TestStartLeavesAPersonsOwnNameAlone is C4 at session start: a session that
// arrives with a title (a name given on the command line) is not named, and
// nothing is written.
func TestStartLeavesAPersonsOwnNameAlone(t *testing.T) {
	m := newMachine(t)
	for _, source := range []string{"startup", "fork"} {
		out := Start(m.at(m.launch(100)), Request{Root: t.TempDir(), SessionID: "z", Source: source, Title: "Zed"})
		if out.Name != "" || len(out.Notes) != 0 {
			t.Fatalf("%s with a title: %+v, want nothing", source, out)
		}
	}
	if m.register() != nil {
		t.Fatal("a session the person named wrote the register")
	}
}

// TestStartNamesNoSubAgent is C12: a payload carrying an agent id is a
// sub-agent's, and no name is handed out for it.
func TestStartNamesNoSubAgent(t *testing.T) {
	m := newMachine(t)
	out := Start(m.at(m.launch(100)), Request{Root: t.TempDir(), SessionID: "s", Source: "startup", AgentID: "agent-1"})
	if out.Name != "" || len(out.Notes) != 0 || m.register() != nil {
		t.Fatalf("a sub-agent was named: %+v", out)
	}
}

// TestStartForksNameTheForkedSession: a fork without a title is a new session
// and takes the next name, as a startup does.
func TestStartForksNameTheForkedSession(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "a"), "Alice")
	mustName(t, Start(m.at(m.launch(200)), Request{Root: root, SessionID: "f", Source: "fork"}), "Bob")
}

// TestCompactionChangesNothing is C5's compaction half: the host keeps the
// title, so the hook sets none and the register is left as it was.
func TestCompactionChangesNothing(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	from := m.launch(100)
	mustName(t, Start(m.at(from), Request{Root: root, SessionID: "a", Source: "startup"}), "Alice")
	before := m.register()
	out := Start(m.at(from), Request{Root: root, SessionID: "a", Source: "compact", Title: "Alice"})
	if out.Name != "" || len(out.Notes) != 0 || !bytes.Equal(m.register(), before) {
		t.Fatalf("compaction: %+v, register\n%s\nwant it unchanged from\n%s", out, m.register(), before)
	}
}

// TestResumeHoldsTheNameForTheResumedProcess is C5's resume half: Bob ended
// and resumed keeps Bob, is given no new title, and holds Bob under its new
// process so no other session is handed Bob while it runs.
func TestResumeHoldsTheNameForTheResumedProcess(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "a"), "Alice")
	mustName(t, m.start(200, root, "b"), "Bob")
	m.end(200)
	out := Start(m.at(m.launch(300)), Request{Root: root, SessionID: "b", Source: "resume", Title: "Bob"})
	if out.Name != "" || len(out.Notes) != 0 {
		t.Fatalf("resume set a title or noted: %+v", out)
	}
	// Alice ends: only the resumed Bob runs now, so neither Alice (C7) nor
	// Bob (held by the resumed process) is handed out.
	m.end(100)
	mustName(t, m.start(400, root, "c"), "Carol")
	mustName(t, m.start(500, root, "d"), "Dave")
	// The resumed session is owed no title at its prompt: the host kept it.
	if got := Resend(m.at(301), "b", "Bob"); got.Name != "" {
		t.Fatalf("the resumed session was sent %q", got.Name)
	}
}

// TestResumeAfterEveryoneEndedStillHoldsTheName: naming restarted at Alice
// while Bob was away, so resuming Bob adds Bob back to the names handed out.
func TestResumeAfterEveryoneEndedStillHoldsTheName(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "a"), "Alice")
	mustName(t, m.start(200, root, "b"), "Bob")
	m.end(100)
	m.end(200)
	Start(m.at(m.launch(300)), Request{Root: root, SessionID: "b", Source: "clear", Title: "Bob"})
	mustName(t, m.start(400, root, "c"), "Alice")
	mustName(t, m.start(500, root, "d"), "Carol")
}

// TestResumeOfAPersonsOwnNameHoldsNothing: a title abcd never hands out (a
// name the person chose) is the person's, and resuming it writes nothing.
func TestResumeOfAPersonsOwnNameHoldsNothing(t *testing.T) {
	m := newMachine(t)
	out := Start(m.at(m.launch(100)), Request{Root: t.TempDir(), SessionID: "z", Source: "resume", Title: "Zed the reviewer"})
	if out.Name != "" || len(out.Notes) != 0 || m.register() != nil {
		t.Fatalf("resuming a person's own name: %+v, register %s", out, m.register())
	}
}

// resends runs the prompt hook's naming for each prompt in turn and fails on
// the first whose title is not the one wanted.
func resends(t *testing.T, m *machine, prompts []struct{ sid, title, want string }) {
	t.Helper()
	for i, c := range prompts {
		got := Resend(m.at(0), c.sid, c.title)
		if got.Name != c.want || len(got.Notes) != 0 {
			t.Fatalf("prompt %d: Resend(%q, %q) = %+v, want %q\nregister:\n%s", i+1, c.sid, c.title, got, c.want, m.register())
		}
	}
}

// TestResendSendsTheReservedNameAtTheFirstPrompt is C1 and C3's register half
// (decision 9): session start reserves a name and sets no title, and the first
// prompt of a session with no title yet sets it, once. The order of the names
// is the order the sessions started in, whichever prompts first.
func TestResendSendsTheReservedNameAtTheFirstPrompt(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "s1"), "Alice")
	mustName(t, m.start(200, root, "s2"), "Bob")
	resends(t, m, []struct{ sid, title, want string }{
		{"s2", "", "Bob"},   // the second session prompts first, and is still Bob
		{"s2", "Bob", ""},   // the host now carries it: never again
		{"s2", "", ""},      // nor on a prompt that somehow lost it
		{"s1", "", "Alice"}, // the first session's first prompt
		{"s1", "Alice", ""},
		{"s3", "", ""}, // a session abcd reserved nothing for
	})
}

// TestResendTakesAMatchingTitleAsSent: a session whose title is already the
// name reserved for it is owed nothing (the host ignores a title equal to the
// one it has), and the entry is marked sent so no later prompt sends it.
func TestResendTakesAMatchingTitleAsSent(t *testing.T) {
	m := newMachine(t)
	mustName(t, m.start(100, t.TempDir(), "s1"), "Alice")
	resends(t, m, []struct{ sid, title, want string }{
		{"s1", "Alice", ""},
		{"s1", "", ""},
	})
	if !bytes.Contains(m.register(), []byte(`"sent": true`)) {
		t.Fatalf("the matching title was not marked sent:\n%s", m.register())
	}
}

// TestResendNeverReplacesAPersonsName is C4 at the prompt: a session the
// person renamed before its first prompt keeps their name, on that prompt and
// every later one.
func TestResendNeverReplacesAPersonsName(t *testing.T) {
	m := newMachine(t)
	mustName(t, m.start(100, t.TempDir(), "s1"), "Alice")
	resends(t, m, []struct{ sid, title, want string }{
		{"s1", "Zed", ""},
		{"s1", "", ""},
		{"s1", "Zed", ""},
	})
}

// TestResendTreatsASymlinkedFolderAsAbsent: a symlinked sessions folder is
// refused at session start, with its one note there; the prompt hook reads it
// as no register at all, rather than repeating the note on every prompt.
func TestResendTreatsASymlinkedFolderAsAbsent(t *testing.T) {
	m := newMachine(t)
	if err := os.MkdirAll(abcdhome.Path(m.home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), abcdhome.Path(m.home, "sessions")); err != nil {
		t.Fatal(err)
	}
	if got := Resend(m.at(0), "s1", ""); got.Name != "" || len(got.Notes) != 0 {
		t.Fatalf("Resend through a symlinked folder = %+v, want nothing", got)
	}
}

// TestResumeOfANameARunningSessionNowHoldsTakesTheNextFreeName is C5's one
// exception (decision 10): every named session ended, so naming restarted and
// a new session was given Alice; the first Alice, resumed, would make two.
// It is given the next free name, and its first prompt sets that name even
// though the session carries a title, because the title is a persona name a
// running session now holds.
func TestResumeOfANameARunningSessionNowHoldsTakesTheNextFreeName(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "a"), "Alice")
	resends(t, m, []struct{ sid, title, want string }{{"a", "", "Alice"}})
	m.end(100)
	mustName(t, m.start(200, root, "b"), "Alice")
	resumed := Start(m.at(m.launch(300)), Request{Root: root, SessionID: "a", Source: "resume", Title: "Alice"})
	mustName(t, resumed, "Bob")
	resends(t, m, []struct{ sid, title, want string }{
		{"b", "", "Alice"},
		{"a", "Alice", "Bob"}, // replaced: the one case a title is
		{"a", "Bob", ""},
		{"a", "Alice", ""}, // once only
	})
	// Bob is held by the resumed session: the next session is Carol.
	mustName(t, m.start(400, root, "c"), "Carol")
}

// TestResumeHoldsTheNameAcrossAClearInTheSameProcess: a clear can give the
// session a new id under the same host process, whose start this reading puts
// a second from the last one. It is the same host, within the tolerance
// liveness uses, so the name is held, not treated as another session's.
func TestResumeHoldsTheNameAcrossAClearInTheSameProcess(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	from := m.launch(100)
	mustName(t, Start(m.at(from), Request{Root: root, SessionID: "a", Source: "startup"}), "Alice")
	for i := range m.procs {
		if m.procs[i].PID == 100 {
			m.procs[i].Age -= time.Second
		}
	}
	out := Start(m.at(from), Request{Root: root, SessionID: "a2", Source: "clear", Title: "Alice"})
	if out.Name != "" || len(out.Notes) != 0 {
		t.Fatalf("clear in the same process: %+v, want the name held", out)
	}
	if reg := m.register(); bytes.Count(reg, []byte(`"session_id"`)) != 1 || !bytes.Contains(reg, []byte(`"a2"`)) {
		t.Fatalf("the register does not hold Alice once, for a2:\n%s", reg)
	}
}

// TestPruneKeepsAnEntryTooRecentToJudge: an entry whose host started after
// the table was read, or within the tolerance before it, is absent from that
// table only because it is too new to be in it. It is a running session, and
// it survives the next start.
func TestPruneKeepsAnEntryTooRecentToJudge(t *testing.T) {
	for name, age := range map[string]time.Duration{
		"after the read":           -5 * time.Second,
		"a second before the read": time.Second,
	} {
		t.Run(name, func(t *testing.T) {
			m := newMachine(t)
			root := t.TempDir()
			m.procs = append(m.procs, proc(100, 1, "claude", age), proc(101, 100, "sh", 0))
			mustName(t, Start(m.at(101), Request{Root: root, SessionID: "a", Source: "startup"}), "Alice")
			m.end(100)
			mustName(t, m.start(200, root, "b"), "Bob")
			if !bytes.Contains(m.register(), []byte(`"session_id": "a"`)) {
				t.Fatalf("the entry too recent to judge was pruned:\n%s", m.register())
			}
		})
	}
}

// TestPruneNothingOnASlowRead: a process table that took more than a second
// to read stamps every process's start time with that much doubt, so no entry
// is judged ended on it.
func TestPruneNothingOnASlowRead(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "a"), "Alice")
	m.end(100)
	slow := m.at(m.launch(200))
	read := slow.ReadTable
	slow.ReadTable = func() (Table, error) {
		tab, err := read()
		tab.Took = 1500 * time.Millisecond
		return tab, err
	}
	mustName(t, Start(slow, Request{Root: root, SessionID: "b", Source: "startup"}), "Bob")
	if !bytes.Contains(m.register(), []byte(`"session_id": "a"`)) {
		t.Fatalf("a slow read pruned an entry:\n%s", m.register())
	}
	// A read within the second judges as before: Alice has ended, Bob runs.
	mustName(t, m.start(300, root, "c"), "Carol")
	if bytes.Contains(m.register(), []byte(`"session_id": "a"`)) {
		t.Fatalf("a fast read kept the ended entry:\n%s", m.register())
	}
}

// TestReadTableStampsTheMomentBeforeTheRead: a process's age is measured back
// from the moment the read began, not the moment it returned, so the time the
// read took under load never moves a start time later than it was; and how
// long it took is kept, for prune to judge.
func TestReadTableStampsTheMomentBeforeTheRead(t *testing.T) {
	t0 := time.Unix(1_800_000_000, 0)
	clock := t0
	tab, err := readTable(func() (machineload.Snapshot, error) {
		clock = clock.Add(3 * time.Second)
		return machineload.Snapshot{HasProcs: true, Procs: []machineload.Proc{proc(100, 1, "claude", time.Hour)}}, nil
	}, func() time.Time { return clock })
	if err != nil {
		t.Fatal(err)
	}
	if !tab.ReadAt.Equal(t0) || tab.Took != 3*time.Second {
		t.Fatalf("ReadAt %s, Took %s; want %s and 3s", tab.ReadAt, tab.Took, t0)
	}
}

// TestStartHandsDistinctNamesToConcurrentSessions: twenty sessions starting at
// once, each under its own host, against one register and its real lock, are
// given twenty different names.
func TestStartHandsDistinctNamesToConcurrentSessions(t *testing.T) {
	const n = 20
	m := newMachine(t)
	m.timeout = 30 * time.Second
	root := t.TempDir()
	froms := make([]int, n)
	for i := range froms {
		froms[i] = m.launch(100 * (i + 1))
	}
	names := make([]Outcome, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			names[i] = Start(m.at(froms[i]), Request{Root: root, SessionID: fmt.Sprintf("s%d", i), Source: "startup"})
		})
	}
	wg.Wait()
	seen := map[string]bool{}
	for i, out := range names {
		if out.Name == "" || seen[out.Name] {
			t.Fatalf("session %d: %+v; names so far %v", i, out, seen)
		}
		seen[out.Name] = true
	}
}

// TestResendWithoutARegisterIsNothing: a machine where no session was ever
// named has no register, and a prompt neither creates one nor notes anything.
func TestResendWithoutARegisterIsNothing(t *testing.T) {
	m := newMachine(t)
	got := Resend(m.at(0), "s1", "Alice")
	if got.Name != "" || len(got.Notes) != 0 {
		t.Fatalf("Resend = %+v, want nothing", got)
	}
	if _, err := os.Lstat(abcdhome.Path(m.home)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a prompt created abcd's home: %v", err)
	}
}

// TestStartLeavesTheSessionUnnamedPastTheLastRound: nine passes through a
// one-name list are nine sessions; the tenth starts unnamed and says why.
func TestStartLeavesTheSessionUnnamedPastTheLastRound(t *testing.T) {
	m := newMachine(t)
	root := writePersonas(t, `{"personas":[{"name":"Zed"}]}`)
	for i, name := range Sequence([]string{"Zed"}) {
		mustName(t, m.start(100*(i+1), root, name), name)
	}
	out := m.start(5000, root, "tenth")
	if out.Name != "" || len(out.Notes) != 1 {
		t.Fatalf("past the last round: %+v, want unnamed with one note", out)
	}
}

// TestTheRegisterIsPrivate: the folder and the file are the account's alone.
func TestTheRegisterIsPrivate(t *testing.T) {
	m := newMachine(t)
	mustName(t, m.start(100, t.TempDir(), "a"), "Alice")
	for p, want := range map[string]os.FileMode{
		abcdhome.Path(m.home, "sessions"):               abcdhome.DirMode,
		abcdhome.Path(m.home, "sessions", "names.json"): abcdhome.FileMode,
	} {
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != want {
			t.Errorf("%s has mode %04o, want %04o", p, fi.Mode().Perm(), want)
		}
	}
}

// TestNamingNeverFailsASession is C11: a register that cannot be read, locked,
// parsed or written, a process table that cannot be read, a platform whose
// table abcd does not read and a missing home each leave the session unnamed
// with exactly one note, and hand out no name.
func TestNamingNeverFailsASession(t *testing.T) {
	sessions := func(m *machine) string { return abcdhome.Path(m.home, "sessions") }
	names := func(m *machine) string { return filepath.Join(sessions(m), "names.json") }
	write := func(t *testing.T, p, body string, mode os.FileMode) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(p, mode); err != nil {
			t.Fatal(err)
		}
	}
	for name, c := range map[string]struct {
		arrange func(t *testing.T, m *machine)
		mention string
	}{
		"unparsable register": {func(t *testing.T, m *machine) { write(t, names(m), "{not json", 0o600) }, "names.json"},
		"register writable by others": {func(t *testing.T, m *machine) {
			write(t, names(m), `{"schema_version":1,"handed":[],"live":[]}`, 0o622)
		}, "names.json"},
		"register owned by another account": {func(t *testing.T, m *machine) {
			write(t, names(m), `{"schema_version":1,"handed":[],"live":[]}`, 0o600)
			t.Cleanup(fsutil.SwapOwnerUIDForTest(func(string) (uint32, error) { return 4242, nil }))
		}, "names.json"},
		"register is a folder": {func(t *testing.T, m *machine) {
			if err := os.MkdirAll(names(m), 0o700); err != nil {
				t.Fatal(err)
			}
		}, "names.json"},
		"sessions folder is a symlink": {func(t *testing.T, m *machine) {
			if err := os.MkdirAll(abcdhome.Path(m.home), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(t.TempDir(), sessions(m)); err != nil {
				t.Fatal(err)
			}
		}, "symlink"},
		"sessions folder writable by others": {func(t *testing.T, m *machine) {
			if err := os.MkdirAll(sessions(m), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(sessions(m), 0o777); err != nil {
				t.Fatal(err)
			}
		}, "sessions"},
		"process table unreadable": {func(t *testing.T, m *machine) {
			m.tableErr = errors.New("could not read the process table (/bin/ps: exit status 1)")
		}, "process table"},
		"platform not read": {func(t *testing.T, m *machine) { m.tableErr = machineload.ErrUnsupported }, "platform"},
		"no home":           {func(t *testing.T, m *machine) { m.home = "" }, "home"},
		"lock held": {func(t *testing.T, m *machine) {
			m.timeout = 100 * time.Millisecond
			holdLock(t, m.home)
		}, "lock"},
	} {
		t.Run(name, func(t *testing.T) {
			m := newMachine(t)
			c.arrange(t, m)
			began := time.Now()
			out := m.start(100, t.TempDir(), "a")
			if took := time.Since(began); took > 1500*time.Millisecond {
				t.Fatalf("naming took %s", took)
			}
			if out.Name != "" || len(out.Notes) != 1 {
				t.Fatalf("Start = %+v, want unnamed with one note", out)
			}
			if !strings.Contains(out.Notes[0], c.mention) {
				t.Fatalf("the note %q does not say %q", out.Notes[0], c.mention)
			}
			if m.home != "" && strings.Contains(out.Notes[0], m.home) {
				t.Fatalf("the note %q carries the home path", out.Notes[0])
			}
		})
	}
}

// holdLock takes the register's lock from another descriptor and holds it
// until the test ends: flock conflicts between two opens even in one process.
func holdLock(t *testing.T, home string) {
	t.Helper()
	dir, err := fsutil.EnsureHomeScope(home, abcdhome.Rel("sessions"), abcdhome.DirMode)
	if err != nil {
		t.Fatal(err)
	}
	held, release := make(chan struct{}), make(chan struct{})
	go func() {
		_ = fsutil.WithFileLockIn(dir, lockName, time.Second, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	t.Cleanup(func() { close(release); dir.Close() })
}

// TestLiveWaitsTwoSecondsForTheLock pins the production machine: the lock's
// wait is the spec's two seconds, and the host walk starts at the hook's
// parent.
func TestLiveWaitsTwoSecondsForTheLock(t *testing.T) {
	m := Live()
	if m.LockTimeout != 2*time.Second {
		t.Fatalf("Live().LockTimeout = %s, want 2s", m.LockTimeout)
	}
	if m.From != os.Getppid() {
		t.Fatalf("Live().From = %d, want the parent %d", m.From, os.Getppid())
	}
}

// slowly makes mm's process-table read report that it took took.
func slowly(mm Machine, took time.Duration) Machine {
	read := mm.ReadTable
	mm.ReadTable = func() (Table, error) {
		tab, err := read()
		tab.Took = took
		return tab, err
	}
	return mm
}

// age sets the age the table reports for the process at pid.
func (m *machine) age(pid int, age time.Duration) {
	for i := range m.procs {
		if m.procs[i].PID == pid {
			m.procs[i].Age = age
		}
	}
}

// startAfterASlowRead runs Alice's session start under a table read that took
// three seconds and sampled her host's age at its end, so the start time it
// records is three seconds early; then a fast later reading puts the host's
// start where it truly is.
func startAfterASlowRead(t *testing.T, m *machine, root string) {
	t.Helper()
	from := m.launch(100)
	m.age(100, time.Hour+3*time.Second)
	mustName(t, Start(slowly(m.at(from), 3*time.Second), Request{Root: root, SessionID: "a", Source: "startup"}), "Alice")
	m.age(100, time.Hour)
}

// TestASlowReadAtReservationDoesNotEndTheSession: the entry records how long
// the read that stamped its host took, and its tolerance widens by that, so
// the still-running session is alive under a fast later read and Bob is the
// next name, not a second Alice.
func TestASlowReadAtReservationDoesNotEndTheSession(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	startAfterASlowRead(t, m, root)
	recorded := m.register()
	mustName(t, m.start(200, root, "b"), "Bob")
	if !bytes.Contains(recorded, []byte(`"host_read_ms": 3000`)) {
		t.Fatalf("the entry does not record the read's length:\n%s", recorded)
	}
	if !bytes.Contains(m.register(), []byte(`"session_id": "a"`)) {
		t.Fatalf("the running session was judged ended:\n%s", m.register())
	}
}

// TestAnEntryWithoutTheReadLengthKeepsTheTolerance: the same entry as written
// before the read's length was recorded is judged with the tolerance alone,
// which is what the widening changes: three seconds early is a different
// process, so the entry is pruned and naming restarts at Alice.
func TestAnEntryWithoutTheReadLengthKeepsTheTolerance(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	startAfterASlowRead(t, m, root)
	p := abcdhome.Path(m.home, "sessions", "names.json")
	legacy := bytes.Replace(m.register(), []byte(`,
      "host_read_ms": 3000`), nil, 1)
	if bytes.Contains(legacy, []byte("host_read_ms")) {
		t.Fatalf("the fixture did not strip the field:\n%s", legacy)
	}
	if err := os.WriteFile(p, legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	mustName(t, m.start(200, root, "b"), "Alice")
}

// TestPruneKeepsATooRecentEntryRecordedBySlowRead: an entry whose host started
// after this table was read is kept unjudged; recorded by a three-second read,
// its start may sit three seconds before this reading, and it is still too
// recent to judge.
func TestPruneKeepsATooRecentEntryRecordedBySlowRead(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(m.home, "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	reg := fmt.Sprintf(`{"schema_version":1,"handed":["Alice"],"live":[{"name":"Alice","session_id":"a","host_pid":100,"host_start_unix":%d,"sent":false,"host_read_ms":3000}]}`, readAt.Unix()-3)
	if err := os.WriteFile(abcdhome.Path(m.home, "sessions", "names.json"), []byte(reg), 0o600); err != nil {
		t.Fatal(err)
	}
	mustName(t, m.start(200, root, "b"), "Bob")
}

// TestAClearReadSlowlyStillHoldsTheName: a clear in the same process whose
// own read was slow puts the host's start early by that read; matched against
// the entry with the clear's read length, it is the same host, and the name
// is held rather than replaced.
func TestAClearReadSlowlyStillHoldsTheName(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	from := m.launch(100)
	mustName(t, Start(m.at(from), Request{Root: root, SessionID: "a", Source: "startup"}), "Alice")
	m.age(100, time.Hour+3*time.Second)
	out := Start(slowly(m.at(from), 3*time.Second), Request{Root: root, SessionID: "a2", Source: "clear", Title: "Alice"})
	if out.Name != "" || len(out.Notes) != 0 {
		t.Fatalf("a slowly read clear in the same process: %+v, want the name held", out)
	}
	if reg := m.register(); bytes.Count(reg, []byte(`"session_id"`)) != 1 || !bytes.Contains(reg, []byte(`"a2"`)) {
		t.Fatalf("the register does not hold Alice once, for a2:\n%s", reg)
	}
}

// TestAResumeBeforeTheFirstPromptKeepsTheReservedName: a session reserved
// Alice and its host ended before its first prompt, so it carries no title.
// Resumed or cleared, its entry is re-keyed to the process now running it,
// keeping the name and the title still owed; a later start does not prune it
// and hand Alice out again, and the first prompt sets Alice.
func TestAResumeBeforeTheFirstPromptKeepsTheReservedName(t *testing.T) {
	for _, source := range []string{"resume", "clear"} {
		t.Run(source, func(t *testing.T) {
			m := newMachine(t)
			root := t.TempDir()
			mustName(t, m.start(100, root, "a"), "Alice")
			m.end(100)
			mustName(t, Start(m.at(m.launch(300)), Request{Root: root, SessionID: "a", Source: source}), "Alice")
			reg := m.register()
			if !bytes.Contains(reg, []byte(`"host_pid": 300`)) || bytes.Contains(reg, []byte(`"host_pid": 100`)) || !bytes.Contains(reg, []byte(`"sent": false`)) {
				t.Fatalf("the entry was not re-keyed to the running host, still owed:\n%s", reg)
			}
			mustName(t, m.start(400, root, "b"), "Bob")
			resends(t, m, []struct{ sid, title, want string }{
				{"a", "", "Alice"},
				{"b", "", "Bob"},
			})
		})
	}
}

// TestAResumeWithoutATitleKeepsASentEntrySent: an entry whose title the
// prompt hook already dealt with stays dealt with when it is re-keyed, so a
// resume that somehow arrives without the title is not sent the name again.
func TestAResumeWithoutATitleKeepsASentEntrySent(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "a"), "Alice")
	resends(t, m, []struct{ sid, title, want string }{{"a", "", "Alice"}})
	m.end(100)
	out := Start(m.at(m.launch(300)), Request{Root: root, SessionID: "a", Source: "resume"})
	if out.Name != "" || len(out.Notes) != 0 {
		t.Fatalf("re-keying a sent entry: %+v, want nothing owed", out)
	}
	if reg := m.register(); !bytes.Contains(reg, []byte(`"host_pid": 300`)) || !bytes.Contains(reg, []byte(`"sent": true`)) {
		t.Fatalf("the sent entry was not re-keyed as sent:\n%s", reg)
	}
	mustName(t, m.start(400, root, "b"), "Bob")
	resends(t, m, []struct{ sid, title, want string }{{"a", "", ""}})
}

// TestAResumeWithoutATitleOrAnEntryTakesAFreshName: a session with no title
// and no entry (its start reserved nothing, or the entry is gone) reserves the
// next free name at a resume, a clear or a compaction, exactly as a startup
// does, and its first prompt sets it.
func TestAResumeWithoutATitleOrAnEntryTakesAFreshName(t *testing.T) {
	for _, source := range []string{"resume", "clear", "compact"} {
		t.Run(source, func(t *testing.T) {
			m := newMachine(t)
			root := t.TempDir()
			mustName(t, m.start(100, root, "a"), "Alice")
			mustName(t, Start(m.at(m.launch(300)), Request{Root: root, SessionID: "z", Source: source}), "Bob")
			resends(t, m, []struct{ sid, title, want string }{{"z", "", "Bob"}})
		})
	}
}

// TestACompactionBeforeTheFirstPromptKeepsItsEntry: a compaction runs in the
// same process, so the entry already names this host; it is kept exactly as it
// is, and the first prompt still sets the name.
func TestACompactionBeforeTheFirstPromptKeepsItsEntry(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	from := m.launch(100)
	mustName(t, Start(m.at(from), Request{Root: root, SessionID: "a", Source: "startup"}), "Alice")
	before := m.register()
	mustName(t, Start(m.at(from), Request{Root: root, SessionID: "a", Source: "compact"}), "Alice")
	if !bytes.Equal(m.register(), before) {
		t.Fatalf("compaction changed the register:\n%s\nwant\n%s", m.register(), before)
	}
	resends(t, m, []struct{ sid, title, want string }{{"a", "", "Alice"}})
}
