package sessionname

import (
	"errors"
	"testing"
	"time"
)

// NameOf reads the register for a session's persona name, and a session it
// does not hold, or a machine with none, has no name.
func TestNameOfReadsTheRegister(t *testing.T) {
	m := newMachine(t)
	got := m.start(100, "/repo", "sess-1")
	shell := 101
	if got.Name == "" {
		t.Fatal("the fixture session was given no name")
	}
	if name := NameOf(m.at(shell), "sess-1"); name != got.Name {
		t.Errorf("NameOf = %q, want %q", name, got.Name)
	}
	if name := NameOf(m.at(shell), "sess-2"); name != "" {
		t.Errorf("an unknown session's name = %q", name)
	}
	if name := NameOf(newMachine(t).at(shell), "sess-1"); name != "" {
		t.Errorf("a machine with no register names %q", name)
	}
}

// SessionAbove finds the session a process descends from, however much stands
// between them: a `make` that is neither a shell nor abcd does not hide it.
func TestSessionAboveWalksPastWhateverIsBetween(t *testing.T) {
	m := newMachine(t)
	got := m.start(100, "/repo", "sess-1")
	shell := 101
	m.procs = append(m.procs, proc(300, shell, "make", time.Minute), proc(301, 300, "abcd", time.Second))
	id, name := SessionAbove(m.at(301))
	if id != "sess-1" || name != got.Name {
		t.Errorf("SessionAbove = %q %q, want sess-1 %q", id, name, got.Name)
	}
}

// A process under no named session, one whose host has ended and had its id
// reused, and a table that cannot be read all answer none.
func TestSessionAboveAnswersNoneWhenNoSessionIsAbove(t *testing.T) {
	m := newMachine(t)
	m.start(100, "/repo", "sess-1")
	m.procs = append(m.procs, proc(500, 1, "login", time.Hour), proc(501, 500, "abcd", time.Second))
	if id, _ := SessionAbove(m.at(501)); id != "" {
		t.Errorf("a plain terminal's process found session %q", id)
	}
	m.end(100)
	m.procs = append(m.procs, proc(100, 1, "claude", time.Second), proc(601, 100, "abcd", time.Second))
	if id, _ := SessionAbove(m.at(601)); id != "" {
		t.Errorf("a reused process id found session %q", id)
	}
	m.tableErr = errTestTable
	if id, _ := SessionAbove(m.at(601)); id != "" {
		t.Errorf("an unreadable table found session %q", id)
	}
}

var errTestTable = errors.New("no table")
