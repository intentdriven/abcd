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
	"github.com/intentdriven/abcd/internal/localtier"
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

// sharers returns the ids of the sessions whose marker names checkout and
// whose host is still running, one per host process, sorted. The process
// table is read, through readTable, only when two or more markers name
// checkout, so a checkout no one else works in costs one folder read and a
// stat of each marker spelt otherwise. A marker whose host has ended is
// removed.
func sharers(home, checkout string, readTable func() (sessionname.Table, error)) ([]string, error) {
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
	wantInfo, _ := os.Stat(want)
	var named []found
	for _, f := range all {
		if f.m.Checkout == want || sameDir(f.m.Checkout, wantInfo) {
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

// sameDir reports whether the directory path names is the one want
// describes, so a checkout spelt in another case on a case-insensitive
// filesystem, or in another Unicode normalization, is the same checkout.
func sameDir(path string, want os.FileInfo) bool {
	if want == nil {
		return false
	}
	fi, err := os.Stat(path)
	return err == nil && os.SameFile(fi, want)
}

// mainWorkingTree reports whether root, a working-tree top, is a repository's
// main working tree rather than a linked worktree: git keeps a directory at a
// main tree's .git and a file at a linked one's.
func mainWorkingTree(root string) bool {
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
	if !mainWorkingTree(root) {
		return nil
	}
	ids, err := sharers(home, root, readTable)
	if err != nil {
		return err
	}
	if len(ids) < 2 {
		return nil
	}
	return &SharedError{Sessions: len(ids)}
}

// treeOf returns the working-tree top a file at path would be written into
// on its cleaned reading: the nearest ancestor of path holding a .git entry,
// a directory at a main working tree's top and a file at a linked one's. The
// path is cleaned and its symlinks resolved first, so a write through a link
// is judged by the tree it lands in; a ".." after a link can land elsewhere
// on the raw reading, which CheckPath judges too. It runs no git subprocess,
// so the hook on the host's file tools stays cheap on every write
// (iss-2610280051498465). ok is false for a relative path and for one in no
// checkout.
func treeOf(path string) (root string, ok bool) {
	if !filepath.IsAbs(path) {
		return "", false
	}
	return treeAt(landing(filepath.Clean(path)))
}

// treeAt is treeOf's walk up from a path already resolved.
func treeAt(dir string) (string, bool) {
	for {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// maxHops bounds the symlinks landing follows, as the kernel's own limit
// does; a chain longer than that is one the write fails on anyway.
const maxHops = 40

// landing is where a write to the absolute path p lands: each component that
// exists is taken as the kernel takes it, a symlink followed (one whose
// target is not yet made included, since writing through it makes that
// target) and a ".." stepping up from the directory reached so far, not from
// the name spelt before it. The part not yet made is joined as written.
func landing(p string) string {
	hops := 0
	return follow(p, &hops)
}

func follow(p string, hops *int) string {
	vol := filepath.VolumeName(p)
	sep := string(filepath.Separator)
	cur := vol + sep
	for _, c := range strings.Split(p[len(vol):], sep) {
		switch c {
		case "", ".":
			continue
		case "..":
			cur = filepath.Dir(cur)
			continue
		}
		next := filepath.Join(cur, c)
		fi, err := os.Lstat(next)
		if err != nil || fi.Mode()&os.ModeSymlink == 0 || *hops >= maxHops {
			cur = next
			continue
		}
		target, err := os.Readlink(next)
		if err != nil {
			cur = next
			continue
		}
		*hops++
		if !filepath.IsAbs(target) {
			target = cur + sep + target
		}
		cur = follow(target, hops)
	}
	return cur
}

// CheckPath is Check on the working tree a file at path would be written
// into, for a writer that names a file rather than a checkout: the hook on
// the host's own file tools. A path in no checkout is nil, and so is one in
// that tree's local tier: it is ignored and kept for that checkout alone, so
// a file written there lands in no peer's index, and the SCRATCH rule sends
// a linked worktree's handovers and helper scripts to the main checkout's.
//
// A path is judged twice when its spelling and the kernel can disagree: as
// cleaned before it is resolved (how a writer that normalizes the path first
// reads it) and as written (how the kernel resolves a ".." after a symlink).
// Either reading landing in a shared main checkout outside its tier refuses
// the write, so no spelling opens the exemption.
func CheckPath(home, path string, readTable func() (sessionname.Table, error)) error {
	if !filepath.IsAbs(path) {
		return nil
	}
	readings := []string{landing(filepath.Clean(path))}
	if raw := landing(path); raw != readings[0] {
		readings = append(readings, raw)
	}
	tier := filepath.FromSlash(localtier.RelDir)
	var failed error
	for _, at := range readings {
		root, ok := treeAt(at)
		if !ok {
			continue
		}
		if rel, err := filepath.Rel(root, at); err == nil &&
			(rel == tier || strings.HasPrefix(rel, tier+string(filepath.Separator))) {
			continue
		}
		var shared *SharedError
		switch err := Check(home, root, readTable); {
		case errors.As(err, &shared):
			return err
		case err != nil && failed == nil:
			failed = err
		}
	}
	return failed
}
