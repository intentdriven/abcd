package sessionname

import (
	"bytes"
	"context"
	"encoding/json"
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
	want := append(bundledList(), "Alice Two", "Bob Two")
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
// arrives with a title is not named. A startup's title is a name given on the
// command line, and the register records the session as the person's, with no
// name given and nothing held, so a later resume never reads the title as one
// inherited from another session. A fork's title abcd never gave is the
// person's too, and writes nothing.
func TestStartLeavesAPersonsOwnNameAlone(t *testing.T) {
	m := newMachine(t)
	out := Start(m.at(m.launch(100)), Request{Root: t.TempDir(), SessionID: "z", Source: "startup", Title: "Zed"})
	if out.Name != "" || len(out.Notes) != 0 {
		t.Fatalf("startup with a title: %+v, want nothing", out)
	}
	reg := m.register()
	if !bytes.Contains(reg, []byte(`"z": ""`)) || bytes.Contains(reg, []byte(`"session_id"`)) || bytes.Contains(reg, []byte(`"Zed"`)) {
		t.Fatalf("startup with a title: the register does not record z as the person's alone:\n%s", reg)
	}
	f := newMachine(t)
	out = Start(f.at(f.launch(100)), Request{Root: t.TempDir(), SessionID: "f", Source: "fork", Title: "Zed"})
	if out.Name != "" || len(out.Notes) != 0 || f.register() != nil {
		t.Fatalf("fork with the person's title: %+v, register %s; want nothing", out, f.register())
	}
}

// TestActsGatesExactlyThePersonaRead is iss-2610152324354563: the front door
// resolves the repository root, a git call, only when Acts says so, so Acts
// must answer yes for every payload whose start reads the persona list and no
// for every other. A titled startup only records the session as the person's
// (keep), which needs no repository: Acts answers no for it, and Start with no
// root still records it. The persona read is seen through a list that cannot
// be used, whose note a start that reads it always carries.
func TestActsGatesExactlyThePersonaRead(t *testing.T) {
	bad := writePersonas(t, `{"personas": []}`)
	for _, c := range []struct {
		name string
		req  Request
	}{
		{"titled startup", Request{SessionID: "p", Source: "startup", Title: "Zed"}},
		{"untitled startup", Request{SessionID: "p", Source: "startup"}},
		{"untitled fork", Request{SessionID: "p", Source: "fork"}},
		{"titled fork", Request{SessionID: "p", Source: "fork", Title: "Zed"}},
		{"untitled resume", Request{SessionID: "p", Source: "resume"}},
		{"titled resume", Request{SessionID: "p", Source: "resume", Title: "Zed"}},
		{"untitled clear", Request{SessionID: "p", Source: "clear"}},
		{"titled clear", Request{SessionID: "p", Source: "clear", Title: "Zed"}},
		{"untitled compact", Request{SessionID: "p", Source: "compact"}},
		{"titled compact", Request{SessionID: "p", Source: "compact", Title: "Zed"}},
		{"sub-agent", Request{SessionID: "p", Source: "startup", AgentID: "agent-1"}},
		{"no session id", Request{Source: "startup"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := newMachine(t)
			req := c.req
			req.Root = bad
			out := Start(m.at(m.launch(100)), req)
			read := slices.ContainsFunc(out.Notes, func(n string) bool { return strings.Contains(n, PersonasRel) })
			if got := Acts(c.req); got != read {
				t.Fatalf("Acts = %v, but the start read the persona list: %v (%+v)", got, read, out)
			}
		})
	}

	m := newMachine(t)
	out := Start(m.at(m.launch(100)), Request{SessionID: "z", Source: "startup", Title: "Zed"})
	if out.Name != "" || len(out.Notes) != 0 {
		t.Fatalf("titled startup with no root: %+v, want nothing", out)
	}
	if reg := m.register(); !bytes.Contains(reg, []byte(`"z": ""`)) {
		t.Fatalf("titled startup with no root: the register does not record z as the person's:\n%s", reg)
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
	resends(t, m, []struct{ sid, title, want string }{{"b", "", "Bob"}})
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
	if got := resend(m.at(301), "b", "Bob"); got.Name != "" {
		t.Fatalf("the resumed session was sent %q", got.Name)
	}
}

// TestResumeAfterEveryoneEndedStillHoldsTheName: naming restarted at Alice
// while Bob was away, so resuming Bob adds Bob back to the names handed out.
// The record of the names abcd gave outlives that restart: it is what says
// the title Bob is abcd's.
func TestResumeAfterEveryoneEndedStillHoldsTheName(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "a"), "Alice")
	mustName(t, m.start(200, root, "b"), "Bob")
	resends(t, m, []struct{ sid, title, want string }{{"b", "", "Bob"}})
	m.end(100)
	m.end(200)
	Start(m.at(m.launch(300)), Request{Root: root, SessionID: "b", Source: "resume", Title: "Bob"})
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
		got := resend(m.at(0), c.sid, c.title)
		if got.Name != c.want || len(got.Notes) != 0 {
			t.Fatalf("prompt %d: resend(%q, %q) = %+v, want %q\nregister:\n%s", i+1, c.sid, c.title, got, c.want, m.register())
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
	if got := resend(m.at(0), "s1", ""); got.Name != "" || len(got.Notes) != 0 {
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
	resends(t, m, []struct{ sid, title, want string }{{"a", "", "Alice"}})
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
	got := resend(m.at(0), "s1", "Alice")
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
	for i, name := range sequenceOf([]string{"Zed"}) {
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
	t.Cleanup(lockRegisterFor(t, home))
}

// lockRegisterFor takes the register's lock from another descriptor and
// returns the function that lets it go.
func lockRegisterFor(t *testing.T, home string) (release func()) {
	t.Helper()
	dir, err := fsutil.EnsureHomeScope(home, abcdhome.Rel("sessions"), abcdhome.DirMode)
	if err != nil {
		t.Fatal(err)
	}
	held, done, gone := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var err2 error
	go func() {
		defer close(gone)
		err2 = fsutil.WithFileLockIn(dir, lockName, time.Second, func() error {
			close(held)
			<-done
			return nil
		})
	}()
	select {
	case <-held:
	case <-gone:
		dir.Close()
		t.Fatalf("the register's lock could not be taken: %v", err2)
	}
	return func() { close(done); <-gone; dir.Close() }
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
	resends(t, m, []struct{ sid, title, want string }{{"a", "", "Alice"}})
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
// Resumed under the same session id in a new process, its entry is re-keyed to
// the process now running it, keeping the name and the title still owed; a
// later start does not prune it and hand Alice out again, and the first prompt
// sets Alice.
func TestAResumeBeforeTheFirstPromptKeepsTheReservedName(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "a"), "Alice")
	m.end(100)
	mustName(t, Start(m.at(m.launch(300)), Request{Root: root, SessionID: "a", Source: "resume"}), "Alice")
	reg := m.register()
	if !bytes.Contains(reg, []byte(`"host_pid": 300`)) || bytes.Contains(reg, []byte(`"host_pid": 100`)) || !bytes.Contains(reg, []byte(`"sent": false`)) {
		t.Fatalf("the entry was not re-keyed to the running host, still owed:\n%s", reg)
	}
	mustName(t, m.start(400, root, "b"), "Bob")
	resends(t, m, []struct{ sid, title, want string }{
		{"a", "", "Alice"},
		{"b", "", "Bob"},
	})
}

// TestAClearBeforeTheFirstPromptKeepsTheReservedName is the host's clear as
// it was seen live: the same host process, a NEW session id (7344b67a became
// 315e82fe), and, before the first prompt, no title. The session is the one
// that reserved Alice, so its entry is re-keyed to the new id, still owed: it
// does not reserve a second name, and Alice is not left held by an id no
// session carries any more.
func TestAClearBeforeTheFirstPromptKeepsTheReservedName(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	from := m.launch(100)
	mustName(t, Start(m.at(from), Request{Root: root, SessionID: "a", Source: "startup"}), "Alice")
	mustName(t, Start(m.at(from), Request{Root: root, SessionID: "a2", Source: "clear"}), "Alice")
	reg := m.register()
	if bytes.Count(reg, []byte(`"session_id"`)) != 1 || !bytes.Contains(reg, []byte(`"session_id": "a2"`)) || !bytes.Contains(reg, []byte(`"sent": false`)) {
		t.Fatalf("the register does not hold Alice once, for a2, still owed:\n%s", reg)
	}
	mustName(t, m.start(400, root, "b"), "Bob")
	resends(t, m, []struct{ sid, title, want string }{
		{"a2", "", "Alice"},
		{"a", "", ""},
		{"b", "", "Bob"},
	})
}

// TestAClearMatchesAnEntryASlowReadRecorded: the entry was recorded by a slow
// table read, so its host's start time is early by that read; the clear, in
// the same process, reads the table fast and puts the start where it truly is.
// The entry's own slack is what lets the two readings name one host: without
// it the clear would not find its session, and would take a second name.
func TestAClearMatchesAnEntryASlowReadRecorded(t *testing.T) {
	t.Run("after the first prompt", func(t *testing.T) {
		m := newMachine(t)
		root := t.TempDir()
		startAfterASlowRead(t, m, root)
		resends(t, m, []struct{ sid, title, want string }{{"a", "", "Alice"}})
		out := Start(m.at(101), Request{Root: root, SessionID: "a2", Source: "clear", Title: "Alice"})
		if out.Name != "" || len(out.Notes) != 0 {
			t.Fatalf("a fast clear of a slowly recorded entry: %+v, want the name held", out)
		}
		if reg := m.register(); bytes.Count(reg, []byte(`"session_id"`)) != 1 || !bytes.Contains(reg, []byte(`"a2"`)) {
			t.Fatalf("the register does not hold Alice once, for a2:\n%s", reg)
		}
	})
	t.Run("before the first prompt", func(t *testing.T) {
		m := newMachine(t)
		root := t.TempDir()
		startAfterASlowRead(t, m, root)
		mustName(t, Start(m.at(101), Request{Root: root, SessionID: "a2", Source: "clear"}), "Alice")
		if reg := m.register(); bytes.Count(reg, []byte(`"session_id"`)) != 1 || !bytes.Contains(reg, []byte(`"a2"`)) {
			t.Fatalf("the register does not hold Alice once, for a2:\n%s", reg)
		}
	})
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

// TestAPersonaNameThePersonGaveIsNeverReplaced is C4 and decision 10's last
// sentence on resume: a title that happens to be a persona name, but that abcd
// did not give this session, is the person's. Resumed while a running session
// holds that name, it is neither held nor replaced, and no name is reserved:
// before that session's first prompt, and after it, when the record of given
// names holds Bob for that other session, which is what a fork's inherited
// name looks like.
func TestAPersonaNameThePersonGaveIsNeverReplaced(t *testing.T) {
	for name, arrange := range map[string]func(m *machine, root string){
		// The person started the session with --name Bob: abcd never named it.
		"started with --name": func(m *machine, root string) {
			from := m.launch(100)
			if out := Start(m.at(from), Request{Root: root, SessionID: "p", Source: "startup", Title: "Bob"}); out.Name != "" {
				t.Fatalf("a session the person named was named %q", out.Name)
			}
		},
		// abcd named it Alice, and the person renamed it Bob.
		"renamed by hand": func(m *machine, root string) {
			mustName(t, m.start(100, root, "p"), "Alice")
			resends(t, m, []struct{ sid, title, want string }{{"p", "", "Alice"}, {"p", "Bob", ""}})
		},
		// abcd reserved Alice, and the person renamed it Bob before its
		// first prompt set Alice.
		"renamed before the first prompt": func(m *machine, root string) {
			mustName(t, m.start(100, root, "p"), "Alice")
			resends(t, m, []struct{ sid, title, want string }{{"p", "Bob", ""}})
		},
	} {
		for _, prompted := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s, Bob prompted %v", name, prompted), func(t *testing.T) {
				m := newMachine(t)
				root := t.TempDir()
				arrange(m, root)
				m.end(100)
				mustName(t, m.start(200, root, "a"), "Alice")
				mustName(t, m.start(300, root, "b"), "Bob")
				if prompted {
					resends(t, m, []struct{ sid, title, want string }{{"a", "", "Alice"}, {"b", "", "Bob"}})
				}
				before := m.register()
				out := Start(m.at(m.launch(400)), Request{Root: root, SessionID: "p", Source: "resume", Title: "Bob"})
				if out.Name != "" || len(out.Notes) != 0 {
					t.Fatalf("resuming the person's Bob: %+v, want nothing", out)
				}
				if !bytes.Equal(m.register(), before) {
					t.Fatalf("resuming the person's Bob changed the register:\n%s\nwant\n%s", m.register(), before)
				}
				resends(t, m, []struct{ sid, title, want string }{{"p", "Bob", ""}})
				mustName(t, m.start(500, root, "c"), "Carol")
			})
		}
	}
}

// TestAClearOfAPersonsNameIsNeverAFork: a clear gives the session a new id in
// the same process, so the new id has no record of its own; a person's own
// title must not then be read as a name inherited from another session, even
// while abcd's session of that name runs elsewhere (C4). Found by the final
// review of 2026-10-16.
func TestAClearOfAPersonsNameIsNeverAFork(t *testing.T) {
	for name, arrange := range map[string]func(m *machine, root string){
		"started with --name": func(m *machine, root string) {
			if out := Start(m.at(m.launch(100)), Request{Root: root, SessionID: "p", Source: "startup", Title: "Bob"}); out.Name != "" {
				t.Fatalf("a session the person named was named %q", out.Name)
			}
			// abcd's Alice, so that abcd's next name is Bob.
			mustName(t, m.start(200, root, "a"), "Alice")
			resends(t, m, []struct{ sid, title, want string }{{"a", "", "Alice"}})
		},
		"renamed by hand": func(m *machine, root string) {
			mustName(t, m.start(100, root, "p"), "Alice")
			resends(t, m, []struct{ sid, title, want string }{{"p", "", "Alice"}, {"p", "Bob", ""}})
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := newMachine(t)
			root := t.TempDir()
			arrange(m, root)
			// abcd's own Bob, running in another process.
			mustName(t, m.start(300, root, "b"), "Bob")
			resends(t, m, []struct{ sid, title, want string }{{"b", "", "Bob"}})
			// The person clears their Bob: a new id, the same process (the
			// hook shell under host 100 is 101).
			out := Start(m.at(101), Request{Root: root, SessionID: "p2", Source: "clear", Title: "Bob"})
			if out.Name != "" {
				t.Fatalf("clearing the person's Bob reserved %q, want nothing", out.Name)
			}
			resends(t, m, []struct{ sid, title, want string }{{"p2", "Bob", ""}})
			// Later the person quits and resumes the cleared session while
			// abcd's Bob still runs: the title is still theirs.
			m.end(100)
			if out := Start(m.at(m.launch(400)), Request{Root: root, SessionID: "p2", Source: "resume", Title: "Bob"}); out.Name != "" {
				t.Fatalf("resuming the person's cleared Bob reserved %q, want nothing", out.Name)
			}
		})
	}
}

// TestAClearBeforeAbcdGaveTheNameStillRecordsItAsThePersons: the person's Bob
// is cleared while abcd has given Bob to no one, so the clear has nothing to
// match; abcd later gives Bob to another session, and the cleared session is
// resumed. The clear recorded the title as the person's, so it is kept (C4).
func TestAClearBeforeAbcdGaveTheNameStillRecordsItAsThePersons(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	if out := Start(m.at(m.launch(100)), Request{Root: root, SessionID: "p", Source: "startup", Title: "Bob"}); out.Name != "" {
		t.Fatalf("a session the person named was named %q", out.Name)
	}
	if out := Start(m.at(101), Request{Root: root, SessionID: "p2", Source: "clear", Title: "Bob"}); out.Name != "" {
		t.Fatalf("clearing the person's Bob reserved %q, want nothing", out.Name)
	}
	m.end(100)
	mustName(t, m.start(200, root, "a"), "Alice")
	mustName(t, m.start(300, root, "b"), "Bob")
	resends(t, m, []struct{ sid, title, want string }{{"a", "", "Alice"}, {"b", "", "Bob"}})
	if out := Start(m.at(m.launch(400)), Request{Root: root, SessionID: "p2", Source: "resume", Title: "Bob"}); out.Name != "" {
		t.Fatalf("resuming the person's cleared Bob reserved %q, want nothing", out.Name)
	}
}

// TestAResumeOfAPersonsNameWithNoRecordIsNeverReplaced: a resume is never a
// fork (the host starts a fork with source "fork"), so a person's title on a
// session the register has no record of (one named before naming existed, or
// whose record aged out) is kept while abcd's session of that name runs (C4).
// Found by the doc-fidelity review of 2026-10-16.
func TestAResumeOfAPersonsNameWithNoRecordIsNeverReplaced(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(200, root, "a"), "Alice")
	mustName(t, m.start(300, root, "b"), "Bob")
	resends(t, m, []struct{ sid, title, want string }{{"a", "", "Alice"}, {"b", "", "Bob"}})
	from := m.launch(400)
	if out := Start(m.at(from), Request{Root: root, SessionID: "old", Source: "resume", Title: "Bob"}); out.Name != "" {
		t.Fatalf("resuming the person's unrecorded Bob reserved %q, want nothing", out.Name)
	}
	// Its prompt finds abcd's Bob accounting for the title on another host:
	// the title is the person's, recorded as theirs, and kept.
	if out := m.prompt(from, root, "old", "Bob"); out.Name != "" || out.Notice != "" || len(out.Notes) != 0 {
		t.Fatalf("the person's unrecorded Bob prompting: %+v, want nothing", out)
	}
	if reg := m.register(); !bytes.Contains(reg, []byte(`"old": ""`)) || bytes.Contains(reg, []byte(`"session_id": "old"`)) {
		t.Fatalf("the person's Bob is not recorded as theirs alone:\n%s", reg)
	}
	reads := 0
	if out := Prompt(counted(m.at(from), &reads), Request{Root: root, SessionID: "old", Title: "Bob"}); out.Name != "" || reads != 0 {
		t.Fatalf("the next prompt: %+v after %d table reads, want nothing and none", out, reads)
	}
}

// TestResumeHoldsANameFromAnEarlierPersonaList: the name abcd gave is held on
// resume even when the repository's persona list no longer has it, because
// what makes the title abcd's is the record of the names it gave, not the
// list as it reads today.
func TestResumeHoldsANameFromAnEarlierPersonaList(t *testing.T) {
	m := newMachine(t)
	root := writePersonas(t, `{"personas":[{"name":"Zed"},{"name":"Yan"}]}`)
	mustName(t, m.start(100, root, "z"), "Zed")
	resends(t, m, []struct{ sid, title, want string }{{"z", "", "Zed"}})
	m.end(100)
	p := filepath.Join(root, filepath.FromSlash(PersonasRel))
	if err := os.WriteFile(p, []byte(`{"personas":[{"name":"Yan"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out := Start(m.at(m.launch(200)), Request{Root: root, SessionID: "z", Source: "resume", Title: "Zed"})
	if out.Name != "" || len(out.Notes) != 0 {
		t.Fatalf("resuming Zed after the list changed: %+v, want the name held", out)
	}
	// Zed is held: a session in a repository whose list is Zed alone is given
	// the next round's name.
	mustName(t, m.start(300, writePersonas(t, `{"personas":[{"name":"Zed"}]}`), "n"), "Zed Two")
}

// TestTheGivenNameFollowsAClearToALaterResume is the live sequence: a session
// is named at its first prompt, cleared (a new id in the same process, the
// title kept), and later resumed under the new id in a new process. The name
// abcd gave follows the id through the clear, so the resume holds it.
func TestTheGivenNameFollowsAClearToALaterResume(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	from := m.launch(100)
	mustName(t, Start(m.at(from), Request{Root: root, SessionID: "a", Source: "startup"}), "Alice")
	resends(t, m, []struct{ sid, title, want string }{{"a", "", "Alice"}})
	if out := Start(m.at(from), Request{Root: root, SessionID: "a2", Source: "clear", Title: "Alice"}); out.Name != "" || len(out.Notes) != 0 {
		t.Fatalf("clear: %+v, want the name held", out)
	}
	m.end(100)
	if out := Start(m.at(m.launch(200)), Request{Root: root, SessionID: "a2", Source: "resume", Title: "Alice"}); out.Name != "" || len(out.Notes) != 0 {
		t.Fatalf("resume of the cleared id: %+v, want the name held", out)
	}
	mustName(t, m.start(300, root, "b"), "Bob")
	if reg := m.register(); !bytes.Contains(reg, []byte(`"a2": "Alice"`)) || bytes.Contains(reg, []byte(`"a": "Alice"`)) {
		t.Fatalf("the given name did not move to the cleared id:\n%s", reg)
	}
}

// TestARegisterWithoutGivenStillLoads: a register written before the record of
// given names existed loads, its owed title is still sent, and the send is
// recorded.
func TestARegisterWithoutGivenStillLoads(t *testing.T) {
	m := newMachine(t)
	m.launch(100)
	if err := os.MkdirAll(abcdhome.Path(m.home, "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := fmt.Sprintf(`{"schema_version":1,"handed":["Alice"],"live":[{"name":"Alice","session_id":"a","host_pid":100,"host_start_unix":%d,"sent":false}]}`, readAt.Add(-time.Hour).Unix())
	if err := os.WriteFile(abcdhome.Path(m.home, "sessions", "names.json"), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	resends(t, m, []struct{ sid, title, want string }{{"a", "", "Alice"}})
	if !bytes.Contains(m.register(), []byte(`"a": "Alice"`)) {
		t.Fatalf("the send was not recorded:\n%s", m.register())
	}
	mustName(t, m.start(200, t.TempDir(), "b"), "Bob")
}

// TestGivenKeepsTheNewestNames: the record of given names is bounded at
// maxGiven, dropping the oldest written first; a name given again to a
// session already in it moves that session to the newest end; and the order
// survives a write and a read of the register, which is what makes "oldest"
// mean anything in a JSON object.
func TestGivenKeepsTheNewestNames(t *testing.T) {
	var g givenNames
	for i := range maxGiven + 1 {
		g.give(fmt.Sprintf("s%d", i), "Alice")
	}
	if len(g) != maxGiven || g[0].SessionID != "s1" || g[len(g)-1].SessionID != fmt.Sprintf("s%d", maxGiven) {
		t.Fatalf("after %d gives: %d kept, first %q, last %q; want %d, s1 first", maxGiven+1, len(g), g[0].SessionID, g[len(g)-1].SessionID, maxGiven)
	}
	g.give("s1", "Bob")
	if g[len(g)-1] != (givenName{"s1", "Bob"}) || g[0].SessionID != "s2" || len(g) != maxGiven {
		t.Fatalf("giving s1 again did not move it to the newest end: first %+v, last %+v", g[0], g[len(g)-1])
	}
	raw, err := json.Marshal(register{SchemaVersion: registerSchema, Given: g})
	if err != nil {
		t.Fatal(err)
	}
	var back register
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(back.Given, g) {
		t.Fatalf("the order did not survive a round trip:\n%s", raw)
	}
	if name, ok := back.Given.of("s1"); !ok || name != "Bob" {
		t.Fatalf("Given.of(s1) = %q, %v; want Bob", name, ok)
	}
}

// forkSources are the sources a fork's session start arrives with: seen live
// on 2026-10-16, a fork made with --resume and --fork-session starts with
// source "fork".
var forkSources = []string{"fork"}

// TestAForkOfARunningSessionTakesTheNextFreeName is the live sequence of
// 2026-10-16: Alice is named at its first prompt and, while it runs, is forked
// into a new session id that the host starts with Alice's title. The title is
// abcd's, given to the parent, so the fork does not share it while the parent
// runs (C6, C7): it reserves the next free name, which its first prompt sets
// in place of the inherited title, once. The parent keeps Alice.
func TestAForkOfARunningSessionTakesTheNextFreeName(t *testing.T) {
	for _, source := range forkSources {
		t.Run(source, func(t *testing.T) {
			m := newMachine(t)
			root := t.TempDir()
			mustName(t, m.start(100, root, "p"), "Alice")
			resends(t, m, []struct{ sid, title, want string }{{"p", "", "Alice"}})
			mustName(t, Start(m.at(m.launch(200)), Request{Root: root, SessionID: "f", Source: source, Title: "Alice"}), "Bob")
			resends(t, m, []struct{ sid, title, want string }{
				{"f", "Alice", "Bob"},
				{"f", "Bob", ""},
				{"f", "Alice", ""}, // once only
				{"p", "Alice", ""},
			})
			mustName(t, m.start(300, root, "c"), "Carol")
		})
	}
}

// TestAForkOfAnEndedSessionKeepsTheInheritedName: the parent has ended, so no
// running session holds Alice and the fork keeps it. The fork holds Alice for
// its own process, so no new session is given Alice while it runs, and the
// record of given names gains the fork's id beside the parent's: the parent's
// is kept, because the parent still carries Alice and may be resumed. Resumed
// while the fork runs, the parent is the one that takes the next free name
// (decision 10).
func TestAForkOfAnEndedSessionKeepsTheInheritedName(t *testing.T) {
	for _, source := range forkSources {
		t.Run(source, func(t *testing.T) {
			m := newMachine(t)
			root := t.TempDir()
			mustName(t, m.start(100, root, "p"), "Alice")
			resends(t, m, []struct{ sid, title, want string }{{"p", "", "Alice"}})
			m.end(100)
			out := Start(m.at(m.launch(200)), Request{Root: root, SessionID: "f", Source: source, Title: "Alice"})
			if out.Name != "" || len(out.Notes) != 0 {
				t.Fatalf("fork of an ended session: %+v, want the name held", out)
			}
			reg := m.register()
			if !bytes.Contains(reg, []byte(`"p": "Alice"`)) || !bytes.Contains(reg, []byte(`"f": "Alice"`)) || bytes.Count(reg, []byte(`"session_id"`)) != 1 {
				t.Fatalf("the fork does not hold Alice beside the parent's record:\n%s", reg)
			}
			resends(t, m, []struct{ sid, title, want string }{{"f", "Alice", ""}})
			mustName(t, m.start(300, root, "c"), "Bob")
			mustName(t, Start(m.at(m.launch(400)), Request{Root: root, SessionID: "p", Source: "resume", Title: "Alice"}), "Carol")
			// The fork, resumed later, is held under its own record.
			m.end(200)
			if out := Start(m.at(m.launch(500)), Request{Root: root, SessionID: "f", Source: "resume", Title: "Alice"}); out.Name != "" || len(out.Notes) != 0 {
				t.Fatalf("resuming the fork: %+v, want the name held", out)
			}
		})
	}
}

// TestAClearIsNotAFork: a clear gives the session a new id in the same host
// process with the name abcd gave the old id, which is what an inherited name
// looks like. It is the same session, so it keeps its name even while a fork
// of it runs elsewhere, and a clear of a fork before its first prompt keeps the
// name the fork reserved rather than reserving a second one.
func TestAClearIsNotAFork(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	from := m.launch(100)
	mustName(t, Start(m.at(from), Request{Root: root, SessionID: "p", Source: "startup"}), "Alice")
	resends(t, m, []struct{ sid, title, want string }{{"p", "", "Alice"}})
	forkFrom := m.launch(200)
	mustName(t, Start(m.at(forkFrom), Request{Root: root, SessionID: "f", Source: "fork", Title: "Alice"}), "Bob")

	// The fork is cleared before its first prompt: still Bob, owed to the
	// first prompt under the new id.
	mustName(t, Start(m.at(forkFrom), Request{Root: root, SessionID: "f2", Source: "clear", Title: "Alice"}), "Bob")
	// The parent is cleared: still Alice, nothing owed.
	if out := Start(m.at(from), Request{Root: root, SessionID: "p2", Source: "clear", Title: "Alice"}); out.Name != "" || len(out.Notes) != 0 {
		t.Fatalf("clearing the parent: %+v, want the name held", out)
	}
	if reg := m.register(); bytes.Count(reg, []byte(`"session_id"`)) != 2 {
		t.Fatalf("the clears left other than one entry each:\n%s", reg)
	}
	resends(t, m, []struct{ sid, title, want string }{
		{"f2", "Alice", "Bob"},
		{"p2", "Alice", ""},
	})
	mustName(t, m.start(300, root, "c"), "Carol")
}

// TestAClearOfAPersonsOwnTitleReadsNoProcessTable: a clear is never read as
// inheriting, so a title the person gave, cleared while no running session's
// entry accounts for it, is theirs whoever else abcd once gave that name to.
// The unlocked read settles it: the clear records the new id as given nothing
// and reads no process table (iss-2610152329345301).
func TestAClearOfAPersonsOwnTitleReadsNoProcessTable(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	if out := Start(m.at(m.launch(100)), Request{Root: root, SessionID: "p", Source: "startup", Title: "Bob"}); out.Name != "" {
		t.Fatalf("a session the person named was named %q", out.Name)
	}
	// abcd gives Bob to another session, which ends; the next start prunes
	// its entry, and the record of the given name stays.
	mustName(t, m.start(200, root, "a"), "Alice")
	mustName(t, m.start(300, root, "b"), "Bob")
	resends(t, m, []struct{ sid, title, want string }{{"a", "", "Alice"}, {"b", "", "Bob"}})
	m.end(300)
	mustName(t, m.start(400, root, "c"), "Carol")

	// The person clears their Bob: a new id, the same process.
	clear := m.at(101)
	reads := 0
	read := clear.ReadTable
	clear.ReadTable = func() (Table, error) {
		reads++
		return read()
	}
	if out := Start(clear, Request{Root: root, SessionID: "p2", Source: "clear", Title: "Bob"}); out.Name != "" || len(out.Notes) != 0 {
		t.Fatalf("clearing the person's Bob: %+v, want nothing", out)
	}
	if reads != 0 {
		t.Fatalf("clearing the person's own title read the process table %d time(s), want none", reads)
	}
	// The clear still recorded the title as the person's: resumed while
	// abcd's Bob runs again, it is kept (C4).
	m.end(100)
	m.end(200)
	m.end(400)
	mustName(t, m.start(500, root, "d"), "Alice")
	mustName(t, m.start(600, root, "e"), "Bob")
	resends(t, m, []struct{ sid, title, want string }{{"d", "", "Alice"}, {"e", "", "Bob"}})
	if out := Start(m.at(m.launch(700)), Request{Root: root, SessionID: "p2", Source: "resume", Title: "Bob"}); out.Name != "" {
		t.Fatalf("resuming the person's cleared Bob reserved %q, want nothing", out.Name)
	}
}

// counted makes mm count its process-table reads in reads.
func counted(mm Machine, reads *int) Machine {
	read := mm.ReadTable
	mm.ReadTable = func() (Table, error) {
		*reads++
		return read()
	}
	return mm
}

// prompt runs the prompt hook's naming for session sid, carrying title, from
// the hook shell from, in root.
func (m *machine) prompt(from int, root, sid, title string) Outcome {
	m.t.Helper()
	return Prompt(m.at(from), Request{Root: root, SessionID: sid, Title: title})
}

// resumeWithTheLockHeld resumes session sid, carrying title, under a new host
// at pid while another process holds the register's lock, so the hold is not
// made: session start notes the lock and holds nothing. It returns the new
// host's hook shell.
func resumeWithTheLockHeld(t *testing.T, m *machine, root string, pid int, sid, title string) int {
	t.Helper()
	release := lockRegisterFor(t, m.home)
	timeout := m.timeout
	m.timeout = 50 * time.Millisecond
	from := m.launch(pid)
	out := Start(m.at(from), Request{Root: root, SessionID: sid, Source: "resume", Title: title})
	m.timeout = timeout
	release()
	if out.Name != "" || len(out.Notes) != 1 || !strings.Contains(out.Notes[0], "lock") {
		t.Fatalf("the resume under a held lock: %+v, want no hold and one note on the lock", out)
	}
	return from
}

// TestAPromptHoldsAResumeWhoseHoldFailed is iss-2611141504101439's first
// proof: Bob is resumed in a new process while the register's lock is held, so
// session start holds nothing, and the next session start drops Bob's entry,
// whose old process has ended. Bob's next prompt makes the entry again for the
// process now running it, so Bob is held: when every other session has ended,
// naming does not restart and hand Bob to a new session while Bob runs.
func TestAPromptHoldsAResumeWhoseHoldFailed(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "a"), "Alice")
	mustName(t, m.start(200, root, "b"), "Bob")
	resends(t, m, []struct{ sid, title, want string }{{"a", "", "Alice"}, {"b", "", "Bob"}})
	m.end(200)
	from := resumeWithTheLockHeld(t, m, root, 300, "b", "Bob")
	mustName(t, m.start(400, root, "c"), "Carol")
	if bytes.Contains(m.register(), []byte(`"session_id": "b"`)) {
		t.Fatalf("the fixture did not lose Bob's entry:\n%s", m.register())
	}

	out := m.prompt(from, root, "b", "Bob")
	if out.Name != "" || out.Notice != "" || len(out.Notes) != 0 {
		t.Fatalf("Bob's prompt: %+v, want the name held with nothing set or said", out)
	}
	if reg := m.register(); !bytes.Contains(reg, []byte(`"session_id": "b"`)) || !bytes.Contains(reg, []byte(`"host_pid": 300`)) {
		t.Fatalf("Bob is not held for the process now running it:\n%s", reg)
	}
	m.end(100)
	m.end(400)
	mustName(t, m.start(500, root, "d"), "Dave")

	// Held, a later prompt is the common case: no process table, no write.
	before := m.register()
	reads := 0
	if out := Prompt(counted(m.at(from), &reads), Request{Root: root, SessionID: "b", Title: "Bob"}); out.Name != "" || out.Notice != "" || len(out.Notes) != 0 {
		t.Fatalf("Bob's next prompt: %+v, want nothing", out)
	}
	if reads != 0 || !bytes.Equal(m.register(), before) {
		t.Fatalf("a held session's prompt read the process table %d time(s) or changed the register:\n%s", reads, m.register())
	}
}

// TestTwoRunningSessionsWithOneNameEndWithTwoNamesAndOneNotice is the record's
// sequence and its second proof: Alice is resumed while the lock is held, so
// her hold is not made; every other session having ended, a new session starts
// and is given Alice. Alice's next prompt finds a running session holding her
// name: hers is the later claim, so she takes Bob, the prompt sets it with one
// notice, and the new session keeps Alice. No later prompt of either renames
// or says anything.
func TestTwoRunningSessionsWithOneNameEndWithTwoNamesAndOneNotice(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "x"), "Alice")
	resends(t, m, []struct{ sid, title, want string }{{"x", "", "Alice"}})
	m.end(100)
	xFrom := resumeWithTheLockHeld(t, m, root, 200, "x", "Alice")
	mustName(t, m.start(300, root, "y"), "Alice")
	yFrom := 301
	resends(t, m, []struct{ sid, title, want string }{{"y", "", "Alice"}})

	notices := 0
	for i, c := range []struct {
		from       int
		sid, title string
		want       string
	}{
		{xFrom, "x", "Alice", "Bob"},
		{xFrom, "x", "Bob", ""},
		{yFrom, "y", "Alice", ""},
		{xFrom, "x", "Alice", ""}, // the person took the old title back after Bob showed: not fought
		{yFrom, "y", "Alice", ""},
		{xFrom, "x", "Bob", ""},
	} {
		out := m.prompt(c.from, root, c.sid, c.title)
		if out.Name != c.want || len(out.Notes) != 0 {
			t.Fatalf("prompt %d (%s carrying %q): %+v, want %q\n%s", i+1, c.sid, c.title, out, c.want, m.register())
		}
		if out.Notice != "" {
			notices++
			if !strings.Contains(out.Notice, "Alice") || !strings.Contains(out.Notice, "now Bob") {
				t.Fatalf("the notice %q does not say Alice is held and the session is now Bob", out.Notice)
			}
		}
	}
	if notices != 1 {
		t.Fatalf("%d notices, want exactly one", notices)
	}
	reg := m.register()
	if !bytes.Contains(reg, []byte(`"x": "Bob"`)) || !bytes.Contains(reg, []byte(`"y": "Alice"`)) || bytes.Count(reg, []byte(`"session_id"`)) != 2 {
		t.Fatalf("the register does not hold Bob for x and Alice for y:\n%s", reg)
	}
	mustName(t, m.start(400, root, "z"), "Carol")
}

// plantTwoHolders writes a register in which two running sessions, x then y,
// each hold Alice, the title both carry: what nothing in abcd writes, and the
// one shape whose later claim the register's own order alone decides.
func plantTwoHolders(t *testing.T, m *machine) {
	t.Helper()
	m.launch(100)
	m.launch(200)
	start := readAt.Add(-time.Hour).Unix()
	reg := fmt.Sprintf(`{"schema_version":1,"handed":["Alice"],"live":[`+
		`{"name":"Alice","session_id":"x","host_pid":100,"host_start_unix":%d,"sent":true},`+
		`{"name":"Alice","session_id":"y","host_pid":200,"host_start_unix":%d,"sent":true}],`+
		`"given":{"x":"Alice","y":"Alice"}}`, start, start)
	if err := os.MkdirAll(abcdhome.Path(m.home, "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(m.home, "sessions", "names.json"), []byte(reg), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestTheLaterOfTwoHoldersIsTheOneRenamed: whichever of two running sessions
// holding one name prompts first, the later entry in the register is the one
// that takes the next free name, once, with one notice; the earlier keeps its
// name and its prompts never touch the register.
func TestTheLaterOfTwoHoldersIsTheOneRenamed(t *testing.T) {
	for name, order := range map[string][]string{"earlier first": {"x", "y", "x", "y"}, "later first": {"y", "x", "y", "x"}} {
		t.Run(name, func(t *testing.T) {
			m := newMachine(t)
			root := t.TempDir()
			plantTwoHolders(t, m)
			from := map[string]int{"x": 101, "y": 201}
			title := map[string]string{"x": "Alice", "y": "Alice"}
			notices := 0
			for i, sid := range order {
				before := m.register()
				out := m.prompt(from[sid], root, sid, title[sid])
				if len(out.Notes) != 0 {
					t.Fatalf("prompt %d (%s): notes %q", i+1, sid, out.Notes)
				}
				if sid == "x" && (out.Name != "" || out.Notice != "" || !bytes.Equal(m.register(), before)) {
					t.Fatalf("prompt %d: the earlier holder was displaced or told something: %+v\n%s", i+1, out, m.register())
				}
				if out.Name != "" {
					title[sid] = out.Name
				}
				if out.Notice != "" {
					notices++
				}
			}
			if notices != 1 || title["x"] != "Alice" || title["y"] != "Bob" {
				t.Fatalf("%d notices, titles %v; want one notice, x Alice and y Bob", notices, title)
			}
		})
	}
}

// TestAPromptThatTimesOutChangesNothing is the record's third proof: a prompt
// whose hold has to be made again, or whose session has to take a new name,
// and whose lock wait or process-table read runs out, keeps the title,
// allocates nothing, gives no notice, and leaves the register byte for byte
// as it was, with one note. The next prompt that completes does the work.
func TestAPromptThatTimesOutChangesNothing(t *testing.T) {
	for name, c := range map[string]struct {
		arrange func(t *testing.T, m *machine, mm *Machine)
		mention string
	}{
		"the lock is held": {func(t *testing.T, m *machine, mm *Machine) {
			mm.LockTimeout = 50 * time.Millisecond
			holdLock(t, m.home)
		}, "lock"},
		"the table read runs out": {func(t *testing.T, m *machine, mm *Machine) {
			mm.ReadTableContext = func(ctx context.Context) (Table, error) {
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > maxReadTook {
					t.Errorf("the prompt's table read is not bounded at %s (deadline %v, %v)", maxReadTook, deadline, ok)
				}
				return Table{}, context.DeadlineExceeded
			}
		}, "process table"},
		"the table read is slow": {func(t *testing.T, m *machine, mm *Machine) {
			*mm = slowly(*mm, maxReadTook+time.Millisecond)
		}, "process table"},
	} {
		for _, route := range []string{"hold lost", "name shared"} {
			t.Run(name+", "+route, func(t *testing.T) {
				m := newMachine(t)
				root := t.TempDir()
				sid, from := "x", 0
				if route == "hold lost" {
					mustName(t, m.start(100, root, "x"), "Alice")
					resends(t, m, []struct{ sid, title, want string }{{"x", "", "Alice"}})
					m.end(100)
					from = resumeWithTheLockHeld(t, m, root, 200, "x", "Alice")
					mustName(t, m.start(300, root, "y"), "Alice")
				} else {
					plantTwoHolders(t, m)
					sid, from = "y", 201
				}
				before := m.register()
				mm := m.at(from)
				c.arrange(t, m, &mm)
				out := Prompt(mm, Request{Root: root, SessionID: sid, Title: "Alice"})
				if out.Name != "" || out.Notice != "" || len(out.Notes) != 1 || !strings.Contains(out.Notes[0], c.mention) {
					t.Fatalf("a prompt that ran out: %+v, want nothing set or said and one note on the %s", out, c.mention)
				}
				if !bytes.Equal(m.register(), before) {
					t.Fatalf("a prompt that ran out changed the register:\n%s\nwant\n%s", m.register(), before)
				}
			})
		}
	}
}

// TestAPromptJudgesAHolderByItsStartTime: a name is free only when its last
// holder's process is proven gone, by its id and its start time together.
// The holder's id now belonging to a process started at another time is a
// holder gone, so a session whose hold was lost takes its name back with no
// notice; the holder alive with its own start time is never displaced, and
// the session whose hold was lost takes the next free name instead.
func TestAPromptJudgesAHolderByItsStartTime(t *testing.T) {
	for name, reused := range map[string]bool{"id reused": true, "holder running": false} {
		t.Run(name, func(t *testing.T) {
			m := newMachine(t)
			root := t.TempDir()
			mustName(t, m.start(100, root, "x"), "Alice")
			resends(t, m, []struct{ sid, title, want string }{{"x", "", "Alice"}})
			m.end(100)
			from := resumeWithTheLockHeld(t, m, root, 200, "x", "Alice")
			mustName(t, m.start(300, root, "y"), "Alice")
			resends(t, m, []struct{ sid, title, want string }{{"y", "", "Alice"}})
			if reused {
				m.end(300)
				m.procs = append(m.procs, proc(300, 1, "vim", 10*time.Minute))
			}
			out := m.prompt(from, root, "x", "Alice")
			reg := m.register()
			switch {
			case reused && (out.Name != "" || out.Notice != "" || bytes.Contains(reg, []byte(`"session_id": "y"`))):
				t.Fatalf("a holder whose id was reused kept the name: %+v\n%s", out, reg)
			case !reused && (out.Name != "Bob" || out.Notice == "" || !bytes.Contains(reg, []byte(`"session_id": "y"`))):
				t.Fatalf("a running holder was displaced: %+v\n%s", out, reg)
			}
			if !reused {
				if out := m.prompt(301, root, "y", "Alice"); out.Name != "" || out.Notice != "" {
					t.Fatalf("the running holder's prompt: %+v, want nothing", out)
				}
			}
		})
	}
}

// renameX runs the record's sequence up to the prompt that renames: x is
// named Alice, resumed under host 200 while the lock is held, and y starts
// and is given Alice. It returns x's hook shell.
func renameX(t *testing.T, m *machine, root string) int {
	t.Helper()
	mustName(t, m.start(100, root, "x"), "Alice")
	resends(t, m, []struct{ sid, title, want string }{{"x", "", "Alice"}})
	m.end(100)
	from := resumeWithTheLockHeld(t, m, root, 200, "x", "Alice")
	mustName(t, m.start(300, root, "y"), "Alice")
	resends(t, m, []struct{ sid, title, want string }{{"y", "", "Alice"}})
	return from
}

// TestARenameWhoseOutputWasDroppedIsSentOnceMore: the host discards a hook's
// whole output when it overruns, so the prompt that renamed x may never reach
// it, and x's next prompt still carries Alice. That prompt is sent Bob and
// the notice again; a third still carrying Alice is the person's choice and
// is sent nothing, and changes nothing.
func TestARenameWhoseOutputWasDroppedIsSentOnceMore(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	from := renameX(t, m, root)
	for i, want := range []string{"Bob", "Bob", ""} {
		before := m.register()
		out := m.prompt(from, root, "x", "Alice")
		if out.Name != want || (out.Notice != "") != (want != "") || len(out.Notes) != 0 {
			t.Fatalf("prompt %d carrying Alice: %+v, want %q with a notice when set", i+1, out, want)
		}
		if want == "" && !bytes.Equal(m.register(), before) {
			t.Fatalf("prompt %d changed the register:\n%s", i+1, m.register())
		}
	}
	// Seen once, the rename is owed no more.
	n := newMachine(t)
	from = renameX(t, n, root)
	for i, c := range []struct{ title, want string }{{"Alice", "Bob"}, {"Bob", ""}, {"Alice", ""}} {
		if out := n.prompt(from, root, "x", c.title); out.Name != c.want || len(out.Notes) != 0 {
			t.Fatalf("prompt %d carrying %q: %+v, want %q", i+1, c.title, out, c.want)
		}
	}
}

// TestAForkWhoseTableReadFailedIsRenamedAtItsPrompt: a fork of a running
// Alice whose start cannot read the process table holds nothing, but its
// inherited title is recorded as given to it, so its first prompt finds the
// parent holding Alice and takes Bob, with the notice. The parent keeps Alice.
func TestAForkWhoseTableReadFailedIsRenamedAtItsPrompt(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "p"), "Alice")
	resends(t, m, []struct{ sid, title, want string }{{"p", "", "Alice"}})
	from := m.launch(200)
	m.tableErr = errors.New("ps failed")
	if out := Start(m.at(from), Request{Root: root, SessionID: "f", Source: "fork", Title: "Alice"}); out.Name != "" || len(out.Notes) != 1 {
		t.Fatalf("the fork's start under a failed table read: %+v, want no name and one note", out)
	}
	m.tableErr = nil
	if reg := m.register(); !bytes.Contains(reg, []byte(`"f": "Alice"`)) {
		t.Fatalf("the fork's inherited title was not recorded as given to it:\n%s", reg)
	}
	out := m.prompt(from, root, "f", "Alice")
	if out.Name != "Bob" || out.Notice == "" || len(out.Notes) != 0 {
		t.Fatalf("the fork's prompt: %+v, want Bob with the notice", out)
	}
	if out := m.prompt(101, root, "p", "Alice"); out.Name != "" || out.Notice != "" {
		t.Fatalf("the parent's prompt: %+v, want nothing", out)
	}
	mustName(t, m.start(300, root, "c"), "Carol")
}

// clearWithTheTableUnread clears session a (host 100, Alice) into a2 in the
// same process while the process table cannot be read, so the clear's hold
// is not made.
func clearWithTheTableUnread(t *testing.T, m *machine, root string) {
	t.Helper()
	mustName(t, Start(m.at(m.launch(100)), Request{Root: root, SessionID: "a", Source: "startup"}), "Alice")
	resends(t, m, []struct{ sid, title, want string }{{"a", "", "Alice"}})
	m.tableErr = errors.New("ps failed")
	if out := Start(m.at(101), Request{Root: root, SessionID: "a2", Source: "clear", Title: "Alice"}); out.Name != "" || len(out.Notes) != 1 {
		t.Fatalf("the clear under a failed table read: %+v, want no name and one note", out)
	}
	m.tableErr = nil
}

// TestAClearWhoseHoldFailedIsHeldAtItsPrompt: the cleared id has no entry and
// no record of its own, and the entry on its own host process accounts for
// its title: that entry is the session before the clear. The prompt moves its
// record and its name to the new id, sets nothing and says nothing.
func TestAClearWhoseHoldFailedIsHeldAtItsPrompt(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	clearWithTheTableUnread(t, m, root)
	if out := m.prompt(101, root, "a2", "Alice"); out.Name != "" || out.Notice != "" || len(out.Notes) != 0 {
		t.Fatalf("the cleared session's prompt: %+v, want the name held with nothing set or said", out)
	}
	reg := m.register()
	if !bytes.Contains(reg, []byte(`"a2": "Alice"`)) || bytes.Contains(reg, []byte(`"a": "Alice"`)) ||
		bytes.Count(reg, []byte(`"session_id"`)) != 1 || !bytes.Contains(reg, []byte(`"session_id": "a2"`)) {
		t.Fatalf("the clear's record and name did not move to a2:\n%s", reg)
	}
}

// TestAClearWhoseHoldFailedKeepsItsNameWhenResumed: the same clear, healed at
// its prompt; its process then ends and the new id is resumed elsewhere. The
// resume holds Alice under the new id, so the next session is not given
// Alice while it runs.
func TestAClearWhoseHoldFailedKeepsItsNameWhenResumed(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	clearWithTheTableUnread(t, m, root)
	m.prompt(101, root, "a2", "Alice")
	m.end(100)
	if out := Start(m.at(m.launch(300)), Request{Root: root, SessionID: "a2", Source: "resume", Title: "Alice"}); out.Name != "" || len(out.Notes) != 0 {
		t.Fatalf("resuming the cleared id: %+v, want the name held", out)
	}
	mustName(t, m.start(400, root, "b"), "Bob")
}

// TestAForkRenamedAtStartAndResumedDoesNotShareItsParentsName: a fork of a
// running Alice reserves Bob at its start, and its process ends before its
// first prompt, so its entry is dropped. Resumed later carrying Alice while
// the parent still runs, it takes the next free name: the name it inherited
// is recorded as given to it, so the resume reads the title as abcd's.
func TestAForkRenamedAtStartAndResumedDoesNotShareItsParentsName(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "p"), "Alice")
	resends(t, m, []struct{ sid, title, want string }{{"p", "", "Alice"}})
	mustName(t, Start(m.at(m.launch(200)), Request{Root: root, SessionID: "f", Source: "fork", Title: "Alice"}), "Bob")
	m.end(200)
	mustName(t, m.start(300, root, "c"), "Carol")
	if bytes.Contains(m.register(), []byte(`"session_id": "f"`)) {
		t.Fatalf("the fixture did not drop the fork's entry:\n%s", m.register())
	}
	from := m.launch(400)
	mustName(t, Start(m.at(from), Request{Root: root, SessionID: "f", Source: "resume", Title: "Alice"}), "Dave")
	if out := m.prompt(from, root, "f", "Alice"); out.Name != "Dave" || out.Notice == "" {
		t.Fatalf("the resumed fork's prompt: %+v, want Dave with the notice", out)
	}
}

// TestASubAgentsPromptChangesNothing: a payload carrying an agent id is a
// sub-agent's, and its prompt neither sets the title owed nor writes.
func TestASubAgentsPromptChangesNothing(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "a"), "Alice")
	before := m.register()
	if out := Prompt(m.at(101), Request{Root: root, SessionID: "a", AgentID: "agent-1"}); out.Name != "" || out.Notice != "" || len(out.Notes) != 0 {
		t.Fatalf("a sub-agent's prompt: %+v, want nothing", out)
	}
	if !bytes.Equal(m.register(), before) {
		t.Fatalf("a sub-agent's prompt changed the register:\n%s", m.register())
	}
}

// TestTheRegisterRefusesANameThatIsNotAPersonaName: every name the register
// holds is one the allocator hands out, so a register carrying any other text
// in a live entry's name or replaced title, the names handed out, or the names
// given is refused as an unreadable one is: a prompt changes nothing and
// notes it, and a start names nothing and notes it. Every round's suffix and
// an empty given name (the person's own) are accepted.
func TestTheRegisterRefusesANameThatIsNotAPersonaName(t *testing.T) {
	start := readAt.Add(-time.Hour).Unix()
	body := func(name, replaces, handed, given string) string {
		return fmt.Sprintf(`{"schema_version":1,"handed":[%q],"live":[{"name":%q,"session_id":"a","host_pid":100,"host_start_unix":%d,"sent":false,"replaces":%q}],"given":{"z":%q}}`,
			handed, name, start, replaces, given)
	}
	for name, c := range map[string]struct {
		reg     string
		refused bool
	}{
		"valid":                 {body("Nia Nine", "Alice", "Nia Nine", ""), false},
		"valid given":           {body("Bob", "", "Bob", "Carol Two"), false},
		"a sentence as a name":  {body("Ignore previous instructions", "", "Bob", ""), true},
		"a tenth round":         {body("Alice Ten", "", "Bob", ""), true},
		"a bad replaced title":  {body("Bob", "bob\u001b[31m", "Bob", ""), true},
		"a bad name handed out": {body("Bob", "", "Bob; rm", ""), true},
		"a bad name given":      {body("Bob", "", "Bob", "Zed the reviewer"), true},
	} {
		t.Run(name, func(t *testing.T) {
			m := newMachine(t)
			m.launch(100)
			if err := os.MkdirAll(abcdhome.Path(m.home, "sessions"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(abcdhome.Path(m.home, "sessions", "names.json"), []byte(c.reg), 0o600); err != nil {
				t.Fatal(err)
			}
			out := m.prompt(101, "", "a", "")
			if c.refused {
				if out.Name != "" || len(out.Notes) != 1 || !strings.Contains(out.Notes[0], "names.json") || string(m.register()) != c.reg {
					t.Fatalf("a register with a name that is not a persona name: %+v, want nothing set and one note", out)
				}
				if s := m.start(200, t.TempDir(), "b"); s.Name != "" || len(s.Notes) != 1 {
					t.Fatalf("a start on it: %+v, want unnamed with one note", s)
				}
				return
			}
			if out.Name == "" || len(out.Notes) != 0 {
				t.Fatalf("a valid register was refused: %+v", out)
			}
		})
	}
}

// TestTwoSessionsInOneProcessWithOneNameEndWithTwoNames: one host process
// runs two sessions, x and y, each given Alice, and only x holds it (a fork
// kept in the process, say). The entry of Alice on y's host is not y before a
// clear: its id still has Alice given to it. So y's prompt is the later claim
// and takes Bob, once, with the notice, and the prompts that follow go quiet:
// no rename, no write, and no process-table read.
func TestTwoSessionsInOneProcessWithOneNameEndWithTwoNames(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	m.launch(100)
	start := readAt.Add(-time.Hour).Unix()
	reg := fmt.Sprintf(`{"schema_version":1,"handed":["Alice"],"live":[{"name":"Alice","session_id":"x","host_pid":100,"host_start_unix":%d,"sent":true}],"given":{"x":"Alice","y":"Alice"}}`, start)
	if err := os.MkdirAll(abcdhome.Path(m.home, "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(m.home, "sessions", "names.json"), []byte(reg), 0o600); err != nil {
		t.Fatal(err)
	}
	title := map[string]string{"x": "Alice", "y": "Alice"}
	notices := 0
	for i, sid := range []string{"y", "x", "y", "x", "y", "x"} {
		before := m.register()
		reads := 0
		out := Prompt(counted(m.at(101), &reads), Request{Root: root, SessionID: sid, Title: title[sid]})
		if len(out.Notes) != 0 {
			t.Fatalf("prompt %d (%s): notes %q", i+1, sid, out.Notes)
		}
		if out.Name != "" {
			title[sid] = out.Name
		}
		if out.Notice != "" {
			notices++
		}
		// The first prompt renames y and the third sees the rename; every
		// other prompt reads no table and writes nothing.
		if i != 0 && i != 2 && (reads != 0 || !bytes.Equal(m.register(), before)) {
			t.Fatalf("prompt %d (%s) read the table %d time(s) or wrote:\n%s", i+1, sid, reads, m.register())
		}
	}
	if notices != 1 || title["x"] != "Alice" || title["y"] != "Bob" {
		t.Fatalf("%d notices, titles %v; want one notice, x Alice and y Bob", notices, title)
	}
}

// resend is Prompt for a session whose repository is not named: a session
// that has to take a new name takes it from the bundled list.
func resend(m Machine, sessionID, title string) Outcome {
	return Prompt(m, Request{SessionID: sessionID, Title: title})
}
