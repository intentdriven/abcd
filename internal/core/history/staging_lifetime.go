package history

// The LIFETIME half of staging (iss-2609090722466403).
//
// staging.go answers "how do raw bytes get onto disk and off it again". This
// file answers the question that was never asked: what happens when they do not
// come off again. Three mechanisms, and one thing none of them does.
//
//   - AGE. StagedTTL is the point past which a staged file has outlived
//     staging's promise. Age escalates: an overdue entry sorts to the front of
//     every drain and is named in every notice. Age does NOT delete, redact
//     down, or otherwise degrade — see below.
//   - QUARANTINE. A transcript the fail-closed scanner refuses is refused
//     identically forever. Retrying it on every drain re-reads and re-scans
//     unredacted text to reach the same answer, and hides the retryable
//     failures behind a permanent one that will never clear. Such an entry is
//     moved to quarantine/ with a written reason, out of the drain queue and
//     into a named terminal state that the listings report.
//   - SURVEY. The store is keyed per repository, so a per-repository listing is
//     blind to precisely the pile that goes wrong: the one in a repository
//     nobody opens. SurveyBacklog reads every key in the store.
//
// What none of them does is delete a transcript. Losing the only copy is worse
// than keeping it — that is the premise staging is built on, and an expiry that
// quietly discarded raw text to make a warning go away would trade a loud
// privacy fact for a silent data loss, which is the trade this whole subsystem
// exists to refuse. Deletion has exactly one door, Discard, and only an
// operator opens it; core never calls it and never asks.

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

	"github.com/intentdriven/abcd/internal/fsutil"
)

// StagedTTL is how long a staged raw transcript may sit before it is reported
// as overdue.
//
// Seven days, and the number is a reporting threshold rather than a deadline
// anything enforces, so it is chosen to be quiet in the normal case and loud in
// the broken one. An ordinary repository drains within a prompt or two of the
// stage — the live drain runs on UserPromptSubmit — so an entry reaching a
// WEEK means no session has opened that repository in a week, which is exactly
// the failure mode this constant exists to name. A shorter TTL would flag
// weekends and holidays; a longer one lets a fortnight of raw transcripts
// accumulate before anything says so, which is what actually happened.
const StagedTTL = 7 * 24 * time.Hour

// stagingNow is the clock the age tests drive. Production never sets it.
var stagingNow = time.Now

// overdue reports whether a stage timestamp has outlived StagedTTL. A zero
// timestamp is NOT overdue: it means the entry's age is unknown (a sidecar that
// recorded none, a filesystem that lost the mtime), and guessing "ancient" from
// missing data would raise an alarm about a file that may have arrived a second
// ago.
func overdue(stagedAt time.Time) bool {
	if stagedAt.IsZero() {
		return false
	}
	return stagingNow().UTC().Sub(stagedAt.UTC()) > StagedTTL
}

// ---------------------------------------------------------------------------
// quarantine
// ---------------------------------------------------------------------------

// quarantineSuffix marks the reason file beside a quarantined transcript. Like
// the staging sidecar it deliberately does NOT end in stagedSuffix, so the
// listing filter never mistakes one for a transcript.
const quarantineSuffix = ".quarantine.json"

// quarantineSchema is the reason file's written schema version, for the same
// reason stageSidecarSchema is written rather than inferred.
const quarantineSchema = 1

// maxQuarantineNoteBytes caps a reason-file read. A reason is a sentence and a
// handful of scalars.
const maxQuarantineNoteBytes = 64 << 10

// quarantineNote is the reason file written beside a quarantined transcript. It
// is what makes quarantine a state rather than a second staging directory: an
// operator reading the directory can tell WHY these bytes stopped moving
// without re-running the scanner over them.
type quarantineNote struct {
	Schema        int       `json:"schema"`
	SessionID     string    `json:"session_id"`
	AgentID       string    `json:"agent_id,omitempty"`
	Reason        string    `json:"reason"`
	StagedAt      time.Time `json:"staged_at"`
	QuarantinedAt time.Time `json:"quarantined_at"`
}

// Quarantined is one transcript that will never pass redaction, parked out of
// the drain queue. Its bytes are still RAW and still on disk — quarantine is a
// change of state, not of exposure — which is why the listings report it as
// loudly as they report a staged entry.
type Quarantined struct {
	SessionID     string    `json:"session_id"`
	AgentID       string    `json:"agent_id,omitempty"`
	Path          string    `json:"path"`
	SidecarPath   string    `json:"sidecar_path,omitempty"`
	ReasonPath    string    `json:"reason_path,omitempty"`
	Reason        string    `json:"reason,omitempty"`
	Bytes         int64     `json:"bytes"`
	StagedAt      time.Time `json:"staged_at,omitempty"`
	QuarantinedAt time.Time `json:"quarantined_at,omitempty"`
	// Err is set when the reason file is present but unreadable. The entry is
	// still listed: the bytes are what matter, and an entry hidden because its
	// metadata is broken is an entry nobody disposes of.
	Err string `json:"error,omitempty"`
}

// quarantineDirPath returns <lane>/quarantine for this repo's store.
func quarantineDirPath(repoRoot, rootSHA string) (string, error) {
	lane, err := laneDir(repoRoot, rootSHA)
	if err != nil {
		return "", err
	}
	return filepath.Join(lane, "quarantine"), nil
}

// quarantineDirReal creates the quarantine leaf under the resolved store, on
// the same terms and for the same reasons as stagingDirReal: 0o700, and never
// created through or as a symlink.
func quarantineDirReal(repoRoot, rootSHA string) (string, error) {
	qdir, err := quarantineDirPath(repoRoot, rootSHA)
	if err != nil {
		return "", err
	}
	if err := fsutil.EnsureRealDir(qdir, storeDirPerm); err != nil {
		return "", storeDirFault(qdir, err)
	}
	return qdir, nil
}

func classifyDrainFailure(sdir, repoRoot, rootSHA string, s Staged, stagedBytes []byte, capErr error) DrainFailure {
	f := DrainFailure{SessionID: s.SessionID, Path: s.Path, Err: capErr.Error()}
	var rerr *RedactionResidualError
	if !errors.As(capErr, &rerr) {
		return f
	}
	f.Permanent = true
	qdir, err := quarantineDirReal(repoRoot, rootSHA)
	if err != nil {
		f.Err += fmt.Sprintf("; could not open quarantine (%v), so the raw copy stays staged", err)
		return f
	}
	qpath, err := quarantineStaged(sdir, qdir, s, stagedBytes, capErr.Error())
	if err != nil {
		f.Err += fmt.Sprintf("; could not quarantine the raw copy (%v), so it stays staged", err)
		return f
	}
	f.Quarantined, f.QuarantinePath = true, qpath
	return f
}

// quarantineStaged moves one staged transcript and its sidecar into quarantine/
// and writes the reason beside them, returning the quarantined transcript's
// path.
//
// The move runs UNDER THE STAGING LOCK and only while the file still holds the
// bytes the drain read, for the identical reason removeStagedIfUnchanged does:
// a Stage that replaced the copy while Capture ran wrote FRESHER bytes, which
// have not been offered to the scanner yet and must not be parked on the
// strength of an older copy's refusal. A replaced copy is left alone and the
// next pass judges it on its own bytes.
//
// The reason file is written BEFORE the transcript is moved in, mirroring the
// sidecar-before-transcript ordering in stageLocked and for the same reason:
// the .raw is what the listing iterates, so it must never be the newest thing
// in the directory. A reason file with no transcript beside it is invisible to
// every reader and is cleaned up on the failure path.
func quarantineStaged(sdir, qdir string, s Staged, read []byte, reason string) (string, error) {
	base := filepath.Base(s.Path)
	qpath := filepath.Join(qdir, base)
	var moved string
	err := withStagingLock(sdir, func() error {
		current, err := fsutil.ReadGuarded(s.Path, maxTranscriptBytes)
		if err != nil {
			return err
		}
		if !bytes.Equal(current, read) {
			return errors.New("the staged copy was replaced while it was being captured; it is left for the next pass")
		}
		if fi, err := os.Lstat(qpath); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			return &StorePathError{Path: qpath, Msg: "quarantine path is a symlink; refusing"}
		}
		note := quarantineNote{
			Schema:    quarantineSchema,
			SessionID: s.SessionID, AgentID: s.AgentID,
			Reason:        reason,
			StagedAt:      s.StagedAt,
			QuarantinedAt: stagingNow().UTC(),
		}
		data, err := json.Marshal(note)
		if err != nil {
			return err
		}
		notePath := quarantineNotePathFor(qpath)
		// 0o600 like everything else describing an unredacted transcript.
		if err := fsutil.WriteFileAtomic(notePath, append(data, '\n'), 0o600); err != nil {
			return err
		}
		// Rename, not copy: both paths are inside ~/.abcd/history/<rootSHA>, so
		// this is one filesystem and the bytes are never duplicated. A copy
		// would briefly put a second unredacted copy on disk, which is the one
		// thing this subsystem must not do casually.
		if err := os.Rename(s.Path, qpath); err != nil {
			_ = os.Remove(notePath)
			return err
		}
		if s.SidecarPath != "" {
			// Best effort: the sidecar is provenance, and a quarantined
			// transcript whose sidecar could not follow it is still correctly
			// quarantined. A sidecar left in staging beside no transcript is
			// invisible to listStaged, so it strands nothing.
			_ = os.Rename(s.SidecarPath, sidecarPathFor(qpath))
		}
		moved = qpath
		return nil
	})
	if err != nil {
		return "", err
	}
	return moved, nil
}

// quarantineNotePathFor returns the .quarantine.json beside a quarantined .raw.
func quarantineNotePathFor(rawPath string) string {
	return strings.TrimSuffix(rawPath, stagedSuffix) + quarantineSuffix
}

// ListQuarantined returns this repo's quarantined transcripts, oldest first. An
// absent quarantine dir is not an error: it means nothing has ever been parked.
func ListQuarantined(repoRoot, rootSHA string) ([]Quarantined, error) {
	qdir, err := quarantineDirPath(repoRoot, rootSHA)
	if err != nil {
		return nil, err
	}
	return listQuarantined(qdir)
}

func listQuarantined(qdir string) ([]Quarantined, error) {
	entries, err := os.ReadDir(qdir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("history: read quarantine dir: %w", err)
	}
	var out []Quarantined
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), stagedSuffix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(qdir, e.Name())
		q := Quarantined{Path: path, Bytes: info.Size(), QuarantinedAt: info.ModTime().UTC()}
		if sp := sidecarPathFor(path); fileReadable(sp) {
			q.SidecarPath = sp
		}
		notePath := quarantineNotePathFor(path)
		data, err := fsutil.ReadGuarded(notePath, maxQuarantineNoteBytes)
		switch {
		case errors.Is(err, os.ErrNotExist):
			// A transcript parked without a readable reason is still parked.
			// Recover what the filename can say and no more.
			q.SessionID = sessionIDFromStaged(e.Name())
		case err != nil:
			q.ReasonPath, q.Err = notePath, err.Error()
		default:
			var note quarantineNote
			if jerr := json.Unmarshal(data, &note); jerr != nil {
				q.ReasonPath, q.Err = notePath, "quarantine note does not parse: "+jerr.Error()
				break
			}
			q.ReasonPath = notePath
			q.SessionID, q.AgentID, q.Reason = note.SessionID, note.AgentID, note.Reason
			q.StagedAt = note.StagedAt.UTC()
			if !note.QuarantinedAt.IsZero() {
				q.QuarantinedAt = note.QuarantinedAt.UTC()
			}
		}
		out = append(out, q)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// fileReadable reports whether path is a regular file this process can stat. It
// is presentational only — it decides whether a listing mentions a sidecar —
// so a race here costs a blank column, never a wrong action.
func fileReadable(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode().IsRegular()
}

// ---------------------------------------------------------------------------
// discard — the one door that deletes
// ---------------------------------------------------------------------------

// DiscardResult reports what one Discard removed.
type DiscardResult struct {
	Path    string   `json:"path"`
	Bytes   int64    `json:"bytes"`
	Removed []string `json:"removed"`
}

// Discard permanently deletes ONE staged or quarantined raw transcript, named
// by its bare filename, together with the sidecar and reason file describing
// it.
//
// This is the only code path in abcd that destroys a transcript nothing has
// stored, and it exists because quarantine without an exit is a room with no
// door: a transcript the scanner will never pass would otherwise sit unredacted
// forever with no legitimate way to be rid of it. It is never called by a hook,
// a drain, or an expiry. The judgement that these particular bytes may go is a
// human one, and the confirmation for it belongs to the front door, not here —
// core neither prompts nor prints.
//
// name is a bare filename by design: a caller passing a path could delete
// outside the store, and this function's whole risk profile is that it deletes.
// It must end in the staged suffix and contain no separator.
func Discard(repoRoot, rootSHA, name string) (DiscardResult, error) {
	if name == "" || name != filepath.Base(name) || strings.ContainsRune(name, filepath.Separator) ||
		name == "." || name == ".." || strings.HasPrefix(name, ".") {
		return DiscardResult{}, fmt.Errorf("history: %q is not a staged transcript filename", name)
	}
	if !strings.HasSuffix(name, stagedSuffix) {
		return DiscardResult{}, fmt.Errorf("history: %q is not a staged transcript (it does not end in %s)", name, stagedSuffix)
	}
	store, err := Resolve(repoRoot, rootSHA)
	if err != nil {
		return DiscardResult{}, err
	}
	sdir := store.Staging
	qdir, err := quarantineDirPath(repoRoot, rootSHA)
	if err != nil {
		return DiscardResult{}, err
	}
	for _, dir := range []string{sdir, qdir} {
		path := filepath.Join(dir, name)
		fi, err := os.Lstat(path)
		if err != nil || !fi.Mode().IsRegular() {
			continue
		}
		res := DiscardResult{Path: path, Bytes: fi.Size()}
		remove := func() error {
			if err := os.Remove(path); err != nil {
				return err
			}
			res.Removed = append(res.Removed, path)
			for _, side := range []string{sidecarPathFor(path), quarantineNotePathFor(path)} {
				if err := os.Remove(side); err == nil {
					res.Removed = append(res.Removed, side)
				} else if !errors.Is(err, os.ErrNotExist) {
					return err
				}
			}
			return nil
		}
		// The staging lock covers the staging directory's mutators; taking it
		// for a quarantined file too is harmless (it is the same per-repo lock)
		// and keeps a discard from racing a drain that is mid-move. When the
		// staging directory does not exist there is no lock file to take and no
		// drain to race — a drain requires it — so the removal runs unlocked
		// rather than creating a directory in order to delete something else.
		derr := remove
		if fsutil.IsRealDir(sdir) {
			derr = func() error { return withStagingLock(sdir, remove) }
		}
		if err := derr(); err != nil {
			return DiscardResult{}, fmt.Errorf("history: discard %s: %w", name, err)
		}
		return res, nil
	}
	return DiscardResult{}, fmt.Errorf("history: %q is neither staged nor quarantined for this repo", name)
}

// ---------------------------------------------------------------------------
// survey — the cross-repository view
// ---------------------------------------------------------------------------

// RepoBacklog is one repository's holding of unredacted transcript text.
//
// It is deliberately COUNTS AND SIZES and no content: the survey's whole point
// is that it is read from a repository other than the one it describes, and a
// notice that quoted another repository's session ids into this repository's
// session would be leaking exactly the thing the store redacts.
type RepoBacklog struct {
	RootSHA string `json:"root_sha"`
	// Name is the repository's directory name from its meta.json, when the
	// store has one. It is what makes the notice actionable — "13 MB in a
	// repository whose root commit begins a1b2c3" is a riddle. Empty when the
	// repo was never registered.
	Name             string    `json:"name,omitempty"`
	Staged           int       `json:"staged"`
	StagedBytes      int64     `json:"staged_bytes"`
	Overdue          int       `json:"overdue"`
	OldestStagedAt   time.Time `json:"oldest_staged_at,omitempty"`
	Quarantined      int       `json:"quarantined"`
	QuarantinedBytes int64     `json:"quarantined_bytes"`
}

// Total is the unredacted bytes this repository is holding, staged and
// quarantined alike. Both are raw; the distinction is whether anything will
// ever try to redact them again.
func (b RepoBacklog) Total() int64 { return b.StagedBytes + b.QuarantinedBytes }

// SurveyBacklog reports every repository in the store that is holding
// unredacted transcript text, newest-oldest by nothing — the order is by root
// SHA, which is stable and meaningless, so a caller that wants a ranking sorts
// for itself.
//
// This exists because the store is keyed per repository and every reader of it
// was too: `abcd history staged` answers for the repository the operator is
// standing in, which is by construction a repository someone is using, and
// therefore one whose staged files are being drained. The pile that grows
// unbounded is the one in the repository nobody has opened for a fortnight, and
// no per-repo verb can ever see it (iss-2609090722466403).
//
// An unreadable repository directory is SKIPPED rather than fatal: one broken
// key must not blind the survey to the other forty. A store that does not exist
// yet returns nothing and no error.
func SurveyBacklog() ([]RepoBacklog, error) {
	root, err := userStoreBase()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("history: read store root: %w", err)
	}
	var out []RepoBacklog
	for _, e := range entries {
		// Only a directory whose name IS a root-commit SHA is a repository key.
		// index.json, meta files and anything else a future version drops here
		// are not, and a survey that tried to read them would report noise.
		if !e.IsDir() || !rootSHARe.MatchString(e.Name()) {
			continue
		}
		// The corpus moved out of ahoy's namespace but the per-repo meta.json
		// did not (iss-95), so the lane and the NAME of the repository that
		// owns it are read from two different roots.
		meta, _ := ahoyRepoMetaPath(e.Name())
		b := RepoBacklog{RootSHA: e.Name(), Name: storedRepoName(meta)}
		staged, err := listStaged(filepath.Join(root, e.Name(), stagingDirName))
		if err == nil {
			for _, s := range staged {
				b.Staged++
				b.StagedBytes += s.Bytes
				if s.Overdue {
					b.Overdue++
				}
				if !s.StagedAt.IsZero() && (b.OldestStagedAt.IsZero() || s.StagedAt.Before(b.OldestStagedAt)) {
					b.OldestStagedAt = s.StagedAt
				}
			}
		}
		quar, err := listQuarantined(filepath.Join(root, e.Name(), "quarantine"))
		if err == nil {
			for _, q := range quar {
				b.Quarantined++
				b.QuarantinedBytes += q.Bytes
			}
		}
		if b.Staged == 0 && b.Quarantined == 0 {
			continue
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RootSHA < out[j].RootSHA })
	return out, nil
}

// maxRepoMetaBytes caps the per-repo meta.json read. It holds four scalars.
const maxRepoMetaBytes = 64 << 10

// storedRepoName reads the `name` field of a per-repo meta.json, or "" when the
// file is absent, unreadable, or holds no such string. Best effort by design:
// the name is a label on a notice, and a survey that failed because one
// repository's metadata was corrupt would report nothing about the other
// repositories' raw transcripts.
func storedRepoName(path string) string {
	data, err := fsutil.ReadGuarded(path, maxRepoMetaBytes)
	if err != nil {
		return ""
	}
	var meta map[string]any
	if err := json.Unmarshal(data, &meta); err != nil {
		return ""
	}
	name, _ := meta["name"].(string)
	return name
}
