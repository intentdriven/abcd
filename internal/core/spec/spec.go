// Package spec is abcd's transport-agnostic native spec store (intent
// lifecycle, itd-64). It owns the in-memory model of spec records and the
// disk operations that mint, load, and transition them. Every function takes a
// structured request and returns a structured result; nothing here writes to
// stdout or knows about a CLI, MCP, or hook surface — the front doors under
// internal/surface/* marshal these results for their transport.
//
// A spec record is a markdown file under
// <repoRoot>/.abcd/development/specs/{open,closed}/spc-N-<slug>.md. The bucket
// directory IS the status (directory-as-truth: open/ vs closed/, with no
// status: frontmatter field), mirroring how intents encode their lifecycle by
// bucket. The load-bearing field is intent: itd-N, the link a spec declares to
// the intent it realises.
//
// Frontmatter is read by the shared internal/core/frontmatter line scanner, not
// a YAML parser — the package pulls in zero new dependencies. Ids are validated
// against strict regexes before any path is built, so a hostile id can never
// traverse out of the spec store.
package spec

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/provenance"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// Bucket directory names. The directory a spec file lives in is its status.
const (
	StatusOpen   = "open"
	StatusClosed = "closed"
)

// Repo-relative store locations.
const (
	// SpecsRelDir is the spec-store root, relative to the repo worktree.
	SpecsRelDir = ".abcd/development/specs"
)

// maxSpecFileBytes caps any spec markdown file read (trust boundary).
const maxSpecFileBytes = 256 * 1024

// A spec id and its load-bearing intent link constrain what path this package
// will build (path-traversal defence); the predicates that decide it live in
// internal/core/recordid, because the record-lint gate has to refuse exactly the
// set this package refuses — content exemption or not — and a second copy of the
// pattern is a copy that can drift (iss-2608270500207987).

var (
	// slugRe constrains a slug to kebab-case, since a slug becomes a filename.
	slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	// specNumRe extracts the numeric N from a spec id or a spec_id value that may
	// carry a trailing slug (spc-1, spc-2-thing).
	specNumRe = regexp.MustCompile(`^spc-([0-9]+)`)
	// specFileRe matches a spec-store filename and captures its id.
	specFileRe = regexp.MustCompile(`^(spc-[0-9]+)-.*\.md$`)
)

// Spec is one spec record. Status is the bucket it was found in; Path is
// repo-relative (never an absolute local path).
type Spec struct {
	ID     string `json:"id"`     // spc-N
	Slug   string `json:"slug"`   // kebab-case
	Intent string `json:"intent"` // itd-N, the load-bearing link
	Status string `json:"status"` // open | closed (directory-as-truth)
	Path   string `json:"path"`   // repo-relative markdown path
}

// Store is the in-memory set of spec records discovered under both buckets.
type Store struct {
	Specs []Spec `json:"specs"`
}

// Lookup returns the spec the given reference names; ok is false when absent.
//
// Matching is on the spec NUMBER, not the literal string, because that is the
// comparison record-lint makes: a spec_id is written bare (spc-9), with its slug
// (spc-9-widget), and zero-padded (spc-009) across the record, and all three are
// lint-green. A literal-only compare would let the lifecycle verbs refuse a
// record the lint accepts. An exact string match still wins when the store holds
// one, so a caller that names a record precisely gets that record; two specs
// sharing a number is a record defect the lint's spec_id_unique rule flags (the
// mint never produces one: it redraws while a candidate names a spec here, and
// this lookup is the presence check it redraws against).
func (s Store) Lookup(specID string) (Spec, bool) {
	for _, sp := range s.Specs {
		if sp.ID == specID {
			return sp, true
		}
	}
	for _, sp := range s.Specs {
		if SameNum(sp.ID, specID) {
			return sp, true
		}
	}
	return Spec{}, false
}

// ByIntent returns the spec linked to the given intent id; ok is false when no
// spec realises that intent.
func (s Store) ByIntent(intentID string) (Spec, bool) {
	for _, sp := range s.Specs {
		if sp.Intent == intentID {
			return sp, true
		}
	}
	return Spec{}, false
}

// Validate enforces the id regexes and that intent is a well-formed itd-N. It
// is the fail-closed guard both Load and the minting path run before trusting a
// record's id in a filesystem path.
func Validate(s Spec) error {
	if !recordid.ValidSpecID(s.ID) {
		return fmt.Errorf("spec: id %q must match ^spc-[0-9]+$", s.ID)
	}
	if !recordid.ValidIntentID(s.Intent) {
		return fmt.Errorf("spec %s: intent %q must match ^itd-[0-9]+$", s.ID, s.Intent)
	}
	return nil
}

// specNum extracts the numeric N from a spec id or spec_id value, or 0 if none.
func specNum(id string) int {
	n, _ := parseSpecNum(id)
	return n
}

// parseSpecNum extracts the numeric N from a spec id or spec_id value; ok is
// false when the value carries no usable number.
func parseSpecNum(id string) (int, bool) {
	m := specNumRe.FindStringSubmatch(id)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		// An over-int64 (or otherwise unparseable) number names nothing: Atoi
		// returns the clamped MaxInt64 alongside the error, and keeping it would
		// let every such spelling compare equal to the clamp, so SameNum would
		// equate two values that name no spec. Treat it as no number.
		return 0, false
	}
	return n, true
}

// SameNum reports whether two spec references name the same spec number — the
// canonical comparison the record lint makes, so a verb using it can never
// refuse a spelling the lint accepts (spc-9, spc-9-widget, spc-009 are one
// spec). A value carrying no usable number (null, "spc-", "spc-abc", an
// over-int64 N) matches nothing, including another such value.
func SameNum(a, b string) bool {
	an, aok := parseSpecNum(a)
	bn, bok := parseSpecNum(b)
	return aok && bok && an == bn
}

// HasNum reports whether v is a spec reference carrying a usable number. It is
// the tolerant counterpart of the ^spc-[0-9]+$ argument grammar, for validating
// a STORED spec_id, which the record lint lets carry a slug or zero padding.
func HasNum(v string) bool {
	_, ok := parseSpecNum(v)
	return ok
}

// stubMarker is the opening of the author-guidance placeholder renderSpec mints.
// BodyIsStub keys on the same constant, so the template and the detector cannot
// drift (lockstep-tested).
const stubMarker = "_Draft: describe what "

// BodyIsStub reports whether a spec body still carries the minted placeholder —
// i.e. no human has written the design record yet.
func BodyIsStub(content string) bool {
	return strings.Contains(content, stubMarker)
}

// renderSpec is the minimal spec-file body: frontmatter carrying the id, slug,
// and the load-bearing intent link, plus a title and a Summary placeholder.
func renderSpec(id, slug, intentID string, stamp provenance.Stamp) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "id: %s\n", id)
	fmt.Fprintf(&b, "slug: %s\n", slug)
	fmt.Fprintf(&b, "intent: %s\n", intentID)
	// The disclosure pair (itd-178), bare like every other scalar in this block
	// and written together — a lone key is a state no write path produces.
	fmt.Fprintf(&b, "%s: %s\n", provenance.KeyOrigin, stamp.OriginValue())
	fmt.Fprintf(&b, "%s: %s\n", provenance.KeyProductionMode, stamp.ModeValue())
	b.WriteString("---\n")
	fmt.Fprintf(&b, "# %s\n\n", slug)
	b.WriteString("## Summary\n\n")
	// A clear author-guidance placeholder, not a bare "TODO" that reads as drift:
	// the spec body is the design record the intent's fidelity review audits against.
	fmt.Fprintf(&b, "%s%s delivers for %s — scope, approach, and how "+
		"it satisfies the intent's Acceptance Criteria. This spec is the design record "+
		"the fidelity review audits against._\n", stubMarker, id, intentID)
	return b.String()
}
