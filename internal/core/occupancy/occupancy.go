// Package occupancy records the checkout each running host session started
// in, so a verb that writes records can tell when the main working tree it is
// about to write into is shared with another live session
// (iss-2610240859590333).
//
// On 2026-10-24 a session filed an issue, an intent, an ADR and two
// regenerated indexes into a main checkout another live session was working
// in, folding the peer's untracked draft into the indexes. The rule that a
// concurrent session works in its own worktree held only as prose. The
// session-start hook now writes one marker per session under abcd's home,
// naming the checkout and the host process; a record-writing verb counts the
// markers for its checkout whose host is still running.
//
// A marker is keyed on the host process, not on a heartbeat, so a session
// that ended without a SessionEnd hook (a crash, a killed terminal, a hook the
// host cut short) stops counting the moment its process is gone. One host
// runs one session at a time, so a new marker replaces any other under the
// same host (a /clear or a resume whose SessionEnd never ran), and each Note
// prunes every marker whose host has ended.
//
// The folder is reached through one walk from home and every file through
// that descriptor, with the guards the session-name register has: a symlinked
// ~/.abcd.noindex or folder is refused, and a marker another account owns or
// can write is not read.
package occupancy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// dirLeaf is the markers' folder under abcd's home.
const dirLeaf = "occupancy"

// maxMarkerBytes bounds one marker's read: a marker is a few hundred bytes.
const maxMarkerBytes = 4 << 10

// sessionIDRe is the shape a session id must have to name a file.
var sessionIDRe = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

// marker is one running session's record.
type marker struct {
	SessionID     string `json:"session_id"`
	Checkout      string `json:"checkout"`
	HostPID       int    `json:"host_pid"`
	HostStartUnix int64  `json:"host_start_unix"`
	HostReadMS    int64  `json:"host_read_ms,omitempty"`
}

func (m marker) host() sessionname.Host {
	return sessionname.Host{PID: m.HostPID, StartUnix: m.HostStartUnix, Slack: time.Duration(m.HostReadMS) * time.Millisecond}
}

func validID(id string) bool {
	return sessionIDRe.MatchString(id) && id != "." && id != ".."
}

// found is one marker as read: its file name, its bytes, and what they say.
type found struct {
	leaf string
	raw  []byte
	m    marker
}

// open walks to the markers' folder from home, creating it when create is
// set. An absent folder opened without create is (nil, nil).
func open(home string, create bool) (*os.Root, error) {
	if home == "" || !filepath.IsAbs(home) {
		return nil, errors.New("there is no home folder to keep the markers in")
	}
	rel := abcdhome.Rel(dirLeaf)
	if !create {
		if _, err := os.Lstat(filepath.Join(home, filepath.FromSlash(rel))); errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
	}
	return fsutil.EnsureHomeScope(home, rel, abcdhome.DirMode)
}

// readAll reads every marker in dir that passes the guards. A file that does
// not, or that does not parse as the marker its name says, is skipped: it
// names no session this package can vouch for.
func readAll(dir *os.Root, home string) ([]found, error) {
	f, err := dir.Open(".")
	if err != nil {
		return nil, err
	}
	entries, err := f.ReadDir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	var out []found
	for _, e := range entries {
		leaf := e.Name()
		if !e.Type().IsRegular() || !strings.HasSuffix(leaf, ".json") {
			continue
		}
		raw, refusal, err := fsutil.ReadHomeDeclarationDenyingIn(dir, home, abcdhome.Rel(dirLeaf, leaf), maxMarkerBytes, 0)
		if err != nil || refusal != fsutil.DeclarationOK {
			continue
		}
		var m marker
		if json.Unmarshal(raw, &m) != nil || m.SessionID+".json" != leaf {
			continue
		}
		out = append(out, found{leaf, raw, m})
	}
	return out, nil
}

// removeIfUnchanged removes f's file only when it still holds the bytes f was
// read with, so a marker a session rewrote after the read is kept.
func removeIfUnchanged(dir *os.Root, home string, f found) {
	raw, refusal, err := fsutil.ReadHomeDeclarationDenyingIn(dir, home, abcdhome.Rel(dirLeaf, f.leaf), maxMarkerBytes, 0)
	if err == nil && refusal == fsutil.DeclarationOK && bytes.Equal(raw, f.raw) {
		_ = dir.Remove(f.leaf)
	}
}

func sameHost(a, b marker) bool {
	return a.HostPID == b.HostPID && a.HostStartUnix == b.HostStartUnix
}

// Note records that the session sessionID, run by host, started in checkout,
// an absolute working-tree top. It replaces any earlier marker for the
// session, removes every other marker under the same host, and prunes those
// whose host tab no longer shows running.
func Note(home, sessionID, checkout string, host sessionname.Host, tab sessionname.Table) error {
	if !validID(sessionID) {
		return errors.New("occupancy: the session id cannot name a marker")
	}
	if !filepath.IsAbs(checkout) || host.PID <= 1 {
		return errors.New("occupancy: a marker needs an absolute checkout and a host process")
	}
	dir, err := open(home, true)
	if err != nil {
		return fmt.Errorf("occupancy: %w", err)
	}
	defer dir.Close()
	mine := marker{
		SessionID: sessionID, Checkout: canonical(checkout),
		HostPID: host.PID, HostStartUnix: host.StartUnix, HostReadMS: host.Slack.Milliseconds(),
	}
	data, err := json.Marshal(mine)
	if err != nil {
		return err
	}
	if err := fsutil.WriteFileAtomicInRoot(dir, sessionID+".json", data, abcdhome.FileMode); err != nil {
		return fmt.Errorf("occupancy: %w", err)
	}
	others, err := readAll(dir, home)
	if err != nil {
		return nil // the marker is written; pruning waits for the next start
	}
	for _, f := range others {
		if f.m.SessionID == sessionID {
			continue
		}
		if sameHost(f.m, mine) || !tab.Alive(f.m.host()) {
			removeIfUnchanged(dir, home, f)
		}
	}
	return nil
}

// Forget removes the session's marker. A missing marker or folder is not an
// error.
func Forget(home, sessionID string) error {
	if !validID(sessionID) {
		return nil
	}
	dir, err := open(home, false)
	if err != nil || dir == nil {
		return err
	}
	defer dir.Close()
	if err := dir.Remove(sessionID + ".json"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Sharers returns the ids of the sessions whose marker names checkout and
// whose host is still running, one per host process, sorted. The process
// table is read, through readTable, only when two or more markers name
// checkout, so a checkout no one else works in costs one folder read. A
// marker whose host has ended is removed.
func Sharers(home, checkout string, readTable func() (sessionname.Table, error)) ([]string, error) {
	dir, err := open(home, false)
	if err != nil {
		return nil, fmt.Errorf("occupancy: %w", err)
	}
	if dir == nil {
		return nil, nil
	}
	defer dir.Close()
	all, err := readAll(dir, home)
	if err != nil {
		return nil, fmt.Errorf("occupancy: %w", err)
	}
	want := canonical(checkout)
	var named []found
	for _, f := range all {
		if f.m.Checkout == want {
			named = append(named, f)
		}
	}
	if len(named) < 2 {
		return nil, nil
	}
	tab, err := readTable()
	if err != nil {
		return nil, err
	}
	var live []found
	for _, f := range named {
		if !tab.Alive(f.m.host()) {
			removeIfUnchanged(dir, home, f)
			continue
		}
		dup := false
		for _, l := range live {
			if sameHost(l.m, f.m) {
				dup = true
				break
			}
		}
		if !dup {
			live = append(live, f)
		}
	}
	ids := make([]string, 0, len(live))
	for _, f := range live {
		ids = append(ids, f.m.SessionID)
	}
	sort.Strings(ids)
	return ids, nil
}

// canonical is path with its symlinks resolved where they can be, so a
// checkout reached as /tmp/x and as /private/tmp/x is one checkout.
func canonical(path string) string {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real
	}
	return filepath.Clean(path)
}

// MainWorkingTree reports whether root, a working-tree top, is a repository's
// main working tree rather than a linked worktree: git keeps a directory at a
// main tree's .git and a file at a linked one's.
func MainWorkingTree(root string) bool {
	fi, err := os.Stat(filepath.Join(root, ".git"))
	return err == nil && fi.IsDir()
}

// SharedError is Check's refusal: the main working tree a record would be
// written into is one Sessions live sessions work in.
type SharedError struct{ Sessions int }

func (e *SharedError) Error() string {
	return fmt.Sprintf("%d live sessions work in this main checkout, so a record written here lands under all of them. "+
		"Work in a worktree of your own in the machine-scoped store: "+
		"git worktree add %s -b <branch> origin/main, where <root-sha> is what `git rev-list --max-parents=0 HEAD` prints, "+
		"and run this from there. Nothing was written.",
		e.Sessions, abcdhome.Display("worktrees", "<root-sha>", "<name>"))
}

// Check is the one judgement every record writer makes before it writes into
// root (iss-2610240859590333): a *SharedError when root is a main working tree
// two or more live sessions work in, nil when it is a linked worktree or one
// session or none works there, and another error when the count could not be
// made, which the caller reports and writes past: a count it cannot make is
// not evidence that the checkout is shared.
func Check(home, root string, readTable func() (sessionname.Table, error)) error {
	if !MainWorkingTree(root) {
		return nil
	}
	ids, err := Sharers(home, root, readTable)
	if err != nil {
		return err
	}
	if len(ids) < 2 {
		return nil
	}
	return &SharedError{Sessions: len(ids)}
}
