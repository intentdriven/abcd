package capture

// The reading-record and disposition write paths (itd-180, spc-58).
//
// A reading record is what an instrument returned under a recorded visible
// world; a disposition is the researcher's answer to one such item, written
// SEPARATELY and keyed to it. The two are never one write, so the ledger can
// always show that a finding existed before it was answered.
//
// Every refusal below refuses rather than accepting-and-flagging, and names the
// rule it enforces. A record the writer would refuse must never reach the
// committed tree, because the reader refuses it too — and a record no reader can
// read is not a lax record, it is a lost one.
//
// One limit on that, stated rather than implied: the committed-tree gate
// (record_schema) declares no required fields for these two families yet, so it
// judges their SHAPE — bucket, filename, id — and not their content. A record
// these functions would refuse still reaches the tree if it is written by hand
// rather than through them. Closing that means declaring the families' required
// fields to the gate; until then the writer and review are what stand behind the
// content, and saying so is better than letting the header claim cover for it.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The disposition family's id grammar. It is checked BEFORE the value is used to
// build a path, so a traversal id can never touch the filesystem.
//
// The reading run's and the reading item's grammars are NOT restated here: they
// are recordid.ValidReadingRunID and recordid.ValidReadingItemID. The
// cold-reading ingest verb builds a directory name out of a run id arriving off
// an untrusted payload, and its rollback DELETES by the item id — one rule
// refusing a traversal id, and bounding a delete, on both sides is the point of
// that package.
var reDispositionID = regexp.MustCompile(`^` + issueschema.DispositionFamily + `-[0-9]+$`)

// ReadingItem is one thing the instrument returned: the pattern it named (an
// envelope field, because a universal core condition must not live in a variant
// part) plus the position-typed body.
type ReadingItem struct {
	// Pattern is the pattern named — required at every position and in every
	// supply regime.
	Pattern string
	// Body carries exactly the fields the run's position declares
	// (issueschema.ReadingBodyFields). A field from another position's body is
	// refused: one record type, four bodies, and an item belongs to one of them.
	Body map[string]string
	// OccasionedBy is RESERVED and dormant — the join key of the surprise entry,
	// populated in Iteration 2. A populated value is refused until the shape is
	// ruled, so the reservation is a behaviour rather than a comment.
	OccasionedBy string
}

// IngestReadingRequest writes one run's items into the ledger. Position and
// Regime come from the reading's DEFINITION, never from operator input, and are
// therefore run-level rather than per-item.
type IngestReadingRequest struct {
	RepoRoot   string
	IssuesRoot string
	// Run is the run identifier (rdg-N) minted by the assembler.
	Run string
	// Manifest is the content-hash reference to the run's committed manifest,
	// which is what makes the assembly re-runnable and diffable.
	Manifest string
	Position string
	Regime   string
	Items    []ReadingItem
}

// ReadingRecordRef is one written reading record: its minted id and its
// repo-relative path.
type ReadingRecordRef struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// IngestReadingResult is the outcome of a successful ingest.
type IngestReadingResult struct {
	Run     string             `json:"run"`
	Records []ReadingRecordRef `json:"records"`
	// Redacted counts the spans the ledger redactor rewrote across every item
	// written, and Degraded is non-empty when it ran with a weakened pattern set.
	// Both exist so a surface can SAY the text was altered.
	Redacted int    `json:"redacted,omitempty"`
	Degraded string `json:"redaction_degraded,omitempty"`
}

// DispositionRequest writes one disposition, keyed to one reading item.
type DispositionRequest struct {
	RepoRoot   string
	IssuesRoot string
	// Item is the reading item this answers (rdi-N).
	Item string
	// State is one of issueschema.DispositionStates, subject to the per-position
	// availability rule.
	State string
	// Grounds is disposition_grounds: free text, required on every state except
	// held.
	Grounds string
	// ExitCondition is required on held and refused elsewhere.
	ExitCondition string
	// Supersedes names the standing disposition this one replaces (dsp-N) — the
	// only exit from a hold.
	Supersedes string
	// Recurs cites prior item ids: the recorded form of the researcher's warm
	// recognition of a persistence. Never a mechanical join, and never a state.
	Recurs []string
	// HoldFrameLocation / HoldMoscow are the RESERVED two-axis hold field. Both
	// are refused while populated; the grammars are stated and dormant.
	HoldFrameLocation string
	HoldMoscow        string
}

// DispositionResult is the outcome of a successful Disposition.
type DispositionResult struct {
	ID       string `json:"id"`
	Item     string `json:"item"`
	State    string `json:"state"`
	Position string `json:"position"`
	Path     string `json:"path"`
	Redacted int    `json:"redacted,omitempty"`
	Degraded string `json:"redaction_degraded,omitempty"`
}

// IngestReading writes one reading record per item under
// readings/<run-id>/rdi-<N>.md. Every id is minted (adr-45) and never derived
// from the item's content, so two runs returning the same tension carry
// different ids for free — a re-raise stays distinguishable from its first
// appearance, and the recurrence signal survives.
//
// It is the ledger's record WRITER, not a payload validator: the output contract
// that "validated" refers to belongs to the cold-reading output contract, which
// calls this and adds no second validation path. What is enforced here is the
// RECORD schema — the envelope, the position's own body, and the reserved
// fields — which is this package's to refuse.
func IngestReading(req IngestReadingRequest) (IngestReadingResult, error) {
	repoRoot, issuesRoot, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return IngestReadingResult{}, err
	}
	if !recordid.ValidReadingRunID(req.Run) {
		return IngestReadingResult{}, fmt.Errorf("%w: run %q does not match ^%s-[0-9]+$",
			ErrMalformedFrontmatter, req.Run, issueschema.ReadingRunFamily)
	}
	// A run with no items is a run with an EMPTY ITEM SET, and it commits.
	//
	// This refused once, on the reading that an ingest with nothing to write was
	// not an ingest. The design framework's section 13 says otherwise and the
	// maintainer's corrections ruling of 2026-09-02 (4) restates it: a run that
	// returns no items is committed at every position as a run with an empty item
	// set, never refused, and refusal is for a malformed payload. The comparative
	// position's not-exercised outcome is one instance of that rule rather than a
	// carve-out — a widening run of fewer than two candidates yields a committed
	// comparative run with no items, which is what makes the outcome of a widening
	// run one shape whether the position ran or not (iss-2609021153269181).
	//
	// Nothing below needs a special path: the staging loop writes no record, the
	// mint is never called, and the result carries the run with an empty record
	// list. The run's ledger directory is still provisioned, so a reader finds an
	// empty bucket rather than an absence it has to interpret.
	if err := mutationPreamble(repoRoot, issuesRoot); err != nil {
		return IngestReadingResult{}, err
	}

	// Everything below runs UNDER the ledger lock, mint included. The mint has to
	// see the tree it is about to write into: an id is a UTC second plus four
	// uniform digits (adr-45) and nothing sequences two ingests, so two runs
	// landing in one second can draw the same suffix. Minting outside the lock and
	// probing only the current run's directory let exactly that through — one
	// rdi-N in two run directories, an item that could afterwards be neither
	// dispositioned nor promoted, and no tree gate to refuse it.
	type staged struct {
		id      string
		content string
	}
	result := IngestReadingResult{Run: req.Run}

	// Redaction happens HERE, outside the lock, with one scanner for the whole
	// batch. A scanner probes the machine identity and shells out to do it, so
	// building one per free-text value per item put seconds inside a lock every
	// other ledger verb waits on — a large batch failed concurrent work with
	// allocator contention. Nothing below the lock needs a scanner.
	redactor := newLedgerRedactor(repoRoot)
	result.Degraded = redactor.Degraded()
	manifest, n := redactor.redact(req.Manifest)
	result.Redacted += n
	items := make([]ReadingItem, 0, len(req.Items))
	for _, item := range req.Items {
		clean, n := redactReadingItem(redactor, item)
		result.Redacted += n
		items = append(items, clean)
	}

	runDir := filepath.Join(issuesRoot, issueschema.ReadingsDir, req.Run)
	err = withLedgerLock(repoRoot, issuesRoot, func() error {
		if err := ensureFamilyDir(issuesRoot, issueschema.ReadingsDir, req.Run); err != nil {
			return err
		}

		// Assemble and validate EVERY item before anything is written: a run that
		// is half-written is a visible world nobody can reconstruct.
		var pending []staged
		minted := map[string]bool{}
		for i, item := range items {
			id, err := mintUnusedItemID(issuesRoot, minted)
			if err != nil {
				return err
			}
			minted[id] = true
			fields, fm := readingFields(id, manifest, req, item)
			if err := validateReadingStrict(fm); err != nil {
				return fmt.Errorf("item %d: %w", i+1, err)
			}
			content, err := buildIssueText(fields, "")
			if err != nil {
				return fmt.Errorf("item %d: %w", i+1, err)
			}
			// The record's size is DECIDED here, on the assembled bytes, because
			// this is the only place the exact count exists: the values are
			// already redacted, already escaped, and this string is what reaches
			// the disk. Every check upstream is an estimate over one of those
			// steps, and an estimate that models one lengthening step and misses
			// the next writes a record past the cap — which every reader of the
			// family then refuses, leaving the item durable and unanswerable.
			// That is the split RecordReadLimit exists to prevent, so the
			// decision belongs where guessing is unnecessary.
			if n := len(content); n > issueschema.RecordReadLimit {
				return fmt.Errorf("item %d: %w: the record would be %d bytes, past the %d-byte limit "+
					"every reader of this family applies; a record written past it is durable and "+
					"unreadable, so it can never be dispositioned",
					i+1, ErrInvariantViolation, n, issueschema.RecordReadLimit)
			}
			pending = append(pending, staged{id: id, content: content})
		}

		// The mint above already proved every id free across the whole ledger, so
		// this pass can only fire against a writer that did not take the lock — a
		// hand-edit, or another tool. The lock is advisory and scoped to abcd, so
		// the guard is the last thing between a foreign file and an atomic write
		// that would replace it without a trace.
		for _, p := range pending {
			if err := refuseExistingRecord(filepath.Join(runDir, p.id+".md"), p.id); err != nil {
				return err
			}
		}
		for _, p := range pending {
			path := filepath.Join(runDir, p.id+".md")
			if err := writeReadingRecord(path, []byte(p.content)); err != nil {
				// Name what LANDED. A bare error leaves the caller unable to say
				// what is on disk, and a retry then mints fresh ids for the items
				// that already wrote — duplicating them inside the run directory.
				return fmt.Errorf("wrote %s before failing on %s: %w",
					renderList(recordIDs(result.Records)), p.id, err)
			}
			result.Records = append(result.Records, ReadingRecordRef{
				ID: p.id, Path: fsutil.RepoRel(repoRoot, path),
			})
		}
		return nil
	})
	if err != nil {
		// result carries the records that landed before the failure, so a caller
		// can see the partial state rather than guess at it.
		return result, err
	}
	return result, nil
}

// Disposition writes the researcher's answer to one reading item, into a
// directory keyed by the ITEM and a file keyed by the disposition's own id.
//
// The two keys settle requirements that pull against each other. Status is the
// presence of the keyed directory — one probe, never a folder-membership
// question. And a disposition still has an id of its own, so the only exit from
// a held state (a superseding disposition that CITES the one it replaces) has
// something to cite. The superseded record stays in place: a hold that vanished
// when it was answered would take its own exit condition with it (adr-3 — the
// location is the state, and git is the history).
func Disposition(req DispositionRequest) (DispositionResult, error) {
	repoRoot, issuesRoot, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return DispositionResult{}, err
	}
	if !recordid.ValidReadingItemID(req.Item) {
		return DispositionResult{}, fmt.Errorf("%w: item %q does not match ^%s-[0-9]+$",
			ErrMalformedFrontmatter, req.Item, issueschema.ReadingItemFamily)
	}
	if err := mutationPreamble(repoRoot, issuesRoot); err != nil {
		return DispositionResult{}, err
	}

	// The position comes off the KEYED reading record, never from the caller: the
	// availability rule is a coupling the schema carries, and a caller-supplied
	// position would let a disposition assert the very rule it must satisfy. An
	// orphan disposition (no such item) is refused by this same path, because the
	// check has no position to reason with.
	position, err := readingItemPosition(issuesRoot, req.Item)
	if err != nil {
		return DispositionResult{}, err
	}

	id, err := minter.Mint(issueschema.DispositionFamily)
	if err != nil {
		return DispositionResult{}, err
	}
	fields, fm, redacted, degraded, err := dispositionFields(repoRoot, id, req)
	if err != nil {
		return DispositionResult{}, err
	}
	if err := validateDispositionStrict(fm, position); err != nil {
		return DispositionResult{}, err
	}
	content, err := buildIssueText(fields, "")
	if err != nil {
		return DispositionResult{}, err
	}

	itemDir := filepath.Join(issuesRoot, issueschema.DispositionsDir, req.Item)
	path := filepath.Join(itemDir, id+".md")
	err = withLedgerLock(repoRoot, issuesRoot, func() error {
		if err := ensureFamilyDir(issuesRoot, issueschema.DispositionsDir, req.Item); err != nil {
			return err
		}
		// A second answer to one item must say which one it replaces, and it must
		// be checked HERE, under the lock: the standing disposition is a property of
		// the directory as it is at the moment of the write.
		standing, err := standingDispositions(itemDir)
		if err != nil {
			return err
		}
		// More than one standing answer is refused outright, exactly as promote
		// refuses it. A new disposition cannot untangle a contest: --supersedes
		// retires one id and adds its own, so the set never shrinks and the caller
		// is sent round a loop. Writing supersedes_disposition into the surplus
		// records is the only thing that reduces it, and only a person can decide
		// which of the standing answers is the surplus.
		if len(standing) > 1 {
			return fmt.Errorf("%w: %s has %d standing answers (%s), so a new disposition cannot say which is in force — "+
				"a fresh answer supersedes at most one of them and adds its own. Write `supersedes_disposition` into the "+
				"records that are no longer meant to stand, by hand, until exactly one does",
				ErrInvariantViolation, req.Item, len(standing), renderList(standing))
		}
		if req.Supersedes != "" && !containsString(standing, req.Supersedes) {
			return fmt.Errorf("%w: supersedes_disposition names %q, which is not a standing disposition of %s (standing: %s)",
				ErrInvariantViolation, req.Supersedes, req.Item, renderList(standing))
		}
		if req.Supersedes == "" && len(standing) > 0 {
			return fmt.Errorf("%w: %s already carries a standing disposition (%s); a second answer must cite the one it replaces (supersedes_disposition), so the record can say which is in force",
				ErrInvariantViolation, req.Item, renderList(standing))
		}
		if err := refuseExistingRecord(path, id); err != nil {
			return err
		}
		return fsutil.WriteFileAtomic(path, []byte(content), 0o644)
	})
	if err != nil {
		return DispositionResult{}, err
	}
	return DispositionResult{
		ID: id, Item: req.Item, State: req.State, Position: position,
		Path: fsutil.RepoRel(repoRoot, path), Redacted: redacted, Degraded: degraded,
	}, nil
}

// readingFields assembles one reading record's ordered frontmatter and the map
// its validator reads, redacting every free-text value on the way — a reading's
// text lands in the committed ledger exactly as a capture's does.
func readingFields(id, manifest string, req IngestReadingRequest, item ReadingItem) ([]kv, map[string]any) {
	fields := []kv{
		{"schema_version", 1},
		{"id", id},
		{"run", req.Run},
		{"manifest", manifest},
		{"position", req.Position},
		{"regime", req.Regime},
		{"pattern", item.Pattern},
	}
	fm := map[string]any{
		"schema_version": 1,
		"id":             id,
		"run":            req.Run,
		"manifest":       manifest,
		"position":       req.Position,
		"regime":         req.Regime,
		"pattern":        item.Pattern,
	}

	// Body fields are written in the position's DECLARED order, so two records at
	// one position are byte-comparable. A field the position does not declare is
	// still carried into the map, so the validator reports it rather than dropping
	// it silently.
	for _, f := range issueschema.ReadingBodyFields[req.Position] {
		v, ok := item.Body[f]
		if !ok {
			continue
		}
		fields = append(fields, kv{f, v})
		fm[f] = v
	}
	for f, v := range item.Body {
		if _, already := fm[f]; already {
			continue
		}
		fm[f] = v
	}
	if item.OccasionedBy != "" {
		fm["occasioned_by"] = item.OccasionedBy
	}
	return fields, fm
}

// redactReadingItem returns the item with every free-text value sanitised, using
// the batch's ONE scanner. It runs BEFORE the ledger lock is taken: redaction is
// the expensive part of an ingest, and the lock serialises every mutation in the
// repository, so time spent under it is a budget every other verb pays out of.
func redactReadingItem(r *ledgerRedactor, item ReadingItem) (ReadingItem, int) {
	total := 0
	scrub := func(s string) string {
		out, n := r.redact(s)
		total += n
		return out
	}
	out := ReadingItem{
		Pattern:      scrub(item.Pattern),
		OccasionedBy: item.OccasionedBy,
	}
	if item.Body != nil {
		out.Body = make(map[string]string, len(item.Body))
		for k, v := range item.Body {
			out.Body[k] = scrub(v)
		}
	}
	return out, total
}

// dispositionFields assembles one disposition's ordered frontmatter and map.
func dispositionFields(repoRoot, id string, req DispositionRequest) ([]kv, map[string]any, int, string, error) {
	redacted := 0
	degraded := ""
	scrub := func(s string) string {
		out, n, d := redactLedgerText(repoRoot, s)
		redacted += n
		if d != "" {
			degraded = d
		}
		return out
	}

	fields := []kv{
		{"schema_version", 1},
		{"id", id},
		{"item", req.Item},
		{"state", req.State},
	}
	fm := map[string]any{
		"schema_version": 1,
		"id":             id,
		"item":           req.Item,
		"state":          req.State,
	}
	add := func(key, value string) {
		if value == "" {
			return
		}
		s := scrub(value)
		fields = append(fields, kv{key, s})
		fm[key] = s
	}
	add("disposition_grounds", req.Grounds)
	add("exit_condition", req.ExitCondition)
	if req.Supersedes != "" {
		fields = append(fields, kv{"supersedes_disposition", req.Supersedes})
		fm["supersedes_disposition"] = req.Supersedes
	}
	if len(req.Recurs) > 0 {
		fields = append(fields, kv{"recurs", req.Recurs})
		fm["recurs"] = req.Recurs
	}
	// The reserved axes are carried into the map UNWRITTEN: the validator refuses
	// a populated value, so they never reach a field list.
	if req.HoldFrameLocation != "" {
		fm["hold_frame_location"] = req.HoldFrameLocation
	}
	if req.HoldMoscow != "" {
		fm["hold_moscow"] = req.HoldMoscow
	}
	return fields, fm, redacted, degraded, nil
}

// ValidateReadingRecord parses one committed reading record and validates it
// against the family's schema, returning its frontmatter.
//
// It is the WRITER's own check, offered to a reader. The cold-reading assembler
// admits one widening run's records as a comparative reading's candidate set
// (adr-2609021016272867), and it has to know that what it is about to hand over
// is a reading record at the widening position rather than whatever a file of
// that name happens to hold. Exporting this check rather than writing a second
// one is the rule this file's header already states for the two READERS of the
// disposition question: two readers are tolerable, two answers are not.
func ValidateReadingRecord(content string) (map[string]any, error) {
	fm, _, err := parseFrontmatterAndBody(content)
	if err != nil {
		return nil, err
	}
	if err := validateReadingStrict(fm); err != nil {
		return nil, err
	}
	return fm, nil
}

// validateReadingStrict validates a reading record's frontmatter against the
// schema held in core/issueschema — the same data the committed-tree gate reads.
func validateReadingStrict(fm map[string]any) error {
	if err := requireSchemaVersion(fm); err != nil {
		return err
	}
	for k := range fm {
		if !issueschema.ReadingKnown[k] {
			return fmt.Errorf("%w: unknown property %q on a reading record", ErrMalformedFrontmatter, k)
		}
	}
	for _, req := range issueschema.ReadingRequired[1:] {
		if err := requireNonBlankString(fm, req); err != nil {
			return err
		}
	}
	id := fm["id"].(string)
	if !recordid.ValidReadingItemID(id) {
		return fmt.Errorf("%w: id %q does not match ^%s-[0-9]+$", ErrMalformedFrontmatter, id, issueschema.ReadingItemFamily)
	}
	if run := fm["run"].(string); !recordid.ValidReadingRunID(run) {
		return fmt.Errorf("%w: run %q does not match ^%s-[0-9]+$", ErrMalformedFrontmatter, run, issueschema.ReadingRunFamily)
	}
	position := fm["position"].(string)
	want, known := issueschema.ReadingBodyFields[position]
	if !known {
		return fmt.Errorf("%w: position %q is not one of {%s}",
			ErrMalformedFrontmatter, position, strings.Join(issueschema.Positions, ", "))
	}
	// The supply regime is resolved from the definition THROUGH the position, so
	// no operator input can set it (rulings (4) and (18)). Accepting it as free
	// text would leave the one field whose whole purpose is to be underivable by
	// the caller derivable by the caller — and a record whose regime disagrees
	// with its own position says nothing about what it was read under.
	if regime := fm["regime"].(string); regime != issueschema.ReadingRegime(position) {
		return fmt.Errorf("%w: regime %q is not the regime the %s position implies (%s); the regime is resolved from the reading's definition through its position and is never supplied",
			ErrInvariantViolation, regime, position, issueschema.ReadingRegime(position))
	}

	// One record type, four bodies: the item carries the body its own position
	// declares, and no other. A field belonging to a different position is a known
	// property in the WRONG record, which is exactly the drift a single untyped
	// body has to be gated against.
	declared := map[string]bool{}
	for _, f := range want {
		declared[f] = true
		if err := requireNonBlankString(fm, f); err != nil {
			return fmt.Errorf("%w (the %s body, which the %s position returns, is %s)",
				err, issueschema.ReadingRegime(position), position, strings.Join(want, " · "))
		}
	}
	for k := range fm {
		if declared[k] || isReadingEnvelopeField(k) {
			continue
		}
		if isReadingBodyField(k) {
			return fmt.Errorf("%w: %q belongs to another position's body; an item at the %s position carries %s",
				ErrMalformedFrontmatter, k, position, strings.Join(want, " · "))
		}
	}

	for _, f := range issueschema.ReservedSurpriseFields {
		if v, present := fm[f]; present && strings.TrimSpace(asString(v)) != "" {
			return fmt.Errorf("%w: %q is reserved and dormant — the surprise entry is a distinct record shape, populated once its shape is ruled; a populated value is refused rather than silently accepted",
				ErrInvariantViolation, f)
		}
	}
	if v, present := fm["promoted_to"]; present {
		if !reItdID.MatchString(asString(v)) {
			return fmt.Errorf("%w: promoted_to %q does not match ^itd-[0-9]+$", ErrMalformedFrontmatter, v)
		}
	}
	return nil
}

// validateDispositionStrict validates a disposition against the schema and
// against the position of the item it answers.
func validateDispositionStrict(fm map[string]any, position string) error {
	if err := requireSchemaVersion(fm); err != nil {
		return err
	}
	for k := range fm {
		if !issueschema.DispositionKnown[k] {
			return fmt.Errorf("%w: unknown property %q on a disposition", ErrMalformedFrontmatter, k)
		}
	}
	for _, req := range issueschema.DispositionRequired[1:] {
		if err := requireNonBlankString(fm, req); err != nil {
			return err
		}
	}
	if id := fm["id"].(string); !reDispositionID.MatchString(id) {
		return fmt.Errorf("%w: id %q does not match ^%s-[0-9]+$", ErrMalformedFrontmatter, id, issueschema.DispositionFamily)
	}
	if item := fm["item"].(string); !recordid.ValidReadingItemID(item) {
		return fmt.Errorf("%w: item %q does not match ^%s-[0-9]+$", ErrMalformedFrontmatter, item, issueschema.ReadingItemFamily)
	}
	state := fm["state"].(string)
	if !containsString(issueschema.DispositionStates, state) {
		return fmt.Errorf("%w: state %q is not one of {%s}; nothing meaning \"already covered\" exists at any position — an undispositioned item is reported as outstanding, never named as a state",
			ErrMalformedFrontmatter, state, strings.Join(issueschema.DispositionStates, ", "))
	}

	// The grounds are what make a disposition a judgement. They are required on
	// every state except held, whose exit condition carries the same weight — and
	// an exit condition on any other state names an exit from a state that has
	// none, so it is refused rather than written and ignored.
	if state == issueschema.DispositionHeld {
		if err := requireNonBlankString(fm, "exit_condition"); err != nil {
			return fmt.Errorf("%w — a held disposition is directional, and a hold with no exit condition is the parking space `open` already is", err)
		}
	} else {
		if err := requireNonBlankString(fm, "disposition_grounds"); err != nil {
			return err
		}
		if v, present := fm["exit_condition"]; present && strings.TrimSpace(asString(v)) != "" {
			return fmt.Errorf("%w: exit_condition belongs to a held disposition; %q has no exit to name",
				ErrInvariantViolation, state)
		}
	}

	// The availability rule, read against the position the keyed reading record
	// carries. An UNRULED pair is permitted quietly: `held` at the widening
	// position is deferred, and refusing it here would settle by implementation
	// what the facilitator deferred.
	available, ruled := issueschema.DispositionStateAvailable(position, state)
	if ruled && !available {
		return fmt.Errorf("%w: %q is not available at the %s position (available there: %s); the disposition record validates its state against the envelope's position",
			ErrInvariantViolation, state, position, strings.Join(availableStates(position), ", "))
	}

	for _, f := range issueschema.ReservedHoldFields {
		if v, present := fm[f]; present && strings.TrimSpace(asString(v)) != "" {
			return fmt.Errorf("%w: %q is reserved and dormant — the two-axis hold field is present in the schema (frame-location: free text naming the frame element; MoSCoW: %s) and a populated value is refused until activation is ruled",
				ErrInvariantViolation, f, strings.Join(issueschema.HoldMoscowValues, " / "))
		}
	}
	if v, present := fm["supersedes_disposition"]; present {
		if !reDispositionID.MatchString(asString(v)) {
			return fmt.Errorf("%w: supersedes_disposition %q does not match ^%s-[0-9]+$",
				ErrMalformedFrontmatter, v, issueschema.DispositionFamily)
		}
	}
	if v, present := fm["recurs"]; present {
		items, isList := v.([]string)
		if !isList {
			return fmt.Errorf("%w: recurs must be a list of prior item ids", ErrMalformedFrontmatter)
		}
		for _, it := range items {
			if !recordid.ValidReadingItemID(it) {
				return fmt.Errorf("%w: recurs item %q does not match ^%s-[0-9]+$",
					ErrMalformedFrontmatter, it, issueschema.ReadingItemFamily)
			}
		}
	}
	return nil
}

// readingItemPosition reads the position off a reading record, located by its id
// across every run directory. An id no run returned is an unknown item — the
// same sentinel an unknown issue id raises, because the fault is the same shape.
func readingItemPosition(issuesRoot, item string) (string, error) {
	path, err := findReadingItem(issuesRoot, item)
	if err != nil {
		return "", err
	}
	content, err := readRecordGuarded(path)
	if err != nil {
		return "", err
	}
	fm, _, err := parseFrontmatterAndBody(content)
	if err != nil {
		return "", err
	}
	if err := validateReadingStrict(fm); err != nil {
		return "", err
	}
	return asString(fm["position"]), nil
}

// findReadingItem locates a reading record by id across the run directories.
//
// The search needs no run argument: an id is unique to the LEDGER, not to the run
// that minted it, so a caller dispositioning an item does not have to know which
// run returned it. That uniqueness is enforced, not assumed — the mint probes the
// whole tree before it claims an id (mintUnusedItemID) — and the
// more-than-one-run arm below is what says so if it ever stops holding.
func findReadingItem(issuesRoot, item string) (string, error) {
	matches, err := readingItemPaths(issuesRoot, item)
	if err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%w: %s is not a reading item this ledger holds", ErrUnknownIssueID, item)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("%w: %s is present in more than one run directory", ErrDuplicateIssueID, item)
	}
}

// readingItemPaths returns every file in the ledger that claims item, across all
// run directories. Zero matches means the id is free, which is what the mint
// asks; one is the ordinary case; more is a ledger fault findReadingItem names.
// An absent readings tree is no matches, not an error — a repository that has
// commissioned no reading is in a state, not a fault.
func readingItemPaths(issuesRoot, item string) ([]string, error) {
	if !recordid.ValidReadingItemID(item) {
		return nil, fmt.Errorf("invalid %s-N identifier: %q", issueschema.ReadingItemFamily, item)
	}
	readingsRoot := filepath.Join(issuesRoot, issueschema.ReadingsDir)
	if err := refuseSymlinkedDir(readingsRoot); err != nil {
		return nil, err
	}
	runs, err := os.ReadDir(readingsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var matches []string
	for _, run := range runs {
		if !recordid.ValidReadingRunID(run.Name()) {
			continue
		}
		// Every run directory is checked, not only the ones a walk would descend
		// into: a symlink IS a directory to ReadDir, and following one is how a
		// read — or promote's stamp, which writes back to whatever this returns —
		// leaves the tree that is supposed to contain it.
		runDir := filepath.Join(readingsRoot, run.Name())
		if err := refuseSymlinkedDir(runDir); err != nil {
			return nil, err
		}
		if !run.IsDir() {
			continue
		}
		cand := filepath.Join(runDir, item+".md")
		if fi, err := os.Lstat(cand); err == nil && fi.Mode().IsRegular() {
			matches = append(matches, cand)
		}
	}
	return matches, nil
}

// refuseSymlinkedDir is safeMkdirLeaf's guard without the mkdir: it refuses a
// path that exists and is not a real directory. The write paths provision their
// directories and meet that guard on the way in; the READ paths never did, and a
// read here is not read-only in consequence — promote stamps back into whatever
// findReadingItem returns, so a symlinked readings root or run directory sent
// that write outside the ledger. An absent path is not a fault: an unpopulated
// tree is a state.
func refuseSymlinkedDir(dir string) error {
	fi, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: lstat failed for %s: %v", ErrPathUnsafe, dir, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return fmt.Errorf("%w: not a real directory: %s", ErrPathUnsafe, dir)
	}
	return nil
}

// standingDispositions lists the dispositions of one item that no sibling
// supersedes — the answers currently in force. An empty result means the item is
// undispositioned, which the outstanding report says out loud and no state names.
//
// The WALK is here; the JUDGEMENT is not. Which records stand is decided by
// issueschema.StandingDispositionIDs, the one reader core/lint calls too. Two
// readers of that question diverged twice in review — first on a duplicated key,
// then on a file led by a comment or a blank line — and each time the board said
// "answered" while this verb said "two standing", over the same bytes.
func standingDispositions(itemDir string) ([]string, error) {
	records, err := readDispositions(itemDir)
	if err != nil {
		return nil, err
	}
	standing := issueschema.StandingDispositionIDs(records)
	// Records present and NOTHING standing is a supersession cycle: every answer
	// retired by another, so an item carrying several reads as carrying none. That
	// is a ledger fault, not an unanswered item, and the difference is the whole
	// point — treating it as unanswered would let the verb write a fresh uncited
	// answer on top of a tangle nobody has untied. It can only be reached by hand
	// (the verb refuses a second answer that supersedes nothing, and --supersedes
	// must name a STANDING id), so untying it by hand is the remedy.
	if len(records) > 0 && len(standing) == 0 {
		return nil, fmt.Errorf("%w: every disposition of %s is superseded by another, so none stands — a supersession cycle in %s, which no write path can produce and only a hand edit can repair",
			ErrInvariantViolation, filepath.Base(itemDir), filepath.Base(itemDir))
	}
	return standing, nil
}

// readDispositions reads one item's disposition directory into the shared record
// shape. A directory that does not exist is an unanswered item, not a fault.
func readDispositions(itemDir string) ([]issueschema.DispositionRecord, error) {
	// The FAMILY root as well as the item leaf. Guarding only the leaf left
	// promote reading an item's standing state through a symlinked dispositions/,
	// so the answer that licenses its stamp came from outside the ledger.
	if err := refuseSymlinkedDir(filepath.Dir(itemDir)); err != nil {
		return nil, err
	}
	if err := refuseSymlinkedDir(itemDir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(itemDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var records []issueschema.DispositionRecord
	for _, e := range entries {
		id, ok := issueschema.DispositionFileID(e.Name())
		if !ok {
			continue
		}
		path := filepath.Join(itemDir, e.Name())
		content, err := readRecordGuarded(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		records = append(records, issueschema.ParseDisposition(id, content))
	}
	return records, nil
}

// mintUnusedItemID draws an item id that neither this batch nor the LEDGER has
// already taken, redrawing on a repeat within a bounded budget.
//
// Both halves matter and for one reason: the mint reads no maximum (adr-45), so
// two draws in one second are separated by four random digits alone. Within a
// batch that is a few percent at 25 items; across two runs ingested in the same
// second it is the same coincidence with no batch to notice it. The ledger probe
// walks every run directory, because an id is unique to the LEDGER, not to the
// run that happened to mint it.
//
// A redraw rather than a refusal, exactly as the issue allocator's reservation
// does (spc-33 ruling 2): redrawing keeps candidates independent and uniform,
// where re-deriving from occupancy would be a miniature max+1. The budget is the
// allocator's own — a draw that keeps colliding is a broken entropy source, and
// looping forever on one would hang the ingest instead of reporting it.
//
// It must be called under the ledger lock: the probe and the write that claims
// the id are only one decision if nothing can land between them.
func mintUnusedItemID(issuesRoot string, minted map[string]bool) (string, error) {
	for attempt := 0; attempt < placeholderRetryBudget; attempt++ {
		id, err := minter.Mint(issueschema.ReadingItemFamily)
		if err != nil {
			return "", err
		}
		if minted[id] {
			continue
		}
		taken, err := readingItemPaths(issuesRoot, id)
		if err != nil {
			return "", err
		}
		if len(taken) == 0 {
			return id, nil
		}
	}
	return "", fmt.Errorf("%w: could not mint a free %s id after %d draws",
		ErrAllocatorContention, issueschema.ReadingItemFamily, placeholderRetryBudget)
}

// recordIDs projects written records to their ids, for an error that says what
// landed.
func recordIDs(records []ReadingRecordRef) []string {
	out := make([]string, 0, len(records))
	for _, r := range records {
		out = append(out, r.ID)
	}
	return out
}

// readingWriteHook, when non-nil, replaces the atomic write inside IngestReading.
// It is a test-only seam (nil in production, zero overhead) used to force a
// deterministic mid-batch write failure, mirroring stampWriteHook in promote.go.
var readingWriteHook func(path string, data []byte) error

func writeReadingRecord(path string, data []byte) error {
	if readingWriteHook != nil {
		return readingWriteHook(path, data)
	}
	return fsutil.WriteFileAtomic(path, data, 0o644)
}

// refuseExistingRecord fails a write whose target is already taken. The id space
// is a UTC second plus four random digits, so two same-second draws CAN coincide
// — rare, and rare is exactly why it must refuse rather than overwrite: a
// committed record silently replaced by another leaves no trace that either
// happened, which is the one outcome a ledger must not produce. The check runs
// under the ledger lock, so between it and the write no other abcd process can
// claim the path.
func refuseExistingRecord(path, id string) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("%w: %s already exists in this ledger; the mint collided with a record already committed, and nothing in this write is applied",
			ErrDuplicateIssueID, id)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ensureFamilyDir provisions <issuesRoot>/<family>/<key>, refusing a symlinked
// leaf at either level exactly as the status directories are provisioned.
func ensureFamilyDir(issuesRoot, family, key string) error {
	if err := safeMkdirLeaf(filepath.Join(issuesRoot, family)); err != nil {
		return err
	}
	return safeMkdirLeaf(filepath.Join(issuesRoot, family, key))
}

// requireSchemaVersion is the version check every record in this ledger opens
// with: a reader that cannot name the version it is reading cannot claim to have
// validated anything.
func requireSchemaVersion(fm map[string]any) error {
	sv, ok := fm["schema_version"]
	if !ok {
		return fmt.Errorf("%w: missing required property 'schema_version'", ErrMissingRequiredField)
	}
	if n, isInt := sv.(int); !isInt || n != 1 {
		return fmt.Errorf("%w: unsupported schema_version %v (this reader only handles 1)", ErrMissingRequiredField, sv)
	}
	return nil
}

// requireNonBlankString demands a present, string-typed, non-blank property. A
// property present but blank counts as missing for the same reason it does on an
// issue: the reader cannot make a value out of it either.
func requireNonBlankString(fm map[string]any, key string) error {
	v, present := fm[key]
	if !present {
		return fmt.Errorf("%w: missing required property %q", ErrMissingRequiredField, key)
	}
	s, isStr := v.(string)
	if !isStr {
		return fmt.Errorf("%w: %q must be a string", ErrMalformedFrontmatter, key)
	}
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("%w: %q must be non-empty", ErrMissingRequiredField, key)
	}
	return nil
}

// isReadingEnvelopeField reports whether key is part of the envelope every
// reading record carries whatever its position.
func isReadingEnvelopeField(key string) bool {
	if containsString(issueschema.ReadingRequired, key) {
		return true
	}
	if containsString(issueschema.ReservedSurpriseFields, key) {
		return true
	}
	return key == "promoted_to"
}

// isReadingBodyField reports whether key belongs to SOME position's body.
func isReadingBodyField(key string) bool {
	for _, p := range issueschema.ReadingPositions {
		if containsString(p.Fields, key) {
			return true
		}
	}
	return false
}

// availableStates renders the states available at a position, for a refusal that
// names the rule rather than merely refusing.
func availableStates(position string) []string {
	var out []string
	for _, s := range issueschema.DispositionStates {
		if available, ruled := issueschema.DispositionStateAvailable(position, s); available && ruled {
			out = append(out, s)
		}
	}
	return out
}

func containsString(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

func renderList(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}
