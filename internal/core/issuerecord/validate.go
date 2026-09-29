package issuerecord

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// The reader's refusal sentinels. core/capture re-exports each under its own
// name, so a caller testing errors.Is against either spelling tests one value.
var (
	// ErrInvariantViolation means frontmatter passed the schema but violates a
	// folder-status cross-field invariant.
	ErrInvariantViolation = errors.New("invariant violation")
	// ErrMalformedFrontmatter means frontmatter could not be parsed or failed
	// schema validation.
	ErrMalformedFrontmatter = errors.New("malformed frontmatter")
	// ErrMissingRequiredField means a schema-required field was absent.
	ErrMissingRequiredField = errors.New("missing required field")
	// ErrPathUnsafe means the ledger root or a status dir is a symlink, or a
	// record leaf is not a regular file the guarded read will open.
	ErrPathUnsafe = errors.New("path unsafe")
)

// The id shapes the schema's id-list fields take, mirroring issue.schema.json.
var (
	IssIDRe = regexp.MustCompile(`^iss-[0-9]+$`)
	ItdIDRe = regexp.MustCompile(`^itd-[0-9]+$`)
	SpcIDRe = regexp.MustCompile(`^spc-[0-9]+$`)
	// LinkIDRe is the shape of a typed link's target (duplicates, refines): an
	// issue or an intent, the two families the filing-time match compares with.
	LinkIDRe = regexp.MustCompile(`^(iss|itd)-[0-9]+$`)
)

// issFamily is the issue family's filename prefix, without its hyphen.
const issFamily = "iss"

// The enum-membership sets, derived from the ONE copy of the value lists in
// core/issueschema.
var (
	severities = stringSet(issueschema.Severities)
	categories = stringSet(issueschema.Categories)
	sources    = stringSet(issueschema.Sources)
)

func stringSet(vals []string) map[string]bool {
	m := make(map[string]bool, len(vals))
	for _, v := range vals {
		m[v] = true
	}
	return m
}

// AcceptedValues renders a closed enum's legal set for a refusal message.
//
// One helper rather than three literal lists, so the message and the membership
// test read the same slice: a value added to core/issueschema appears in the
// refusal without anyone remembering to add it.
func AcceptedValues(vals []string) string {
	return "accepted values: " + strings.Join(vals, " | ")
}

// uniqueItemsFields are the array properties issue.schema.json flags
// uniqueItems:true.
var uniqueItemsFields = []string{"related_intents", "related_specs", "related_issues", "synthesis_clusters", "blocked_by", "duplicates", "refines"}

// ValidateStrict validates a frontmatter map against the issue schema. It
// special-cases schema_version first (mirrors _validate_strict) and rejects
// unknown keys (additionalProperties:false).
func ValidateStrict(fm map[string]any) error {
	sv, ok := fm["schema_version"]
	if !ok {
		return fmt.Errorf("%w: missing required property 'schema_version'", ErrMissingRequiredField)
	}
	if n, isInt := sv.(int); !isInt || n != 1 {
		return fmt.Errorf("%w: unsupported schema_version %v (this reader only handles 1)", ErrMissingRequiredField, sv)
	}

	for k := range fm {
		if successor, retired := issueschema.Retired[k]; retired {
			return fmt.Errorf("%w: retired property %q (renamed to %q); %s",
				ErrMalformedFrontmatter, k, successor, issueschema.MigrateHint)
		}
		if !issueschema.Known[k] {
			return fmt.Errorf("%w: unknown property %q", ErrMalformedFrontmatter, k)
		}
	}

	// Required strings — the schema's own list (core/issueschema) minus
	// schema_version, which the version check above already answered for. The
	// record lint reads the SAME list, so the reader and the committed-ledger gate
	// cannot disagree about what a well-formed record carries.
	for _, req := range issueschema.RequiredStrings {
		v, present := fm[req]
		if !present {
			return fmt.Errorf("%w: missing required property %q", ErrMissingRequiredField, req)
		}
		if _, isStr := v.(string); !isStr {
			return fmt.Errorf("%w: %q must be a string", ErrMalformedFrontmatter, req)
		}
	}

	id := fm["id"].(string)
	if !IssIDRe.MatchString(id) {
		return fmt.Errorf("%w: id %q does not match ^iss-[0-9]+$", ErrMalformedFrontmatter, id)
	}
	if !issueschema.SlugRe.MatchString(fm["slug"].(string)) {
		return fmt.Errorf("%w: slug %q is not kebab-case", ErrMalformedFrontmatter, fm["slug"])
	}
	// A closed enum's refusal NAMES THE SET IT ACCEPTS. It has the legal values
	// in hand, and withholding them turns one round trip into several: an
	// operator told only that their value was rejected has to go looking, and
	// the field report behind iss-2609100519128005 is what that costs — a
	// refusal naming an invalid category with no accepted set, arriving as JSON
	// on a stream the operator was not reading, made them doubt the store rather
	// than the flag. The set is rendered from the ONE copy in core/issueschema,
	// so it can never drift from the membership test on the line above it.
	if !severities[fm["severity"].(string)] {
		return fmt.Errorf("%w: invalid severity %q; %s", ErrMalformedFrontmatter, fm["severity"], AcceptedValues(issueschema.Severities))
	}
	if !categories[fm["category"].(string)] {
		return fmt.Errorf("%w: invalid category %q; %s", ErrMalformedFrontmatter, fm["category"], AcceptedValues(issueschema.Categories))
	}
	if !sources[fm["source"].(string)] {
		return fmt.Errorf("%w: invalid source %q; %s", ErrMalformedFrontmatter, fm["source"], AcceptedValues(issueschema.Sources))
	}
	if strings.TrimSpace(fm["found_during"].(string)) == "" {
		return fmt.Errorf("%w: found_during must be non-empty", ErrMalformedFrontmatter)
	}

	// impact is optional but, when written, is checked against the ONE shared
	// enum (internal/core/changelog) rather than a private copy — the same enum
	// the record lint gates on and the release derivation consumes, so all three
	// can never disagree about what a legal judgement is.
	//
	// A YAML null reads as ABSENT rather than as a value. The parser is where
	// bare parts from quoted (parse.go maps every bare null spelling to "" and
	// deliberately keeps a QUOTED null as the string it spells), so by here the
	// only null left is the empty string. Re-testing frontmatter.IsNull on the
	// unquoted value would swallow the quoted spellings the parser just
	// preserved, accepting `impact: "null"` here while the record-lint blocker
	// issue_impact_valid — which reads the raw scalar — refuses it. Both gates
	// must reach the same verdict on one value.
	if v, present := fm["impact"]; present {
		s, isStr := v.(string)
		if !isStr {
			return fmt.Errorf("%w: %q must be a string", ErrMalformedFrontmatter, "impact")
		}
		if s != "" {
			if _, err := changelog.ParseImpact(s); err != nil {
				return fmt.Errorf("%w: %v", ErrMalformedFrontmatter, err)
			}
		}
	}

	// Optional scalar strings.
	for _, opt := range []string{"found_at", "lapsed_at", "details", "remedy", "suggested_fix", "wontfix_reason", "resolution"} {
		if v, present := fm[opt]; present {
			if _, isStr := v.(string); !isStr {
				return fmt.Errorf("%w: %q must be a string", ErrMalformedFrontmatter, opt)
			}
		}
	}
	// lapsed_at is optional for every category, lapse included, and an RFC 3339
	// instant whenever it is present. spc-60 made it REQUIRED on a lapse; that
	// refusal is parked (iss-2609091009111294) until the rethink of the reading
	// work settles what a lapse record must carry. The format half is checked
	// after the type loop above, so a non-string value is reported as the type
	// error it is, and it reads the ONE shared definition in core/issueschema —
	// the same one the committed-ledger gate reads.
	lapsedRaw, _ := fm["lapsed_at"].(string)
	lapsedAt := strings.TrimSpace(lapsedRaw)
	if lapsedAt != "" && !issueschema.ValidLapsedAt(lapsedAt) {
		return fmt.Errorf("%w: lapsed_at %q is not an RFC 3339 instant (want 2026-08-28T00:00:00Z)",
			ErrMalformedFrontmatter, lapsedAt)
	}

	// Optional id-list fields.
	idListFields := []struct {
		field string
		re    *regexp.Regexp
		desc  string
	}{
		{"related_intents", ItdIDRe, "itd-N"},
		{"related_specs", SpcIDRe, "spc-N"},
		{"related_issues", IssIDRe, "iss-N"},
		{"blocked_by", IssIDRe, "iss-N"},
		// The filing-time match's typed links name an issue or an intent.
		{"duplicates", LinkIDRe, "iss-N or itd-N"},
		{"refines", LinkIDRe, "iss-N or itd-N"},
	}
	for _, f := range idListFields {
		v, present := fm[f.field]
		if !present {
			continue
		}
		items, isList := v.([]string)
		if !isList {
			return fmt.Errorf("%w: %q must be a list", ErrMalformedFrontmatter, f.field)
		}
		for _, it := range items {
			if !f.re.MatchString(it) {
				return fmt.Errorf("%w: %q item %q does not match %s", ErrMalformedFrontmatter, f.field, it, f.desc)
			}
		}
	}
	if v, present := fm["synthesis_clusters"]; present {
		if _, isList := v.([]string); !isList {
			return fmt.Errorf("%w: synthesis_clusters must be a list", ErrMalformedFrontmatter)
		}
	}
	if v, present := fm["resolved_by"]; present {
		m, isMap := v.(map[string]any)
		if !isMap {
			return fmt.Errorf("%w: resolved_by must be an object", ErrMalformedFrontmatter)
		}
		for k, sv := range m {
			if k != "intent" && k != "spec" && k != "commit" {
				return fmt.Errorf("%w: resolved_by has unknown key %q", ErrMalformedFrontmatter, k)
			}
			// Type-check the sub-value: issueFromFrontmatter reads it via asString,
			// which coerces a non-string (a number, a list) to "". Without this
			// check a malformed resolved_by validates cleanly and then silently
			// loses its value on read, so the round-trip is lossy and undetected.
			if _, isStr := sv.(string); !isStr {
				return fmt.Errorf("%w: resolved_by.%s must be a string", ErrMalformedFrontmatter, k)
			}
		}
	}
	return nil
}

// ValidateInvariants enforces folder<->field invariants and filename<->id
// match, mirroring _issue_lib._validate_invariants. Assumes ValidateStrict ran.
func ValidateInvariants(fm map[string]any, status, path string) error {
	id, _ := fm["id"].(string)
	if !IssIDRe.MatchString(id) {
		return fmt.Errorf("%w: id %q does not match ^iss-[0-9]+$", ErrInvariantViolation, id)
	}
	name := filepath.Base(path)
	fnID, fnSlug, ok := recordid.SplitRecordFilename(issFamily, name)
	if !ok {
		return fmt.Errorf("%w: filename %q does not match iss-N[-slug].md", ErrInvariantViolation, name)
	}
	if fnID != id {
		return fmt.Errorf("%w: filename id %q does not match frontmatter id %q", ErrInvariantViolation, fnID, id)
	}
	// The slug is the other half of the same agreement, and it is checked the
	// same way: exactly, not leniently. Capture derives the slug once (deriveSlug
	// is where the 60-char cap is applied), normalises it once, and hands that one
	// string to both writers — reservePath builds issID+"-"+slug+".md" while
	// commitCapture stores the identical variable as fm["slug"]. The cap therefore
	// lands BEFORE the fork, so a filename is never a truncated form of a longer
	// field, and a prefix match would only license the drift this check exists to
	// catch. A name with no slug segment at all likewise disagrees: the schema
	// requires a non-empty slug, so "" is a value that names a different record.
	//
	// Without this, a record renamed by hand keeps a stale handle in its filename
	// while its frontmatter says something else, and the readers that locate a
	// record by name and the readers that trust the field part company in silence.
	// record-lint's record_schema asks the SAME question of the committed corpus
	// through the same splitter, so the drift is a red gate rather than a silent
	// skip here.
	slug, _ := fm["slug"].(string)
	if fnSlug != slug {
		return fmt.Errorf("%w: filename slug %q does not match frontmatter slug %q", ErrInvariantViolation, fnSlug, slug)
	}

	_, hasResolution := fm["resolution"]
	_, hasWontfix := fm["wontfix_reason"]
	switch status {
	case "open":
		if hasResolution {
			return fmt.Errorf("%w: resolution must not appear in open/", ErrInvariantViolation)
		}
		if hasWontfix {
			return fmt.Errorf("%w: wontfix_reason must not appear in open/", ErrInvariantViolation)
		}
	case "resolved":
		if !hasResolution {
			return fmt.Errorf("%w: resolution required in resolved/", ErrMissingRequiredField)
		}
		if isBlank(fm["resolution"]) {
			return fmt.Errorf("%w: resolution required non-empty in resolved/", ErrInvariantViolation)
		}
		if hasWontfix {
			return fmt.Errorf("%w: wontfix_reason must not appear in resolved/", ErrInvariantViolation)
		}
	case "wontfix":
		if !hasWontfix {
			return fmt.Errorf("%w: wontfix_reason required in wontfix/", ErrMissingRequiredField)
		}
		if isBlank(fm["wontfix_reason"]) {
			return fmt.Errorf("%w: wontfix_reason required non-empty in wontfix/", ErrInvariantViolation)
		}
		if hasResolution {
			return fmt.Errorf("%w: resolution must not appear in wontfix/", ErrInvariantViolation)
		}
	default:
		return fmt.Errorf("%w: unknown status directory %q", ErrInvariantViolation, status)
	}

	for _, field := range uniqueItemsFields {
		v, present := fm[field]
		if !present {
			continue
		}
		items, ok := v.([]string)
		if !ok {
			continue
		}
		seen := map[string]bool{}
		for _, it := range items {
			if seen[it] {
				return fmt.Errorf("%w: %s contains duplicate items", ErrInvariantViolation, field)
			}
			seen[it] = true
		}
	}
	return nil
}

func isBlank(v any) bool {
	s, ok := v.(string)
	return ok && strings.TrimSpace(s) == ""
}
