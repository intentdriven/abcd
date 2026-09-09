package history

// Migrating the composite records.
//
// Before the lineage fields existed, a sub-agent's transcript was filed under a
// hand-made identifier: the spawning session TRUNCATED to a prefix, glued to
// the agent id. That fails in both directions — a reader holding the full
// session id the harness gives them cannot match the stored value, and a reader
// holding the stored value cannot recover the full session id from it, because
// the truncation is lossy.
//
// The truncated half is not gone, though. The record's own body is the
// harness's line-delimited transcript, and every one of its lines carries the
// full session identifier. So the migration READS the record it is repairing:
// the body is the source, and the stored prefix is only the check that the two
// describe the same session. A body that disagrees leaves the record untouched
// and is reported, because a prefix that is a lossy truncation can confirm a
// candidate but can never produce one.
//
// Three things this deliberately does not do. It does not recompute
// source_sha256, which is taken over the raw source and is what makes a
// migrated record still dedup against a re-capture of the same bytes. It does
// not rename the file, because listRecords parses frontmatter and never the
// filename, so a rename would break a path a reader already holds and buy
// nothing. And it does not write at all unless it is asked to: the store holds
// the only copy of these records.

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// compositeAgentSep is the marker the pre-lineage binary put between the
// session half and the agent half of the identifier it composed.
const compositeAgentSep = "--agent-"

// compositeSegmentSep separates the session prefix from anything the composing
// binary wedged between it and the agent marker. The store on this machine
// holds two shapes: `<prefix>--agent-<agent>`, and — for an agent launched
// inside a workflow — `<prefix>--wf_<workflow>--agent-<agent>`. The spec
// describes only the first, and splitting on the LAST separator rather than the
// first is what keeps the second from being read as a session prefix that
// nothing in the body can confirm.
const compositeSegmentSep = "--"

// HarnessLineage is what the harness's own per-agent metadata can tell a
// migration about an agent it recorded: the kind of agent, how deep it was
// spawned, the tool call that launched it, and — at depth two and deeper — the
// agent that spawned it.
//
// The spec says these fields "were never held" anywhere and so must stay empty
// on a migrated record. That is wrong on this machine and, as far as the store
// can tell, on any machine whose harness still has its per-agent metadata:
// the files are keyed by agent id and outlive the transcripts. Where they
// answer, the migration uses them; where they do not, the record says its
// lineage is unknown through spawn_attribution rather than by looking empty.
type HarnessLineage struct {
	AgentType      string
	ParentAgentID  string
	SpawnDepth     int
	SpawnToolUseID string
}

// LineageRef identifies the agent a lookup is being asked about. SourcePath is
// set only when the caller is holding the transcript file itself (ingest does,
// migrate does not), so a resolver can take the cheap route of deriving the
// harness's metadata file from the transcript path before falling back to a
// search.
type LineageRef struct {
	SessionID  string
	AgentID    string
	SourcePath string
}

// LineageLookup answers what the harness recorded about one agent.
//
// It is a function rather than a path because core holds no knowledge of the
// harness's on-disk layout and must not acquire any: a design that reads a
// vendor's directory structure is one the vendor can silently break. The front
// door supplies the resolver, and a nil lookup simply means the rung does not
// answer — never an error.
type LineageLookup func(LineageRef) (HarnessLineage, bool)

// MigrateOptions carries what the migration needs beyond the store key.
type MigrateOptions struct {
	// RepoRoot is the repository whose scanner configuration governs this
	// store. It is required in BOTH modes: the lineage a lookup returns is
	// externally supplied and lands in frontmatter, so it is redacted before it
	// is reported, not only before it is written.
	RepoRoot string
	// Apply switches the run from reporting to writing.
	Apply bool
	// Lineage is the attribution ladder's first rung. Optional.
	Lineage LineageLookup
}

// MigrateEntry is one composite record's outcome.
type MigrateEntry struct {
	Path            string `json:"path"`
	StoredSessionID string `json:"stored_session_id"`
	SessionID       string `json:"session_id,omitempty"`
	AgentID         string `json:"agent_id,omitempty"`
	// ExtraSegment is whatever the composing binary put between the session
	// prefix and the agent marker (a workflow id, on this machine). It is
	// reported rather than stored: the schema has no field for it, and dropping
	// it silently would lose the only trace that these records were composed
	// differently from the rest.
	ExtraSegment     string `json:"extra_segment,omitempty"`
	AgentType        string `json:"agent_type,omitempty"`
	ParentAgentID    string `json:"parent_agent_id,omitempty"`
	SpawnDepth       int    `json:"spawn_depth,omitempty"`
	SpawnToolUseID   string `json:"spawn_tool_use_id,omitempty"`
	SpawnAttribution string `json:"spawn_attribution,omitempty"`
	Wrote            bool   `json:"wrote"`
	Refused          string `json:"refused,omitempty"`
}

// MigrateResult is what one migration run saw and did.
type MigrateResult struct {
	Applied   bool           `json:"applied"`
	Scanned   int            `json:"scanned"`
	Composite int            `json:"composite"`
	Migrated  []MigrateEntry `json:"migrated"`
	Refused   []MigrateEntry `json:"refused"`
}

// Migrate repairs every record whose session_id is a composite identifier.
//
// It reports by default and writes only under opts.Apply. Re-running it is a
// no-op: a record that already carries an agent id has been migrated, and a
// record whose session id is not a composite was never this verb's business.
func Migrate(rootSHA string, opts MigrateOptions) (MigrateResult, error) {
	if !rootSHARe.MatchString(rootSHA) {
		return MigrateResult{}, errors.New(rootSHAErrMsg)
	}
	if opts.RepoRoot == "" {
		return MigrateResult{}, errors.New("history: migrate needs the destination repository root; the lineage it recovers is redacted under that repository's scanner configuration")
	}
	tdir, err := ownedDirsReal(rootSHA)
	if err != nil {
		return MigrateResult{}, err
	}
	// Fail closed on a degraded scanner exactly as Capture does. A migration
	// that could not redact what it learned would write externally supplied
	// text into frontmatter with less coverage than the repository asked for.
	sc, err := scanner.New(opts.RepoRoot)
	if err != nil {
		return MigrateResult{}, fmt.Errorf("history: scanner init: %w", err)
	}
	if unavail, reason := sc.Unavailable(); unavail {
		return MigrateResult{}, fmt.Errorf("history: refusing to migrate with a degraded scanner: %s", reason)
	}

	release, err := repoLock(tdir)
	if err != nil {
		return MigrateResult{}, err
	}
	defer release()

	records, err := listRecords(tdir)
	if err != nil {
		return MigrateResult{}, err
	}
	res := MigrateResult{Applied: opts.Apply, Scanned: len(records)}
	for _, r := range records {
		prefix, extra, agentID, ok := splitComposite(r.SessionID)
		if !ok || r.AgentID != "" {
			continue
		}
		res.Composite++
		entry := MigrateEntry{Path: r.Path, StoredSessionID: r.SessionID, AgentID: agentID, ExtraSegment: extra}
		if err := migrateOne(sc, opts, r, prefix, agentID, &entry); err != nil {
			entry.Refused = err.Error()
			res.Refused = append(res.Refused, entry)
			continue
		}
		res.Migrated = append(res.Migrated, entry)
	}
	return res, nil
}

// migrateOne repairs one record, filling entry with what it recovered. It
// returns an error only for a refusal that leaves the record untouched.
func migrateOne(sc *scanner.Scanner, opts MigrateOptions, r Record, prefix, agentID string, entry *MigrateEntry) error {
	data, err := fsutil.ReadGuarded(r.Path, maxTranscriptBytes)
	if err != nil {
		return fmt.Errorf("record unreadable: %w", err)
	}
	rec, body, err := parseRecord(data)
	if err != nil {
		return fmt.Errorf("record unparseable: %w", err)
	}
	full, err := recoverSessionID(body, prefix)
	if err != nil {
		return err
	}
	entry.SessionID = full

	meta := CaptureMeta{
		SessionID:        full,
		Kind:             rec.SourceKind,
		AgentID:          agentID,
		LineageSource:    "migrated",
		SpawnAttribution: "unattributed",
	}
	if opts.Lineage != nil {
		if h, ok := opts.Lineage(LineageRef{SessionID: full, AgentID: agentID}); ok && h.SpawnDepth > 0 {
			meta.SpawnAttribution = "sidecar"
			meta.AgentType = h.AgentType
			meta.SpawnDepth = h.SpawnDepth
			meta.SpawnToolUseID = h.SpawnToolUseID
			// A parent id is written into a record that names an agent with a
			// record of its own, so it is held to the same shape as any other
			// agent id; a value that fails is dropped rather than sinking the
			// whole repair.
			if agentIDRe.MatchString(h.ParentAgentID) {
				meta.ParentAgentID = h.ParentAgentID
			}
		}
	}
	// Everything the lookup returned is externally supplied and lands in
	// frontmatter, which the read path never scans. It goes through the same
	// sanitise-then-verify pass a captured body does. The recovered session id
	// does NOT need one: it came out of a body that was already redacted on the
	// way in.
	meta, err = redactLineage(sc, meta)
	if err != nil {
		return err
	}
	if err := meta.validate(); err != nil {
		return err
	}

	entry.AgentID = meta.AgentID
	entry.AgentType = meta.AgentType
	entry.ParentAgentID = meta.ParentAgentID
	entry.SpawnDepth = meta.SpawnDepth
	entry.SpawnToolUseID = meta.SpawnToolUseID
	entry.SpawnAttribution = meta.SpawnAttribution
	if !opts.Apply {
		return nil
	}

	rec.SessionID = meta.SessionID
	rec.AgentID = meta.AgentID
	rec.ParentAgentID = meta.ParentAgentID
	rec.AgentType = meta.AgentType
	rec.SpawnDepth = meta.SpawnDepth
	rec.SpawnToolUseID = meta.SpawnToolUseID
	rec.LineageSource = meta.LineageSource
	rec.SpawnAttribution = meta.SpawnAttribution
	// Preserve-mode, in place, at the path the record already has: the filename
	// is left alone on purpose.
	if err := fsutil.WriteFileAtomicPreserveMode(r.Path, marshalRecord(rec, body)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	entry.Wrote = true
	return nil
}

// splitComposite takes a stored session_id apart into the session prefix, any
// segment the composing binary wedged in the middle, and the agent id.
//
// The agent marker is found from the RIGHT: everything left of it is the
// composed session half, whose first "--"-delimited segment is the truncated
// session prefix. A left-to-right split would read a workflow id as part of the
// prefix and refuse every workflow-launched agent in the store.
func splitComposite(stored string) (prefix, extra, agentID string, ok bool) {
	i := strings.LastIndex(stored, compositeAgentSep)
	if i < 0 {
		return "", "", "", false
	}
	head, agent := stored[:i], stored[i+len(compositeAgentSep):]
	if head == "" || !agentIDRe.MatchString(agent) {
		return "", "", "", false
	}
	prefix = head
	if j := strings.Index(head, compositeSegmentSep); j >= 0 {
		prefix, extra = head[:j], head[j+len(compositeSegmentSep):]
	}
	if prefix == "" {
		return "", "", "", false
	}
	return prefix, extra, agent, true
}

// transcriptLineIdentity is the sliver of a harness transcript line this
// package reads: which session it belongs to, which agent produced it, and
// where it was running. Every other key is ignored — the line's shape is the
// harness's to change, and reading four fields out of it is the smallest
// dependency that answers the question.
type transcriptLineIdentity struct {
	SessionID       string `json:"sessionId"`
	ParentSessionID string `json:"parentSessionId"`
	AgentID         string `json:"agentId"`
	Cwd             string `json:"cwd"`
}

// recoverSessionID finds the full session identifier in a record body.
//
// Only a value that BEGINS with the stored prefix is a candidate, and exactly
// one distinct candidate is required. Two would mean the record body spans two
// sessions, which is not a thing to guess between; none means the body cannot
// confirm the prefix, and a prefix alone is not a session id.
func recoverSessionID(body, prefix string) (string, error) {
	var found []string
	seen := map[string]struct{}{}
	forEachJSONLine(body, func(line transcriptLineIdentity) {
		for _, id := range []string{line.SessionID, line.ParentSessionID} {
			if id == "" || !strings.HasPrefix(id, prefix) || !sessionIDRe.MatchString(id) {
				continue
			}
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			found = append(found, id)
		}
	})
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		return "", fmt.Errorf("no session identifier in the body begins with the stored prefix %q; the record is left untouched", prefix)
	default:
		return "", fmt.Errorf("the body carries %d session identifiers beginning with the stored prefix %q; refusing to guess which session owns this transcript", len(found), prefix)
	}
}

// forEachJSONLine walks a line-delimited transcript, handing the caller the few
// identity fields of every line that parses as a JSON object. Lines are sliced
// out of the text rather than split into a slice, because a transcript body can
// run to tens of megabytes and nothing here needs them all at once.
func forEachJSONLine(text string, fn func(transcriptLineIdentity)) {
	for rest := text; rest != ""; {
		line := rest
		if i := strings.IndexByte(rest, '\n'); i >= 0 {
			line, rest = rest[:i], rest[i+1:]
		} else {
			rest = ""
		}
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var parsed transcriptLineIdentity
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			continue
		}
		fn(parsed)
	}
}

// redactLineage runs a CaptureMeta's scalars through the same two-stage,
// fail-closed pass Capture runs over a transcript, and returns the meta with
// the redacted values in place.
//
// It reuses frameLineage/unframeLineage rather than scanning the fields
// individually, so the write paths that do not go through Capture — the
// migration's in-place rewrite — cannot drift from the one that does. The
// gitleaks augmentation is deliberately NOT run here: it is a subprocess with a
// 30-second timeout, these are a handful of short identifiers rather than a
// transcript, and a per-record subprocess would make a 176-record migration
// unusable. A record whose BODY needs that coverage got it when it was
// captured.
func redactLineage(sc *scanner.Scanner, m CaptureMeta) (CaptureMeta, error) {
	text := frameLineage(m, nil)
	redacted, _ := scanner.Redact(text, sc.ScanText(text, "transcript"))
	if home := scanner.CallerHome(); home != "" {
		redacted = scanner.SweepCallerHome(redacted, home)
		var resid []scanner.Finding
		redacted, resid = scanner.SurvivingCallerHome(redacted, home)
		if len(resid) > 0 {
			return CaptureMeta{}, &RedactionResidualError{Residual: resid}
		}
	}
	if resid := scanner.BlockingResidual(sc.ScanText(redacted, "transcript")); len(resid) > 0 {
		return CaptureMeta{}, &RedactionResidualError{Residual: resid}
	}
	scalars, _, err := unframeLineage(redacted)
	if err != nil {
		return CaptureMeta{}, err
	}
	return applyLineageScalars(m, scalars), nil
}
