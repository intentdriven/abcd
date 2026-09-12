package history

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// rootSHARe is the immutable repo key: a lowercase hex commit SHA, 40 chars for
// git's SHA-1 object format or 64 for SHA-256. Accepting only 40 made every
// history verb (Capture/List/Read) fail for a SHA-256 repo, whose root SHA the
// ahoy layer derives at 64 chars — mirrors the voyage-ledger key fix.
var rootSHARe = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// rootSHAErrMsg is the diagnostic for a value that fails rootSHARe; it names both
// accepted widths so it cannot drift from the regex (which accepts SHA-256's 64
// as well as SHA-1's 40).
const rootSHAErrMsg = "history: rootSHA must be a 40- or 64-character lowercase hex commit SHA"

// sessionIDRe restricts a vendor session id to filesystem-safe characters so it
// can be embedded verbatim in a record filename with no path-traversal or
// separator surprises.
var sessionIDRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// agentIDRe restricts a harness-supplied agent id to the same filesystem-safe
// charset as a session id, and for the same reason: a sub-agent's agent id is
// embedded verbatim in its record filename, so a separator or a traversal
// segment in it is a path hazard rather than a cosmetic problem. A parent agent
// id is held to the same shape because it names an agent that has, or will
// have, a record of its own.
var agentIDRe = sessionIDRe

// validKinds are the accepted source_kind values.
var validKinds = map[string]struct{}{
	"native":           {},
	"specstory-import": {},
}

// validLineageSources are the accepted lineage_source values: which rung of the
// attribution ladder answered. It is what distinguishes an agent_type that was
// never recoverable from one that was never there.
var validLineageSources = map[string]struct{}{
	"hook":     {},
	"ingest":   {},
	"migrated": {},
}

// validSpawnAttributions are the accepted spawn_attribution values: which rung
// of the attribution ladder placed this agent's spawn point.
//
// This field exists because the field set could otherwise not tell two
// different empties apart. An empty parent_agent_id was read as "the main
// thread spawned it", but a hook-sourced record captured with no harness
// sidecar has it empty too — and spawn_depth zero with it — so a genuine
// depth-1 child of the main thread and a record whose lineage was never
// recovered were the same bytes. lineage_source cannot separate them: it names
// which DOOR the record came through, and both came through the hook. Making it
// carry the rung as well would overload one field with two facts, which is the
// defect adr-2609090636172016 removed; so it is a field, in the same flat
// one-scalar-per-line frontmatter idiom as the rest.
var validSpawnAttributions = map[string]struct{}{
	"sidecar":      {}, // the harness's per-agent sidecar answered
	"transcript":   {}, // the spawning transcript's own tool result answered
	"unattributed": {}, // nothing answered; the spawn fields carry no information
}

// validate checks CaptureMeta's external inputs at the store's boundary. Every
// field here comes off a harness payload or a configuration file, and every one
// of them is written into a record — into a filename, in the agent id's case,
// and into one-scalar-per-line frontmatter in the rest. A line break in a scalar
// would let externally supplied text forge a frontmatter field, so it is refused
// here rather than escaped downstream.
func (m CaptureMeta) validate() error {
	if !sessionIDRe.MatchString(m.SessionID) {
		return fmt.Errorf("history: sessionID must be non-empty and match [A-Za-z0-9._-]+")
	}
	if _, ok := validKinds[m.Kind]; !ok {
		return fmt.Errorf("history: source kind %q is not one of native, specstory-import", m.Kind)
	}
	if m.AgentID != "" && !agentIDRe.MatchString(m.AgentID) {
		return fmt.Errorf("history: agentID must match [A-Za-z0-9._-]+")
	}
	if m.ParentAgentID != "" && !agentIDRe.MatchString(m.ParentAgentID) {
		return fmt.Errorf("history: parentAgentID must match [A-Za-z0-9._-]+")
	}
	if m.SpawnDepth < 0 {
		return fmt.Errorf("history: spawnDepth must not be negative, got %d", m.SpawnDepth)
	}
	if m.LineageSource != "" {
		if _, ok := validLineageSources[m.LineageSource]; !ok {
			return fmt.Errorf("history: lineage source %q is not one of hook, ingest, migrated", m.LineageSource)
		}
	}
	if err := m.validateSpawnAttribution(); err != nil {
		return err
	}
	for _, f := range []struct{ name, value string }{
		{"agentType", m.AgentType},
		{"spawnToolUseID", m.SpawnToolUseID},
		{"adoptedProject", m.AdoptedProject},
	} {
		if strings.ContainsAny(f.value, "\r\n") {
			return fmt.Errorf("history: %s must not contain a line break (record frontmatter is one scalar per line)", f.name)
		}
	}
	return nil
}

// validateSpawnAttribution holds the invariant that makes "no parent" and
// "unknown parent" structurally distinct for everything written from here on.
// A sub-agent record MUST say which rung placed it — an unset field would be
// the ambiguity itself — and a record claiming nothing placed it cannot also
// carry spawn detail. A main-thread record has no spawn to attribute.
func (m CaptureMeta) validateSpawnAttribution() error {
	if m.AgentID == "" {
		if m.SpawnAttribution != "" {
			return fmt.Errorf("history: spawnAttribution %q is meaningless on a main-thread record (no agent id)", m.SpawnAttribution)
		}
		return nil
	}
	if _, ok := validSpawnAttributions[m.SpawnAttribution]; !ok {
		return fmt.Errorf("history: a sub-agent record needs a spawnAttribution of sidecar, transcript or unattributed, got %q", m.SpawnAttribution)
	}
	if m.SpawnAttribution == "unattributed" &&
		(m.ParentAgentID != "" || m.SpawnDepth != 0 || m.SpawnToolUseID != "") {
		return fmt.Errorf("history: an unattributed spawn cannot also name a parent, a depth or a spawning tool call")
	}
	return nil
}

// maxTranscriptBytes caps a single guarded record read from the store. It
// matches the transcript-capture cap on the write side; a record grown past it
// out of band is refused rather than read wholly into memory.
const maxTranscriptBytes = 64 << 20 // 64 MiB

// repoLock takes a per-<rootSHA> advisory lock on records/.lock, disjoint
// from ahoy's index lock. The lock file is opened O_NOFOLLOW mode 0o600 so a
// pre-planted lock-file symlink is refused. The returned release closes the fd
// (which drops the flock). Ports the two-domain lock model from
// history_store.py.
func repoLock(tdir string) (func(), error) {
	lockPath := filepath.Join(tdir, ".lock")
	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, &StorePathError{Path: lockPath, Msg: "lock file open refused (symlinked or unwritable): " + err.Error()}
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("history: acquire lock %s: %w", lockPath, err)
	}
	return func() { f.Close() }, nil
}

// recordFilename is <compact-utc>-<session-id>.md for a main-thread record and
// <compact-utc>-<session-id>-agent-<agent-id>.md for a sub-agent's. It sorts
// chronologically and, with nanosecond precision, does not collide within a
// session.
//
// The agent segment is readable convenience ONLY: an operator listing the
// directory can tell a session's spine from its branches. Nothing decodes this
// string back into fields — listRecords parses frontmatter and never the
// filename — which is what keeps the composite-identifier defect this schema
// removed from reappearing one directory later.
func recordFilename(capturedAt time.Time, sessionID, agentID string) string {
	name := capturedAt.UTC().Format("20060102T150405.000000000Z") + "-" + sessionID
	if agentID != "" {
		name += "-agent-" + agentID
	}
	return name + ".md"
}

// frontmatter fields (flat, one scalar per line) — a small fixed schema parsed
// by a line reader, so no YAML dependency is pulled in.
const (
	fmSchema      = "schema"
	fmSessionID   = "session_id"
	fmRootCommit  = "root_commit"
	fmCapturedAt  = "captured_at"
	fmSourceKind  = "source_kind"
	fmSourceSHA   = "source_sha256"
	fmRedSecrets  = "redacted_secrets"
	fmRedHomePath = "redacted_home_paths"

	// Lineage (schema 2). Every one of these is omitted when empty, so a
	// main-thread record is byte-identical to its schema-1 shape but for the
	// version stamp.
	fmAgentID          = "agent_id"
	fmParentAgentID    = "parent_agent_id"
	fmAgentType        = "agent_type"
	fmSpawnDepth       = "spawn_depth"
	fmSpawnToolUseID   = "spawn_tool_use_id"
	fmLineageSource    = "lineage_source"
	fmSpawnAttribution = "spawn_attribution"

	// Adoption (schema 3).
	fmAdoptedProject = "adopted_project"
)

// marshalRecord renders a record file: YAML frontmatter then the redacted body.
func marshalRecord(r Record, body string) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "%s: %d\n", fmSchema, recordSchemaVersion)
	fmt.Fprintf(&b, "%s: %s\n", fmSessionID, r.SessionID)
	fmt.Fprintf(&b, "%s: %s\n", fmRootCommit, r.RootCommit)
	fmt.Fprintf(&b, "%s: %s\n", fmCapturedAt, r.CapturedAt.UTC().Format(time.RFC3339Nano))
	fmt.Fprintf(&b, "%s: %s\n", fmSourceKind, r.SourceKind)
	fmt.Fprintf(&b, "%s: %s\n", fmSourceSHA, r.SourceSHA256)
	fmt.Fprintf(&b, "%s: %d\n", fmRedSecrets, r.Secrets)
	fmt.Fprintf(&b, "%s: %d\n", fmRedHomePath, r.HomePaths)
	// Lineage, omitted when absent: an empty field would be a trailing-space
	// line, and a main-thread record has nothing to say here.
	for _, f := range []struct{ key, value string }{
		{fmAgentID, r.AgentID},
		{fmParentAgentID, r.ParentAgentID},
		{fmAgentType, r.AgentType},
		{fmSpawnToolUseID, r.SpawnToolUseID},
		{fmLineageSource, r.LineageSource},
		{fmSpawnAttribution, r.SpawnAttribution},
		{fmAdoptedProject, r.AdoptedProject},
	} {
		if f.value != "" {
			fmt.Fprintf(&b, "%s: %s\n", f.key, f.value)
		}
	}
	if r.SpawnDepth > 0 {
		fmt.Fprintf(&b, "%s: %d\n", fmSpawnDepth, r.SpawnDepth)
	}
	b.WriteString("---\n")
	b.WriteString(marshalBody(body))
	return []byte(b.String())
}

// marshalBody is the body exactly as a record file holds it: newline-terminated.
// It is a named seam because supersession compares a candidate body against what
// is already on disk, and a comparison against a differently-terminated string
// would miss the prefix relation it exists to find.
func marshalBody(body string) string {
	if strings.HasSuffix(body, "\n") {
		return body
	}
	return body + "\n"
}

// parseRecord splits a record file into its metadata and redacted body. The
// Path field is set by the caller. Returns an error when the frontmatter fence
// is missing or a required field is malformed.
func parseRecord(data []byte) (Record, string, error) {
	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		return Record{}, "", fmt.Errorf("history: record missing frontmatter fence")
	}
	rest := text[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return Record{}, "", fmt.Errorf("history: record frontmatter not terminated")
	}
	head := rest[:end]
	body := rest[end+len("\n---\n"):]

	fields := map[string]string{}
	for _, line := range strings.Split(head, "\n") {
		if line == "" {
			continue
		}
		i := strings.Index(line, ": ")
		if i < 0 {
			continue
		}
		fields[line[:i]] = line[i+2:]
	}

	var r Record
	r.SessionID = fields[fmSessionID]
	r.RootCommit = fields[fmRootCommit]
	r.SourceKind = fields[fmSourceKind]
	r.SourceSHA256 = fields[fmSourceSHA]
	if r.SessionID == "" || r.RootCommit == "" || r.SourceSHA256 == "" {
		return Record{}, "", fmt.Errorf("history: record frontmatter missing a required field")
	}
	if ts := fields[fmCapturedAt]; ts != "" {
		t, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			return Record{}, "", fmt.Errorf("history: record captured_at unparseable: %w", err)
		}
		r.CapturedAt = t.UTC()
	}
	r.Secrets, _ = strconv.Atoi(fields[fmRedSecrets])
	r.HomePaths, _ = strconv.Atoi(fields[fmRedHomePath])
	// Lineage is optional in BOTH directions: absent on a main-thread record and
	// absent on every schema-1 record, which is why a schema-1 record parses as a
	// main-thread record rather than as a fault.
	r.AgentID = fields[fmAgentID]
	r.ParentAgentID = fields[fmParentAgentID]
	r.AgentType = fields[fmAgentType]
	r.SpawnToolUseID = fields[fmSpawnToolUseID]
	r.LineageSource = fields[fmLineageSource]
	r.SpawnAttribution = fields[fmSpawnAttribution]
	r.AdoptedProject = fields[fmAdoptedProject]
	r.SpawnDepth, _ = strconv.Atoi(fields[fmSpawnDepth])
	return r, body, nil
}

// listRecords reads every *.md record under tdir, newest first. A record file
// that fails to parse is skipped (an individual corrupt transcript is not
// fatal to the rest), mirroring ScanBundle's per-file tolerance.
func listRecords(tdir string) ([]Record, error) {
	entries, err := os.ReadDir(tdir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Record
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		p := filepath.Join(tdir, e.Name())
		// Guarded read: the transcripts dir is a cross-repo store under HOME,
		// so a *.md symlink or FIFO planted by anything running as the caller
		// must not be followed or block the listing (iss-383). A refused leaf
		// is skipped like an unparseable one.
		data, err := fsutil.ReadGuarded(p, maxTranscriptBytes)
		if err != nil {
			continue
		}
		r, _, err := parseRecord(data)
		if err != nil {
			continue
		}
		r.Path = p
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].CapturedAt.Equal(out[j].CapturedAt) {
			return out[i].CapturedAt.After(out[j].CapturedAt)
		}
		return out[i].Path > out[j].Path
	})
	return out, nil
}

// StorePathError is a preflight fault: an owned store path is absent, a symlink,
// or otherwise unsafe. It is returned (never panicked) so the caller can surface
// a clean diagnostic and refuse the operation.
type StorePathError struct {
	Path string
	Msg  string
}

func (e *StorePathError) Error() string { return "history: " + e.Msg + " (" + e.Path + ")" }
