package history

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/sessionkind"
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

// A record names where its transcript came from with two separate labels
// (ruling J13 on iss-2608230752354928): the ROUTE it reached the store by, in
// source_kind, and the TOOL that produced it, in source_tool.
//
// The route vocabulary is closed. `native` is abcd's own capture of the
// transcript the host harness wrote — its hooks, or its capture and ingest
// verbs reading that transcript. `import` is a transcript another tool
// exported, brought in through abcd. The tool vocabulary is open: any
// lowercase slug naming the producing tool, with `host` reserved for the
// harness abcd is installed in, which is what a native capture that names no
// tool records.
const (
	RouteNative = "native"
	RouteImport = "import"
	ToolHost    = "host"
)

// validRoutes are the accepted source_kind values.
var validRoutes = map[string]struct{}{
	RouteNative: {},
	RouteImport: {},
}

// legacyKinds are the source_kind values a record written before the split
// may carry that fused a tool into the route. Each maps to the two labels it
// always meant; a record carrying one is read under them, never rewritten
// here.
var legacyKinds = map[string][2]string{
	"specstory-import": {RouteImport, "specstory"},
}

// errToolRedacted refuses a source_tool label the redaction pass rewrote. The
// label is a name, and a name redaction changed is not the tool's name, so
// Capture and migrate stop rather than store a record under a masked label.
var errToolRedacted = errors.New("history: the source tool label was redacted; name the tool with a label the scanner does not match")

// toolRe is the source_tool shape: a lowercase slug, so the label can be
// neither a sentence nor a second frontmatter line.
var toolRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// sourceLabels derives a record's two labels from what its frontmatter or its
// caller supplied. It is the one place the pre-split vintage is read: a legacy
// fused kind splits into its route and tool, and a native record that names no
// tool names the host. Anything else passes through for validateSource to
// judge.
//
// A fused kind already names its tool, so a tool that names a different one is
// a second answer to the same question. It is refused, never resolved in favour
// of either: the store cannot know which one the caller meant. Naming the same
// tool twice is one answer and passes.
func sourceLabels(kind, tool string) (route, sourceTool string, err error) {
	if pair, ok := legacyKinds[kind]; ok {
		if tool != "" && tool != pair[1] {
			return "", "", fmt.Errorf("history: source kind %q already names the tool %q, which conflicts with the tool %q; name the two labels separately: kind %s with tool %s, or kind %s with tool %s",
				kind, pair[1], tool, pair[0], tool, pair[0], pair[1])
		}
		kind, tool = pair[0], pair[1]
	}
	if kind == RouteNative && tool == "" {
		tool = ToolHost
	}
	return kind, tool, nil
}

// validateSource holds the two labels apart. The route is one of the closed
// set, so a tool name cannot stand in it; the tool is a slug that is not a
// route word and not a fused `<tool>-import` value, so a route cannot stand in
// it; and an import names the tool that exported it, which is never the host,
// because the host's own transcript is what the native route records.
func validateSource(route, tool string) error {
	if _, ok := validRoutes[route]; !ok {
		return fmt.Errorf("history: source route %q is not one of native, import", route)
	}
	if !toolRe.MatchString(tool) {
		if route == RouteImport && tool == "" {
			return fmt.Errorf("history: an import must name the tool that exported the transcript")
		}
		return fmt.Errorf("history: source tool %q must be a lowercase slug matching [a-z0-9][a-z0-9._-]*", tool)
	}
	if _, ok := validRoutes[tool]; ok {
		return fmt.Errorf("history: source tool %q is a route, not a tool", tool)
	}
	if _, ok := legacyKinds[tool]; ok || strings.HasSuffix(tool, "-"+RouteImport) {
		return fmt.Errorf("history: source tool %q fuses a route into the tool; name the tool alone", tool)
	}
	if route == RouteImport && tool == ToolHost {
		return fmt.Errorf("history: an import cannot name the host as its tool; the host's own transcript is a native capture")
	}
	return nil
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
	// Judged as they will be stored: staging validates a meta long before
	// Capture settles its labels, and must accept exactly what Capture will.
	route, tool, err := sourceLabels(m.Kind, m.Tool)
	if err != nil {
		return err
	}
	if err := validateSource(route, tool); err != nil {
		return err
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

// repoLockTimeout bounds how long a writer of one repo's records waits for
// records/.lock. The holder runs a whole capture — the scanner's two-stage
// redaction of a transcript up to maxTranscriptBytes — or a whole migration
// under it, so the budget is minutes rather than the seconds of a single-file
// lock; what it removes is the wait with no end, behind a holder that never
// lets go. A var so a test can shorten it.
var repoLockTimeout = 2 * time.Minute

// withRepoLock runs fn holding the per-<rootSHA> advisory lock on
// records/.lock, disjoint from ahoy's index lock. The lock is
// fsutil.WithFileLock, the one inter-process lock-file primitive: the file is
// opened O_NOFOLLOW at mode 0o600 and proved a regular file on the descriptor,
// so a pre-planted lock-file symlink is refused (a *StorePathError), and a
// holder past repoLockTimeout is fsutil.ErrLockContention naming the lock.
// fn's own error passes through unchanged. Ports the two-domain lock model from
// history_store.py.
func withRepoLock(tdir string, fn func() error) error {
	lockPath := filepath.Join(tdir, ".lock")
	ran := false
	err := fsutil.WithFileLock(lockPath, repoLockTimeout, func() error {
		ran = true
		return fn()
	})
	switch {
	case ran || err == nil:
		return err
	case errors.Is(err, fsutil.ErrLockContention):
		return fmt.Errorf("history: acquire lock %s: %w", lockPath, err)
	}
	return &StorePathError{Path: lockPath, Msg: "lock file open refused (symlinked or unwritable): " + err.Error()}
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
	fmSourceTool  = "source_tool"
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

	// The per-run context stamps a transcript carried, comma-joined. Optional
	// and omitted when empty, so it does not move the schema version: every
	// reader parses by field presence (adr-2609021016275803).
	fmContextStamps = "context_stamps"
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
	if r.SourceTool != "" {
		fmt.Fprintf(&b, "%s: %s\n", fmSourceTool, r.SourceTool)
	}
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
	if stamps := wellFormedStamps(r.ContextStamps); len(stamps) > 0 {
		fmt.Fprintf(&b, "%s: %s\n", fmContextStamps, strings.Join(stamps, ","))
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
//
// The delimiters are matched byte-exact, deliberately NOT by
// frontmatter.IsDelimiter: this is the store's own format, written only by this
// file, so a record whose fence is not the one the writer emits was not written
// here and is refused rather than read leniently (iss-2608270908348042).
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
	// Both vintages read: a record written before the route and the tool were
	// split carries source_kind alone, and its labels are derived here. No
	// writer pairs a fused kind with a different tool, so a record that does is
	// malformed and is not read under either answer.
	var err error
	r.SourceKind, r.SourceTool, err = sourceLabels(fields[fmSourceKind], fields[fmSourceTool])
	if err != nil {
		return Record{}, "", err
	}
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
	if raw := fields[fmContextStamps]; raw != "" {
		r.ContextStamps = wellFormedStamps(strings.Split(raw, ","))
	}
	return r, body, nil
}

// wellFormedStamps keeps the entries that are stamps, distinct and in order.
// Anything else in the field — a hand edit, a truncation — is not evidence of a
// context a session held, so it is dropped on both the write and the read
// rather than counted.
func wellFormedStamps(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if _, ok := sessionkind.Parse(s); !ok || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
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
