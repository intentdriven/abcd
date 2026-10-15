package sessionname

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	// The resumed session is not owed a re-sent title: the host kept it.
	if got := Resend(m.at(301), "b", "Bob"); got.Name != "" {
		t.Fatalf("the resumed session was re-sent %q", got.Name)
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

// TestResendCarriesTheTitleOnce is C3's register half: the first prompt of a
// session abcd named is owed the title once, so the session list catches up;
// later prompts, a renamed session and another session are owed nothing.
func TestResendCarriesTheTitleOnce(t *testing.T) {
	m := newMachine(t)
	root := t.TempDir()
	mustName(t, m.start(100, root, "s1"), "Alice")
	mustName(t, m.start(200, root, "s2"), "Bob")
	for _, c := range []struct {
		sid, title, want string
	}{
		{"s1", "", ""},
		{"s1", "Zed", ""},   // renamed by the person: their name wins (C4)
		{"s2", "Alice", ""}, // another session's name
		{"s1", "Alice", "Alice"},
		{"s1", "Alice", ""}, // once
		{"s2", "Bob", "Bob"},
	} {
		got := Resend(m.at(0), c.sid, c.title)
		if got.Name != c.want || len(got.Notes) != 0 {
			t.Fatalf("Resend(%q, %q) = %+v, want %q", c.sid, c.title, got, c.want)
		}
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
