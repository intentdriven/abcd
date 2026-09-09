package history

// Locating a transcript's owning repository, and recording what the harness did
// not deliver.
//
// The store is keyed on a repository's root-commit SHA, which every writer so
// far resolved from its own working directory. A SubagentStop hook cannot rely
// on that. A sub-agent given its own worktree records that worktree as its cwd,
// and the harness REMOVES the worktree when the agent stops — so by the time
// the hook runs, the directory the payload names may not exist, `ahoy.Detect`
// on it fails, and the hook would exit 0 having staged nothing. That loses
// exactly the isolated implementation-lane agents, which are the ones whose
// transcripts are worth the most.
//
// The fallback is the spawning session. A session that started in a repository
// leaves a note here saying so, and any later hook holding that session's id can
// find the store without a working directory at all. The note is a zero-byte
// file named for the session: the lookup is one stat per known repository, and
// it reads no transcript.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// sessionNoteTTL bounds how long a session→store note is kept. A note is a
// zero-byte file, so the cost of keeping one is an inode; the cost of keeping
// them forever is an unbounded directory. Well past any session's life, and
// well short of forever.
const sessionNoteTTL = 30 * 24 * time.Hour

// subagentGapFilename is the marker recording that this harness delivered a
// SubagentStop payload with no agent_transcript_path. It sits beside the
// staging directory rather than inside it: staging holds transcripts, and this
// is the record of a transcript that never arrived.
const subagentGapFilename = "subagent-payload-gap.json"

// safeIDSegment reports whether an identifier may be used as a single path
// segment. sessionIDRe admits "." and "..", which as a whole segment would walk
// out of the directory being written into; every other character it admits is
// inert. The regex is not widened, because it is also the record's field
// validator, where the traversal question does not arise.
func safeIDSegment(id string) bool {
	return sessionIDRe.MatchString(id) && id != "." && id != ".."
}

// sessionsDirReal returns ~/.abcd/history/<rootSHA>/sessions, creating the leaf
// if absent. Like staging, the parents are NOT created: a note written under a
// store nothing bootstrapped would be a note nothing ever reads.
func sessionsDirReal(rootSHA string) (string, error) {
	root, err := historyRoot()
	if err != nil {
		return "", err
	}
	repoDir := filepath.Join(root, rootSHA)
	for _, d := range []string{root, repoDir} {
		if !fsutil.IsRealDir(d) {
			return "", &StorePathError{Path: d, Msg: "not a real directory (absent or symlink); run `abcd ahoy install` to bootstrap the store"}
		}
	}
	dir := filepath.Join(repoDir, "sessions")
	if err := os.Mkdir(dir, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return "", &StorePathError{Path: dir, Msg: "cannot create sessions dir: " + err.Error()}
	}
	if !fsutil.IsRealDir(dir) {
		return "", &StorePathError{Path: dir, Msg: "sessions path is not a real directory (symlink?); refusing"}
	}
	return dir, nil
}

// NoteSessionRepo records that sessionID is running against the store keyed on
// rootSHA, so a later hook holding only that session id can find the store.
//
// It is written by the hooks that CAN resolve a repository — SessionStart from
// the session's own cwd, and SubagentStop when its cwd still resolves — and read
// by the one that sometimes cannot. Writing it is best-effort by construction:
// a failure here degrades a fallback, never a capture.
func NoteSessionRepo(rootSHA, sessionID string) error {
	if !rootSHARe.MatchString(rootSHA) {
		return errors.New(rootSHAErrMsg)
	}
	if !safeIDSegment(sessionID) {
		return fmt.Errorf("history: sessionID must be non-empty, match [A-Za-z0-9._-]+ and not be a directory reference")
	}
	dir, err := sessionsDirReal(rootSHA)
	if err != nil {
		return err
	}
	pruneSessionNotes(dir)
	// Zero bytes: the filename IS the content, and an empty file cannot carry
	// anything worth redacting.
	return fsutil.WriteFileAtomic(filepath.Join(dir, sessionID), nil, 0o600)
}

// pruneSessionNotes drops notes older than sessionNoteTTL. Best-effort: it runs
// on the write path, where a failure to tidy must never fail the write.
func pruneSessionNotes(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-sessionNoteTTL)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		_ = os.Remove(filepath.Join(dir, e.Name()))
	}
}

// SessionRepo returns the root-commit SHA of the store that noted sessionID.
//
// It reads no transcript: one directory listing of ~/.abcd/history and one stat
// per repository under it. A session id that two stores claim is refused rather
// than guessed — a transcript filed against the wrong repository is redacted by
// the wrong repository's scanner configuration, which is a privacy fault, not a
// misfiling. An unknown session returns an error naming that.
func SessionRepo(sessionID string) (string, error) {
	if !safeIDSegment(sessionID) {
		return "", fmt.Errorf("history: sessionID must be non-empty, match [A-Za-z0-9._-]+ and not be a directory reference")
	}
	root, err := historyRoot()
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("history: no store has seen session %s", sessionID)
		}
		return "", err
	}
	var found []string
	for _, e := range entries {
		if !e.IsDir() || !rootSHARe.MatchString(e.Name()) {
			continue
		}
		note := filepath.Join(root, e.Name(), "sessions", sessionID)
		fi, err := os.Lstat(note)
		if err != nil || !fi.Mode().IsRegular() {
			continue
		}
		found = append(found, e.Name())
	}
	switch len(found) {
	case 0:
		return "", fmt.Errorf("history: no store has seen session %s", sessionID)
	case 1:
		return found[0], nil
	default:
		return "", fmt.Errorf("history: session %s is claimed by %d stores; refusing to guess which repository owns its transcripts", sessionID, len(found))
	}
}

// SubagentGapNote is the recorded fact that this harness fired SubagentStop
// without an agent_transcript_path. Absence of sub-agent records then has an
// explanation on disk instead of looking like an absence of sub-agents.
type SubagentGapNote struct {
	Schema    int       `json:"schema"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	Count     int       `json:"count"`
	Event     string    `json:"event"`
}

// gapNotePath returns the marker's path beside the staging directory.
func gapNotePath(rootSHA string) (string, error) {
	root, err := historyRoot()
	if err != nil {
		return "", err
	}
	repoDir := filepath.Join(root, rootSHA)
	if !fsutil.IsRealDir(repoDir) {
		return "", &StorePathError{Path: repoDir, Msg: "not a real directory (absent or symlink); run `abcd ahoy install` to bootstrap the store"}
	}
	return filepath.Join(repoDir, subagentGapFilename), nil
}

// NoteSubagentGap records a SubagentStop payload that carried no
// agent_transcript_path. The first sighting sets FirstSeen and every sighting
// moves LastSeen and the count, so a reader can tell a harness that has never
// delivered the field from one that stopped, or started.
func NoteSubagentGap(rootSHA, event string) error {
	if !rootSHARe.MatchString(rootSHA) {
		return errors.New(rootSHAErrMsg)
	}
	path, err := gapNotePath(rootSHA)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	note := SubagentGapNote{Schema: 1, FirstSeen: now, Event: event}
	if prior, ok, err := SubagentGap(rootSHA); err == nil && ok {
		note.FirstSeen = prior.FirstSeen
		note.Count = prior.Count
	}
	note.LastSeen = now
	note.Count++
	data, err := json.Marshal(note)
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, append(data, '\n'), 0o600)
}

// SubagentGap reads the marker. A missing marker is not an error: it means this
// harness has always delivered the field, or has never fired the event.
func SubagentGap(rootSHA string) (SubagentGapNote, bool, error) {
	if !rootSHARe.MatchString(rootSHA) {
		return SubagentGapNote{}, false, errors.New(rootSHAErrMsg)
	}
	root, err := historyRoot()
	if err != nil {
		return SubagentGapNote{}, false, err
	}
	data, err := fsutil.ReadGuarded(filepath.Join(root, rootSHA, subagentGapFilename), maxStageSidecarBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SubagentGapNote{}, false, nil
		}
		return SubagentGapNote{}, false, err
	}
	var note SubagentGapNote
	if err := json.Unmarshal(data, &note); err != nil {
		return SubagentGapNote{}, false, fmt.Errorf("history: sub-agent payload marker does not parse: %w", err)
	}
	return note, true, nil
}
