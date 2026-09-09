package history

// Staging is the write-ahead half of transcript capture (iss-2608230817034768).
//
// SessionEnd cannot afford redaction. It costs roughly 0.7s per MB, and the host
// cancels a shutdown hook rather than wait for it, so every transcript past a
// couple of MB was dropped silently and permanently enough to matter: nine of
// this repo's own ended sessions were absent from its store before this existed.
//
// Staging splits the work across the two hooks that can each afford their half.
// SessionEnd copies the raw bytes into ~/.abcd/history/<rootSHA>/staging/ at
// write speed and returns; the next SessionStart drains that directory through
// the same fail-closed Capture path, where there is a real time budget.
//
// The store's invariant is untouched: every transcript in transcripts/ is still
// redacted on write, because staging is NOT the store. Staged bytes are raw, so
// this directory is the one place in abcd that holds unredacted transcript text
// on purpose. It is created 0o700 and its files 0o600, and nothing reads it but
// Drain, Discard and the read-only listings.
//
// HOW LONG A STAGED FILE LIVES. This comment used to say "only until the next
// session starts". That was false, and the falsehood was load-bearing
// (iss-2609090722466403): the drain runs from a hook of the repository the
// staged file belongs to, so a repository nobody opens again keeps its raw
// transcripts for as long as the disk lasts. Four files, thirteen megabytes,
// the oldest a fortnight old, were on the author's own machine when this was
// found. What is true now:
//
//   - Every prompt of a LIVE session drains a little (a one-entry budget on
//     UserPromptSubmit), so a session that spawns sub-agents redacts its own
//     branches as it goes rather than leaving them for a start that may never
//     come.
//   - Every session start reports the backlog of EVERY repository in the store,
//     not just the one the operator is standing in, so a quiet repository's pile
//     is visible from wherever work is actually happening (SurveyBacklog).
//   - Past StagedTTL an entry is OVERDUE: it sorts to the front of every drain
//     and is named in the notices. Age alone never deletes and never degrades —
//     the only copy of a transcript is not discarded to make a warning go away.
//   - A transcript the fail-closed scanner will never pass is QUARANTINED rather
//     than retried forever, and only an explicit operator Discard removes bytes.
//
// A staged file is also the outcome record the store never had. Before this,
// "absent from the store" spanned never-ended, ended-before-the-store-existed,
// ended-and-captured and ended-and-lost, and no caller could tell them apart —
// which is why the loss went unnoticed for a week. A file in staging says
// exactly one thing: this session ended and its capture has not completed yet.

import (
	"bytes"
	"crypto/sha256"
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

// stagedSuffix marks a staged raw transcript. The extension is deliberately not
// .md: a staged file is unredacted and must never be mistaken for a record.
const stagedSuffix = ".raw"

// stageSidecarSuffix marks the metadata file beside a staged transcript. It
// does NOT end in stagedSuffix, so listStaged's filter never mistakes one for a
// transcript.
const stageSidecarSuffix = ".stage.json"

// stageSidecarSchema is the sidecar's schema version. It is a written field
// rather than an inference from the key set, because a reader that guesses the
// version from which keys are present cannot tell an older writer from a
// corrupted file.
const stageSidecarSchema = 1

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
//
// Everything past Bytes comes from the .stage.json sidecar beside the .raw
// file. A staged file written by an older binary has no sidecar, so those
// fields are zero on it and it drains as a main-thread transcript — which is
// exactly what it is.
type Staged struct {
	SessionID string    `json:"session_id"`
	StagedAt  time.Time `json:"staged_at"`
	Path      string    `json:"path"`
	Bytes     int64     `json:"bytes"`

	// Lineage, verbatim from the sidecar. All empty on a main-thread stage.
	AgentID          string `json:"agent_id,omitempty"`
	ParentAgentID    string `json:"parent_agent_id,omitempty"`
	AgentType        string `json:"agent_type,omitempty"`
	SpawnDepth       int    `json:"spawn_depth,omitempty"`
	SpawnToolUseID   string `json:"spawn_tool_use_id,omitempty"`
	LineageSource    string `json:"lineage_source,omitempty"`
	SpawnAttribution string `json:"spawn_attribution,omitempty"`

	// SourcePath is the file the staged bytes were read from. The drain re-reads
	// it, which is one of the two mitigations for a transcript staged before the
	// harness finished flushing it. Empty when the stage did not record one.
	SourcePath string `json:"source_path,omitempty"`
	// SidecarPath is the .stage.json beside Path; empty on a legacy entry.
	SidecarPath string `json:"sidecar_path,omitempty"`

	// Overdue is true once the entry has been staged longer than StagedTTL. It
	// is a REPORTING fact and a queue-ordering one, never a licence to delete:
	// the entry sorts to the front of the next drain and is named in the
	// session notices, and nothing else about it changes.
	Overdue bool `json:"overdue,omitempty"`

	// Err is set when the entry cannot be trusted — a sidecar that is present
	// but unreadable or unparseable. Such an entry is NOT drained: falling back
	// to the filename would read a sub-agent's key as a session id and file the
	// transcript under a session that does not exist. Drain reports it as a
	// failure and leaves the file alone.
	Err string `json:"error,omitempty"`
}

// StageMeta is what a stage records about a transcript besides its bytes: the
// lineage Capture will eventually be handed, and the path the bytes came from.
//
// Lineage is the same CaptureMeta the drain passes to Capture, so the staging
// sidecar and the record carry one shape and not two that can drift.
type StageMeta struct {
	Lineage CaptureMeta
	// SourcePath is the transcript file the bytes were read from, recorded so
	// the drain can re-read it. It is never used to locate the staged bytes.
	SourcePath string
}

// DrainBudget bounds one drain pass. A zero value is unbounded, which is what
// the explicit `abcd history drain` verb asks for.
//
// The bound is a PAIR because the two costs are different. Redaction time
// tracks BYTES (roughly 0.7s per MB), so the byte bound is the one that
// protects an interactive session start; the count bound keeps a pathological
// many-tiny-transcripts case bounded too, which bytes alone would not. An
// intent that admits one staged file per sub-agent completion makes the second
// case ordinary rather than pathological.
type DrainBudget struct {
	MaxEntries int   // entries attempted; <= 0 means unbounded
	MaxBytes   int64 // staged bytes attempted; <= 0 means unbounded
}

// exhausted reports whether attempting an entry of size next, with consumed
// bytes and attempted entries already spent, would break the budget.
//
// attempted == 0 always returns false: a pass must make progress. Otherwise a
// single staged transcript larger than MaxBytes would be skipped by every pass
// forever, and the raw unredacted file it names would never leave the disk —
// the exact failure the budget exists to bound, inverted.
func (b DrainBudget) exhausted(attempted int, consumed, next int64) bool {
	if attempted == 0 {
		return false
	}
	if b.MaxEntries > 0 && attempted >= b.MaxEntries {
		return true
	}
	return b.MaxBytes > 0 && consumed+next > b.MaxBytes
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
// bytes are never deleted by a failure: a failure here is recoverable by hand,
// and deleting the only copy abcd holds would convert a reported problem into
// the silent permanent loss this whole mechanism exists to end.
//
// The two kinds of failure are NOT the same fact, and reporting them alike was
// its own defect (iss-2609090722466403). A store path that is momentarily
// unwritable, a lock a peer holds, a source file that vanished — those are
// RETRYABLE: the next drain may well succeed, so the entry stays staged and
// stays queued. A transcript the fail-closed scanner refuses to pass is
// DETERMINISTIC: the same bytes will be refused by every future drain, forever,
// so leaving it queued buys nothing and costs a re-read and a re-scan of
// unredacted text on every pass while its raw copy sits there anyway. Permanent
// marks the second kind; such an entry is moved out of the drain queue into
// quarantine/ and reported there until an operator decides what becomes of it.
type DrainFailure struct {
	SessionID string `json:"session_id"`
	Path      string `json:"path"`
	Err       string `json:"error"`
	// Permanent is true when the refusal is a property of the transcript's own
	// bytes rather than of the environment — today, exactly a
	// *RedactionResidualError. A permanent failure is never retried.
	Permanent bool `json:"permanent,omitempty"`
	// Quarantined records that the staged bytes were moved into quarantine/ as
	// a consequence, and where they went. False on every retryable failure, and
	// false on a permanent one whose move itself failed (which is reported in
	// Err rather than swallowed).
	Quarantined    bool   `json:"quarantined,omitempty"`
	QuarantinePath string `json:"quarantine_path,omitempty"`
}

// DrainResult reports one drain pass.
type DrainResult struct {
	Captured  []Record       `json:"captured"`
	Failed    []DrainFailure `json:"failed"`
	Remaining int            `json:"remaining"` // staged entries not attempted, budget exhausted
	// Extended counts the entries whose recorded source file had grown into a
	// strict superset of the staged bytes by the time the drain read it — a
	// transcript that was staged before the harness finished writing it, caught
	// and completed. It is reported rather than left silent because the rate is
	// the measurement that decides whether the flush race is real: a zero here
	// across a corpus is evidence the event fires after the flush, and a
	// non-zero one is the count of transcripts that would otherwise have been
	// stored short.
	Extended int `json:"extended"`
	// Overdue counts the entries this pass SAW that were already past
	// StagedTTL, whether or not the pass reached them. It is the number a
	// notice must be able to say out loud: an overdue entry is unredacted text
	// that has outlived the guarantee staging makes about it.
	Overdue int `json:"overdue"`
}

// stagingDirPath returns ~/.abcd/history/<rootSHA>/staging.
func stagingDirPath(rootSHA string) (string, error) {
	root, err := historyRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, rootSHA, "staging"), nil
}

// stagingDirReal verifies the owned path down to staging/ and creates the leaf
// if absent. The parents are NOT created: the store proper is bootstrapped by
// `abcd ahoy install`, and staging into a repo that was never installed would
// accumulate raw transcripts nothing would ever drain.
func stagingDirReal(rootSHA string) (string, error) {
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
	sdir := filepath.Join(repoDir, "staging")
	// 0o700: staged transcripts are unredacted.
	if err := os.Mkdir(sdir, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return "", &StorePathError{Path: sdir, Msg: "cannot create staging dir: " + err.Error()}
	}
	if !fsutil.IsRealDir(sdir) {
		return "", &StorePathError{Path: sdir, Msg: "staging path is not a real directory (symlink?); refusing"}
	}
	return sdir, nil
}

// stagedFilename is <compact-utc>-<key>.raw, matching recordFilename's shape so
// staging and transcripts sort and read alike. The key is the agent id when
// there is one and the session id otherwise — it is a NAME, not an identifier:
// nothing decodes it back into lineage except the legacy path below, which
// applies only where no sidecar exists and where the key can only be a session.
func stagedFilename(at time.Time, key string) string {
	return at.UTC().Format("20060102T150405.000000000Z") + "-" + key + stagedSuffix
}

// stagedKey is the filename key for a stage: the agent id when there is one,
// the session id otherwise.
func stagedKey(m CaptureMeta) string {
	if m.AgentID != "" {
		return m.AgentID
	}
	return m.SessionID
}

// sidecarPathFor returns the .stage.json beside a staged .raw file.
func sidecarPathFor(rawPath string) string {
	return strings.TrimSuffix(rawPath, stagedSuffix) + stageSidecarSuffix
}

// maxStageSidecarBytes caps a sidecar read. A sidecar is a dozen short scalars;
// anything larger is not one, and the cap keeps a planted file from being read
// into memory whole.
const maxStageSidecarBytes = 64 << 10

// stageSidecar is the on-disk metadata beside a staged transcript. It exists so
// that lineage is never encoded in a filename: overloading an identifier with
// structure is the defect adr-2609090636172016 removed from the record, and
// rebuilding it one directory earlier would be the same mistake with a shorter
// blast radius.
type stageSidecar struct {
	Schema           int       `json:"schema"`
	SessionID        string    `json:"session_id"`
	AgentID          string    `json:"agent_id,omitempty"`
	ParentAgentID    string    `json:"parent_agent_id,omitempty"`
	AgentType        string    `json:"agent_type,omitempty"`
	SpawnDepth       int       `json:"spawn_depth,omitempty"`
	SpawnToolUseID   string    `json:"spawn_tool_use_id,omitempty"`
	LineageSource    string    `json:"lineage_source,omitempty"`
	SpawnAttribution string    `json:"spawn_attribution,omitempty"`
	SourcePath       string    `json:"source_path,omitempty"`
	StagedAt         time.Time `json:"staged_at"`
}

// writeStageSidecar writes the sidecar for one stage, mode 0o600 like the
// transcript it describes: a source path and an agent type are still facts
// about the caller's machine.
func writeStageSidecar(path string, meta StageMeta, at time.Time) error {
	m := meta.Lineage
	data, err := json.Marshal(stageSidecar{
		Schema:           stageSidecarSchema,
		SessionID:        m.SessionID,
		AgentID:          m.AgentID,
		ParentAgentID:    m.ParentAgentID,
		AgentType:        m.AgentType,
		SpawnDepth:       m.SpawnDepth,
		SpawnToolUseID:   m.SpawnToolUseID,
		LineageSource:    m.LineageSource,
		SpawnAttribution: m.SpawnAttribution,
		SourcePath:       meta.SourcePath,
		StagedAt:         at,
	})
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, append(data, '\n'), 0o600)
}

// sessionIDFromStaged recovers the session id from a staged filename. The stamp
// is fixed-width and session ids cannot contain "-"... except they can, so the
// split is on the FIRST "-" after the stamp, which is a fixed offset.
//
// This is the LEGACY path only: it runs for a .raw with no sidecar, which can
// only have been written by a binary that staged main threads and nothing else.
// A sub-agent's key is its agent id, so reading a key as a session id where a
// sidecar was expected would file the transcript under a session that does not
// exist — which is why a sidecar that is present but unreadable is an error
// rather than a fallback to here.
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
func Stage(rootSHA string, meta StageMeta, raw []byte) (StageResult, error) {
	if !rootSHARe.MatchString(rootSHA) {
		return StageResult{}, errors.New(rootSHAErrMsg)
	}
	if meta.Lineage.Kind == "" {
		meta.Lineage.Kind = "native"
	}
	// The same validation Capture applies, applied at the door the bytes come
	// in through: a lineage the store would refuse must not be written into a
	// sidecar the drain will only discover it cannot use.
	if err := meta.Lineage.validate(); err != nil {
		return StageResult{}, err
	}
	if strings.ContainsAny(meta.SourcePath, "\r\n") {
		return StageResult{}, errors.New("history: sourcePath must not contain a line break")
	}
	if len(raw) == 0 {
		return StageResult{}, errors.New("history: refusing to stage an empty transcript")
	}
	sdir, err := stagingDirReal(rootSHA)
	if err != nil {
		return StageResult{}, err
	}
	var res StageResult
	err = withStagingLock(sdir, func() error {
		var err error
		res, err = stageLocked(sdir, meta, raw)
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
// a (session, agent) has several copies (a staging dir written before the lock
// existed) the newest is the one compared and replaced; the drain retires the
// rest.
//
// The idempotency key is the PAIR, not the session. Keyed on the session alone,
// a session's second sub-agent completion would replace the first one's staged
// transcript at its path, and the first would be gone with nothing to say it
// ever existed — silent loss, which is the failure staging exists to end.
func stageLocked(sdir string, meta StageMeta, raw []byte) (StageResult, error) {
	m := meta.Lineage
	existing, err := listStaged(sdir)
	if err != nil {
		return StageResult{}, err
	}
	var prior *Staged
	for i := range existing {
		// An entry whose sidecar could not be read carries no trustworthy key,
		// so it can neither match nor be replaced: it is left for the drain to
		// report.
		if existing[i].Err == "" && existing[i].SessionID == m.SessionID && existing[i].AgentID == m.AgentID {
			prior = &existing[i]
		}
	}
	at := time.Now().UTC()
	if prior != nil {
		// ReadGuarded is O_NOFOLLOW, so a symlink planted at the staged path
		// is refused here rather than replaced or read through.
		current, err := fsutil.ReadGuarded(prior.Path, maxTranscriptBytes)
		if err != nil {
			return StageResult{}, fmt.Errorf("history: read staged copy of %s: %w", stagedKey(m), err)
		}
		if bytes.Equal(current, raw) {
			return StageResult{Staged: *prior, Wrote: false}, nil
		}
		// The sidecar goes first, for the same reason it does on a first stage:
		// the .raw is what listStaged iterates, so it must never be newer than
		// the metadata describing it.
		side := sidecarPathFor(prior.Path)
		if err := writeStageSidecar(side, meta, at); err != nil {
			return StageResult{}, fmt.Errorf("history: re-stage sidecar: %w", err)
		}
		// 0o600: unredacted. The rename lands at the existing path, so the
		// listing keeps one entry for the pair AND its place in the drain
		// queue: listStaged orders on the filename, whose stamp is when the
		// transcript first arrived, so a re-stage never pushes an older entry
		// behind newer ones under a budget. Only StagedAt moves to the newer
		// bytes.
		if err := fsutil.WriteFileAtomic(prior.Path, raw, 0o600); err != nil {
			return StageResult{}, fmt.Errorf("history: re-stage transcript: %w", err)
		}
		return StageResult{
			Staged:        stagedEntry(meta, prior.Path, side, at, int64(len(raw))),
			Wrote:         true,
			Replaced:      true,
			ReplacedBytes: prior.Bytes,
		}, nil
	}
	path := filepath.Join(sdir, stagedFilename(at, stagedKey(m)))
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return StageResult{}, &StorePathError{Path: path, Msg: "staged path is a symlink; refusing"}
	}
	// Sidecar BEFORE the transcript, and the transcript's failure takes the
	// sidecar back down with it. listStaged iterates .raw files and reads the
	// sidecar beside each, so a .raw that briefly exists without one would be
	// read down the legacy path — and for a sub-agent that means its agent id
	// parsed as a session id. A sidecar with no .raw beside it is invisible to
	// every reader, which is the harmless half of the pair to leave lying
	// around; it is still cleaned up rather than left to accumulate.
	side := sidecarPathFor(path)
	if err := writeStageSidecar(side, meta, at); err != nil {
		return StageResult{}, fmt.Errorf("history: stage sidecar: %w", err)
	}
	// 0o600: unredacted.
	if err := fsutil.WriteFileAtomic(path, raw, 0o600); err != nil {
		_ = os.Remove(side)
		return StageResult{}, fmt.Errorf("history: stage transcript: %w", err)
	}
	return StageResult{
		Staged: stagedEntry(meta, path, side, at, int64(len(raw))),
		Wrote:  true,
	}, nil
}

// stagedEntry is the Staged view of a stage that just happened, assembled from
// what was written rather than re-read from disk.
func stagedEntry(meta StageMeta, path, sidecar string, at time.Time, size int64) Staged {
	m := meta.Lineage
	return Staged{
		SessionID:        m.SessionID,
		StagedAt:         at,
		Path:             path,
		Bytes:            size,
		AgentID:          m.AgentID,
		ParentAgentID:    m.ParentAgentID,
		AgentType:        m.AgentType,
		SpawnDepth:       m.SpawnDepth,
		SpawnToolUseID:   m.SpawnToolUseID,
		LineageSource:    m.LineageSource,
		SpawnAttribution: m.SpawnAttribution,
		SourcePath:       meta.SourcePath,
		SidecarPath:      sidecar,
	}
}

// captureMeta is the CaptureMeta the drain hands Capture for this entry. The
// lineage round-trips through the sidecar unchanged; the drain adds nothing and
// infers nothing.
func (s Staged) captureMeta() CaptureMeta {
	return CaptureMeta{
		SessionID:        s.SessionID,
		Kind:             "native",
		AgentID:          s.AgentID,
		ParentAgentID:    s.ParentAgentID,
		AgentType:        s.AgentType,
		SpawnDepth:       s.SpawnDepth,
		SpawnToolUseID:   s.SpawnToolUseID,
		LineageSource:    s.LineageSource,
		SpawnAttribution: s.SpawnAttribution,
	}
}

// listStaged reads the staging dir, oldest first so a drain processes entries
// in the order they arrived.
//
// Each .raw is described by the .stage.json beside it. Three cases:
//
//   - sidecar present and readable: its fields ARE the entry's lineage.
//   - sidecar absent: a staged file from an older binary. Its session id is
//     parsed from the filename exactly as before and it drains as a main-thread
//     transcript, because that is the only kind that binary staged. An upgrade
//     therefore never strands a backlog.
//   - sidecar present but unreadable or unparseable: the entry is marked with
//     Err and drained by nobody. The legacy fallback is NOT available here: a
//     sub-agent's filename key is its AGENT id, so parsing it as a session id
//     would file the transcript under a session that does not exist, and a
//     confident wrong attribution is worse than a reported one.
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
		info, err := e.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(sdir, e.Name())
		s := Staged{StagedAt: info.ModTime().UTC(), Path: path, Bytes: info.Size()}
		side, err := readStageSidecar(sidecarPathFor(path))
		switch {
		case errors.Is(err, os.ErrNotExist):
			id := sessionIDFromStaged(e.Name())
			if id == "" || !sessionIDRe.MatchString(id) {
				continue
			}
			s.SessionID = id
		case err != nil:
			s.Err = err.Error()
		default:
			s.SessionID = side.SessionID
			s.AgentID = side.AgentID
			s.ParentAgentID = side.ParentAgentID
			s.AgentType = side.AgentType
			s.SpawnDepth = side.SpawnDepth
			s.SpawnToolUseID = side.SpawnToolUseID
			s.LineageSource = side.LineageSource
			s.SpawnAttribution = side.SpawnAttribution
			s.SourcePath = side.SourcePath
			s.SidecarPath = sidecarPathFor(path)
			if !side.StagedAt.IsZero() {
				s.StagedAt = side.StagedAt.UTC()
			}
		}
		s.Overdue = overdue(s.StagedAt)
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// readStageSidecar reads and validates one sidecar. An absent file returns a
// wrapped os.ErrNotExist so the caller can take the legacy path; every other
// fault returns an error the caller reports rather than works around.
func readStageSidecar(path string) (stageSidecar, error) {
	data, err := fsutil.ReadGuarded(path, maxStageSidecarBytes)
	if err != nil {
		return stageSidecar{}, err
	}
	var side stageSidecar
	if err := json.Unmarshal(data, &side); err != nil {
		return stageSidecar{}, fmt.Errorf("staging sidecar does not parse: %v", err)
	}
	if side.Schema != stageSidecarSchema {
		return stageSidecar{}, fmt.Errorf("staging sidecar schema %d is not %d", side.Schema, stageSidecarSchema)
	}
	// The sidecar is a file on disk that anything running as the caller can
	// write, and its fields go into a record. Validate them at the same
	// boundary Capture would, so a bad one is a reported entry rather than a
	// drain failure discovered a step later.
	m := CaptureMeta{
		SessionID: side.SessionID, Kind: "native",
		AgentID: side.AgentID, ParentAgentID: side.ParentAgentID,
		AgentType: side.AgentType, SpawnDepth: side.SpawnDepth,
		SpawnToolUseID: side.SpawnToolUseID, LineageSource: side.LineageSource,
		SpawnAttribution: side.SpawnAttribution,
	}
	if err := m.validate(); err != nil {
		return stageSidecar{}, fmt.Errorf("staging sidecar is not a valid lineage: %v", err)
	}
	return side, nil
}

// ListStaged returns the transcripts awaiting redaction for this repo, oldest
// first. An absent staging dir is not an error: it means nothing is pending.
func ListStaged(rootSHA string) ([]Staged, error) {
	if !rootSHARe.MatchString(rootSHA) {
		return nil, errors.New(rootSHAErrMsg)
	}
	sdir, err := stagingDirPath(rootSHA)
	if err != nil {
		return nil, err
	}
	return listStaged(sdir)
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
func Drain(repoRoot, rootSHA string, budget DrainBudget) (DrainResult, error) {
	if !rootSHARe.MatchString(rootSHA) {
		return DrainResult{}, errors.New(rootSHAErrMsg)
	}
	sdir, err := stagingDirPath(rootSHA)
	if err != nil {
		return DrainResult{}, err
	}
	staged, err := listStaged(sdir)
	if err != nil {
		return DrainResult{}, err
	}
	ordered := drainOrder(staged)
	var res DrainResult
	// Counted over EVERY entry the pass saw, before the budget can cut the loop
	// short: an overdue entry the budget did not reach is still unredacted text
	// past its guaranteed lifetime, and a count that only reported the ones a
	// pass happened to touch would go quiet exactly when the backlog was worst.
	for _, s := range ordered {
		if s.Overdue {
			res.Overdue++
		}
	}
	var consumed int64
	attempted := 0
	for i, s := range ordered {
		if budget.exhausted(attempted, consumed, s.Bytes) {
			res.Remaining = len(ordered) - i
			break
		}
		attempted++
		consumed += s.Bytes
		if s.Err != "" {
			res.Failed = append(res.Failed, DrainFailure{SessionID: s.SessionID, Path: s.Path, Err: s.Err})
			continue
		}
		stagedBytes, err := fsutil.ReadGuarded(s.Path, maxTranscriptBytes)
		if err != nil {
			res.Failed = append(res.Failed, DrainFailure{SessionID: s.SessionID, Path: s.Path,
				Err: fmt.Sprintf("cannot read staged transcript: %v", err)})
			continue
		}
		body, extended := refreshedFromSource(s, stagedBytes)
		if extended {
			res.Extended++
		}
		cr, err := Capture(repoRoot, rootSHA, body, s.captureMeta())
		if err != nil {
			res.Failed = append(res.Failed, classifyDrainFailure(sdir, rootSHA, s, stagedBytes, err))
			continue
		}
		// Stored (or already stored): the staged copy has done its job, and it is
		// unredacted, so it goes now rather than lingering — unless a concurrent
		// Stage replaced it while Capture ran, in which case the newer bytes stay
		// staged for the next pass. The comparison is against what was READ from
		// staging, not against what was captured: a drain that took fresher bytes
		// from the source path would otherwise never recognise its own staged
		// copy and would leave it behind forever.
		if err := removeStagedIfUnchanged(sdir, s, stagedBytes); err != nil {
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

// drainOrder puts OVERDUE entries before fresh ones, and within each of those
// halves main-thread entries before sub-agent ones, every quarter keeping
// listStaged's chronological order.
//
// A session's main thread stages LAST, because it ends last, so a chronological
// pass under a budget would drain a session's branches and leave its spine —
// the one transcript that makes the branches legible. Whatever a truncated pass
// stores should be the part the rest can be read against.
//
// The overdue split sits ABOVE that one and is the reason a budgeted drain
// eventually finishes at all. Ordering by arrival alone, a busy repository
// whose sessions stage faster than one pass drains can starve its own oldest
// entry indefinitely — and the oldest entry is precisely the raw text that has
// been on disk longest, which is the privacy fact, not a scheduling one. Age
// therefore buys priority. It buys nothing else: an overdue entry is drained
// through the same fail-closed Capture as any other, and is never deleted or
// degraded for being old.
//
// An entry whose sidecar could not be read sorts with the main-thread half; it
// is reported rather than captured either way, and reporting it early is what a
// budgeted pass should spend an attempt on.
func drainOrder(staged []Staged) []Staged {
	out := make([]Staged, 0, len(staged))
	for _, wantOverdue := range []bool{true, false} {
		for _, wantMain := range []bool{true, false} {
			for _, s := range staged {
				if s.Overdue == wantOverdue && (s.AgentID == "") == wantMain {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

// refreshedFromSource is the drain-time half of the flush-race mitigation. If a
// transcript was staged before the harness finished writing it, the file it was
// read from now holds more of it.
//
// The source replaces the staged bytes ONLY when it is strictly longer AND the
// staged bytes are a byte-prefix of it. A harness that recycles a path for a
// different transcript therefore cannot substitute one for another: a divergent
// file fails the prefix test and is ignored. Anything else — an absent source,
// an unreadable one, a shorter one — leaves the staged bytes exactly as staged.
//
// This does NOT settle whether SubagentStop fires before the flush. It bounds
// the damage if it does; measuring the rate is a separate step, and the two
// re-reads are what make it measurable, because a staged copy that the source
// strictly extends is a truncation that was caught.
func refreshedFromSource(s Staged, stagedBytes []byte) (body []byte, extended bool) {
	if s.SourcePath == "" {
		return stagedBytes, false
	}
	current, err := fsutil.ReadGuarded(s.SourcePath, maxTranscriptBytes)
	if err != nil {
		return stagedBytes, false
	}
	if len(current) > len(stagedBytes) && bytes.HasPrefix(current, stagedBytes) {
		return current, true
	}
	return stagedBytes, false
}

// removeStagedIfUnchanged removes the staged file under the staging lock, and
// only if it still holds exactly the bytes the drain read from it. A file that
// is already gone is fine (a peer drain retired it); a file whose bytes differ
// was re-staged since the capture read it and is left in place — its transcript
// is not yet in the store, so deleting it would be the silent loss this
// mechanism exists to end.
//
// The sidecar goes with the transcript and only with it: a sidecar left behind
// describes a staged file that no longer exists, and one removed while its
// transcript stays would send the next pass down the legacy path with an agent
// id where a session id belongs.
func removeStagedIfUnchanged(sdir string, s Staged, read []byte) error {
	want := sha256.Sum256(read)
	return withStagingLock(sdir, func() error {
		current, err := fsutil.ReadGuarded(s.Path, maxTranscriptBytes)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if sha256.Sum256(current) != want {
			return nil // replaced mid-drain; the next pass captures the newer bytes
		}
		if err := os.Remove(s.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.Remove(sidecarPathFor(s.Path)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	})
}

// TranscriptSettled reports whether a JSONL transcript's last non-blank line is
// a complete JSON value.
//
// It is the predicate behind the stage-time half of the flush-race mitigation:
// a hook that fires the instant an agent stops may read a transcript the
// harness has not finished appending to, and a severed final line is the shape
// that has. The caller waits and re-reads on false; the wait is deliberately
// NOT in this package, because it must never happen inside the staging lock —
// a burst of simultaneous completions would serialise on a lock whose timeout
// was tuned for one SessionEnd.
//
// Empty input is unsettled: there is nothing to have finished writing.
//
// Whether SubagentStop actually fires before the flush is UNMEASURED. This is
// a mitigation, not an answer, and the measurement is a separate step.
func TranscriptSettled(raw []byte) bool {
	trimmed := bytes.TrimRight(raw, " \t\r\n")
	if len(trimmed) == 0 {
		return false
	}
	last := trimmed
	if i := bytes.LastIndexByte(trimmed, '\n'); i >= 0 {
		last = trimmed[i+1:]
	}
	return json.Valid(bytes.TrimSpace(last))
}
