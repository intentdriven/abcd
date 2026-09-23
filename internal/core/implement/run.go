// Package implement is the run machinery an autonomous implementation run
// calls: the run's shared state in the machine-scoped store, the sessions that
// join it, the claim that makes a record the unit of exclusion between them, the
// bounds that keep a second session from becoming a hazard, and the run log the
// comparison of division modes is derived from (itd-2609221656373558,
// spc-2609221657588816).
//
// Everything lives under one directory per repository:
//
//	~/.abcd/runs/<root-sha>/<YYYY-MM-DD>.jsonl   the run log, one event per line
//	~/.abcd/runs/<root-sha>/claims/<record>.json one claim per record
//	~/.abcd/runs/<root-sha>/sessions/<id>.json   one record per joined session
//
// The key is the repository's root-commit SHA, the same key and the same full
// form the transcript, history and voyage stores use (internal/core/history's
// location.go says why: a checkout moves, is renamed and is cloned twice, while
// its root commit changes under none of that). Two sessions in two worktrees of
// one repository therefore share one directory and no repository file.
//
// The package is the seed the implement loop builds on (itd-2609201916151817:
// `implement step` and `implement receipt`) and the pacing verb reads
// (itd-2609201925079472): a Run is the handle a later state file hangs off, and
// the log is the measurement both inherit. Core never writes to stdout; the CLI
// front door formats what these functions return.
package implement

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// runsRelPath is the store's root relative to the caller's home.
const runsRelPath = ".abcd/runs"

// storeDirPerm is the mode a run directory is created with: the run state is the
// caller's own and nobody else's business.
const storeDirPerm = 0o700

// fileMode is the mode every file in the run state is created with.
const fileMode = 0o600

// Sub-directory names inside one run.
const (
	claimsDirName   = "claims"
	sessionsDirName = "sessions"
	lockFileName    = ".lock"
)

// lockTimeout bounds how long a mutation waits for another session's mutation
// to finish before it reports contention. Every mutation is a handful of small
// file operations, so a lock held longer than this is a stuck writer, and the
// caller is told to back off rather than made to wait on it.
const lockTimeout = 3 * time.Second

// ErrRefused is the class of every refusal: an input the verb does not
// recognise, a session that has not joined, or a bound the caller's role does
// not permit. Nothing is written for the refused act (the refusal itself may be
// logged). The CLI maps it to exit 2.
var ErrRefused = errors.New("refused")

// ErrContention is the class of every "someone else holds it" outcome: a record
// another session has claimed, or the run state locked by another session's
// mutation. It is not a fault: the caller backs off and tries other work. The
// CLI maps it to exit 3.
var ErrContention = errors.New("contention")

// refusal builds an ErrRefused-classed error with a message.
func refusal(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrRefused}, a...)...)
}

// Run is one repository's run state. The zero value is not usable; construct it
// with Open (which creates the directories a mutation needs) or Peek (which
// creates nothing, for the read-only renders).
type Run struct {
	// Dir is the run directory, ~/.abcd/runs/<root-sha>.
	Dir string
	// RootSHA is the repository's root-commit SHA the directory is keyed on.
	RootSHA string
	// RepoRoot is the checkout the run was opened from. The reading corpus the
	// second session's bounds check against is read from its preset file.
	RepoRoot string
	// Now is the clock. Tests set it; nil means time.Now.
	Now func() time.Time
	// exists records whether Dir was present (Peek) or made (Open).
	exists bool
}

// now returns the run's clock reading in UTC, truncated to the second so every
// timestamp the run writes round-trips through RFC 3339 exactly.
func (r *Run) now() time.Time {
	if r.Now != nil {
		return r.Now().UTC().Truncate(time.Second)
	}
	return time.Now().UTC().Truncate(time.Second)
}

// runDir resolves the run directory for rootSHA under the caller's home, without
// touching the filesystem.
func runDir(rootSHA string) (home, dir string, err error) {
	if !gitutil.IsFullSHA(rootSHA) {
		return "", "", refusal("the repository has no root commit to key the run on (a repository with no commits has no run state)")
	}
	home, err = os.UserHomeDir()
	if err != nil || home == "" {
		return "", "", fmt.Errorf("cannot resolve the caller's home directory: %v", err)
	}
	return home, filepath.Join(home, filepath.FromSlash(runsRelPath), rootSHA), nil
}

// Open returns the run for rootSHA, creating its directories when absent. Every
// level from ~/.abcd down is created one at a time and proved a real directory,
// never through a symlink (fsutil.EnsureRealDirAll), so a planted redirect is
// refused before anything is written beneath it.
func Open(rootSHA string) (*Run, error) {
	home, dir, err := runDir(rootSHA)
	if err != nil {
		return nil, err
	}
	for _, sub := range []string{claimsDirName, sessionsDirName} {
		rel := runsRelPath + "/" + rootSHA + "/" + sub
		if err := fsutil.EnsureRealDirAll(home, rel, storeDirPerm); err != nil {
			return nil, fmt.Errorf("cannot create the run state: %w", err)
		}
	}
	return &Run{Dir: dir, RootSHA: rootSHA, exists: true}, nil
}

// OpenJoined returns the run for rootSHA on behalf of a session that must
// already have joined it: every writer but join. A run directory that does not
// exist holds no session, so the caller is refused before anything is created —
// no directory, no lock, no log line. An existing run is opened as Open opens
// it, and the verb itself refuses a session it does not hold.
func OpenJoined(rootSHA, session string) (*Run, error) {
	if err := validName("session", session); err != nil {
		return nil, err
	}
	peek, err := Peek(rootSHA)
	if err != nil {
		return nil, err
	}
	if !peek.exists {
		return nil, refusal("session %s has not joined this run (run `abcd implement join` first)", session)
	}
	return Open(rootSHA)
}

// Peek returns the run for rootSHA without creating anything. A run directory
// that does not exist reads as an empty run: no sessions, no claims, no log.
func Peek(rootSHA string) (*Run, error) {
	_, dir, err := runDir(rootSHA)
	if err != nil {
		return nil, err
	}
	r := &Run{Dir: dir, RootSHA: rootSHA}
	if fsutil.IsRealDir(dir) {
		r.exists = true
	} else if ok, _ := fsutil.ExistsNoFollow(dir); ok {
		return nil, fmt.Errorf("the run state path is not a real directory (a symlink or a file occupies it); refusing")
	}
	return r, nil
}

// root opens the run directory as an os.Root, so every file operation below it
// is contained against a symlinked component.
func (r *Run) root() (*os.Root, error) {
	return os.OpenRoot(r.Dir)
}

// withLock runs fn holding the run's advisory lock. Every mutation of the claim
// and session state takes it, so a read-decide-write sequence (a lapse and a
// re-claim, a cap count and a claim) is never interleaved with another
// session's. The exclusive create underneath remains the exclusion itself: the
// lock orders the sequences, the create decides the race.
func (r *Run) withLock(fn func() error) error {
	err := fsutil.WithFileLock(filepath.Join(r.Dir, lockFileName), lockTimeout, fn)
	if errors.Is(err, fsutil.ErrLockContention) {
		return fmt.Errorf("%w: the run state is locked by another session's change; back off and retry", ErrContention)
	}
	return err
}

// nameRe is the shape of a session id and a lane name: a path segment that can
// be neither empty, hidden, nor a traversal.
var nameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// validName refuses anything that is not exactly a name, naming which one.
func validName(what, v string) error {
	if !nameRe.MatchString(v) {
		return refusal("%s %q is not a name (letters, digits, '.', '_' and '-', starting with a letter or digit, at most 64)", what, v)
	}
	return nil
}
