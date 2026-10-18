package autonomy

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"regexp"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// SessionsFile is the away store's note of each session's start copy: the
// worktree a session started in, keyed by session id, so a hook run after the
// session changed directory into a worktree it made still reads the start
// copy's key, local tier and session state ("Which copy").
const SessionsFile = "sessions.json"

// MaxSessionNotes bounds the notes kept; the oldest go first.
const MaxSessionNotes = 256

// SessionNoteTTL is how long a note is kept: a session older than this is
// long over.
const SessionNoteTTL = 30 * 24 * time.Hour

// sessionIDRe is the shape of a session id abcd keys anything on.
var sessionIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

// ErrSessionID is a session id outside [A-Za-z0-9_-]{1,128}.
var ErrSessionID = errors.New("autonomy: the session id is not [A-Za-z0-9_-]{1,128}")

// ValidSessionID reports whether id is one abcd keys a file on.
func ValidSessionID(id string) bool { return sessionIDRe.MatchString(id) }

// SessionNote is one session's start copy.
type SessionNote struct {
	ID   string    `json:"id"`
	Copy string    `json:"copy"`
	At   time.Time `json:"at"`
}

type sessionNotes struct {
	SchemaVersion int           `json:"schema_version"`
	Sessions      []SessionNote `json:"sessions"`
}

func (s Store) readNotes() (sessionNotes, error) {
	raw, err := s.ReadFile(SessionsFile)
	if errors.Is(err, fs.ErrNotExist) && !errors.Is(err, ErrUnsafe) {
		return sessionNotes{SchemaVersion: SchemaVersion}, nil
	}
	if err != nil {
		return sessionNotes{}, err
	}
	var n sessionNotes
	if err := json.Unmarshal(raw, &n); err != nil || n.SchemaVersion != SchemaVersion {
		return sessionNotes{}, errors.Join(ErrUnsafe, errors.New(s.Display(SessionsFile)+" does not parse"))
	}
	return n, nil
}

// NoteSessionCopy notes copy as the session's start copy, under the store's
// lock, unless the session already has a note: the first note is the start,
// so a compaction or resume run from a worktree the session made keeps it.
// It reports whether it wrote. Notes past SessionNoteTTL, and the oldest past
// MaxSessionNotes, are dropped.
func (s Store) NoteSessionCopy(sessionID, copy string, now time.Time) (bool, error) {
	if !ValidSessionID(sessionID) {
		return false, ErrSessionID
	}
	wrote := false
	err := s.WithLock(func() error {
		n, err := s.readNotes()
		if err != nil {
			return err
		}
		kept := n.Sessions[:0]
		for _, e := range n.Sessions {
			if e.ID == sessionID {
				return nil
			}
			if now.Sub(e.At) < SessionNoteTTL {
				kept = append(kept, e)
			}
		}
		kept = append(kept, SessionNote{ID: sessionID, Copy: copy, At: now.Truncate(time.Second)})
		if len(kept) > MaxSessionNotes {
			kept = kept[len(kept)-MaxSessionNotes:]
		}
		n.SchemaVersion, n.Sessions = SchemaVersion, kept
		raw, err := json.MarshalIndent(n, "", "  ")
		if err != nil {
			return err
		}
		if err := s.WriteFile(SessionsFile, append(raw, '\n')); err != nil {
			return err
		}
		wrote = true
		return nil
	})
	return wrote, err
}

// SessionCopy is the session's noted start copy, when it has one that is
// still a directory.
func (s Store) SessionCopy(sessionID string) (string, bool) {
	if !ValidSessionID(sessionID) {
		return "", false
	}
	n, err := s.readNotes()
	if err != nil {
		return "", false
	}
	for _, e := range n.Sessions {
		if e.ID == sessionID {
			if fi, err := os.Stat(e.Copy); err == nil && fi.IsDir() {
				return e.Copy, true
			}
			return "", false
		}
	}
	return "", false
}

// ResolveCopy is the working copy a session acts from: the start copy its
// session-start noted, else the worktree of cwd; and the away store its
// repository keys. The store is keyed on the root commit, so every worktree
// of the repository opens the same one and finds the note.
func ResolveCopy(cwd, sessionID string, o Options) (string, Store, error) {
	root, st, err := Resolve(cwd, o)
	if err != nil {
		return root, st, err
	}
	if c, ok := st.SessionCopy(sessionID); ok {
		if top, err := gitutil.Toplevel(c); err == nil && gitutil.RootCommit(top) == st.RootSHA() {
			return top, st, nil
		}
	}
	return root, st, nil
}
