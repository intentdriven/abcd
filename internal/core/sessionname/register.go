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
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The register's place: one machine-scoped file in abcd's home, with its lock
// beside it. One file for the whole machine is what makes a name unique across
// repositories (decision 7); it lives in the account's home, so a second
// account on the machine keeps its own sequence (the fifth scope condition).
const (
	registerDir  = "sessions"
	registerFile = "names.json"
	// lockName is a separate file, never the register itself: the register is
	// replaced by a rename on every write, and a lock held on the replaced
	// inode would admit a second holder of the new one.
	lockName = registerFile + ".lock"
)

// registerSchema is the register's schema_version.
const registerSchema = 1

// maxRegisterBytes caps the register's read. An entry is about a hundred
// bytes, so the cap holds a couple of thousand running sessions and bounds a
// planted device or an endless file.
const maxRegisterBytes = 256 << 10

// maxReadTook is the longest a process-table read may take and still be used
// to judge which sessions have ended (prune). A start time read from the table
// is early by at most the read's length, and two readings of one process,
// each early by up to a second, stay within startTolerance of each other.
const maxReadTook = time.Second

// lockTimeout is the spec's two seconds: the longest a session start waits
// for another session's turn at the register before it starts unnamed (C11).
const lockTimeout = 2 * time.Second

// notePrefix opens every note, so a line on stderr says what wrote it.
const notePrefix = "abcd session name: "

// Machine is everything the register reads beyond its arguments.
type Machine struct {
	// Home is the person's home directory, absolute; abcd's home is under it.
	// Empty names no home, and nothing is named.
	Home string
	// From is the process the host walk starts from: the hook process's
	// parent.
	From int
	// ReadTable reads the process table once.
	ReadTable func() (Table, error)
	// ReadTableContext is ReadTable bound to ctx, which the prompt hook reads
	// through so that its read ends at maxReadTook (Prompt). Nil reads
	// through ReadTable, unbounded but for the reader's own limit.
	ReadTableContext func(ctx context.Context) (Table, error)
	// LockTimeout bounds the wait for the register's lock.
	LockTimeout time.Duration
}

// Live is this machine as the hooks see it: the account's home, the hook's
// parent, the process table machineload reads (one ps run on macOS, /proc on
// Linux, ErrUnsupported elsewhere) and the two-second lock wait.
func Live() Machine {
	home, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) {
		home = ""
	}
	return Machine{Home: home, From: os.Getppid(), ReadTable: readLiveTable, ReadTableContext: readLiveTableContext, LockTimeout: lockTimeout}
}

// readLiveTable reads the machine's process table and notes when the read
// began, which is what each process's age is measured back from, and how long
// it took.
func readLiveTable() (Table, error) { return readTable(machineload.Read, time.Now) }

// readLiveTableContext is readLiveTable bound to ctx. On macOS a ctx that ends
// while ps runs kills it, and the read fails; on Linux the /proc read is not
// stopped part way, and a read that took longer than the bound is refused
// afterwards by its length (promptHost).
func readLiveTableContext(ctx context.Context) (Table, error) {
	return readTable(func() (machineload.Snapshot, error) { return machineload.ReadContext(ctx) }, time.Now)
}

// readTable is readLiveTable with its reader and clock passed in.
//
// The moment is taken BEFORE the read. Under load a read can take seconds,
// and a moment taken after it would put every start time computed from the
// table that much later than it was: a session that started while the table
// was being read would look started before the read, so absent from a table
// that should have held it, and ended. Measured from before the read, a start
// time can only come out early, by at most the read's length, which Took
// records so prune can refuse to judge a slow read.
func readTable(read func() (machineload.Snapshot, error), now func() time.Time) (Table, error) {
	before := now()
	snap, err := read()
	took := now().Sub(before)
	if err != nil {
		return Table{}, err
	}
	if !snap.HasProcs {
		return Table{}, errors.New("the process table was not read")
	}
	return Table{Procs: snap.Procs, ReadAt: before, Took: took}, nil
}

// Request is what a session-start hook's payload says about the session.
type Request struct {
	// Root is the repository root the session works in, whose persona list
	// names it; empty takes the bundled list.
	Root string
	// SessionID is the host's id for the session. A session without one is
	// never named: the prompt hook finds its entry by it.
	SessionID string
	// Source is why the session started: startup, resume, clear, compact or
	// fork.
	Source string
	// Title is the session's current custom title: a name the person gave it,
	// or one a hook set earlier. Empty when it has none.
	Title string
	// AgentID is set on a sub-agent's payload, which is never named (C12).
	AgentID string
}

// Outcome is what a hook does with naming: the name (from Start, the name
// reserved; from Prompt, the title to set on this prompt; "" for none) and the
// notes for stderr, raw (the front door masks them as it prints).
type Outcome struct {
	Name  string
	Notes []string
	// Notice is the one line the prompt that sets a name in place of the
	// title the session carried says to the session and the person: which
	// name it now has, and why. Empty on every other outcome.
	Notice string
}

// entry is one running named session.
type entry struct {
	Name      string `json:"name"`
	SessionID string `json:"session_id"`
	// HostPID and HostStartUnix are the host process (Host).
	HostPID       int   `json:"host_pid"`
	HostStartUnix int64 `json:"host_start_unix"`
	// HostReadMS is how long, in milliseconds, the table read that stamped
	// HostStartUnix took (capped at maxReadSlack): how far that start time may
	// be early, which widens the tolerance this entry is judged by. Absent on
	// a fast read and on an entry written before it was recorded, where the
	// tolerance is unchanged.
	HostReadMS int64 `json:"host_read_ms,omitempty"`
	// Sent records that the prompt hook has dealt with the title: set it,
	// found it already set, or found the person's own name there. Session
	// start reserves the name and sets no title (decision 9); the title is
	// owed once, at the first prompt.
	Sent bool `json:"sent"`
	// Replaces is the title a resumed session or a fork carried that a
	// running session now holds (decision 10): the one title the prompt hook
	// replaces with Name. Empty on every other entry.
	Replaces string `json:"replaces,omitempty"`
}

// host is the host process the entry was recorded for, as that reading saw it.
func (e entry) host() Host {
	return Host{PID: e.HostPID, StartUnix: e.HostStartUnix, Slack: readSlack(time.Duration(e.HostReadMS) * time.Millisecond)}
}

// newEntry is a live entry for sessionID under host.
func newEntry(name, sessionID string, host Host) entry {
	return entry{Name: name, SessionID: sessionID, HostPID: host.PID, HostStartUnix: host.StartUnix,
		HostReadMS: readSlack(host.Slack).Milliseconds()}
}

// register is the file's shape.
type register struct {
	SchemaVersion int `json:"schema_version"`
	// Handed is every name given out since the last moment no named session
	// ran. A name stays here after its session ends, so it is not handed out
	// again while any named session runs (C7).
	Handed []string `json:"handed"`
	// Live is one entry per running named session.
	Live []entry `json:"live"`
	// Given is the name abcd's prompt hook set as each session's title, by
	// session id, and "" for a session abcd found carrying the person's own
	// name (given on the command line, or before its first prompt). It is
	// what tells abcd's title from the person's on resume, clear and fork
	// (C4, decision 10): a title is abcd's when it is the name given to that
	// session id, or, for a session id with no record of its own (a fork), a
	// name given to another session. It is not emptied with Handed, since a
	// session resumed after every session ended still carries abcd's name,
	// and it is bounded at maxGiven. A register written before it existed
	// reads as having given nothing.
	Given givenNames `json:"given"`
}

// maxGiven bounds the record of given names. A session that is never resumed
// stays in it until newer ones push it out; 512 is far more sessions than a
// person resumes from, and keeps the record within a few tens of kilobytes,
// well inside maxRegisterBytes.
const maxGiven = 512

// givenName is one session's given name.
type givenName struct {
	SessionID string
	Name      string
}

// givenNames is the record of given names, oldest first. On disk it is one
// JSON object, {session_id: name}, written and read in this order, so the
// order of its keys is the order the names were given. That order is what
// give drops by: when the record is past maxGiven, the oldest written goes
// first, and giving a session a name again moves it to the newest end.
type givenNames []givenName

// of is the name given to sessionID, and whether there is one.
func (g givenNames) of(sessionID string) (string, bool) {
	for _, n := range g {
		if n.SessionID == sessionID {
			return n.Name, true
		}
	}
	return "", false
}

// gave reports whether abcd gave sessionID the name title.
func (g givenNames) gave(sessionID, title string) bool {
	name, ok := g.of(sessionID)
	return ok && name == title
}

// inherits reports whether title is a name abcd gave another session while
// sessionID has no record of its own: a fork, which the host starts as a new
// session id carrying its parent's title. hold asks it for source fork alone;
// a resume or a clear never inherits. A session with a record of its own
// is never read as inheriting, so a title the person gave a session abcd knows
// stays theirs.
func (g givenNames) inherits(sessionID, title string) bool {
	if _, own := g.of(sessionID); own || title == "" {
		return false
	}
	return slices.ContainsFunc(g, func(n givenName) bool { return n.SessionID != sessionID && n.Name == title })
}

// give records name as given to sessionID, newest, and drops the oldest past
// maxGiven.
func (g *givenNames) give(sessionID, name string) {
	*g = append(slices.DeleteFunc(*g, func(n givenName) bool { return n.SessionID == sessionID }),
		givenName{SessionID: sessionID, Name: name})
	if over := len(*g) - maxGiven; over > 0 {
		*g = slices.Delete(*g, 0, over)
	}
}

// move carries the name given to from over to to: a clear gave the session a
// new id. Nothing is recorded when from was given nothing.
func (g *givenNames) move(from, to string) {
	if name, ok := g.of(from); ok && from != to {
		*g = slices.DeleteFunc(*g, func(n givenName) bool { return n.SessionID == from })
		g.give(to, name)
	}
}

// MarshalJSON writes the record as one object, oldest first.
func (g givenNames) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, n := range g {
		k, err := json.Marshal(n.SessionID)
		if err != nil {
			return nil, err
		}
		v, err := json.Marshal(n.Name)
		if err != nil {
			return nil, err
		}
		if i > 0 {
			b.WriteByte(',')
		}
		b.Write(k)
		b.WriteByte(':')
		b.Write(v)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// UnmarshalJSON reads the object in the order its keys are written. null
// reads as an empty record; a key seen twice keeps its later name.
func (g *givenNames) UnmarshalJSON(raw []byte) error {
	if string(bytes.TrimSpace(raw)) == "null" {
		*g = nil
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return errors.New("given is not an object")
	}
	var out givenNames
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		sessionID, ok := tok.(string)
		if !ok {
			return errors.New("given has a key that is not a string")
		}
		var name string
		if err := dec.Decode(&name); err != nil {
			return err
		}
		out.give(sessionID, name)
	}
	if _, err := dec.Token(); err != nil {
		return err
	}
	*g = out
	return nil
}

// Start is the session-start hook's naming.
//
// Start reserves; it never sets a title. The host ignores a prompt-hook title
// equal to the one a session already has, so a title set here would never
// reach the session list (decision 9). The returned Name is the name reserved,
// which the prompt hook sets at the session's first prompt (Resend).
//
// A sub-agent is never named, nor a payload with no session id. A new session
// (startup, or a fork) with no title reserves the next free name. A startup
// with a title is the person's own name: it is not named, and the register
// records that abcd gave it nothing (keep). A resumed or cleared session whose
// title is the name abcd gave it (Given) keeps it, and the register holds the
// name for the process now running it, so no other session is given it (C5).
// A fork, which the host starts as a new session id carrying its parent's
// title with source fork (seen live), inherits the name abcd gave the parent,
// and is held the same way. Either way, when a running session on another
// host holds that name, the session reserves the next free name instead, which
// its first prompt sets in place of the title (decision 10). Any other title
// is the person's, persona name or not, and is neither held nor replaced (C4).
// A compaction of a titled session changes nothing: the host keeps the title
// through it.
//
// A resume, clear or compaction with no title is a session whose first prompt
// has not set its name yet (rejoin): its entry is re-keyed to the process now
// running it, or, when it has none, it reserves the next free name as a new
// session does.
func Start(m Machine, req Request) Outcome {
	if !acts(req) {
		return Outcome{}
	}
	switch {
	case keeps(req):
		return keep(m, req)
	case req.Title != "":
		return hold(m, req)
	case req.Source == "startup" || req.Source == "fork":
		return name(m, req)
	}
	return rejoin(m, req)
}

// Acts reports whether Start reads the repository's persona list for req,
// which is the one thing it needs req.Root for, so a front door resolves the
// root (a git call) only then. It answers no for every payload naming never
// acts on (acts), and for a startup with a title, which only records that the
// title is the person's (keep) and needs no repository (iss-2610152324354563).
func Acts(req Request) bool {
	return acts(req) && !keeps(req)
}

// keeps reports whether req is a startup with a title: a name the person gave
// on the command line, which Start only records as theirs (keep).
func keeps(req Request) bool {
	return req.Title != "" && req.Source == "startup"
}

// acts reports whether Start can do anything for req. It does nothing for a
// sub-agent's payload (C12), one with no session id, or the compaction of a
// titled session, through which the host keeps the title (C5).
func acts(req Request) bool {
	if req.AgentID != "" || req.SessionID == "" {
		return false
	}
	switch req.Source {
	case "compact":
		return req.Title == ""
	case "startup", "fork", "resume", "clear":
		return true
	}
	return false
}

// keep is session start for a new session that starts with a title: a name
// the person gave on the command line, which wins (C4). It is never named. The
// register records that abcd gave this session no name (an empty given
// record), so that a later resume never reads the title as one inherited from
// another session that abcd did give it. No process table is read. A failure
// writes no note: the session is not named either way, and the next session
// start that names one names the refusal.
func keep(m Machine, req Request) Outcome {
	_ = m.locked(func(reg *register) (bool, error) {
		if _, own := reg.Given.of(req.SessionID); own {
			return false, nil
		}
		reg.Given.give(req.SessionID, "")
		return true, nil
	})
	return Outcome{}
}

// name reserves for the session the first name of its persona sequence that
// is not already handed out.
func name(m Machine, req Request) Outcome {
	personas, note := personasOf(req.Root)
	var out Outcome
	if note != "" {
		out.Notes = append(out.Notes, note)
	}
	err := m.update(func(reg *register, tab Table, host Host) (bool, error) {
		reg.prune(tab)
		var err error
		out.Name, err = reg.reserve(personas, req.SessionID, host, "")
		return err == nil, err
	})
	if err != nil {
		out.Name = ""
		out.Notes = append(out.Notes, failed(err))
	}
	return out
}

// rejoin is session start for a resume, clear or compaction that carries no
// title: a session whose name, if it has one, the first prompt has not set.
//
// Its entry, when it has one, is re-keyed to the host process now running it,
// keeping its name and whether the title is still owed: the old host has
// ended, and an entry left under it would be pruned at the next start and the
// name handed out again while this session's first prompt sets it. The entry
// is re-keyed before the prune, so it is judged by the host now running it. A
// compaction runs in the same process, so its entry is kept exactly as it is.
//
// A clear gives the session a new id in the same host process (seen live: a
// clear turned 7344b67a into 315e82fe and kept the process), so a clear whose
// id has no entry takes the entry on its own host (sameHost) as its own, and
// re-keys it to the new id as well; without that it would reserve a second
// name and leave the first held by an id no session carries.
//
// A session with no entry reserves the next free name, as a new session does.
// The returned name is the one its first prompt sets, "" when nothing is owed.
func rejoin(m Machine, req Request) Outcome {
	personas, note := personasOf(req.Root)
	var out Outcome
	if note != "" {
		out.Notes = append(out.Notes, note)
	}
	err := m.update(func(reg *register, tab Table, host Host) (bool, error) {
		i := slices.IndexFunc(reg.Live, func(e entry) bool { return e.SessionID == req.SessionID })
		if i < 0 && req.Source == "clear" {
			i = reg.onHost(host)
		}
		if i >= 0 {
			e := reg.Live[i]
			if !e.Sent {
				out.Name = e.Name
			}
			if req.Source == "compact" {
				return false, nil
			}
			reg.rekey(i, req.SessionID, host)
			reg.prune(tab)
			return true, nil
		}
		reg.prune(tab)
		var err error
		out.Name, err = reg.reserve(personas, req.SessionID, host, "")
		return err == nil, err
	})
	if err != nil {
		out.Name = ""
		out.Notes = append(out.Notes, failed(err))
	}
	return out
}

// rekey moves the live entry at i to sessionID under host, keeping its name,
// whether its title is still owed and the title it replaces, and carries its
// given record over to the new id. It returns the name still owed to the first
// prompt, or "". The entry moves to the end: the newest entry on a host is the
// session it runs now (onHost).
func (reg *register) rekey(i int, sessionID string, host Host) string {
	e := reg.Live[i]
	reg.Given.move(e.SessionID, sessionID)
	r := newEntry(e.Name, sessionID, host)
	r.Sent, r.Replaces = e.Sent, e.Replaces
	reg.Live = append(slices.Delete(reg.Live, i, i+1), r)
	if !slices.Contains(reg.Handed, e.Name) {
		reg.Handed = append(reg.Handed, e.Name)
	}
	if e.Sent {
		return ""
	}
	return e.Name
}

// beforeClear is the index of the live entry that was req's session before a
// clear gave it a new id, or -1; always -1 when req is not a clear. A clear
// keeps the host process, so that entry is on this same host: either it holds
// the title, given to its id, or it is a name reserved to replace the title
// (decision 10, or a fork's inherited name) that the first prompt has not set
// yet.
func (reg *register) beforeClear(req Request, host Host) int {
	if req.Source != "clear" {
		return -1
	}
	return reg.clearedFrom(req.Title, host)
}

// clearedFrom is the index of the live entry on host that accounts for title
// (accountsFor), or -1: on the same host process, the session a clear gave a
// new id.
func (reg *register) clearedFrom(title string, host Host) int {
	return slices.IndexFunc(reg.Live, func(e entry) bool {
		return sameHost(e.host(), host) && reg.accountsFor(e, title)
	})
}

// accountsFor reports whether the live entry e accounts for title as abcd's:
// e holds title as the name given to its id, or e is a name reserved to
// replace title that the first prompt has not set yet.
func (reg *register) accountsFor(e entry, title string) bool {
	return (e.Name == title && reg.Given.gave(e.SessionID, title)) ||
		(!e.Sent && e.Replaces == title)
}

// onHost is the index of the live entry recorded for host (sameHost, with
// each reading's slack), or -1. A host process runs one session at a time, so
// when several entries name it the newest, the last, is the one it runs now.
func (reg *register) onHost(host Host) int {
	for i := len(reg.Live) - 1; i >= 0; i-- {
		if sameHost(reg.Live[i].host(), host) {
			return i
		}
	}
	return -1
}

// reserve hands the session the first name of the persona sequence not
// already handed out, and records it live for this session's host, owed to the
// first prompt. replaces is the title it stands in for (decision 10), or "".
func (reg *register) reserve(personas []string, sessionID string, host Host, replaces string) (string, error) {
	for _, n := range sequenceOf(personas) {
		if slices.Contains(reg.Handed, n) {
			continue
		}
		reg.Handed = append(reg.Handed, n)
		reg.Live = slices.DeleteFunc(reg.Live, func(e entry) bool { return e.SessionID == sessionID })
		e := newEntry(n, sessionID, host)
		e.Replaces = replaces
		reg.Live = append(reg.Live, e)
		return n, nil
	}
	return "", errors.New("every name of this repository's persona sequence, through its ninth round, is held while a named session runs")
}

// claim holds title, the name abcd gave the session, for sessionID under host,
// the process now running it. It returns "" when the session holds title, and
// otherwise the next free name reserved in its place, owed to the prompt with
// title as the one it replaces (decision 10).
//
// Every entry whose session has ended is dropped first (prune), so a name is
// free only when the process that last held it is proven gone, by its id and
// its start time together. This session's own entry is replaced, and so is an
// entry of title under this same host whose id no longer has title given to
// it: the session before a clear gave it a new id, whose record the clear
// moved here. An entry of title on this host whose id still has title given
// to it is another session the same process runs (a fork kept in the
// process), and it is kept. When another running session still holds title,
// this claim is the later one, so it is the one that takes the next free
// name: the holder keeps its name (RFC 6762, section 9).
func (reg *register) claim(personas []string, sessionID, title string, host Host, tab Table) (string, error) {
	reg.prune(tab)
	reg.Live = slices.DeleteFunc(reg.Live, func(e entry) bool {
		return e.SessionID == sessionID ||
			(e.Name == title && sameHost(e.host(), host) && !reg.Given.gave(e.SessionID, title))
	})
	if slices.ContainsFunc(reg.Live, func(e entry) bool { return e.Name == title }) {
		return reg.reserve(personas, sessionID, host, title)
	}
	// Sent: the session already carries the title, so the session list shows
	// it and the prompt hook owes nothing.
	e := newEntry(title, sessionID, host)
	e.Sent = true
	reg.Live = append(reg.Live, e)
	if !slices.Contains(reg.Handed, title) {
		reg.Handed = append(reg.Handed, title)
	}
	return "", nil
}

// hold keeps a resumed session's name held for the process now running it.
// Only a name abcd gave is held, which Given records by session id: any other
// title is the person's own, a persona name the person typed included, and
// abcd neither records nor guards it (C4). Given, not this repository's
// persona sequence, is the test, so a name given before the sequence changed
// is still held.
//
// A clear gives the session a new id in the same process, so on a clear the
// session before it is the live entry on this same host (beforeClear): its
// name, given to its id, counts and is carried over to the new id; or, when
// that entry is a name reserved to replace the title and not set yet, the
// entry itself follows the clear, still owed. A clear is matched first, so the
// same session is never read as a fork of itself.
//
// A fork is a new session id carrying its parent's title, so a session id with
// no record of its own whose title is a name given to another session has
// inherited it (inherits). It is held like a resumed name, and recorded as
// given to the fork too. A session id with a record of its own is never read
// as inheriting: abcd records the person's own name as "" (keep, Resend), so
// a session the person named is not mistaken for a fork when another session
// is given the same name.
//
// The register is first read without the lock, as the prompt hook reads it:
// a title abcd never gave costs no process-table read and writes nothing,
// except on a clear, which records the new id as given nothing (keep), so a
// later resume of it keeps the person's title (C4).
//
// When a running session on another host holds the name, which happens once
// every named session had ended and naming restarted while this one was away,
// or when a fork's parent is still running, the session reserves the next free
// name instead, which its first prompt sets in place of the title it carries
// (decision 10): two running sessions never share a name (claim).
//
// A hold that cannot be made here (the lock or the process table not read in
// time) is made at the session's next prompt instead (Prompt). A fork has no
// record of its own for that prompt to go by, so when the process table cannot
// be read for a fork, its inherited title is recorded as given to it under the
// lock alone, as keep records a person's own name, and its prompt then makes
// the hold, taking the next free name while its parent runs.
func hold(m Machine, req Request) Outcome {
	personas, note := personasOf(req.Root)
	var out Outcome
	if note != "" {
		out.Notes = append(out.Notes, note)
	}
	if !m.mayHold(req) {
		// A clear of a title abcd never gave is the person's own under a
		// new id: record it as theirs, as a titled startup does, so a later
		// resume of the new id never reads it as inherited (C4).
		if req.Source == "clear" {
			keep(m, req)
		}
		return out
	}
	tab, host, err := m.tableAndHost(m.ReadTable)
	if err != nil {
		if req.Source == "fork" {
			_ = m.locked(func(reg *register) (bool, error) {
				if _, own := reg.Given.of(req.SessionID); own || !reg.Given.inherits(req.SessionID, req.Title) {
					return false, nil
				}
				reg.Given.give(req.SessionID, req.Title)
				return true, nil
			})
		}
		out.Notes = append(out.Notes, failed(err))
		return out
	}
	err = m.locked(func(reg *register) (bool, error) {
		inherited := false
		if !reg.Given.gave(req.SessionID, req.Title) {
			switch i := reg.beforeClear(req, host); {
			case i >= 0 && reg.Live[i].Name != req.Title:
				// A name reserved to replace this title, not set yet: it
				// follows the clear to the new id, still owed.
				out.Name = reg.rekey(i, req.SessionID, host)
				reg.prune(tab)
				return true, nil
			case i >= 0:
				reg.Given.move(reg.Live[i].SessionID, req.SessionID)
			// Only a fork inherits: the host starts one with source "fork"
			// (seen live). A clear is this session under a new id, and a
			// resume is this session again, so a title no record accounts
			// for there is the person's own (C4).
			case req.Source == "fork" && reg.Given.inherits(req.SessionID, req.Title):
				inherited = true
			case req.Source == "clear":
				// The person's own title under the clear's new id: recorded
				// as theirs, so a later resume of the new id keeps it (C4).
				if _, own := reg.Given.of(req.SessionID); own {
					return false, nil
				}
				reg.Given.give(req.SessionID, "")
				return true, nil
			default:
				return false, nil
			}
		}
		var err error
		if out.Name, err = reg.claim(personas, req.SessionID, req.Title, host, tab); err != nil {
			return false, err
		}
		// An inherited name is recorded as given to the fork as well, so a
		// later resume of the fork, or its prompt, finds it abcd's under its
		// own id: holding it, or, when the fork took the next free name and
		// its entry is gone before its first prompt, taking a free name again
		// rather than sharing the parent's. The parent's record is copied,
		// not moved: the parent still carries the name and may be resumed,
		// and its own record is what makes that resume abcd's.
		if inherited {
			reg.Given.give(req.SessionID, req.Title)
		}
		return true, nil
	})
	if err != nil {
		out.Name = ""
		out.Notes = append(out.Notes, failed(err))
	}
	return out
}

// mayHold reads the register without the lock and reports whether abcd may
// have given req's title to its session: the name given to its id, or, for an
// id with no record of its own (a new id), to any session.
//
// A clear never inherits, so a name given to some other session says nothing
// about it: only a live entry can account for its title, as the session it was
// before the clear (accountsFor). Which host this session runs on is not known
// without the process table, so an entry on any host admits it and the locked
// read tells this host's from another's (beforeClear); with no such entry the
// title is the person's own, and hold records it without reading the table.
//
// A register that cannot be read answers yes, so the locked read names the
// refusal; no register answers no.
func (m Machine) mayHold(req Request) bool {
	dir, reg, err := m.peek()
	if err != nil {
		return true
	}
	if dir == nil {
		return false
	}
	dir.Close()
	if reg.Given.gave(req.SessionID, req.Title) {
		return true
	}
	if req.Source == "clear" {
		return slices.ContainsFunc(reg.Live, func(e entry) bool { return reg.accountsFor(e, req.Title) })
	}
	_, own := reg.Given.of(req.SessionID)
	return !own && slices.ContainsFunc(reg.Given, func(n givenName) bool { return n.Name == req.Title })
}

// peek opens the sessions folder without creating it and reads the register
// without the lock: the file is only ever replaced by a rename, so an unlocked
// read sees one whole version of it. A machine with no home, no sessions
// folder, or a sessions folder that is a symlink (session start names that
// refusal) has no register: a nil folder and no error. The caller closes the
// folder.
func (m Machine) peek() (*os.Root, register, error) {
	if m.Home == "" || !filepath.IsAbs(m.Home) {
		return nil, register{}, nil
	}
	dir, err := fsutil.OpenHomeScope(m.Home, abcdhome.Rel(registerDir))
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) || errors.Is(err, fsutil.ErrHomeScopeSymlinked) {
		return nil, register{}, nil
	}
	if err != nil {
		return nil, register{}, folderError(err)
	}
	if err := vetFolder(dir); err != nil {
		dir.Close()
		return nil, register{}, err
	}
	reg, err := readRegister(dir, m.Home)
	if err != nil {
		dir.Close()
		return nil, register{}, err
	}
	return dir, reg, nil
}

// Resend is Prompt for a session whose repository is not named: a session
// that has to take a new name takes it from the bundled list.
func Resend(m Machine, sessionID, title string) Outcome {
	return Prompt(m, Request{SessionID: sessionID, Title: title})
}

// Prompt is the prompt hook's naming: the title to set on this prompt (Name,
// "" for none) and, when that title replaces the one the session carries, the
// notice that says so. req.Title is the session's current title from the
// prompt's payload, and req.Root the repository whose persona list names a
// session that has to take a new name here.
//
// The name session start reserved is owed once, at the first prompt (C1, C3,
// decision 9), and only to a session that has no title yet: a title equal to
// the name is already set, and any other title is the person's own, which
// always wins (C4). Either way the entry is marked sent. The one title
// replaced is a name abcd gave that a running session now holds, carried by a
// resumed session abcd named or by a fork (decision 10), and the prompt that
// replaces it carries the notice. The host discards a hook's whole output when
// the hook overruns, so a rename stays owed until a prompt shows the new name:
// a prompt that still carries the replaced title is sent the new name and the
// notice once more, and after that the title is the person's to keep, so a
// person who takes the old title back is not fought. A name set is recorded in
// Given, so a later resume or clear knows the title is abcd's; the person's
// own name is recorded as "", so a later resume never reads it as inherited.
//
// Every prompt also re-asserts the session's hold on its name, so a hold that
// session start could not make (its lock or its process-table read not done
// in time) is made at a later prompt instead of never, as a client re-creates
// its ephemeral state once its session is lost. Three cases call for it, and
// each is settled under the register's lock (reclaim, claim):
//
//   - the session has no entry while it carries the name abcd gave it (Given):
//     the entry is made again for the process now running the session;
//   - the session has no entry and no record of its own, and a live entry
//     accounts for its title (accountsFor): on this same host process that
//     entry is the session before a clear whose hold was not made, and its
//     record and its name follow the clear; on any other host the title is
//     the person's own, and it is recorded as theirs;
//   - the session carries the name of its entry, and an earlier live entry
//     holds the same name: two sessions hold one name.
//
// In either case, when another running session holds the name, this session
// is the later claim and takes the next free name, which this prompt sets in
// place of the title, with the notice. The register's live entries are kept in
// the order their names were claimed (reserve, claim and rekey append), and
// both sessions read that one order under the lock, so exactly one of them is
// the later claim: the other keeps its name, is told nothing, and is never
// displaced while its host runs (prune, by process id and start time).
//
// The register is first read without the lock: the file is only ever replaced
// by a rename, so an unlocked read sees one whole version of it, and on every
// prompt but the rare one it is all that is needed. A session whose entry is
// there, its own and first of its name, with nothing owed, costs that one read:
// no lock and no process table. A title owed takes the lock; only a missing or
// shared hold reads the process table, bounded at maxReadTook, and then takes
// the lock. A machine with no register touches nothing, nor does one whose
// sessions folder is a symlink: session start refuses that with its note, and
// the prompt hook does not repeat it on every prompt.
//
// Nothing here fails the hook. A register that cannot be read or written, a
// lock not free within LockTimeout, or a process table not read, or read more
// slowly than maxReadTook, changes nothing: the title is kept, no name is
// allocated, no notice is given, and one note says why. The next prompt tries
// again.
func Prompt(m Machine, req Request) Outcome {
	if req.SessionID == "" || req.AgentID != "" {
		return Outcome{}
	}
	dir, reg, err := m.peek()
	if err != nil {
		return Outcome{Notes: []string{unchanged(err)}}
	}
	if dir == nil {
		return Outcome{}
	}
	defer dir.Close()
	need := reg.needs(req.SessionID, req.Title)
	if need == needNothing {
		return Outcome{}
	}
	var out Outcome
	var personas []string
	var tab *Table
	var host Host
	if need == needClaim {
		var note string
		if personas, note = personasOf(req.Root); note != "" {
			out.Notes = append(out.Notes, note)
		}
		t, h, err := m.promptHost()
		if err != nil {
			out.Notes = append(out.Notes, unchanged(err))
			return out
		}
		tab, host = &t, h
	}
	err = lockRegister(dir, m.LockTimeout, func() error {
		reg, err := readRegister(dir, m.Home)
		if err != nil {
			return err
		}
		changed, err := reg.assert(personas, req.SessionID, req.Title, tab, host, &out)
		if err != nil || !changed {
			return err
		}
		return writeRegister(dir, reg)
	})
	if err != nil {
		out.Name, out.Notice = "", ""
		out.Notes = append(out.Notes, unchanged(err))
	}
	return out
}

// promptNeed is what a prompt has to do for its session's hold.
type promptNeed int

const (
	// needNothing: the hold is in place and nothing is owed.
	needNothing promptNeed = iota
	// needSend: a title is owed or to be marked dealt with; the lock alone.
	needSend
	// needClaim: the hold is missing or shared; the process table, then the
	// lock (claim).
	needClaim
)

// needs is what a prompt from sessionID carrying title has to do (Prompt).
func (reg *register) needs(sessionID, title string) promptNeed {
	i := slices.IndexFunc(reg.Live, func(e entry) bool { return e.SessionID == sessionID })
	if i < 0 {
		if title == "" {
			return needNothing
		}
		given, own := reg.Given.of(sessionID)
		switch {
		case own && given == title:
			return needClaim
		case !own && slices.ContainsFunc(reg.Live, func(e entry) bool { return reg.accountsFor(e, title) }):
			return needClaim
		}
		return needNothing
	}
	e := reg.Live[i]
	switch {
	case e.Sent && title == e.Name && slices.ContainsFunc(reg.Live[:i], func(o entry) bool { return o.Name == title }):
		return needClaim
	case !e.Sent:
		return needSend
	case e.Replaces != "" && (title == e.Replaces || title == e.Name):
		// A rename sent and not yet seen: sent once more, or seen now.
		return needSend
	}
	return needNothing
}

// assert does what a prompt from sessionID carrying title needs, under the
// lock, and reports whether it changed the register. tab is nil when the
// prompt read no process table: a hold found missing or shared only now, under
// the lock, is then left for a later prompt, and nothing is changed for it.
func (reg *register) assert(personas []string, sessionID, title string, tab *Table, host Host, out *Outcome) (bool, error) {
	changed := false
	switch reg.needs(sessionID, title) {
	case needNothing:
		return false, nil
	case needClaim:
		if tab == nil {
			return false, nil
		}
		if err := reg.reclaim(personas, sessionID, title, host, *tab); err != nil {
			return false, err
		}
		changed = true
	}
	i := slices.IndexFunc(reg.Live, func(e entry) bool { return e.SessionID == sessionID })
	if i < 0 {
		return changed, nil
	}
	e := &reg.Live[i]
	switch {
	case !e.Sent:
		e.Sent = true
		switch {
		case e.Replaces != "" && title == e.Replaces:
			// The rename: sent with its notice, and Replaces kept until a
			// prompt shows the new name, so one dropped output is not the
			// end of it.
			out.Name = e.Name
			out.Notice = renamed(e.Name, e.Replaces)
			reg.Given.give(sessionID, e.Name)
		case title == "":
			out.Name = e.Name
			e.Replaces = ""
			reg.Given.give(sessionID, e.Name)
		case title != e.Name:
			// The person's own name, given before this first prompt: abcd
			// gave this session nothing, so a later resume never reads the
			// title as inherited.
			e.Replaces = ""
			reg.Given.give(sessionID, "")
		default:
			e.Replaces = ""
		}
	case e.Replaces != "" && title == e.Replaces:
		// The rename was sent and the prompt still carries the old title:
		// the output that carried it may have been dropped. Sent once more,
		// and then the title is left to the person.
		out.Name = e.Name
		out.Notice = renamed(e.Name, e.Replaces)
		e.Replaces = ""
	case e.Replaces != "" && title == e.Name:
		// The rename is seen: nothing more is owed.
		e.Replaces = ""
	default:
		return changed, nil
	}
	return true, nil
}

// reclaim makes the hold a prompt found missing or shared (needs, needClaim).
// A session with no entry and no record of its own carries a title a live
// entry accounts for: on this same host that entry is the session before a
// clear whose hold was not made, so its record moves to this id (or, when it
// is a name reserved to replace the title and not set yet, the entry itself
// follows the clear, still owed, as hold does); on any other host the title is
// the person's own, recorded as theirs (C4), and nothing is claimed. Every
// other hold is claimed (claim).
func (reg *register) reclaim(personas []string, sessionID, title string, host Host, tab Table) error {
	_, own := reg.Given.of(sessionID)
	if !own && !slices.ContainsFunc(reg.Live, func(e entry) bool { return e.SessionID == sessionID }) {
		switch i := reg.clearedFrom(title, host); {
		case i >= 0 && reg.Live[i].Name != title:
			reg.rekey(i, sessionID, host)
			reg.prune(tab)
			return nil
		case i >= 0:
			reg.Given.move(reg.Live[i].SessionID, sessionID)
		default:
			reg.Given.give(sessionID, "")
			return nil
		}
	}
	_, err := reg.claim(personas, sessionID, title, host, tab)
	return err
}

// promptHost reads the process table for a prompt, bounded at maxReadTook, and
// finds this session's host in it. A read slower than that is refused, as
// prune would refuse to judge by it: a prompt changes nothing on it.
func (m Machine) promptHost() (Table, Host, error) {
	read := m.ReadTable
	if m.ReadTableContext != nil {
		read = func() (Table, error) {
			ctx, cancel := context.WithTimeout(context.Background(), maxReadTook)
			defer cancel()
			return m.ReadTableContext(ctx)
		}
	}
	tab, host, err := m.tableAndHost(read)
	if err == nil && tab.Took > maxReadTook {
		return Table{}, Host{}, fmt.Errorf("the process table took longer than %s to read, so a running session cannot be told from an ended one", maxReadTook)
	}
	return tab, host, err
}

// prune drops every entry whose session has ended, and empties Handed when
// none is left, so naming starts again at Alice (C8).
//
// An entry whose host started after the table was read (within the start
// tolerance, widened by the slack of the read that recorded the entry) cannot
// be judged by that table: the process is simply not in it yet. Such an entry is a session that started while this one was reading the
// table and was registered first, and it is kept. This is what lets the table
// be read before the register's lock is taken, so a platform or a process
// table abcd cannot read costs no write to the home, without two sessions
// starting together pruning each other.
//
// A table that took longer than maxReadTook to read judges nothing: every
// start time computed from it may be early by that long, more than the
// tolerance allows for, so a running session could read as ended.
func (reg *register) prune(tab Table) {
	if tab.Took > maxReadTook {
		return
	}
	reg.Live = slices.DeleteFunc(reg.Live, func(e entry) bool {
		h := e.host()
		if h.StartUnix+slackSecs(h.Slack) >= tab.ReadAt.Unix()-startTolerance {
			return false
		}
		return !tab.Alive(h)
	})
	if len(reg.Live) == 0 {
		reg.Handed = nil
	}
}

// update reads the process table and finds this session's host, then reads,
// changes and writes the register under its lock. fn reports whether it
// changed the register; an error from fn is returned and nothing is written.
func (m Machine) update(fn func(reg *register, tab Table, host Host) (bool, error)) error {
	if m.Home == "" || !filepath.IsAbs(m.Home) {
		return errors.New("there is no home folder to keep the name register in")
	}
	tab, host, err := m.tableAndHost(m.ReadTable)
	if err != nil {
		return err
	}
	return m.locked(func(reg *register) (bool, error) { return fn(reg, tab, host) })
}

// tableAndHost reads the process table once through read and finds this
// session's host in it.
func (m Machine) tableAndHost(read func() (Table, error)) (Table, Host, error) {
	if read == nil {
		return Table{}, Host{}, errors.New("the process table could not be read")
	}
	tab, err := read()
	if errors.Is(err, machineload.ErrUnsupported) {
		return Table{}, Host{}, errors.New("abcd reads the process table on macOS and Linux only, so on this platform it cannot tell a running session from an ended one")
	}
	if err != nil {
		return Table{}, Host{}, fmt.Errorf("the process table could not be read (%v), so a running session cannot be told from an ended one", err)
	}
	host, err := FindHost(tab, m.From)
	if err != nil {
		return Table{}, Host{}, err
	}
	return tab, host, nil
}

// locked reads, changes and writes the register under its lock, reading no
// process table. fn reports whether it changed the register; an error from fn
// is returned and nothing is written.
func (m Machine) locked(fn func(reg *register) (bool, error)) error {
	if m.Home == "" || !filepath.IsAbs(m.Home) {
		return errors.New("there is no home folder to keep the name register in")
	}
	// The folder is created, judged and opened in one walk from home, and the
	// lock and the file are reached through it, so a link swapped in after the
	// judgement is refused rather than written through.
	dir, err := fsutil.EnsureHomeScope(m.Home, abcdhome.Rel(registerDir), abcdhome.DirMode)
	if err != nil {
		return folderError(err)
	}
	defer dir.Close()
	if err := vetFolder(dir); err != nil {
		return err
	}
	return lockRegister(dir, m.LockTimeout, func() error {
		reg, err := readRegister(dir, m.Home)
		if err != nil {
			return err
		}
		changed, err := fn(&reg)
		if err != nil || !changed {
			return err
		}
		return writeRegister(dir, reg)
	})
}

// lockRegister runs fn under the register's lock, waiting at most timeout.
func lockRegister(dir *os.Root, timeout time.Duration, fn func() error) error {
	err := fsutil.WithFileLockIn(dir, lockName, timeout, fn)
	switch {
	case errors.Is(err, fsutil.ErrLockContention):
		return fmt.Errorf("the lock on %s was not free within %s", abcdhome.Display(registerDir, registerFile), timeout)
	case errors.Is(err, fsutil.ErrLockPathUnsafe):
		return fmt.Errorf("the lock %s is not a regular file (a symlink, or something else); remove it and the next session creates it afresh", abcdhome.Display(registerDir, lockName))
	}
	return err
}

// readRegister reads the register through dir, with the guards the machine's
// rules.json gets: a regular file, never a symlink, owned by this account and
// writable by nobody else, within the cap. An absent register is an empty one.
func readRegister(dir *os.Root, home string) (register, error) {
	shown := abcdhome.Display(registerDir, registerFile)
	raw, refusal, err := fsutil.ReadHomeDeclarationDenyingIn(dir, home, abcdhome.Rel(registerDir, registerFile), maxRegisterBytes, 0)
	switch {
	case refusal == fsutil.DeclarationAbsent && errors.Is(err, os.ErrNotExist):
		return register{SchemaVersion: registerSchema}, nil
	case refusal == fsutil.DeclarationOK:
	case refusal == fsutil.DeclarationWritableByOthers:
		return register{}, fmt.Errorf("%s is writable by others, so it is not necessarily yours; `chmod go-w %s`", shown, shown)
	case refusal == fsutil.DeclarationForeignOwner:
		return register{}, fmt.Errorf("%s is not owned by this account, so it is not read", shown)
	case refusal == fsutil.DeclarationNotRegular:
		return register{}, fmt.Errorf("%s is not a regular file (a symlink, folder or device is refused)", shown)
	case errors.Is(err, fsutil.ErrTooBig):
		return register{}, fmt.Errorf("%s is larger than %d bytes", shown, maxRegisterBytes)
	default:
		return register{}, fmt.Errorf("%s could not be read", shown)
	}
	var reg register
	if err := json.Unmarshal(raw, &reg); err != nil || reg.SchemaVersion != registerSchema || !reg.namesAllocatable() {
		return register{}, fmt.Errorf("%s is not a name register abcd can read; remove it and the next session starts the register afresh", shown)
	}
	return reg, nil
}

// namesAllocatable reports whether every name the register holds is one the
// allocator hands out (allocatable): each live entry's name and the title it
// replaces (which may be empty), each name handed out, and each name given
// (which may be empty, for the person's own). abcd writes no other name into
// these fields: a title is held, replaced or recorded as given only when it
// is a name abcd gave, and the person's own title is recorded as "". The
// register decides the title a session is set, so one carrying any other text
// is refused whole, as an unreadable one is.
func (reg register) namesAllocatable() bool {
	for _, e := range reg.Live {
		if !allocatable(e.Name) || (e.Replaces != "" && !allocatable(e.Replaces)) {
			return false
		}
	}
	for _, n := range reg.Handed {
		if !allocatable(n) {
			return false
		}
	}
	for _, g := range reg.Given {
		if g.Name != "" && !allocatable(g.Name) {
			return false
		}
	}
	return true
}

// writeRegister replaces the register in one rename, at the account-only mode.
func writeRegister(dir *os.Root, reg register) error {
	reg.SchemaVersion = registerSchema
	if reg.Handed == nil {
		reg.Handed = []string{}
	}
	if reg.Live == nil {
		reg.Live = []entry{}
	}
	body, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	if err := fsutil.WriteFileAtomicInRoot(dir, registerFile, append(body, '\n'), abcdhome.FileMode); err != nil {
		return fmt.Errorf("%s could not be written", abcdhome.Display(registerDir, registerFile))
	}
	return nil
}

// vetFolder refuses a sessions folder another account could change: one whose
// group or others can write it, or one this account does not own. Whoever can
// write the folder can put a register of their own making in place of this
// account's, and the register decides what title a session is given.
func vetFolder(dir *os.Root) error {
	fi, err := dir.Stat(".")
	if err != nil {
		return folderError(err)
	}
	shown := abcdhome.Display(registerDir)
	if fsutil.WritableByOthers(fi) {
		return fmt.Errorf("%s can be written by another account (mode %04o), so the register in it is not necessarily yours; `chmod go-w %s`", shown, uint32(fi.Mode().Perm()), shown)
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); !ok || st.Uid != uint32(os.Getuid()) {
		return fmt.Errorf("%s is not owned by this account, so the register in it is not read", shown)
	}
	return nil
}

// folderError words a refusal to create or open the sessions folder. A
// symlinked level's error already names it in tilde form; any other error can
// carry the absolute path, so it is not quoted.
func folderError(err error) error {
	if errors.Is(err, fsutil.ErrHomeScopeSymlinked) {
		return err
	}
	return fmt.Errorf("%s could not be created or opened", abcdhome.Display(registerDir))
}

// failed is the note for a naming that could not complete.
func failed(err error) string {
	return notePrefix + err.Error() + "; this session is not named"
}

// unchanged is the note for a prompt whose naming could not complete: it
// changed nothing, and the next prompt tries again.
func unchanged(err error) string {
	return notePrefix + err.Error() + "; the session's name is left as it is on this prompt"
}

// renamed is the notice for a prompt that sets name in place of the title
// replaced, which another running session holds.
func renamed(name, replaced string) string {
	return notePrefix + "another running session holds the name " + replaced + ", so this session is now " + name + "."
}
