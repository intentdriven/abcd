package history

// Staging is the write-ahead half of transcript capture (iss-2608230817034768).
//
// SessionEnd cannot afford redaction. It costs roughly 0.7s per MB, and the host
// cancels a shutdown hook rather than wait for it, so every transcript past a
// couple of MB was dropped silently and permanently enough to matter: nine of
// this repo's own ended sessions were absent from its store before this existed.
//
// Staging splits the work across the two hooks that can each afford their half.
// SessionEnd copies the raw bytes into the store's staging/ lane at write speed
// and returns; the next SessionStart drains that directory through the same
// fail-closed Capture path, where there is a real time budget.
//
// The store's invariant is untouched: every transcript in records/ is still
// redacted on write, because staging is NOT the store. Staged bytes are raw, so
// this directory is the one place in abcd that holds unredacted transcript text
// on purpose. It is created 0o700 and its files 0o600, it holds each transcript
// only until the next session starts, and nothing reads it but Drain.
//
// A staged file is also the outcome record the store never had. Before this,
// "absent from the store" spanned never-ended, ended-before-the-store-existed,
// ended-and-captured and ended-and-lost, and no caller could tell them apart —
// which is why the loss went unnoticed for a week. A file in staging says
// exactly one thing: this session ended and its capture has not completed yet.

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// stagedSuffix marks a staged raw transcript. The extension is deliberately not
// .md: a staged file is unredacted and must never be mistaken for a record.
const stagedSuffix = ".raw"

// stagingLockFilename is the per-repo staging lock, a sibling of the staged
// files (listStaged filters on stagedSuffix, so the lock is invisible to it).
// Every writer of the staging dir — Stage's list-compare-write and Drain's
// remove-if-unchanged — takes it through fsutil.WithFileLock, the one
// inter-process load-modify-write primitive, so the per-session idempotency
// guarantee holds across concurrent hooks and not just single-threaded
// (GHSA-xq36-hcgf-9wrj). It nests inside nothing: Drain releases it before
// Capture takes the store's repoLock, so the two can never wait on each other.
const stagingLockFilename = ".lock"

// stagingLockTimeout bounds how long a staging writer waits for the lock. It is
// short because the SessionEnd hook must never wedge the session it is ending:
// the critical section is one listing, at most one read and one write, so a
// wait past this is a stuck peer. Contention surfaces as
// fsutil.ErrLockContention, which the hook reports and exits 0 on, exactly as
// it does on any other staging failure.
const stagingLockTimeout = 5 * time.Second

// Staged is one raw transcript awaiting redaction.
type Staged struct {
	SessionID string    `json:"session_id"`
	StagedAt  time.Time `json:"staged_at"`
	Path      string    `json:"path"`
	Bytes     int64     `json:"bytes"`
}

// StageResult reports the outcome of one stage.
type StageResult struct {
	Staged Staged `json:"staged"`
	Wrote  bool   `json:"wrote"` // false when this session is already staged with identical bytes
	// Replaced is true when the session was already staged with DIFFERENT bytes
	// and that copy was replaced by this one; ReplacedBytes is the size of the
	// copy that was replaced. Both are zero on a first stage and on a no-op.
	Replaced      bool  `json:"replaced"`
	ReplacedBytes int64 `json:"replaced_bytes"`
}

// DrainFailure is one staged transcript that could not be captured. The staged
// file is deliberately LEFT in place: a failure here is recoverable by hand, and
// deleting the only copy abcd holds would convert a reported problem into the
// silent permanent loss this whole mechanism exists to end.
type DrainFailure struct {
	SessionID string `json:"session_id"`
	Path      string `json:"path"`
	Err       string `json:"error"`
}

// DrainResult reports one drain pass.
type DrainResult struct {
	Captured  []Record       `json:"captured"`
	Failed    []DrainFailure `json:"failed"`
	Remaining int            `json:"remaining"` // staged entries not attempted, budget exhausted
}

// stagingDirReal resolves the store (creating it when absent, and refusing any
// level that is not a real directory) and creates the staging leaf under it.
// 0o700 throughout, because a staged transcript is unredacted.
func stagingDirReal(repoRoot, rootSHA string) (string, error) {
	store, err := Resolve(repoRoot, rootSHA)
	if err != nil {
		return "", err
	}
	if err := ensureRealDir(store.Staging); err != nil {
		return "", err
	}
	return store.Staging, nil
}

// stagedFilename is <compact-utc>-<session-id>.raw, matching recordFilename's
// shape so staging and transcripts sort and read alike.
func stagedFilename(at time.Time, sessionID string) string {
	return at.UTC().Format("20060102T150405.000000000Z") + "-" + sessionID + stagedSuffix
}

// sessionIDFromStaged recovers the session id from a staged filename. The stamp
// is fixed-width and session ids cannot contain "-"... except they can, so the
// split is on the FIRST "-" after the stamp, which is a fixed offset.
func sessionIDFromStaged(name string) string {
	base := strings.TrimSuffix(name, stagedSuffix)
	// Stamp is "20060102T150405.000000000Z" — 26 chars — then "-".
	const stampLen = 26
	if len(base) <= stampLen+1 || base[stampLen] != '-' {
		return ""
	}
	return base[stampLen+1:]
}

// Stage writes raw transcript bytes into the staging area without redacting
// them. It is the SessionEnd half of capture and must stay cheap: no scanner and
// no read of the store. Cost is one listing of the staging dir, at most one read
// of this session's own staged copy, and one write, so the hook's runtime is
// independent of the store's size — which is the entire point.
//
// It is idempotent per session on CONTENT, and the whole list-compare-write runs
// under the staging lock so the guarantee holds across concurrent hooks
// (GHSA-xq36-hcgf-9wrj): a session already staged with identical bytes is a
// no-op (Wrote=false); a session staged with different bytes has that copy
// replaced at its existing path (Wrote=true, Replaced=true) — last-writer-wins,
// because a re-fired SessionEnd carrying different bytes is the later snapshot
// of the same session, and the fresher end-of-session bytes are the ones worth
// keeping. Either way one session has one staged file, whatever fires.
func Stage(repoRoot, rootSHA, sessionID string, raw []byte) (StageResult, error) {
	if !rootSHARe.MatchString(rootSHA) {
		return StageResult{}, errors.New(rootSHAErrMsg)
	}
	if !sessionIDRe.MatchString(sessionID) {
		return StageResult{}, fmt.Errorf("history: sessionID must be non-empty and match [A-Za-z0-9._-]+")
	}
	if len(raw) == 0 {
		return StageResult{}, errors.New("history: refusing to stage an empty transcript")
	}
	sdir, err := stagingDirReal(repoRoot, rootSHA)
	if err != nil {
		return StageResult{}, err
	}
	var res StageResult
	err = withStagingLock(sdir, func() error {
		var err error
		res, err = stageLocked(sdir, sessionID, raw)
		return err
	})
	if err != nil {
		return StageResult{}, err
	}
	return res, nil
}

// withStagingLock runs fn under the staging lock, naming the lock in the error
// when the primitive itself refuses (contention, or an unsafe lock path); fn's
// own error passes through unchanged.
func withStagingLock(sdir string, fn func() error) error {
	err := fsutil.WithFileLock(filepath.Join(sdir, stagingLockFilename), stagingLockTimeout, fn)
	if errors.Is(err, fsutil.ErrLockContention) || errors.Is(err, fsutil.ErrLockPathUnsafe) {
		return fmt.Errorf("history: staging lock: %w", err)
	}
	return err
}

// stageLocked is Stage's critical section. listStaged is oldest-first, so when
// a session has several copies (a staging dir written before the lock existed)
// the newest is the one compared and replaced; the drain retires the rest.
func stageLocked(sdir, sessionID string, raw []byte) (StageResult, error) {
	existing, err := listStaged(sdir)
	if err != nil {
		return StageResult{}, err
	}
	var prior *Staged
	for i := range existing {
		if existing[i].SessionID == sessionID {
			prior = &existing[i]
		}
	}
	if prior != nil {
		// ReadGuarded is O_NOFOLLOW, so a symlink planted at the staged path
		// is refused here rather than replaced or read through.
		current, err := fsutil.ReadGuarded(prior.Path, maxTranscriptBytes)
		if err != nil {
			return StageResult{}, fmt.Errorf("history: read staged copy of %s: %w", sessionID, err)
		}
		if bytes.Equal(current, raw) {
			return StageResult{Staged: *prior, Wrote: false}, nil
		}
		// 0o600: unredacted. The rename lands at the existing path, so the
		// listing keeps one entry for the session AND its place in the drain
		// queue: listStaged orders on the filename, whose stamp is when the
		// session first ended, so a re-stage never pushes an older session
		// behind newer ones under a budget. Only StagedAt (the file's mtime)
		// moves to the newer bytes.
		if err := fsutil.WriteFileAtomic(prior.Path, raw, 0o600); err != nil {
			return StageResult{}, fmt.Errorf("history: re-stage transcript: %w", err)
		}
		return StageResult{
			Staged:        Staged{SessionID: sessionID, StagedAt: time.Now().UTC(), Path: prior.Path, Bytes: int64(len(raw))},
			Wrote:         true,
			Replaced:      true,
			ReplacedBytes: prior.Bytes,
		}, nil
	}
	at := time.Now().UTC()
	path := filepath.Join(sdir, stagedFilename(at, sessionID))
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return StageResult{}, &StorePathError{Path: path, Msg: "staged path is a symlink; refusing"}
	}
	// 0o600: unredacted.
	if err := fsutil.WriteFileAtomic(path, raw, 0o600); err != nil {
		return StageResult{}, fmt.Errorf("history: stage transcript: %w", err)
	}
	return StageResult{
		Staged: Staged{SessionID: sessionID, StagedAt: at, Path: path, Bytes: int64(len(raw))},
		Wrote:  true,
	}, nil
}

// listStaged reads the staging dir, oldest first so a drain processes sessions
// in the order they ended.
func listStaged(sdir string) ([]Staged, error) {
	entries, err := os.ReadDir(sdir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("history: read staging dir: %w", err)
	}
	var out []Staged
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), stagedSuffix) {
			continue
		}
		id := sessionIDFromStaged(e.Name())
		if id == "" || !sessionIDRe.MatchString(id) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, Staged{
			SessionID: id,
			StagedAt:  info.ModTime().UTC(),
			Path:      filepath.Join(sdir, e.Name()),
			Bytes:     info.Size(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// ListStaged returns the transcripts awaiting redaction for this repo, oldest
// first. An absent staging dir is not an error: it means nothing is pending.
func ListStaged(repoRoot, rootSHA string) ([]Staged, error) {
	store, err := Resolve(repoRoot, rootSHA)
	if err != nil {
		return nil, err
	}
	return listStaged(store.Staging)
}

// Drain captures every staged transcript into the store and removes the ones it
// stored. It is the SessionStart half of capture.
//
// budget bounds how many entries one pass attempts; <= 0 means all of them. The
// bound exists because a backlog drains at redaction speed and SessionStart is
// interactive: a repo with a dozen missed sessions would otherwise stall the
// user's first prompt. Whatever the budget leaves is reported in Remaining
// rather than dropped, so a partial pass is loud and the caller can say so.
//
// A staged file is deleted ONLY when its transcript is in the store — either
// captured now or already present — and only while it still holds the bytes
// that were captured. The staging lock is deliberately NOT held across Capture:
// redaction is the cost staging exists to keep out of SessionEnd, and holding
// the lock through it would stall every concurrent hook for its duration. So a
// Stage that replaces the copy mid-drain wins the race by design: the removal
// re-checks the bytes under the lock, a replaced copy is left for the next
// pass, and the fresher transcript is never lost. Any failure leaves the file
// where it is.
func Drain(repoRoot, rootSHA string, budget int) (DrainResult, error) {
	store, err := Resolve(repoRoot, rootSHA)
	if err != nil {
		return DrainResult{}, err
	}
	sdir := store.Staging
	staged, err := listStaged(sdir)
	if err != nil {
		return DrainResult{}, err
	}
	var res DrainResult
	for i, s := range staged {
		if budget > 0 && i >= budget {
			res.Remaining = len(staged) - i
			break
		}
		raw, err := fsutil.ReadGuarded(s.Path, maxTranscriptBytes)
		if err != nil {
			res.Failed = append(res.Failed, DrainFailure{SessionID: s.SessionID, Path: s.Path,
				Err: fmt.Sprintf("cannot read staged transcript: %v", err)})
			continue
		}
		cr, err := Capture(repoRoot, rootSHA, s.SessionID, raw, "native")
		if err != nil {
			res.Failed = append(res.Failed, DrainFailure{SessionID: s.SessionID, Path: s.Path, Err: err.Error()})
			continue
		}
		// Stored (or already stored): the staged copy has done its job, and it is
		// unredacted, so it goes now rather than lingering — unless a concurrent
		// Stage replaced it while Capture ran, in which case the newer bytes stay
		// staged for the next pass.
		if err := removeStagedIfUnchanged(sdir, s.Path, raw); err != nil {
			res.Failed = append(res.Failed, DrainFailure{SessionID: s.SessionID, Path: s.Path,
				Err: fmt.Sprintf("captured but could not remove staged copy: %v", err)})
			continue
		}
		if cr.Wrote {
			res.Captured = append(res.Captured, cr.Record)
		}
	}
	return res, nil
}

// removeStagedIfUnchanged removes the staged file at path under the staging
// lock, and only if it still holds exactly the captured bytes. A file that is
// already gone is fine (a peer drain retired it); a file whose bytes differ was
// re-staged since the capture read it and is left in place — its transcript is
// not yet in the store, so deleting it would be the silent loss this mechanism
// exists to end.
func removeStagedIfUnchanged(sdir, path string, captured []byte) error {
	want := sha256.Sum256(captured)
	return withStagingLock(sdir, func() error {
		current, err := fsutil.ReadGuarded(path, maxTranscriptBytes)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if sha256.Sum256(current) != want {
			return nil // replaced mid-drain; the next pass captures the newer bytes
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	})
}
