// Package history is abcd's native session-transcript store: the write/read/
// redact engine that populates ~/.abcd/history/<root-sha>/transcripts/ and
// retires the specstory shim (adr-29). It is transport-agnostic — no stdout, no
// os.Exit, no CLI knowledge — so any surface can drive it and marshal its
// structured results.
//
// The index.json registry and per-repo meta.json (the store's substrate) are
// owned by internal/core/ahoy and created at install time; this package only
// writes transcript records into an already-bootstrapped transcripts/ dir.
//
// Redaction is NOT reimplemented here. Every transcript is sanitised through
// internal/adapter/scanner — the same detector and masking discipline the
// launch path uses — in a two-stage, fail-closed capture: sanitise on write,
// then re-scan and refuse to write if any hard_fail secret or self home path
// survived. A stored record can never contain a live secret or an absolute home
// path.
package history

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/gitleaks"
	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// recordSchemaVersion is the frontmatter schema stamped into every record.
// Version 2 added the lineage fields (adr-2609090636172016); version 3 added
// adopted_project, which only an ingest adoption sets. Readers admit ALL of
// them, because parsing is by field presence and every added key is optional: a
// schema-1 record carries no lineage keys at all and parses as a main-thread
// record with empty lineage, so everything already stored keeps working and
// stays readable until a migration touches it.
const recordSchemaVersion = 3

// scanGitleaks is the OPT-IN gitleaks augmentation seam (iss-96). The default
// wiring loads the per-repo .abcd/config/gitleaks.json and, ONLY when the repo
// opted in, shells out to gitleaks over the transcript and returns findings to
// fold into redaction; a repo that did not opt in gets (nil, nil) and pays
// nothing — no lookup, no process, no cost. It is a package var so a test can
// inject a fake without spawning a real binary. When a repo opts in but the
// binary is absent, the default wiring returns gitleaks.ErrConfiguredNotFound,
// which Capture surfaces and fails closed on — never a silent skip.
var scanGitleaks = gitleaks.Scan

// Record is one stored transcript's metadata (its frontmatter). It never
// carries raw content — the redacted body is fetched separately via Read.
//
// SessionID means one thing and one thing only: the session the transcript
// belongs to. On a sub-agent record it is the FULL, untruncated id of the
// spawning session, which is what the parent's own record carries too, so a
// reader holding a session id reaches the whole session by matching one field
// (adr-2609090636172016). Lineage is carried by the fields below, never by
// overloading the session id with a composite.
type Record struct {
	SessionID    string    `json:"session_id"`
	RootCommit   string    `json:"root_commit"`
	CapturedAt   time.Time `json:"captured_at"`
	SourceKind   string    `json:"source_kind"`
	SourceSHA256 string    `json:"source_sha256"`
	Path         string    `json:"path"`
	Secrets      int       `json:"redacted_secrets"`
	HomePaths    int       `json:"redacted_home_paths"`

	// Lineage (schema 2). All optional; all empty/zero on a main-thread
	// record, which is exactly how a schema-1 record parses.
	AgentID          string `json:"agent_id,omitempty"`
	ParentAgentID    string `json:"parent_agent_id,omitempty"`
	AgentType        string `json:"agent_type,omitempty"`
	SpawnDepth       int    `json:"spawn_depth,omitempty"`
	SpawnToolUseID   string `json:"spawn_tool_use_id,omitempty"`
	LineageSource    string `json:"lineage_source,omitempty"`
	SpawnAttribution string `json:"spawn_attribution,omitempty"`

	// AdoptedProject (schema 3) names the harness project directory this
	// record was adopted from, on a transcript ingested into a repository that
	// claims that name because the repository the transcript recorded no longer
	// exists on disk. It is empty on everything else, and its presence is what
	// makes an adoption a property of the artefact rather than of a run's
	// output.
	AdoptedProject string `json:"adopted_project,omitempty"`
}

// CaptureMeta is everything Capture stamps onto a record besides the bytes and
// the store key: the session the transcript belongs to, the source kind, and
// the lineage that says which agent produced it.
//
// It replaces the positional parameter list Capture used to take. Six more
// positional strings on a security-critical call is a transposition waiting to
// happen, and a transposed session and agent id would silently mis-attribute a
// transcript rather than fail.
//
// Every string field here is externally supplied — the harness payload, a
// configuration file a contributor can edit — so every one of them passes
// through the same redaction gate as the transcript body. There is no field on
// this record the scanner does not see.
type CaptureMeta struct {
	SessionID string // the session the transcript belongs to (required)
	Kind      string // source_kind: native | specstory-import (default native)

	AgentID        string // the sub-agent's own id; empty on the main thread
	ParentAgentID  string // the agent that spawned it; empty when the main thread did
	AgentType      string // the kind of agent, as the harness reports it
	SpawnDepth     int    // 0 for the main thread, 1 for a sub-agent of it, and so on
	SpawnToolUseID string // the tool call in the spawning transcript that launched it
	LineageSource  string // hook | ingest | migrated — which door the record came through

	// SpawnAttribution is which rung of the attribution ladder placed this
	// agent's spawn point: sidecar | transcript | unattributed. Required on a
	// sub-agent record, empty on the main thread.
	SpawnAttribution string

	// AdoptedProject names the harness project directory an ingested
	// transcript was adopted from. Set only by an adoption; empty everywhere
	// else.
	AdoptedProject string
}

// CaptureResult reports the outcome of one capture.
type CaptureResult struct {
	Record   Record            `json:"record"`
	Wrote    bool              `json:"wrote"`    // false on an idempotent no-op (source unchanged)
	Residual []scanner.Finding `json:"residual"` // populated only alongside RedactionResidualError
	// Superseded names the record this capture replaced, when a longer
	// transcript for the same (session, agent) arrived and the stored one was a
	// byte-prefix of it. Nil whenever nothing was replaced.
	Superseded *Record `json:"superseded,omitempty"`
}

// RedactionResidualError is returned by Capture when the stage-two re-scan finds
// a BLOCKING span that survived redaction — any identity or network span
// whatever its severity, plus every hard_fail one (scanner.BlockingResidual). NO file is
// written. It carries the surviving findings' kinds/locations only — the scanner
// has already masked their Matched fields, so no raw secret material is exposed.
type RedactionResidualError struct {
	Residual []scanner.Finding
}

func (e *RedactionResidualError) Error() string {
	kinds := make([]string, 0, len(e.Residual))
	for _, f := range e.Residual {
		kinds = append(kinds, f.Kind)
	}
	return fmt.Sprintf("history: redaction left %d blocking span(s) unresolved [%s]; refusing to write",
		len(e.Residual), strings.Join(kinds, ", "))
}

// Capture reads a raw session transcript, redacts it through the scanner
// (two-stage, fail-closed), and writes a record into
// ~/.abcd/history/<rootSHA>/transcripts/.
//
// It is idempotent on the source's sha256: an identical source already stored
// is a no-op (Wrote=false, existing record returned, mtime preserved). It is
// fail-closed: if a blocking span survives redaction it returns a
// *RedactionResidualError and writes nothing.
//
// Precondition: the transcripts/ dir must already exist (abcd ahoy install
// created it). Capture re-validates that the store's owned dirs are real
// directories; it never creates the index or meta.
func Capture(repoRoot, rootSHA string, raw []byte, meta CaptureMeta) (CaptureResult, error) {
	// Boundary validation — external inputs.
	if !rootSHARe.MatchString(rootSHA) {
		return CaptureResult{}, errors.New(rootSHAErrMsg)
	}
	if err := meta.validate(); err != nil {
		return CaptureResult{}, err
	}
	sessionID, kind := meta.SessionID, meta.Kind

	tdir, err := ownedDirsReal(rootSHA)
	if err != nil {
		return CaptureResult{}, err
	}

	release, err := repoLock(tdir)
	if err != nil {
		return CaptureResult{}, err
	}
	defer release()

	sum := sha256.Sum256(raw)
	sourceSHA := hex.EncodeToString(sum[:])

	// Idempotency: re-capturing the SAME source for the SAME session, agent and
	// kind is a no-op. Keying on the source SHA alone would silently attribute a
	// second, distinct session that happens to produce byte-identical bytes to the
	// first session's record — the second session would then have no record at all
	// while Capture reports success. The agent id is in the key for exactly the
	// same reason one directory down: two sub-agents of ONE session can produce
	// byte-identical transcripts (two reviewers handed the same file, both
	// answering "no findings"), and without it the second collapses into the
	// first's record and is lost.
	existing, err := listRecords(tdir)
	if err != nil {
		return CaptureResult{}, err
	}
	for _, r := range existing {
		if r.SourceSHA256 == sourceSHA && r.SessionID == sessionID &&
			r.AgentID == meta.AgentID && r.SourceKind == kind {
			return CaptureResult{Record: r, Wrote: false}, nil
		}
	}

	// Stage one — sanitise on write, using the per-repo merged scanner.
	sc, err := scanner.New(repoRoot)
	if err != nil {
		return CaptureResult{}, fmt.Errorf("history: scanner init: %w", err)
	}
	// Fail closed on a degraded scanner. ScanText/Redact cannot signal the
	// unavailable state in-band (only ScanBundle does), so without this guard a
	// broken per-repo pii.json would silently redact with a weakened pattern set
	// and still report the write as clean — the exact fail-open this store forbids.
	if unavail, reason := sc.Unavailable(); unavail {
		return CaptureResult{}, fmt.Errorf("history: refusing to capture with a degraded scanner: %s", reason)
	}
	// The lineage scalars are scanned WITH the body. Every one of them is
	// externally supplied and every one of them lands in frontmatter, which has
	// never been scanned — so writing them straight through would open a
	// redaction bypass beside the redaction gate. They are prepended to the raw
	// text as ordinary lines and split back off after the pass, so they get
	// exactly the same sanitise-then-verify discipline as the body (the same
	// detectors, the same caller-home backstop, the same fail-closed residual
	// refusal) with no second code path to drift from this one.
	text := frameLineage(meta, raw)
	findings := sc.ScanText(text, "transcript")

	// Opt-in deeper coverage (iss-96). Off by default: for a repo that has not
	// armed the gitleaks adapter this returns (nil, nil) and invokes nothing, so
	// the native path below is byte-for-byte what it was. When armed, the adapter's
	// findings AUGMENT the native ones — masked by the same Redact discipline and
	// counted in the same audit buckets. Fail-closed: an armed-but-absent binary
	// returns an error here and refuses the write, mirroring the degraded-scanner
	// guard above rather than silently capturing with less coverage than the repo
	// asked for.
	extra, err := scanGitleaks(repoRoot, text, "transcript")
	if err != nil {
		return CaptureResult{}, fmt.Errorf("history: %w", err)
	}
	findings = append(findings, extra...)

	redacted, _ := scanner.Redact(text, findings)

	// Stage one-and-a-half — deterministic literal home-path backstop, wholly
	// INDEPENDENT of the scanner heuristic (defence in depth on this trust
	// boundary). The scanner's stage-two re-scan below uses the same detector
	// that produced `findings`, so a span that detector's trailing-boundary
	// heuristic dropped would slip through both stages. This literal sweep
	// collapses every remaining occurrence of the resolved $HOME to "~", then
	// fails closed if any absolute path still reveals the caller's own home.
	if home := scanner.CallerHome(); home != "" {
		redacted = scanner.SweepCallerHome(redacted, home)
		var resid []scanner.Finding
		redacted, resid = scanner.SurvivingCallerHome(redacted, home)
		if len(resid) > 0 {
			return CaptureResult{Residual: resid}, &RedactionResidualError{Residual: resid}
		}
	}

	// Stage two — verify. Re-scan the redacted text; a surviving finding that
	// carries a leak blocks the write (fail-closed). The native re-scan cannot
	// see an augmented span — a different detector found it — so every
	// augmented finding is verified by its bytes instead, and a survivor blocks
	// the write the same way: verification is symmetric with detection
	// (GHSA-j7v5-q7x6-v3rp).
	residual := scanner.BlockingResidual(sc.ScanText(redacted, "transcript"))
	residual = append(residual, unsealedAugmented(redacted, extra)...)
	if len(residual) > 0 {
		return CaptureResult{Residual: residual}, &RedactionResidualError{Residual: residual}
	}

	// Split the redacted scalars back off the redacted body. A frame that did not
	// survive intact is refused rather than guessed at: what would land otherwise
	// is a record whose fields and body are silently offset from each other.
	scalars, body, err := unframeLineage(redacted)
	if err != nil {
		return CaptureResult{}, err
	}
	// From here on the meta carries the REDACTED scalars, never the caller's
	// originals: a scalar whose redaction changed it is stored changed.
	meta = applyLineageScalars(meta, scalars)

	// Supersession: the unit of the store is one (session_id, agent_id), not one
	// transcript. A harness fires its stop event on EVERY stop, so an agent
	// resumed with a follow-up message stops again carrying a longer transcript
	// that BEGINS with the one already stored — and a session that ends more than
	// once does the same on the main thread. Those are two snapshots of one run,
	// not two runs. Their source hashes differ, so the idempotency check above
	// cannot see it, and left alone every reader of the set gets the same agent
	// twice.
	//
	// So the longer body wins and the record it grew out of goes, and a shorter
	// re-arrival (an older staged copy drained after the fuller one landed) is a
	// no-op. Nothing is lost either way, because one body is a byte-prefix of the
	// other. Two bodies where NEITHER is a prefix of the other are not snapshots
	// of one run — a recycled agent id, a rewritten transcript — and they are
	// left side by side rather than collapsed, because the store holds the only
	// copy of both.
	//
	// The comparison is on the redacted body, which is what the store actually
	// holds: the raw source is not kept, and redaction is line-local, so a
	// line-aligned prefix of the source is a prefix of the redacted text too.
	// That costs a redaction pass before the no-op is detected, which is the
	// price of not keeping raw bytes around to compare.
	superseded, prior := resolveSupersession(existing, meta, kind, marshalBody(body))
	if prior != nil {
		return CaptureResult{Record: *prior, Wrote: false}, nil
	}

	secrets, homePaths := countBuckets(findings)
	capturedAt := time.Now().UTC()
	name := recordFilename(capturedAt, sessionID, meta.AgentID)
	path := filepath.Join(tdir, name)

	// Refuse a pre-planted symlink at the leaf record path.
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return CaptureResult{}, &StorePathError{Path: path, Msg: "record path is a symlink; refusing"}
	}

	rec := Record{
		SessionID:        sessionID,
		RootCommit:       rootSHA,
		CapturedAt:       capturedAt,
		SourceKind:       kind,
		SourceSHA256:     sourceSHA,
		Path:             path,
		Secrets:          secrets,
		HomePaths:        homePaths,
		AgentID:          meta.AgentID,
		ParentAgentID:    meta.ParentAgentID,
		AgentType:        meta.AgentType,
		SpawnToolUseID:   meta.SpawnToolUseID,
		LineageSource:    meta.LineageSource,
		SpawnAttribution: meta.SpawnAttribution,
		SpawnDepth:       meta.SpawnDepth,
		AdoptedProject:   meta.AdoptedProject,
	}
	if err := fsutil.WriteFileAtomic(path, marshalRecord(rec, body), 0o644); err != nil {
		return CaptureResult{}, fmt.Errorf("history: write record: %w", err)
	}
	// Retire the records this one grew out of, AFTER the replacement is safely on
	// disk — the reverse order would put the only copy of a transcript in the gap
	// between a remove and a failed write. A removal that fails leaves the agent
	// counted twice, which is the whole defect, so it is an error and not a
	// shrug; the record itself is returned alongside it, and a re-drain of the
	// same bytes is a no-op, so the state is recoverable.
	for i := range superseded {
		if err := os.Remove(superseded[i].Path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return CaptureResult{Record: rec, Wrote: true, Superseded: &superseded[i]},
				fmt.Errorf("history: stored %s but could not retire the record it superseded: %w",
					sessionID, err)
		}
	}
	res := CaptureResult{Record: rec, Wrote: true}
	if len(superseded) > 0 {
		// Newest first, so the head is the record this one directly replaced; a
		// tail is a pre-supersession store's leftovers, retired in the same pass.
		res.Superseded = &superseded[0]
	}
	return res, nil
}

// resolveSupersession compares one capture's redacted body against every record
// already stored for the same (session, agent, kind). It returns the records
// this capture supersedes, or — when the store already holds everything this
// capture carries — the stored record the caller should return as a no-op.
//
// An unreadable prior is skipped rather than fatal: a single corrupt record is
// not a reason to refuse a fresh capture, exactly as listRecords already treats
// one.
func resolveSupersession(existing []Record, meta CaptureMeta, kind, body string) ([]Record, *Record) {
	var superseded []Record
	for _, p := range existing {
		if p.SessionID != meta.SessionID || p.AgentID != meta.AgentID || p.SourceKind != kind {
			continue
		}
		data, err := fsutil.ReadGuarded(p.Path, maxTranscriptBytes)
		if err != nil {
			continue
		}
		_, priorBody, err := parseRecord(data)
		if err != nil {
			continue
		}
		switch {
		case strings.HasPrefix(priorBody, body):
			// Equal, or the stored record is already the longer one.
			stored := p
			return nil, &stored
		case strings.HasPrefix(body, priorBody):
			superseded = append(superseded, p)
		}
	}
	return superseded, nil
}

// List returns the records under <rootSHA>/transcripts/, newest first. It reads
// frontmatter only, never bodies. An absent transcripts dir returns no records
// and no error (the store is simply not populated for this repo yet); a
// symlinked owned dir is refused with an error.
func List(rootSHA string) ([]Record, error) {
	if !rootSHARe.MatchString(rootSHA) {
		return nil, errors.New(rootSHAErrMsg)
	}
	tdir, err := transcriptsDir(rootSHA)
	if err != nil {
		return nil, err
	}
	if _, err := os.Lstat(tdir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !fsutil.IsRealDir(tdir) {
		return nil, &StorePathError{Path: tdir, Msg: "transcripts dir is a symlink; refusing"}
	}
	return listRecords(tdir)
}

// Read returns the metadata and full redacted body of one record. It never
// un-redacts; the stored bytes are already sanitised.
//
// The key is resolved in three steps, most specific first:
//
//  1. an exact record filename — one record, named outright;
//  2. an exact agent id — one sub-agent's transcript;
//  3. a session id, preferring the MAIN-THREAD record and newest first. A
//     reader who names a session is asking for its spine, even when a sub-agent
//     of it was captured more recently; ListForSession is how they get the rest.
func Read(rootSHA, key string) (Record, []byte, error) {
	records, err := List(rootSHA)
	if err != nil {
		return Record{}, nil, err
	}
	// List is newest-first, so within each step the first hit is the newest.
	var match *Record
	for _, accept := range []func(Record) bool{
		func(r Record) bool { return filepath.Base(r.Path) == key },
		func(r Record) bool { return r.AgentID != "" && r.AgentID == key },
		func(r Record) bool { return r.SessionID == key && r.AgentID == "" },
		func(r Record) bool { return r.SessionID == key },
	} {
		for i := range records {
			if accept(records[i]) {
				match = &records[i]
				break
			}
		}
		if match != nil {
			break
		}
	}
	if match == nil {
		return Record{}, nil, fmt.Errorf("history: no record for %q under %s", key, rootSHA)
	}
	data, err := fsutil.ReadGuarded(match.Path, maxTranscriptBytes)
	if err != nil {
		return Record{}, nil, err
	}
	rec, body, err := parseRecord(data)
	if err != nil {
		return Record{}, nil, err
	}
	rec.Path = match.Path
	return rec, []byte(body), nil
}

// ListForSession returns every record belonging to one session — the main
// thread and every sub-agent it spawned, at any depth — main thread first and
// otherwise newest first.
//
// This is what a session identifier buys now that lineage is carried by fields:
// a sub-agent record holds the FULL, untruncated id of the spawning session, so
// one field match reaches the whole session. Under the composite identifier it
// could not, which is the defect adr-2609090636172016 records.
//
// The main thread leads because it is what makes the rest legible: the branches
// are only interpretable against the spine that spawned them. An empty result
// is not an error — the session may simply have no records here.
func ListForSession(rootSHA, sessionID string) ([]Record, error) {
	records, err := List(rootSHA)
	if err != nil {
		return nil, err
	}
	var main, subs []Record
	for _, r := range records {
		if r.SessionID != sessionID {
			continue
		}
		if r.AgentID == "" {
			main = append(main, r)
			continue
		}
		subs = append(subs, r)
	}
	return append(main, subs...), nil
}

// lineageFrameEnd terminates the scalar block Capture prepends to the raw
// transcript for the redaction pass. It is a positive frame rather than a bare
// line count because redaction is line-preserving for every rewrite it performs
// EXCEPT a PEM block, which it collapses to a single placeholder line: a count
// alone would silently mis-split a scalar that happened to look like a PEM
// header, and the record would land with its fields offset from its body. The
// marker is deliberately inert text no detector matches, so it survives the
// pass unchanged whenever the frame itself is intact.
const lineageFrameEnd = "abcd-history-lineage-frame-end"

// lineageScalars is the ordered scalar block that goes through redaction with
// the body. SpawnDepth is absent by construction: it is an integer, so there is
// nothing in it for a detector to find and nothing for a redactor to change.
// The order here IS the contract with unframeLineage and with the Record fields
// Capture fills from it.
func lineageScalars(m CaptureMeta) []string {
	return []string{m.AgentID, m.ParentAgentID, m.AgentType, m.SpawnToolUseID,
		m.LineageSource, m.SpawnAttribution, m.AdoptedProject}
}

// applyLineageScalars is lineageScalars' inverse: it writes the block back onto
// a CaptureMeta in the order lineageScalars produced it. Both directions live
// beside each other so a field added to one is a compile error in the other,
// which is what keeps a record's fields from silently sliding one position
// against the values the redaction pass returned.
func applyLineageScalars(m CaptureMeta, s []string) CaptureMeta {
	m.AgentID, m.ParentAgentID, m.AgentType = s[0], s[1], s[2]
	m.SpawnToolUseID, m.LineageSource, m.SpawnAttribution = s[3], s[4], s[5]
	m.AdoptedProject = s[6]
	return m
}

// frameLineage prepends the lineage scalars, one per line, and the frame marker
// to the raw transcript.
func frameLineage(m CaptureMeta, raw []byte) string {
	var b strings.Builder
	for _, s := range lineageScalars(m) {
		b.WriteString(s)
		b.WriteByte('\n')
	}
	b.WriteString(lineageFrameEnd)
	b.WriteByte('\n')
	b.Write(raw)
	return b.String()
}

// unframeLineage splits the redacted scalars back off the redacted body. A
// frame that is not exactly where frameLineage put it is a refusal, not a
// best-effort split.
func unframeLineage(redacted string) (scalars []string, body string, err error) {
	n := len(lineageScalars(CaptureMeta{}))
	parts := strings.SplitN(redacted, "\n", n+2)
	if len(parts) != n+2 || parts[n] != lineageFrameEnd {
		return nil, "", errors.New("history: the lineage frame did not survive redaction; refusing to write")
	}
	return parts[:n], parts[n+1], nil
}

// unsealedAugmented returns, for every augmented finding whose reported bytes
// still occur anywhere in the redacted text, a finding naming its kind and
// declared position with the bytes withheld (the error it feeds lists kinds
// only). Presence anywhere is the right test, not the declared span: the
// adapter locates every occurrence of a value across the whole text and
// secret kinds are sealed length-preservingly, so after Redact no occurrence
// of a located value can legitimately remain, while a span compare would drift
// under the identity placeholders Redact rewrites after the seal (they change
// line lengths) and would miss a finding whose declared position Redact could
// not apply at all — the exact case in which the record would otherwise count
// a redaction it never performed. Re-running gitleaks over the redacted text
// is the other symmetric shape; it doubles a 30 s-timeout subprocess and is not
// deterministic across rule sets, so the bytes the adapter reported are what
// is checked.
//
// Presence-anywhere is only safe because the adapter detects at the same scope:
// it locates every occurrence of every line of a reported value across the
// whole text, so a line that recurs outside the value is sealed rather than
// left as a survivor this check would then refuse the write on for good
// (iss-2609020231145566).
func unsealedAugmented(redacted string, extra []scanner.Finding) []scanner.Finding {
	var out []scanner.Finding
	for _, f := range extra {
		if f.Matched == "" || !strings.Contains(redacted, f.Matched) {
			continue
		}
		out = append(out, scanner.Finding{
			File:     f.File,
			Line:     f.Line,
			Column:   f.Column,
			Kind:     f.Kind,
			Severity: f.Severity,
		})
	}
	return out
}

// countBuckets rolls the redacted findings into the two audit counters stamped
// into the record frontmatter: home paths (self + third-party) and everything
// else (secret tokens plus real-name/email/username identity spans).
func countBuckets(findings []scanner.Finding) (secrets, homePaths int) {
	for _, f := range findings {
		switch f.Kind {
		case "home_path_self", "home_path_other":
			homePaths++
		default:
			secrets++
		}
	}
	return secrets, homePaths
}
