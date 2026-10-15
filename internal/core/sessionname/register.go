package sessionname

import (
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
	return Machine{Home: home, From: os.Getppid(), ReadTable: readLiveTable, LockTimeout: lockTimeout}
}

// readLiveTable reads the machine's process table and notes when it was read,
// which is what each process's age is measured back from.
func readLiveTable() (Table, error) {
	snap, err := machineload.Read()
	at := time.Now()
	if err != nil {
		return Table{}, err
	}
	if !snap.HasProcs {
		return Table{}, errors.New("the process table was not read")
	}
	return Table{Procs: snap.Procs, ReadAt: at}, nil
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

// Outcome is what a hook does with naming: the title to set ("" for none) and
// the notes for stderr, raw (the front door masks them as it prints).
type Outcome struct {
	Name  string
	Notes []string
}

// entry is one running named session.
type entry struct {
	Name      string `json:"name"`
	SessionID string `json:"session_id"`
	// HostPID and HostStartUnix are the host process (Host).
	HostPID       int   `json:"host_pid"`
	HostStartUnix int64 `json:"host_start_unix"`
	// Resent records that the prompt hook has set the title again, which is
	// what puts it in the session list (C3); it is owed once.
	Resent bool `json:"resent"`
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
}

// Start is the session-start hook's naming.
//
// A sub-agent is never named, nor a payload with no session id. A new session
// (startup, or a fork) with no title takes the next free name. A resumed or
// cleared session whose title is one of this repository's persona names keeps
// it, and the register holds the name for the process now running it, so no
// other session is given it (C5); no title is set. A compaction changes
// nothing: the host keeps the title through it.
func Start(m Machine, req Request) Outcome {
	if !Acts(req) {
		return Outcome{}
	}
	if req.Title == "" {
		return name(m, req)
	}
	return hold(m, req)
}

// Acts reports whether Start can do anything for req, so a front door can skip
// resolving the repository root for a payload naming never acts on: a
// sub-agent's (C12), one with no session id, a new session that already has a
// title (the person's own name wins, C4), a resume or clear without one, and a
// compaction, through which the host keeps the title (C5).
func Acts(req Request) bool {
	if req.AgentID != "" || req.SessionID == "" {
		return false
	}
	switch req.Source {
	case "startup", "fork":
		return req.Title == ""
	case "resume", "clear":
		return req.Title != ""
	}
	return false
}

// name hands the session the first name of its persona sequence that is not
// already handed out.
func name(m Machine, req Request) Outcome {
	personas, note := Personas(req.Root)
	var out Outcome
	if note != "" {
		out.Notes = append(out.Notes, note)
	}
	var chosen string
	err := m.update(func(reg *register, tab Table, host Host) (bool, error) {
		reg.prune(tab)
		for _, n := range Sequence(personas) {
			if slices.Contains(reg.Handed, n) {
				continue
			}
			chosen = n
			reg.Handed = append(reg.Handed, n)
			reg.Live = slices.DeleteFunc(reg.Live, func(e entry) bool { return e.SessionID == req.SessionID })
			reg.Live = append(reg.Live, entry{Name: n, SessionID: req.SessionID, HostPID: host.PID, HostStartUnix: host.StartUnix})
			return true, nil
		}
		return false, errors.New("every name of this repository's persona sequence, through its ninth round, is held while a named session runs")
	})
	if err != nil {
		out.Notes = append(out.Notes, failed(err))
		return out
	}
	out.Name = chosen
	return out
}

// hold keeps a resumed session's name held for the process now running it.
// Only a name this repository's sequence hands out is held: any other title is
// the person's own, and abcd neither records nor guards it.
func hold(m Machine, req Request) Outcome {
	personas, note := Personas(req.Root)
	var out Outcome
	if note != "" {
		out.Notes = append(out.Notes, note)
	}
	if !slices.Contains(Sequence(personas), req.Title) {
		return out
	}
	err := m.update(func(reg *register, tab Table, host Host) (bool, error) {
		reg.prune(tab)
		reg.Live = slices.DeleteFunc(reg.Live, func(e entry) bool {
			return e.SessionID == req.SessionID ||
				(e.Name == req.Title && e.HostPID == host.PID && e.HostStartUnix == host.StartUnix)
		})
		// Resent: the host kept the title as the session's own, so the
		// session list already shows it and the prompt hook owes nothing.
		reg.Live = append(reg.Live, entry{Name: req.Title, SessionID: req.SessionID,
			HostPID: host.PID, HostStartUnix: host.StartUnix, Resent: true})
		if !slices.Contains(reg.Handed, req.Title) {
			reg.Handed = append(reg.Handed, req.Title)
		}
		return true, nil
	})
	if err != nil {
		out.Notes = append(out.Notes, failed(err))
	}
	return out
}

// Resend is the prompt hook's naming: the title to set again on this prompt,
// or "". It is owed once, to the session abcd named, while its title is still
// the name abcd gave it (C3); a session the person renamed is owed nothing
// (C4), because its title no longer equals the entry's name.
//
// A prompt with no title, or on a machine with no register, touches nothing.
// The register is first read without the lock, which is enough to learn that
// nothing is owed, the answer on every prompt but one: the file is only ever
// replaced by a rename, so an unlocked read sees one whole version of it. The
// lock is taken only to mark the title sent.
func Resend(m Machine, sessionID, title string) Outcome {
	if title == "" || sessionID == "" || m.Home == "" || !filepath.IsAbs(m.Home) {
		return Outcome{}
	}
	dir, err := fsutil.OpenHomeScope(m.Home, abcdhome.Rel(registerDir))
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return Outcome{}
	}
	if err != nil {
		return Outcome{Notes: []string{failed(folderError(err))}}
	}
	defer dir.Close()
	if err := vetFolder(dir); err != nil {
		return Outcome{Notes: []string{failed(err)}}
	}
	reg, err := readRegister(dir, m.Home)
	if err != nil {
		return Outcome{Notes: []string{failed(err)}}
	}
	if reg.owed(sessionID, title) < 0 {
		return Outcome{}
	}
	sent := false
	err = lockRegister(dir, m.LockTimeout, func() error {
		reg, err := readRegister(dir, m.Home)
		if err != nil {
			return err
		}
		i := reg.owed(sessionID, title)
		if i < 0 {
			return nil
		}
		reg.Live[i].Resent = true
		if err := writeRegister(dir, reg); err != nil {
			return err
		}
		sent = true
		return nil
	})
	if err != nil {
		return Outcome{Notes: []string{failed(err)}}
	}
	if !sent {
		return Outcome{}
	}
	return Outcome{Name: title}
}

// owed is the index of the live entry that is owed a re-sent title, or -1.
func (reg *register) owed(sessionID, title string) int {
	for i, e := range reg.Live {
		if e.SessionID == sessionID && e.Name == title && !e.Resent {
			return i
		}
	}
	return -1
}

// prune drops every entry whose session has ended, and empties Handed when
// none is left, so naming starts again at Alice (C8).
//
// An entry whose host started after the table was read (within the start
// tolerance) cannot be judged by that table: the process is simply not in it
// yet. Such an entry is a session that started while this one was reading the
// table and was registered first, and it is kept. This is what lets the table
// be read before the register's lock is taken, so a platform or a process
// table abcd cannot read costs no write to the home, without two sessions
// starting together pruning each other.
func (reg *register) prune(tab Table) {
	reg.Live = slices.DeleteFunc(reg.Live, func(e entry) bool {
		if e.HostStartUnix >= tab.ReadAt.Unix()-startTolerance {
			return false
		}
		return !tab.Alive(Host{PID: e.HostPID, StartUnix: e.HostStartUnix})
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
	if m.ReadTable == nil {
		return errors.New("the process table could not be read")
	}
	tab, err := m.ReadTable()
	if errors.Is(err, machineload.ErrUnsupported) {
		return errors.New("abcd reads the process table on macOS and Linux only, so on this platform it cannot tell a running session from an ended one")
	}
	if err != nil {
		return fmt.Errorf("the process table could not be read (%v), so a running session cannot be told from an ended one", err)
	}
	host, err := FindHost(tab, m.From)
	if err != nil {
		return err
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
		changed, err := fn(&reg, tab, host)
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
	if err := json.Unmarshal(raw, &reg); err != nil || reg.SchemaVersion != registerSchema {
		return register{}, fmt.Errorf("%s is not a name register abcd can read; remove it and the next session starts the register afresh", shown)
	}
	return reg, nil
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
