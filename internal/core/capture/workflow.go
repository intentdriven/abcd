package capture

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/provenance"
	"github.com/intentdriven/abcd/internal/core/relink"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// mutationPreamble runs the idempotent pre-mutation steps: sweep orphan
// placeholders and (re-)assert the symlink-refused directory shape. Read-only
// entry points (List, Status) deliberately skip it.
//
// The orphan sweep runs UNDER the ledger lock (iss-102): a >60s-stalled capture
// fills its reserved placeholder via commitCapture, which now also holds the
// ledger lock, so the sweep's classify-then-unlink can no longer interleave with
// a commit's fill and delete a just-committed issue file.
func mutationPreamble(repoRoot, issuesRoot string) error {
	if err := withLedgerLock(repoRoot, issuesRoot, func() error {
		return cleanOrphanPlaceholders(issuesRoot)
	}); err != nil {
		return err
	}
	return ensureLedgerDirs(repoRoot, issuesRoot)
}

// Capture appends a new issue to open/ with an auto-assigned (or forced) iss-N.
// The write is transactional: a zero-byte placeholder is reserved, and on any
// failure it is swept. Returns the committed path under open/.
func Capture(req CaptureRequest) (CaptureResult, error) {
	repoRoot, issuesRoot, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return CaptureResult{}, err
	}
	// A found_at that names a path must name one in THIS checkout
	// (iss-2609120511058115). Checked before the preamble, so a refused capture
	// writes nothing at all — not even the ledger directories.
	if err := checkFoundAt(repoRoot, req.FoundAt); err != nil {
		return CaptureResult{}, err
	}
	if err := mutationPreamble(repoRoot, issuesRoot); err != nil {
		return CaptureResult{}, err
	}
	// The targets go through the ONE blocked_by validator link shares
	// (link.go): shape, no self-edge, existence in any status folder, duplicates
	// collapsed. The subject is the migrator's ForceID when there is one, since
	// a minted id cannot be named before it exists.
	if req.BlockedBy, err = validateBlockers(issuesRoot, "capture", req.ForceID, req.BlockedBy); err != nil {
		return CaptureResult{}, err
	}

	if strings.TrimSpace(req.FoundDuring) == "" {
		return CaptureResult{}, fmt.Errorf("found_during must be a non-empty string")
	}

	// Redact the free-text INPUTS, before the slug is normalised and before the
	// record is rendered. Redacting the rendered file instead treats the
	// structural fields as free text: a home path in the body reaches the
	// caller-derived slug, redaction rewrites it to a bracketed placeholder, and
	// the kebab-case check then REFUSES the capture — turning a leak into a lost
	// finding, which is the outcome this whole path exists to avoid. Normalising
	// after redaction strips the placeholder's brackets instead.
	//
	// Only free text is touched. id, severity, category and source are generated
	// or enum-constrained, so they carry nothing to redact and must not be
	// rewritten.
	var redacted int
	var degraded string
	req.Text, req.Slug, req.FoundAt, req.FoundDuring, redacted, degraded =
		redactCaptureInputs(repoRoot, req.Text, req.Slug, req.FoundAt, req.FoundDuring)

	// When no explicit slug was supplied, derive it HERE — from the text that
	// redaction has already rewritten — never from the raw caller text upstream. A
	// caller that kebab-cased the raw text first would smuggle a home path's
	// username past the redactor (which no longer sees a path shape) and into the
	// committed filename (gh-485). Deriving after redaction is what keeps a finding
	// out of the filename; an explicit slug still flows through redactCaptureInputs
	// above.
	if strings.TrimSpace(req.Slug) == "" {
		req.Slug = deriveSlug(req.Text)
	}

	slugNorm, err := normaliseSlug(req.Slug)
	if err != nil {
		return CaptureResult{}, err
	}

	// The mint is timestamp-numeric (adr-45; mechanics per spc-33): it consults
	// no maximum, so the refs-union scan the max+1 allocator needed (iss-115,
	// iss-120) is gone — time orders the ids and entropy separates same-second
	// minters on branches this working tree cannot see.
	issID, placeholder, err := reservePath(repoRoot, issuesRoot, slugNorm, req.ForceID)
	if err != nil {
		return CaptureResult{}, err
	}

	result, err := commitCapture(repoRoot, issuesRoot, req, issID, slugNorm, placeholder)
	if err != nil {
		_ = cancelReservation(placeholder)
		return CaptureResult{}, err
	}
	result.Redacted, result.Degraded = redacted, degraded
	// Machine output carries a repo-relative locator, never an absolute
	// developer-identity path (iss-81).
	result.Path = fsutil.RepoRel(repoRoot, result.Path)
	return result, nil
}

func commitCapture(repoRoot, issuesRoot string, req CaptureRequest, issID, slug, placeholder string) (CaptureResult, error) {
	// The disclosure pair (itd-178). origin is DERIVED — a capture's text is
	// written directly rather than derived from another record or a reading
	// item, so its route is researcher-authored (which names the route, not who
	// ran the command) and no request member carries it — while the production
	// mode is the closed choice the caller declared, defaulted here so a
	// captured record always carries both keys.
	stamp, err := provenance.NewStamp(provenance.KindResearcherAuthored, req.ProductionMode)
	if err != nil {
		return CaptureResult{}, fmt.Errorf("capture: %w", err)
	}
	fields := []kv{
		{"schema_version", 1},
		{"id", issID},
		{"slug", slug},
		{"severity", string(req.Severity)},
		{"category", string(req.Category)},
		{"source", string(req.Source)},
		{"found_during", req.FoundDuring},
	}
	fm := map[string]any{
		"schema_version": 1,
		"id":             issID,
		"slug":           slug,
		"severity":       string(req.Severity),
		"category":       string(req.Category),
		"source":         string(req.Source),
		"found_during":   req.FoundDuring,
	}
	// Written bare, like every other machine-read scalar: a quoted value reads as
	// a different string to the line scanner the gate compares against.
	fields = append(fields,
		kv{provenance.KeyOrigin, rawScalar(stamp.OriginValue())},
		kv{provenance.KeyProductionMode, rawScalar(stamp.ModeValue())})
	fm[provenance.KeyOrigin] = stamp.OriginValue()
	fm[provenance.KeyProductionMode] = stamp.ModeValue()
	if req.FoundAt != "" {
		fields = append(fields, kv{"found_at", req.FoundAt})
		fm["found_at"] = req.FoundAt
	}
	// lapsed_at is never defaulted: the value is the instant the discipline gave
	// way, and inventing one at write-up would be the reconstruction the lapse log
	// exists to measure (spc-60). It IS trimmed, and the trim is what keeps the two
	// gates agreeing: the reader trims before judging, so an all-whitespace value
	// reads as absent and passes, while the committed-ledger gate reads the same
	// bytes as a present value that is no RFC 3339 instant and blocks. Writing the
	// padding would therefore commit a record capture goes on reading while its own
	// record_schema blocker says it refuses and skips it. Nothing else is
	// normalised — a value that survives the trim is written exactly as given.
	if lapsedAt := strings.TrimSpace(req.LapsedAt); lapsedAt != "" {
		fields = append(fields, kv{"lapsed_at", lapsedAt})
		fm["lapsed_at"] = lapsedAt
	}
	if req.RelatedIntents != nil {
		fields = append(fields, kv{"related_intents", req.RelatedIntents})
		fm["related_intents"] = req.RelatedIntents
	}
	if req.RelatedSpecs != nil {
		fields = append(fields, kv{"related_specs", req.RelatedSpecs})
		fm["related_specs"] = req.RelatedSpecs
	}
	if req.BlockedBy != nil {
		fields = append(fields, kv{"blocked_by", req.BlockedBy})
		fm["blocked_by"] = req.BlockedBy
	}

	if err := validateStrict(fm); err != nil {
		return CaptureResult{}, err
	}
	if err := validateInvariants(fm, StateOpen, placeholder); err != nil {
		return CaptureResult{}, err
	}

	content, err := buildIssueText(fields, req.Text)
	if err != nil {
		return CaptureResult{}, err
	}

	// The checksum re-read + fill runs UNDER the ledger lock (iss-102): the orphan
	// sweep (mutationPreamble) also holds this lock, so a >60s-stalled commit can no
	// longer land its fill in the sweep's classify-then-unlink window and have the
	// just-committed file deleted. If the sweep reclaimed the placeholder first, the
	// re-read fails and the capture reports an error rather than a false success.
	var result CaptureResult
	err = withLedgerLock(repoRoot, issuesRoot, func() error {
		// Guard the overwrite: the placeholder must still be the zero-byte file we
		// reserved (expected_checksum = sha256("")).
		_, checksum, rerr := readWithChecksum(placeholder)
		if rerr != nil {
			return rerr
		}
		if checksum != emptyChecksum {
			return fmt.Errorf("%w: placeholder %s changed since reservation", ErrChecksumMismatch, placeholder)
		}
		if werr := fsutil.WriteFileAtomicPreserveMode(placeholder, []byte(content)); werr != nil {
			return werr
		}
		result = CaptureResult{ID: issID, Slug: slug, Path: placeholder, Status: StateOpen}
		return nil
	})
	if err != nil {
		return CaptureResult{}, err
	}
	return result, nil
}

// Resolve moves an open issue to resolved/, writing the resolution note and the
// product impact. resolved/ is gated by issue_impact_valid, so the impact is
// validated against the shared changelog enum up front (empty or invalid is
// refused, never defaulted) and stamped bare alongside the note — the tool's own
// resolve path can never mint a record its own blocker rejects.
//
// The optional ByIntent/BySpec/ByCommit members land as the structured
// resolved_by object (spc-25) in the same atomic transition. Ids are validated
// for existence in their record store before anything is written; the sha is
// shape-checked only (its home may be the remote, and shallow or rebased
// states must not refuse a legitimate resolution). No members → no
// resolved_by key at all: provenance is optional, never defaulted.
func Resolve(req ResolveRequest) (TransitionResult, error) {
	impact, err := changelog.ParseImpact(req.Impact)
	if err != nil {
		return TransitionResult{}, fmt.Errorf("resolve: %w", err)
	}
	rb, err := resolveProvenance(req)
	if err != nil {
		return TransitionResult{}, err
	}
	// Validated BEFORE the move, like the impact: a resolution is a conjecture
	// being closed, and recording the route without the reasoning is the thing
	// itd-179 exists to stop. resolveRoots is called for the redactor's repo root
	// alone — the transition below resolves them again for the move itself.
	rr, _, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return TransitionResult{}, err
	}
	g, gRedacted, gDegraded, err := optionalGrounds(rr, "resolve", req.Grounds)
	if err != nil {
		return TransitionResult{}, err
	}
	// Validated BEFORE the move, like every other member: the field's job is to
	// take a record OUT of a release, so a value the derivation cannot read is
	// worse than no value at all — it would sit in the ledger looking like an
	// exclusion and never be one.
	if req.ShippedIn != "" && !reShippedIn.MatchString(req.ShippedIn) {
		return TransitionResult{}, fmt.Errorf(
			"resolve: --shipped-in %q is not a release tag (want vMAJOR.MINOR.PATCH); nothing written", req.ShippedIn)
	}
	if err := validateRestampMode(req.ProductionMode); err != nil {
		return TransitionResult{}, fmt.Errorf("resolve: %w", err)
	}
	extras := []kv{{"impact", rawScalar(string(impact))}}
	if req.ShippedIn != "" {
		// rawScalar, like impact above: a plain string is double-quoted by
		// yamlScalar, and the derivation reads the RAW scalar, so a quoted value
		// arrives as `"v0.6.2"` and fails its own shape check. The first draft of
		// this feature wrote it quoted and read it bare, so the field never
		// excluded anything even once the property was allowed.
		extras = append(extras, kv{"shipped_in", rawScalar(req.ShippedIn)})
	}
	if rb != nil {
		var members nested
		if rb.Intent != "" {
			members = append(members, kv{"intent", rb.Intent})
		}
		if rb.Spec != "" {
			members = append(members, kv{"spec", rb.Spec})
		}
		if rb.Commit != "" {
			members = append(members, kv{"commit", rb.Commit})
		}
		extras = append(extras, kv{"resolved_by", members})
	}
	res, err := transition(req.RepoRoot, req.IssuesRoot, req.ID, "resolve", "resolution", req.Resolution,
		extras, g, req.ProductionMode, StateResolved)
	if err != nil {
		return TransitionResult{}, err
	}
	res.ResolvedBy = rb
	res.Redacted += gRedacted
	if res.Degraded == "" {
		res.Degraded = gDegraded
	}
	return res, nil
}

// resolveProvenance validates the resolved_by members before any write: id
// shape and store existence for --intent/--spec (via the shared findRecordFile
// probe promote's link mode also uses), shape only for --commit. Returns nil
// when no member was supplied.
func resolveProvenance(req ResolveRequest) (*ResolvedBy, error) {
	if req.ByIntent == "" && req.BySpec == "" && req.ByCommit == "" {
		return nil, nil
	}
	repoRoot, _, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return nil, err
	}
	if req.ByIntent != "" {
		if !reItdID.MatchString(req.ByIntent) {
			return nil, fmt.Errorf("resolve: --intent %q does not match ^itd-[0-9]+$; nothing written", req.ByIntent)
		}
		if _, ok := findRecordFile(repoRoot, intentStoreRelDirs(), req.ByIntent); !ok {
			return nil, fmt.Errorf("resolve: --intent %s not found in the intent store; nothing written", req.ByIntent)
		}
	}
	if req.BySpec != "" {
		if !reSpcID.MatchString(req.BySpec) {
			return nil, fmt.Errorf("resolve: --spec %q does not match ^spc-[0-9]+$; nothing written", req.BySpec)
		}
		if _, ok := findRecordFile(repoRoot, specStoreRelDirs(), req.BySpec); !ok {
			return nil, fmt.Errorf("resolve: --spec %s not found in the spec store; nothing written", req.BySpec)
		}
	}
	if req.ByCommit != "" && !reCommitSha.MatchString(req.ByCommit) {
		return nil, fmt.Errorf("resolve: --commit %q is not a 7-64 char lowercase hex sha; nothing written", req.ByCommit)
	}
	return &ResolvedBy{Intent: req.ByIntent, Spec: req.BySpec, Commit: req.ByCommit}, nil
}

// reShippedIn is the release-tag shape the derivation accepts. It is duplicated
// from internal/core/changelog deliberately: capture must refuse a value that
// package would silently ignore, and importing it here for one regexp would
// couple the ledger writer to the release deriver. If a third site needs it, it
// wants exporting rather than a third copy.
//
// It sits below Resolve rather than above it: an earlier draft put it between
// Resolve's doc comment and its signature, which silently transferred twelve
// lines of documentation from the function to the regexp.
var reShippedIn = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// Wontfix moves an open issue to wontfix/, writing the wontfix_reason note and
// the `declined:` grounds derived from it. wontfix/ carries no impact
// (issue_impact_valid gates resolved/ only), so no judgement is stamped.
//
// It needs no new required flag: transition already refuses an empty reason, so
// a wontfix could never be recorded without grounds — what it lacked was the
// TYPE. Grounds overrides the text when the conjecture is worth stating
// separately from the user-facing reason.
func Wontfix(req WontfixRequest) (TransitionResult, error) {
	rr, _, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return TransitionResult{}, err
	}
	g, gRedacted, gDegraded, err := wontfixGrounds(rr, req.Grounds, req.Reason)
	if err != nil {
		return TransitionResult{}, err
	}
	if err := validateRestampMode(req.ProductionMode); err != nil {
		return TransitionResult{}, fmt.Errorf("wontfix: %w", err)
	}
	res, err := transition(req.RepoRoot, req.IssuesRoot, req.ID, "wontfix", "wontfix_reason", req.Reason,
		nil, &g, req.ProductionMode, StateWontfix)
	if err != nil {
		return TransitionResult{}, err
	}
	res.Redacted += gRedacted
	if res.Degraded == "" {
		res.Degraded = gDegraded
	}
	return res, nil
}

// validateRestampMode checks a declared restamp against the closed vocabulary
// before the ledger is touched at all, so a typo costs no lock and no read. An
// undeclared mode is not a value to validate: it means "leave the record's
// existing stamp alone".
//
// Whether the record can BE restamped is a separate question, answered inside
// the transition against the record's own bytes (see restampField).
func validateRestampMode(mode string) error {
	if mode == "" {
		return nil
	}
	_, err := provenance.ParseMode(mode)
	return err
}

// restampField returns the production_mode entry a transition writes, or nothing
// when none was declared. It is the ONE place the restamp rule lives, so the two
// transitions cannot come to differ about it.
//
// It REFUSES a restamp of a record carrying no origin. Every record committed
// before the disclosure keys existed is in that state, and forward-only
// population means none of them is backfilled — so appending a lone
// production_mode there produces exactly the shape record_provenance reports as
// "a state no command produced", against a record the command had just written.
// The pair is written together or not at all.
//
// An undeclared mode writes nothing whatever the record carries: an unstamped
// record must stay resolvable, because the refusal is about the restamp and never
// about the transition.
func restampField(fm map[string]any, issID, mode string) ([]kv, error) {
	if mode == "" {
		return nil, nil
	}
	if asString(fm[provenance.KeyOrigin]) == "" {
		return nil, fmt.Errorf(
			"%s carries no %s, so it predates disclosure and there is nothing to restamp: the pair is written together or not at all, and a lone %s is a state no command produces (nothing written — re-run without --production-mode)",
			issID, provenance.KeyOrigin, provenance.KeyProductionMode)
	}
	m, err := provenance.ParseMode(mode)
	if err != nil {
		return nil, err
	}
	return []kv{{provenance.KeyProductionMode, rawScalar(string(m))}}, nil
}

// transition moves an open issue to target, setting the defining note field and
// any extra frontmatter fields (e.g. resolved/'s impact) in one atomic write.
//
// verb is the command the caller is running, and it exists because the grounds
// append needs it. Passing the target STATE instead made one command emit two
// prefixes — `resolve: grounds refused` from the flag check and
// `resolved: grounds refused` from the append — which reads as two different
// commands failing (iss-2608301803425790).
//
// productionMode, when non-empty, restamps the record's production_mode in the
// same write — and is refused against a record that carries no origin, before
// anything is written (restampField).
func transition(repoRoot, issuesRoot, issID, verb, field, note string, extra []kv, g *grounds.Grounds, productionMode string, target State) (TransitionResult, error) {
	rr, ir, err := resolveRoots(repoRoot, issuesRoot)
	if err != nil {
		return TransitionResult{}, err
	}
	if err := mutationPreamble(rr, ir); err != nil {
		return TransitionResult{}, err
	}
	if !reIssID.MatchString(issID) {
		return TransitionResult{}, fmt.Errorf("invalid iss-N identifier: %q", issID)
	}
	if strings.TrimSpace(note) == "" {
		return TransitionResult{}, fmt.Errorf("%s must be a non-empty string", field)
	}

	// The find→read→move critical section runs under the ledger lock, the SAME
	// lock id allocation takes, so two concurrent conflicting transitions (a
	// resolve and a wontfix on one issue) serialize: the second sees the issue
	// already moved out of open/ and conflicts, instead of both passing the
	// checksum re-read and landing the issue in two status dirs (split-brain).
	var result TransitionResult
	err = withLedgerLock(rr, ir, func() error {
		src, status, err := findIssue(ir, issID)
		if err != nil {
			return err
		}
		if status != StateOpen {
			return fmt.Errorf("%w: %s already in %s", ErrTransitionConflict, issID, status)
		}

		content, checksum, err := readWithChecksum(src)
		if err != nil {
			return err
		}
		// The restamp is judged against the record's OWN bytes, read here, and
		// refused before any of the writes below are composed.
		currentFM, _, err := parseFrontmatterAndBody(content)
		if err != nil {
			return err
		}
		restamp, err := restampField(currentFM, issID, productionMode)
		if err != nil {
			return err
		}
		// The note is the only free text a transition adds, so it is what gets
		// redacted — before it is written into the record, never after, so no
		// rewritten span can reach a field the validator has already passed.
		redNote, redacted, degraded := redactLedgerText(rr, note)
		newContent, err := setScalarField(content, field, redNote)
		if err != nil {
			return err
		}
		for _, f := range append(append([]kv{}, extra...), restamp...) {
			if members, isNested := f.val.(nested); isNested {
				newContent, err = setMapField(newContent, f.key, members)
			} else {
				newContent, err = setScalarField(newContent, f.key, f.val)
			}
			if err != nil {
				return err
			}
		}
		// The grounds entry is APPENDED to the body, not set in frontmatter, and
		// it rides the same atomic write as the fields above. A record promoted
		// before it was resolved carries both conjectures afterwards
		// (iss-2608301657354776).
		if g != nil {
			newContent, err = appendGrounds(verb, newContent, *g)
			if err != nil {
				return err
			}
		}

		dst := filepath.Join(ir, statusDirName[target], filepath.Base(src))
		fm, _, err := parseFrontmatterAndBody(newContent)
		if err != nil {
			return err
		}
		if err := validateStrict(fm); err != nil {
			return err
		}
		if err := validateInvariants(fm, target, dst); err != nil {
			return err
		}

		if err := commitTransition(src, dst, newContent, checksum); err != nil {
			return err
		}
		result = TransitionResult{ID: issID, Path: dst, FromStatus: StateOpen, ToStatus: target,
			Redacted: redacted, Degraded: degraded}
		// Repoint every link that named the issue in open/, still under the
		// ledger lock because the links it rewrites include other issues'. A
		// failure is reported, not raised: the issue has moved.
		result.Relinked, result.RelinkError = repointMovedIssue(rr, src, dst)
		return nil
	})
	if err != nil {
		return TransitionResult{}, err
	}
	// Machine output carries a repo-relative locator, never an absolute
	// developer-identity path (iss-81).
	result.Path = fsutil.RepoRel(rr, result.Path)
	return result, nil
}

// repointMovedIssue repoints the links that named an issue's path before a
// transition moved it, through the one primitive every record-moving verb
// shares. A ledger outside the repository (a custom issues root) is linked from
// nowhere the repository's links can reach, so there is nothing to repoint.
func repointMovedIssue(repoRoot, src, dst string) ([]relink.Rewrite, string) {
	from, err1 := filepath.Rel(repoRoot, src)
	to, err2 := filepath.Rel(repoRoot, dst)
	if err1 != nil || err2 != nil || !filepath.IsLocal(from) || !filepath.IsLocal(to) {
		return nil, ""
	}
	rw, err := relink.Repoint(repoRoot, []relink.Move{{From: from, To: to, MovedNow: true}})
	if err != nil {
		return rw, err.Error()
	}
	return rw, ""
}

// removeSourceHook, when non-nil, replaces os.Remove(src) inside
// commitTransition. It is a test-only seam (nil in production, zero overhead)
// used to force a deterministic non-ENOENT remove failure without relying on
// platform-specific filesystem tricks (immutable attributes, read-only
// remounts) that a portable test cannot set up.
var removeSourceHook func(path string) error

// commitTransition re-verifies the source's checksum, writes the destination
// atomically, then removes the source. A concurrent edit surfaces as a
// checksum mismatch or transition conflict rather than a lost update.
//
// A non-ENOENT remove failure (iss-186: EPERM/EROFS/EIO on the source status
// dir) rolls the destination back rather than leaving the issue split across
// both directories — findIssue rejects any id present in more than one status
// dir, so a stranded copy could never again be transitioned without manual
// repair. Rolling back restores the pre-call state (src present, dst absent)
// so the caller can simply retry once the underlying failure clears.
func commitTransition(src, dst, newContent, expected string) error {
	_, current, err := readWithChecksum(src)
	if os.IsNotExist(err) {
		return fmt.Errorf("%w: %s move source missing", ErrTransitionConflict, src)
	}
	if err != nil {
		return err
	}
	if current != expected {
		return fmt.Errorf("%w: %s changed since it was read", ErrChecksumMismatch, src)
	}
	if err := fsutil.WriteFileAtomicPreserveMode(dst, []byte(newContent)); err != nil {
		return err
	}
	removeSrc := os.Remove
	if removeSourceHook != nil {
		removeSrc = removeSourceHook
	}
	if err := removeSrc(src); err != nil && !os.IsNotExist(err) {
		if rbErr := os.Remove(dst); rbErr != nil && !os.IsNotExist(rbErr) {
			return fmt.Errorf("%w (rollback of %s also failed: %v)", err, dst, rbErr)
		}
		return err
	}
	return nil
}

// List scans one state (or all) and returns issues sorted ascending by numeric
// N plus a roster of unparseable files. Read-only: no preamble, no dir
// creation, tolerant of a virgin/absent ledger.
func List(req ListRequest) (ListResult, error) {
	repoRoot, ir, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return ListResult{}, err
	}
	if err := checkOneStatusPerID(repoRoot, ir); err != nil {
		return ListResult{}, err
	}
	state := req.State
	if state == "" {
		state = StateAll
	}
	if state != StateAll && state != StateOpen && state != StateResolved && state != StateWontfix {
		return ListResult{}, fmt.Errorf("state must be all/open/resolved/wontfix, got %q", state)
	}
	issues, skipped := scanLedger(ir, state)
	sortIssues(issues)
	prioritise(issues, openIDSet(ir))
	relativiseLedgerPaths(repoRoot, issues, skipped)
	// A --json collection is an empty list, never bare null: a consumer that
	// iterates the rows (the capture.md contract) errors on null.
	if issues == nil {
		issues = []Issue{}
	}
	if skipped == nil {
		skipped = []SkipRecord{}
	}
	return ListResult{Issues: issues, Skipped: skipped}, nil
}

// relativiseLedgerPaths rewrites every ledger Path in place to a repo-relative
// locator, so no --json envelope echoes an absolute developer-identity path
// (iss-81). It covers both the structured Path field and a SkipRecord.Error
// string that wraps a guarded-read/parse failure carrying the same absolute path:
// the skipped list rides an exit-0 success envelope, which the CLI's error-path
// scrubber never sees, so the abs path is stripped here for the --json surface.
func relativiseLedgerPaths(repoRoot string, issues []Issue, skipped []SkipRecord) {
	prefix := repoRoot + string(filepath.Separator)
	for i := range issues {
		issues[i].Path = fsutil.RepoRel(repoRoot, issues[i].Path)
	}
	for i := range skipped {
		skipped[i].Path = fsutil.RepoRel(repoRoot, skipped[i].Path)
		skipped[i].Error = strings.ReplaceAll(skipped[i].Error, prefix, "")
	}
}

// Status is a pure read: counts per status dir plus up to 10 most-recent open
// issues (newest first). Guaranteed no mutation.
func Status(req StatusRequest) (StatusResult, error) {
	repoRoot, ir, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return StatusResult{}, err
	}
	if err := checkOneStatusPerID(repoRoot, ir); err != nil {
		return StatusResult{}, err
	}
	var res StatusResult
	open, skOpen := scanLedger(ir, StateOpen)
	resolved, skRes := scanLedger(ir, StateResolved)
	wontfix, skWf := scanLedger(ir, StateWontfix)
	res.OpenCount = len(open)
	res.ResolvedCount = len(resolved)
	res.WontfixCount = len(wontfix)
	res.Skipped = append(append(append([]SkipRecord{}, skOpen...), skRes...), skWf...)

	// The same predicate List uses, over the scan already in hand: skOpen carries
	// the records open/ holds and the reader refused, and they block too.
	openIDs := openBlockingIDs(open, skOpen)
	// Newest first: higher N is newer (ids are monotonic with creation).
	sort.SliceStable(open, func(i, j int) bool { return issNumber(open[i].ID) > issNumber(open[j].ID) })
	if len(open) > 10 {
		open = open[:10]
	}
	// Derived-priority view over the recent slice: unblocked first, then severity.
	prioritise(open, openIDs)
	if open == nil {
		open = []Issue{}
	}
	res.RecentOpen = open
	relativiseLedgerPaths(repoRoot, res.RecentOpen, res.Skipped)
	return res, nil
}

// severityRank orders severities for the derived-priority view: higher rank
// sorts earlier (critical is most urgent, nitpick least).
var severityRank = map[Severity]int{
	SeverityCritical: 4, SeverityMajor: 3, SeverityMinor: 2, SeverityNitpick: 1,
}

// prioritise applies the read-time priority projection in place: it fills each
// issue's BlockedByOpen with the blocked_by targets still in open/ (openIDs),
// then stably orders unblocked issues ahead of blocked ones and, within each
// group, higher severity first. There is no stored priority — this is a derived
// view so the CLI and any future front door share one ordering. The caller is
// expected to have pre-sorted issues into a deterministic tiebreak order.
func prioritise(issues []Issue, openIDs map[string]bool) {
	for i := range issues {
		var stillOpen []string
		for _, dep := range issues[i].BlockedBy {
			if openIDs[dep] {
				stillOpen = append(stillOpen, dep)
			}
		}
		issues[i].BlockedByOpen = stillOpen
	}
	sort.SliceStable(issues, func(i, j int) bool {
		bi, bj := len(issues[i].BlockedByOpen) > 0, len(issues[j].BlockedByOpen) > 0
		if bi != bj {
			return !bi // unblocked (false) sorts first
		}
		return severityRank[issues[i].Severity] > severityRank[issues[j].Severity]
	})
}

// openIDSet returns the set of ids currently in open/ — the predicate a
// blocked_by target must satisfy to still count as blocking. Read-only.
func openIDSet(issuesRoot string) map[string]bool {
	return openBlockingIDs(scanLedger(issuesRoot, StateOpen))
}

// openBlockingIDs is that predicate over ONE scan of open/: the ids that listed,
// PLUS the ids of the records the scan had to skip. A record the guarded reader
// refused — a FIFO, a body over the read cap, a symlinked leaf — is still in
// open/, and being unreadable says nothing about whether it was resolved.
// Counting only what parsed rendered every dependent unblocked and sorted it to
// the top of the board while its own blocked_by went on naming the record. The
// unreadable case resolves toward still-blocking: a board that understates
// progress is recoverable, one that invites work whose blocker nobody can read
// is not — and the skip is reported alongside, so the cause is never silent.
func openBlockingIDs(open []Issue, skipped []SkipRecord) map[string]bool {
	set := idSet(open)
	for _, sk := range skipped {
		if id := skippedRecordID(sk.Path); id != "" {
			set[id] = true
		}
	}
	return set
}

// skippedRecordID recovers the id a skipped file's NAME claims, through
// issFileNumRe — the one detection grammar the scan, the resolver and
// record-lint share, so a file the scan counted as a record contributes the id
// that same grammar reads. A name too malformed to carry an ordinal claims no
// id and contributes none: there is nothing to be blocked by.
func skippedRecordID(path string) string {
	m := issFileNumRe.FindStringSubmatch(filepath.Base(path))
	if m == nil {
		return ""
	}
	return issFamily + "-" + m[1]
}

// idSet collects the ids of a slice of issues into a set.
func idSet(issues []Issue) map[string]bool {
	set := make(map[string]bool, len(issues))
	for _, iss := range issues {
		set[iss.ID] = true
	}
	return set
}

// scanLedger reads issues from the requested state(s). Stray/non-matching .md
// files are silently ignored; corrupt matching files go into Skipped.
func scanLedger(issuesRoot string, state State) ([]Issue, []SkipRecord) {
	var targets []State
	if state == StateAll {
		targets = statusDirs
	} else {
		targets = []State{state}
	}
	var issues []Issue
	var skipped []SkipRecord
	for _, sub := range targets {
		dir := filepath.Join(issuesRoot, statusDirName[sub])
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // virgin/absent ledger tolerance
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		sort.Strings(names)
		for _, name := range names {
			path := filepath.Join(dir, name)
			wellFormed := issFileNumRe.MatchString(name)
			if !wellFormed {
				// A file that claims to be a record (family prefix + ordinal) and is
				// not well-formed is REPORTED, not dropped: it sits in the ledger,
				// counted by nothing and reported by nothing, which is how a record
				// gets silently lost. A file claiming nothing — README.md, a stray
				// note, the allocator lock — is silently ignored, as before.
				//
				// Detection uses recordid.FilenameNumRe, the SAME grammar the resolver
				// and record-lint's per-store rule read, so a filename the gate and
				// the resolver treat as a record reaches the reader too rather than
				// being dropped by a stricter local grammar (iss-2608280739112123).
				// The stricter slug shape is still enforced — as a filename<->
				// frontmatter agreement, below in validateInvariants — but that is a
				// judgement on a record, not the question of whether one exists.
				if filepath.Ext(name) == ".md" && reIssNameClaim.MatchString(name) {
					skipped = append(skipped, SkipRecord{Path: path, Error: fmt.Errorf(
						"%w: filename %q is not a well-formed record name (iss-N[-slug].md)",
						ErrInvariantViolation, name).Error()})
				}
				continue
			}
			// A well-formed name always ends .md — the grammar's pattern requires
			// it — so no separate extension check is needed on this path.
			//
			// The read is guarded, not bare: a well-formed NAME says nothing about
			// the leaf behind it, and in a hostile clone that leaf is a FIFO that
			// would hang this scan, a symlink to a file outside the ledger, or a
			// body sized to make the read unbounded. Each is a skipped record the
			// surfaces already render, never a hang and never serialized.
			content, err := readRecordGuarded(path)
			if err != nil {
				skipped = append(skipped, SkipRecord{Path: path, Error: err.Error()})
				continue
			}
			fm, body, err := parseFrontmatterAndBody(content)
			if err != nil {
				skipped = append(skipped, SkipRecord{Path: path, Error: err.Error()})
				continue
			}
			if err := validateStrict(fm); err != nil {
				skipped = append(skipped, SkipRecord{Path: path, Error: err.Error()})
				continue
			}
			if err := validateInvariants(fm, sub, path); err != nil {
				skipped = append(skipped, SkipRecord{Path: path, Error: err.Error()})
				continue
			}
			issues = append(issues, issueFromFrontmatter(fm, sub, path, body))
		}
	}
	return issues, skipped
}

// sortIssues orders issues ascending by numeric N.
func sortIssues(issues []Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		return issNumber(issues[i].ID) < issNumber(issues[j].ID)
	})
}

// issNumber extracts N from an iss-N reference; -1 for non-matching input.
func issNumber(s string) int {
	m := reSortIssID.FindStringSubmatch(s)
	if m == nil {
		return -1
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return -1
	}
	return n
}

// checkOneStatusPerID refuses a ledger in which one id is claimed by more than
// one record file (iss-2609100507430423).
//
// WHY IT IS A REFUSAL rather than a row. This store's status model rests on ONE
// fact: the directory a record sits in IS its lifecycle state (adr-3). The one
// state that model cannot represent is a record in two directories at once —
// such an id is open and resolved simultaneously, which is not a status the
// ledger has a word for. Rendering it would mean either printing the id twice
// with two contradictory statuses or picking one arbitrarily, and both answer a
// question that has no answer. So every read of the ledger refuses until the
// duplicate is gone, and the refusal names both files, which is what the fix
// needs: one of them moves or goes.
//
// HOW IT HAPPENS is not hypothetical, and it is not a hand-edit. Landing a batch
// of worker branches produced it: two records were committed to the default
// branch, in open/, AFTER the branches had been cut, and those branches then
// resolved the same issues. Git's rename detection saw an add on one side and a
// delete-plus-add at a different path on the other, paired neither, and the
// integration branch carried both copies. It was found by reading open/ by hand
// and recognising a slug that had already been closed. The convention that
// avoids it is stated with the rest of the record rules in AGENTS.md: a record is
// resolved on the branch that carries it, never re-added to the default branch
// after a branch was cut from it.
//
// It is the READER's check rather than a new rule because the rule already
// exists on both sides of this one and neither covers a read: findIssue refuses
// a TRANSITION on a duplicated id, and record-lint's issue_id_unique refuses a
// COMMITTED tree that carries one. What had no check was the board in between —
// `capture list`, `capture status` and `abcd <iss-N>`, the surfaces a managed
// repository actually has, which rendered the duplicate twice and said nothing.
//
// The scan reads NAMES only: no file is opened, so it costs one readdir per
// status directory and a hostile leaf behind a well-formed name is irrelevant
// here (the guarded read that handles those is scanLedger's).
func checkOneStatusPerID(repoRoot, issuesRoot string) error {
	claims := map[string][]string{}
	var ids []string
	for _, sub := range statusDirs {
		dir := filepath.Join(issuesRoot, statusDirName[sub])
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // virgin/absent ledger tolerance, as scanLedger has
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		sort.Strings(names)
		for _, name := range names {
			id := recordIDOfFilename(name)
			if id == "" {
				continue
			}
			if len(claims[id]) == 0 {
				ids = append(ids, id)
			}
			claims[id] = append(claims[id], fsutil.RepoRel(repoRoot, filepath.Join(dir, name)))
		}
	}

	// Every conflicting id is named, not the first: an operator fixing a merge
	// artefact should see the whole list in one pass, and a refusal that named one
	// would send them round the loop once per duplicate.
	sort.Strings(ids)
	var conflicts []string
	for _, id := range ids {
		paths := claims[id]
		if len(paths) < 2 {
			continue
		}
		// The two shapes are told apart because the remedy differs: two status
		// folders is a status with no answer, while two files in one folder is a
		// duplicated record whose status is at least legible.
		diagnosis := "the status folder IS the record's status, so an id in two of them at once has no " +
			"defined status at all"
		if oneStatusFolder(paths) {
			diagnosis = "an id is the record's identity across the ledger and must name one record"
		}
		conflicts = append(conflicts, fmt.Sprintf("%s is claimed by %s — %s",
			id, strings.Join(paths, " and "), diagnosis))
	}
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s; move or remove one of each pair so the ledger says which status the record is in",
		ErrDuplicateIssueID, strings.Join(conflicts, "; and "))
}

// oneStatusFolder reports whether every claimant of an id sits in the same status
// directory.
func oneStatusFolder(paths []string) bool {
	for _, p := range paths[1:] {
		if statusDirOfLedgerPath(p) != statusDirOfLedgerPath(paths[0]) {
			return false
		}
	}
	return true
}

// recordIDOfFilename returns the canonical id a ledger filename claims, or "" for
// a file that claims none (README.md, the allocator lock, a stray note).
//
// The number is REBUILT from the parsed ordinal rather than echoed, so a
// zero-padded twin (`iss-007-x.md`) keys with `iss-7` instead of beside it. A
// duplicate detector that keyed on the raw text would fail open on exactly the
// spelling a hand-written file is most likely to carry.
func recordIDOfFilename(name string) string {
	m := issFileNumRe.FindStringSubmatch(name)
	if m == nil {
		return ""
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return ""
	}
	return issFamily + "-" + strconv.Itoa(n)
}

// statusDirOfLedgerPath reads the status directory out of a ledger path — the
// segment before the filename, which is the record's lifecycle state.
func statusDirOfLedgerPath(rel string) string {
	return filepath.Base(filepath.Dir(filepath.FromSlash(rel)))
}
