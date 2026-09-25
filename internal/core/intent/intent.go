// Package intent is abcd's transport-agnostic native intent store (intent
// lifecycle, itd-80). It owns the in-memory model of intent records and the disk
// operations that plan, link, and summarise them. Every function takes a
// structured request and returns a structured result; nothing here writes to
// stdout or knows about a CLI, MCP, or hook surface — the front doors under
// internal/surface/* marshal these results for their transport.
//
// An intent record is a markdown file under
// <repoRoot>/.abcd/development/intents/{drafts,planned,shipped,disciplines,
// superseded}/itd-N-<slug>.md. The bucket directory IS the lifecycle state
// (directory-as-truth: there is no status: frontmatter field), mirroring the
// native spec store. The load-bearing field is spec_id: spc-N, the intent's
// derived side of the bidirectional link to the spec that realises it (the
// spec's reciprocal side is intent: itd-N).
//
// Frontmatter is read by the shared internal/core/frontmatter line scanner, not
// a YAML parser — the package pulls in zero new dependencies. Ids are validated
// against strict regexes before any path is built, so a hostile id can never
// traverse out of the intent store.
package intent

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// IntentsRelDir is the intent-store root, relative to the repo worktree.
const IntentsRelDir = ".abcd/development/intents"

// Lifecycle buckets. The directory an intent file lives in is its state.
const (
	BucketDrafts      = "drafts"
	BucketPlanned     = "planned"
	BucketShipped     = "shipped"
	BucketDisciplines = "disciplines"
	BucketSuperseded  = "superseded"
)

// KindStandalone is the default binding kind Plan writes (a 1:1 intent↔spec).
const KindStandalone = "standalone"

// Buckets is the fixed lifecycle order used for loading and rendering.
var Buckets = []string{BucketDrafts, BucketPlanned, BucketShipped, BucketDisciplines, BucketSuperseded}

// maxIntentFileBytes caps any intent markdown file read (trust boundary).
const maxIntentFileBytes = 256 * 1024

// An intent id and a spec id constrain what path this package will build
// (path-traversal defence); the predicates that decide it live in
// internal/core/recordid, because the record-lint gate has to refuse exactly the
// set this package refuses and a second copy of the pattern is a copy that can
// drift (iss-2608270500198764).

var (
	// slugRe constrains a slug to kebab-case, since a slug becomes a filename.
	slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	// intentFileRe matches an intent-store filename.
	intentFileRe = regexp.MustCompile(`^itd-[0-9]+.*\.md$`)
	// fmKeyRe matches a top-level frontmatter key (column 0) for the writer.
	fmKeyRe = regexp.MustCompile(`^([A-Za-z0-9_]+):(.*)$`)
	// acHeadingRe matches the `## Acceptance Criteria` heading (any heading depth).
	acHeadingRe = regexp.MustCompile(`^#{1,6}\s+Acceptance Criteria\s*$`)
)

// Intent is one intent record. Bucket is the directory it was found in; Path is
// repo-relative (never an absolute local path).
type Intent struct {
	ID     string `json:"id"`      // itd-N
	Slug   string `json:"slug"`    // kebab-case
	Kind   string `json:"kind"`    // standalone | bundle-member | discipline | null
	SpecID string `json:"spec_id"` // spc-N, the derived link (may be null)
	Bucket string `json:"bucket"`  // lifecycle directory (directory-as-truth)
	Path   string `json:"path"`    // repo-relative markdown path
	// RelatedIssues are the ledger records this intent graduated from — an iss-N,
	// or the rdi-N of a dispositioned reading item — the intent half of the
	// two-sided promote join (itd-4 AC3; the record half is the source's
	// `related_intents`). The first entry is the record the intent was promoted
	// from; a later one was linked beside it. Parsed leniently: absent on every
	// record that graduated from nothing.
	RelatedIssues []string `json:"related_issues,omitempty"`
	// Held is the reason the record is held — the value `abcd intent hold`
	// wrote — and empty when it is not. HeldMalformed reports a `held:` key
	// present in a shape no verb writes (blank, null, a list, a map, a block
	// scalar): the loader stays lenient so one hand edit cannot fail-close the
	// whole corpus, and record-lint is what names the line. See hold.go for the
	// trust boundary between the two.
	Held          string `json:"held,omitempty"`
	HeldMalformed bool   `json:"held_malformed,omitempty"`
}

// Corpus is the in-memory set of intent records discovered across every bucket.
type Corpus struct {
	Intents []Intent `json:"intents"`
}

// Lookup returns the intent with the given id; ok is false when absent.
//
// Matching is CANONICAL (recordid.SameID) after an exact hit fails, the same
// two-pass shape spec.Store.Lookup uses and for the same reason: record-lint
// resolves an intent handle on its number with its leading zeros trimmed, so a
// link written `itd-007` is green and names itd-7. A literal-only compare here
// made this verb refuse — "itd-007 not found in any bucket" — a record the lint
// says exists, which left the spec carrying that spelling permanently unclosable
// and its intent permanently unlinkable. An exact match still wins when the
// corpus holds one, so a caller naming a record precisely gets that record.
func (c Corpus) Lookup(id string) (Intent, bool) {
	for _, it := range c.Intents {
		if it.ID == id {
			return it, true
		}
	}
	for _, it := range c.Intents {
		if recordid.SameID(it.ID, id) {
			return it, true
		}
	}
	return Intent{}, false
}

// Validate enforces the id regex — the fail-closed guard Load runs before
// trusting a record's id in a filesystem path.
func Validate(it Intent) error {
	if !recordid.ValidIntentID(it.ID) {
		return fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", it.ID)
	}
	return nil
}

// hasAcceptanceCriteria reports whether content carries a `## Acceptance Criteria`
// section with at least one top-level -/* bullet — the itd-1 discipline Plan
// enforces. It requires a BULLET (not merely non-blank prose) so the Plan gate
// agrees with the ingest gate (countAcceptanceCriteria): an intent Plan accepts
// is one whose criteria the fidelity review can actually enumerate and judge,
// never a prose-only or numbered section that would perpetually dead-letter every
// verdict for having zero positional criteria.
func hasAcceptanceCriteria(content string) bool {
	return countAcceptanceCriteria(content) > 0
}

// setFrontmatterFields returns content with the given frontmatter keys set to
// the given values: an existing top-level key line is rewritten in place, and a
// key not yet present is inserted just before the closing `---` (sorted, for a
// deterministic result). Everything outside the leading frontmatter block — the
// body and untouched keys — is preserved verbatim. An input without a well-formed
// leading frontmatter block is an error (fail closed rather than corrupt a file).
func setFrontmatterFields(content string, updates map[string]string) (string, error) {
	lines := strings.Split(content, "\n")
	// Match frontmatter.Fields's delimiter tolerance exactly: a `---` line may
	// carry trailing whitespace ("--- "). Trimming only "\r" here (stricter than
	// the reader) makes the writer skip a delimiter the reader accepts and insert
	// keys into the body instead of the frontmatter — corrupting the record.
	if len(lines) == 0 || strings.TrimRight(lines[0], " \t\r") != "---" {
		return "", fmt.Errorf("intent: file has no leading frontmatter block")
	}
	closing := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t\r") == "---" {
			closing = i
			break
		}
	}
	if closing < 0 {
		return "", fmt.Errorf("intent: frontmatter block is not closed")
	}

	remaining := make(map[string]string, len(updates))
	for k, v := range updates {
		remaining[k] = v
	}
	for i := 1; i < closing; i++ {
		m := fmKeyRe.FindStringSubmatch(strings.TrimRight(lines[i], "\r"))
		if m == nil {
			continue
		}
		if v, ok := remaining[m[1]]; ok {
			lines[i] = m[1] + ": " + v
			delete(remaining, m[1])
		}
	}
	if len(remaining) > 0 {
		keys := make([]string, 0, len(remaining))
		for k := range remaining {
			keys = append(keys, k)
		}
		sortStrings(keys)
		ins := make([]string, 0, len(keys))
		for _, k := range keys {
			ins = append(ins, k+": "+remaining[k])
		}
		out := make([]string, 0, len(lines)+len(ins))
		out = append(out, lines[:closing]...)
		out = append(out, ins...)
		out = append(out, lines[closing:]...)
		lines = out
	}
	return strings.Join(lines, "\n"), nil
}

// sortStrings sorts in place (small local helper to avoid importing sort for one
// call site).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// PlanOptions parameterises Plan. Both fields are optional: the zero value
// plans a draft exactly as the bare verb does.
type PlanOptions struct {
	// ProductionMode is the disclosure the MINTED SPEC carries (itd-178); it is
	// validated by the spec store before the id is minted, and an empty value
	// takes the vocabulary's default. It has no bearing on the intent record,
	// whose own stamp was written when the draft was created and is never
	// rewritten.
	ProductionMode string
	// Impact is the product-impact judgement to stamp onto the INTENT record
	// (`impact: additive|breaking|fix`), the same field the create path writes
	// from its own --impact and the close that ships demands. Plan is the verb
	// that runs when the judgement is actually made — the planning interview
	// settles the impact class — so it is where a draft filed without one gets
	// it. It is validated at the bar the create and close paths apply, refused
	// when it disagrees with a judgement the record already carries, and
	// accepted as a no-op when it agrees; empty leaves the record unjudged.
	Impact string
}

// PlanResult reports a completed Plan: the updated planned intent and the spec
// minted to realise it.
type PlanResult struct {
	Intent Intent    `json:"intent"`
	Spec   spec.Spec `json:"spec"`
	// ConditionsStamped is how many scope-condition bullets this run gave an
	// identity to.
	ConditionsStamped int `json:"conditions_stamped"`
	// StampOnly reports that this run did the identity step alone, over a record
	// already in planned/: no spec was minted and no bucket moved. It is how a
	// condition written after planning reaches the mint, which is what makes the
	// readiness gate's remedy a command that works.
	StampOnly bool `json:"stamp_only"`
	// ImpactStamped is the impact judgement this run wrote onto the record, and
	// empty when it wrote none — because no --impact was supplied, or because the
	// record already carried the same value.
	ImpactStamped string `json:"impact_stamped"`
}

// LinkResult reports a completed Link: the updated intent and the spec it now
// declares.
type LinkResult struct {
	Intent Intent    `json:"intent"`
	Spec   spec.Spec `json:"spec"`
}

// ReconcileResult reports a completed Reconcile (the deterministic half of
// `abcd spec close`): the closed spec, the linked intent in its post-reconcile
// state, whether the intent moved this call (false on an idempotent re-run), and
// the intent's bucket transition (From → To).
type ReconcileResult struct {
	Spec        spec.Spec `json:"spec"`
	Intent      Intent    `json:"intent"`
	IntentMoved bool      `json:"intent_moved"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	// OpenSpecs names the specs that still realise the intent after this close,
	// in store order. Empty is the ordinary case and the one that ships: the
	// intent moves planned/ -> shipped/ on the close after which no open spec
	// names it (adr-2609151513118583). A non-empty list is the visible reason the
	// intent did NOT move, and the surface prints it.
	OpenSpecs []string `json:"open_specs,omitempty"`
	// Remainder is the follow-on spec this close minted for the part of the
	// intent the closed spec did not deliver (the zero value when none was
	// asked for). It is attached to the same intent and lands in open/.
	Remainder spec.Spec `json:"remainder,omitzero"`
	// RemainderMinted says whether THIS invocation wrote that remainder. The mint
	// is idempotent — a retry after a failure downstream of it reuses the spec the
	// previous attempt left behind — so without this the surface reports a record
	// it did not write as one it just wrote.
	RemainderMinted bool `json:"remainder_minted,omitempty"`
	// RemainderSteps are the steps of the closing spec that were not marked
	// landed, which this close carried into the remainder it minted, in order
	// and renumbered from one (itd-2609212103565953). Empty when the closing
	// spec lists no steps, when every step it lists has landed, and when the
	// remainder was reused rather than minted: a reused spec is left as found.
	RemainderSteps []spec.Step `json:"remainder_steps,omitempty"`
	// ReceiptID is the deterministic fidelity-review receipt parked in the
	// shipped intent's Audit Notes (empty if the emit failed).
	ReceiptID string `json:"receipt_id,omitempty"`
	// ReceiptStatus says what the emit did: "owed" on the close that actually
	// parked a new OWED stub, and "already_owed"/"already_ingested"/
	// "already_dead_letter" when the intent had shipped before and the receipt
	// was already there. A close is idempotent, so the same receipt id comes back
	// on every re-run; without this the surface announced a fresh review on each
	// one, which reads as a new obligation the operator has to discharge.
	ReceiptStatus string `json:"receipt_status,omitempty"`
	// AuditEmitError is a NON-FATAL report of a failed review emit. The review is
	// report-only, so the intent still ships; the surface prints this loudly.
	AuditEmitError string `json:"audit_emit_error,omitempty"`
}

// RemainderRequest asks a close to mint a follow-on spec for the part of the
// intent the closing spec did not deliver, attached to that same intent. The
// zero value asks for none, which is the ordinary close.
//
// It is how the honest path is taken in one operation: the visible state after
// a partial delivery is "spec closed X, spec open Y, intent still planned", and
// minting Y by hand afterwards is the step that gets forgotten
// (adr-2609151513118583).
type RemainderRequest struct {
	// Slug is the kebab-case slug of the spec to mint. Empty means no remainder.
	Slug string
	// ProductionMode is the disclosure mode stamped on the minted spec; empty
	// takes the vocabulary's default (provenance.DefaultMode).
	ProductionMode string
}

// LinkedPair is one intent↔spec link in the lifecycle summary.
type LinkedPair struct {
	Intent string `json:"intent"`
	Spec   string `json:"spec"`
}

// StatusView is the read-only lifecycle summary: intent counts by bucket, spec
// counts by status, and the linked intent↔spec pairs.
type StatusView struct {
	Buckets     map[string]int `json:"buckets"`
	SpecsOpen   int            `json:"specs_open"`
	SpecsClosed int            `json:"specs_closed"`
	Linked      []LinkedPair   `json:"linked"`
}
