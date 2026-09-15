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
	// PromotedFrom is the iss-N this intent graduated from (spc-24's two-sided
	// promote edge). Parsed leniently: absent on every non-promoted record.
	PromotedFrom string `json:"promoted_from,omitempty"`
}

// Corpus is the in-memory set of intent records discovered across every bucket.
type Corpus struct {
	Intents []Intent `json:"intents"`
}

// Lookup returns the intent with the given id; ok is false when absent.
func (c Corpus) Lookup(id string) (Intent, bool) {
	for _, it := range c.Intents {
		if it.ID == id {
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
	// ReceiptID is the deterministic fidelity-review receipt parked in the
	// shipped intent's Audit Notes (empty if the emit failed).
	ReceiptID string `json:"receipt_id,omitempty"`
	// AuditEmitError is a NON-FATAL report of a failed review emit. The review is
	// report-only, so the intent still ships; the surface prints this loudly.
	AuditEmitError string `json:"audit_emit_error,omitempty"`
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
