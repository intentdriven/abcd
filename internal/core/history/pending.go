package history

// Pending is the session-end half of staging with no repository in hand
// (iss-2610241519088123).
//
// Staging (staging.go) took redaction off the exit path, but it still needed
// the repository's root-commit SHA to know which lane to write into, and
// finding it is git: a detection and a root-commit lookup, retried when git
// failed once. The host gives a SessionEnd hook 1.5 seconds in all, and a
// timeout a plugin sets does not raise it. On a loaded machine starting the
// binary takes about a third of a second and each git call a sixth, so the
// hook was cancelled before it staged anything, and the transcript was lost
// with nothing to say so.
//
// So the session-end hook resolves nothing. It writes the raw transcript into
// one machine-level directory keyed by session id, with the payload's cwd and
// transcript path beside it, and returns. The next SessionStart, which has
// the time, places each pending transcript: it finds the store from the
// session note SessionStart wrote (NoteSessionRepo), or else from the staged
// cwd, and stages it there, where the ordinary drain redacts and stores it
// through the same fail-closed Capture. A pending transcript it cannot place
// stays where it is and is named, never dropped.
//
// Like staging, this directory holds unredacted text on purpose: 0o700, its
// files 0o600, and nothing reads it but the placement and its listing.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// pendingDirName is the machine-level directory, under the user-level store
// root, that holds transcripts no repository has been found for yet. It is
// not a root-commit SHA, so every reader that walks the store's lanes passes
// over it.
const pendingDirName = "pending"

// pendingSidecarSuffix marks the metadata beside a pending transcript.
const pendingSidecarSuffix = ".pending.json"

// pendingSidecarSchema is the pending sidecar's schema version.
const pendingSidecarSchema = 1

// PendingMeta is what the session-end hook knows about a transcript without
// asking git: whose it is, where the session ran, and where the bytes came
// from.
type PendingMeta struct {
	SessionID  string
	Cwd        string
	SourcePath string
}

// Pending is one transcript awaiting placement in a store.
type Pending struct {
	SessionID  string    `json:"session_id"`
	Cwd        string    `json:"cwd,omitempty"`
	SourcePath string    `json:"source_path,omitempty"`
	StagedAt   time.Time `json:"staged_at"`
	Path       string    `json:"path"`
	Bytes      int64     `json:"bytes"`
	// Err is set when the entry's sidecar is missing or does not parse: it
	// carries no trustworthy cwd, so it is never placed.
	Err string `json:"error,omitempty"`
}

// pendingSidecar is the on-disk metadata beside a pending transcript.
type pendingSidecar struct {
	Schema     int       `json:"schema"`
	SessionID  string    `json:"session_id"`
	Cwd        string    `json:"cwd,omitempty"`
	SourcePath string    `json:"source_path,omitempty"`
	StagedAt   time.Time `json:"staged_at"`
}

// pendingDirReal returns the pending directory, creating abcd's home, the
// user-level store root and the leaf as real 0o700 directories. It reads no
// repository: the session-end hook calls it with none in hand.
func pendingDirReal() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", fmt.Errorf("history: cannot resolve the caller's home directory: %v", err)
	}
	if err := fsutil.EnsureRealDir(abcdhome.Path(home), abcdhome.DirMode); err != nil {
		return "", storeDirFault(abcdhome.Path(home), err)
	}
	base := filepath.Join(home, filepath.FromSlash(userStoreRelPath))
	dir := filepath.Join(base, pendingDirName)
	for _, d := range []string{base, dir} {
		if err := fsutil.EnsureRealDir(d, storeDirPerm); err != nil {
			return "", storeDirFault(d, err)
		}
	}
	return dir, nil
}

// pendingPaths returns the transcript and sidecar paths for one session.
func pendingPaths(dir, sessionID string) (raw, side string) {
	raw = filepath.Join(dir, sessionID+stagedSuffix)
	return raw, strings.TrimSuffix(raw, stagedSuffix) + pendingSidecarSuffix
}

// StagePending writes a session's raw transcript into the pending directory.
// It runs no git and reads no store: one stat, at most one read of this
// session's own pending copy, and two writes.
//
// It is idempotent on content per session, under the same per-key lock the
// staging lane uses: identical bytes are a no-op (Wrote=false), different
// bytes replace the pending copy (Replaced=true), because a re-fired
// SessionEnd carries the later snapshot of the same session.
func StagePending(meta PendingMeta, raw []byte) (StageResult, error) {
	if !safeIDSegment(meta.SessionID) {
		return StageResult{}, errors.New("history: sessionID must be non-empty, match [A-Za-z0-9._-]+ and not be a directory reference")
	}
	if strings.ContainsAny(meta.Cwd, "\r\n") || strings.ContainsAny(meta.SourcePath, "\r\n") {
		return StageResult{}, errors.New("history: a pending transcript's cwd and source path must not contain a line break")
	}
	if len(raw) == 0 {
		return StageResult{}, errors.New("history: refusing to stage an empty transcript")
	}
	dir, err := pendingDirReal()
	if err != nil {
		return StageResult{}, err
	}
	var res StageResult
	err = withStagingLock(dir, meta.SessionID, func() error {
		path, side := pendingPaths(dir, meta.SessionID)
		at := time.Now().UTC()
		entry := Staged{SessionID: meta.SessionID, StagedAt: at, Path: path, Bytes: int64(len(raw)), SourcePath: meta.SourcePath}
		current, rerr := fsutil.ReadGuarded(path, maxTranscriptBytes)
		switch {
		case rerr == nil && bytes.Equal(current, raw):
			res = StageResult{Staged: entry, Wrote: false}
			return nil
		case rerr == nil:
			res = StageResult{Staged: entry, Wrote: true, Replaced: true, ReplacedBytes: int64(len(current))}
		case errors.Is(rerr, os.ErrNotExist):
			res = StageResult{Staged: entry, Wrote: true}
		default:
			// O_NOFOLLOW: a symlink planted at the path is refused, not
			// replaced or read through.
			return fmt.Errorf("history: read pending copy of %s: %w", meta.SessionID, rerr)
		}
		// The sidecar first, as in the staging lane: the listing iterates the
		// .raw files, so a transcript must never be newer than its metadata.
		data, err := json.Marshal(pendingSidecar{Schema: pendingSidecarSchema, SessionID: meta.SessionID,
			Cwd: meta.Cwd, SourcePath: meta.SourcePath, StagedAt: at})
		if err != nil {
			return err
		}
		if err := fsutil.WriteFileAtomic(side, append(data, '\n'), 0o600); err != nil {
			return fmt.Errorf("history: pending sidecar: %w", err)
		}
		if err := fsutil.WriteFileAtomic(path, raw, 0o600); err != nil {
			if !res.Replaced {
				_ = os.Remove(side)
			}
			return fmt.Errorf("history: pending transcript: %w", err)
		}
		return nil
	})
	if err != nil {
		return StageResult{}, err
	}
	return res, nil
}

// ListPending returns the transcripts awaiting placement, oldest first. A
// pending directory that does not exist holds nothing.
func ListPending() ([]Pending, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil, fmt.Errorf("history: cannot resolve the caller's home directory: %v", err)
	}
	return listPending(filepath.Join(home, filepath.FromSlash(userStoreRelPath), pendingDirName))
}

func listPending(dir string) ([]Pending, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []Pending
	for _, e := range entries {
		name := e.Name()
		if !e.Type().IsRegular() || !strings.HasSuffix(name, stagedSuffix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		p := Pending{SessionID: strings.TrimSuffix(name, stagedSuffix), Path: filepath.Join(dir, name),
			Bytes: info.Size(), StagedAt: info.ModTime().UTC()}
		_, side := pendingPaths(dir, p.SessionID)
		data, err := fsutil.ReadGuarded(side, maxStageSidecarBytes)
		var sc pendingSidecar
		switch {
		case err != nil:
			p.Err = fmt.Sprintf("its metadata cannot be read (%v)", err)
		case json.Unmarshal(data, &sc) != nil:
			p.Err = "its metadata does not parse"
		case sc.SessionID != p.SessionID:
			p.Err = "its metadata names another session"
		default:
			p.Cwd, p.SourcePath, p.StagedAt = sc.Cwd, sc.SourcePath, sc.StagedAt
		}
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StagedAt.Before(out[j].StagedAt) })
	return out, nil
}

// RepoResolver finds the repository a working directory belongs to: its root
// and its root-commit SHA. The caller supplies it, because finding a
// repository is git and the front door owns that.
type RepoResolver func(cwd string) (repoRoot, rootSHA string, err error)

// Placed is one pending transcript filed in a store's staging lane.
type Placed struct {
	SessionID string `json:"session_id"`
	RootSHA   string `json:"root_sha"`
	// Via says how the store was found: "session-note" or "cwd".
	Via string `json:"via"`
}

// Unplaced is one pending transcript no store could be found for. It stays
// pending, unredacted, until a later start places it.
type Unplaced struct {
	SessionID string `json:"session_id"`
	Cwd       string `json:"cwd,omitempty"`
	Path      string `json:"path"`
	Reason    string `json:"reason"`
}

// PlaceResult reports one placement pass.
type PlaceResult struct {
	Placed   []Placed   `json:"placed"`
	Unplaced []Unplaced `json:"unplaced"`
}

// PlacePending files every pending transcript in its store's staging lane,
// where the drain redacts and stores it. The store is the one the session's
// note names (NoteSessionRepo, written when the session started), or else the
// one resolve finds from the cwd the session ended in. The note comes first
// because it was written while the session's directory certainly existed; a
// worktree removed since leaves the cwd resolving to nothing.
//
// A pending copy is removed only once its bytes are in the lane, and only if
// they are still the bytes that were staged: a SessionEnd re-firing during the
// placement wins. A transcript no store is found for stays, and is reported.
func PlacePending(resolve RepoResolver) (PlaceResult, error) {
	res := PlaceResult{Placed: []Placed{}, Unplaced: []Unplaced{}}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return res, fmt.Errorf("history: cannot resolve the caller's home directory: %v", err)
	}
	dir := filepath.Join(home, filepath.FromSlash(userStoreRelPath), pendingDirName)
	entries, err := listPending(dir)
	if err != nil || len(entries) == 0 {
		return res, err
	}
	for _, p := range entries {
		unplaced := func(reason string) {
			res.Unplaced = append(res.Unplaced, Unplaced{SessionID: p.SessionID, Cwd: p.Cwd, Path: p.Path, Reason: reason})
		}
		if p.Err != "" {
			unplaced(p.Err)
			continue
		}
		repoRoot, rootSHA, via, why := placeTarget(p, resolve)
		if rootSHA == "" {
			unplaced(why)
			continue
		}
		err := withStagingLock(dir, p.SessionID, func() error {
			raw, err := fsutil.ReadGuarded(p.Path, maxTranscriptBytes)
			if err != nil {
				return err
			}
			if _, err := Stage(repoRoot, rootSHA, StageMeta{
				Lineage:    CaptureMeta{SessionID: p.SessionID, Kind: RouteNative, LineageSource: "hook"},
				SourcePath: p.SourcePath,
			}, raw); err != nil {
				return err
			}
			// Still the bytes just staged: nothing re-fired in between, since
			// this lock is the one StagePending takes.
			_, side := pendingPaths(dir, p.SessionID)
			if err := os.Remove(p.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			_ = os.Remove(side)
			retireStagingLock(dir, p.SessionID)
			return nil
		})
		if err != nil {
			unplaced(fmt.Sprintf("it could not be staged in its store (%v)", err))
			continue
		}
		res.Placed = append(res.Placed, Placed{SessionID: p.SessionID, RootSHA: rootSHA, Via: via})
	}
	return res, nil
}

// placeTarget finds the store for one pending transcript. With the session's
// note, the cwd is still asked for its root so a store kept in the checkout
// is honoured, but only when it names the same repository; otherwise the
// note's store is the user-level one, which needs no root.
func placeTarget(p Pending, resolve RepoResolver) (repoRoot, rootSHA, via, why string) {
	noted, noteErr := SessionRepo(p.SessionID)
	var cwdRoot, cwdSHA string
	var cwdErr error
	switch {
	case p.Cwd == "":
		cwdErr = errors.New("the session recorded no working directory")
	case resolve == nil:
		cwdErr = errors.New("nothing was given to resolve its working directory")
	default:
		cwdRoot, cwdSHA, cwdErr = resolve(p.Cwd)
		if cwdErr == nil && !rootSHARe.MatchString(cwdSHA) {
			cwdErr = errors.New("its working directory is in no repository with a root commit")
		}
	}
	if noteErr == nil {
		if cwdErr == nil && cwdSHA == noted {
			return cwdRoot, noted, "session-note", ""
		}
		return "", noted, "session-note", ""
	}
	if cwdErr == nil {
		return cwdRoot, cwdSHA, "cwd", ""
	}
	return "", "", "", fmt.Sprintf("no store noted the session, and %v", cwdErr)
}
