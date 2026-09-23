package capture

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/grounds"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// knownFields is the additionalProperties:false allow-list from
// issue.schema.json — the ONE copy in core/issueschema, the same set the record
// lint reads, so the reader and the committed-ledger gate cannot disagree about
// which keys a well-formed record may carry.
var knownFields = issueschema.Known

// uniqueItemsFields are the array properties issue.schema.json flags
// uniqueItems:true.
var uniqueItemsFields = []string{"related_intents", "related_specs", "related_issues", "synthesis_clusters", "blocked_by"}

// validateStrict validates a frontmatter map against the issue schema. It
// special-cases schema_version first (mirrors _validate_strict) and rejects
// unknown keys (additionalProperties:false).
func validateStrict(fm map[string]any) error {
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
		if !knownFields[k] {
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
	if !reIssID.MatchString(id) {
		return fmt.Errorf("%w: id %q does not match ^iss-[0-9]+$", ErrMalformedFrontmatter, id)
	}
	if !reSlug.MatchString(fm["slug"].(string)) {
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
	if !validSeverities[Severity(fm["severity"].(string))] {
		return fmt.Errorf("%w: invalid severity %q; %s", ErrMalformedFrontmatter, fm["severity"], acceptedValues(issueschema.Severities))
	}
	if !validCategories[Category(fm["category"].(string))] {
		return fmt.Errorf("%w: invalid category %q; %s", ErrMalformedFrontmatter, fm["category"], acceptedValues(issueschema.Categories))
	}
	if !validSources[Source(fm["source"].(string))] {
		return fmt.Errorf("%w: invalid source %q; %s", ErrMalformedFrontmatter, fm["source"], acceptedValues(issueschema.Sources))
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
	for _, opt := range []string{"found_at", "lapsed_at", "details", "suggested_fix", "wontfix_reason", "resolution"} {
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
	lapsedAt := strings.TrimSpace(asString(fm["lapsed_at"]))
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
		{"related_intents", reItdID, "itd-N"},
		{"related_specs", reSpcID, "spc-N"},
		{"related_issues", reIssID, "iss-N"},
		{"blocked_by", reIssID, "iss-N"},
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

// validateInvariants enforces folder<->field invariants and filename<->id
// match, mirroring _issue_lib._validate_invariants. Assumes validateStrict ran.
func validateInvariants(fm map[string]any, status State, path string) error {
	id, _ := fm["id"].(string)
	if !reIssID.MatchString(id) {
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
	case StateOpen:
		if hasResolution {
			return fmt.Errorf("%w: resolution must not appear in open/", ErrInvariantViolation)
		}
		if hasWontfix {
			return fmt.Errorf("%w: wontfix_reason must not appear in open/", ErrInvariantViolation)
		}
	case StateResolved:
		if !hasResolution {
			return fmt.Errorf("%w: resolution required in resolved/", ErrMissingRequiredField)
		}
		if isBlank(fm["resolution"]) {
			return fmt.Errorf("%w: resolution required non-empty in resolved/", ErrInvariantViolation)
		}
		if hasWontfix {
			return fmt.Errorf("%w: wontfix_reason must not appear in resolved/", ErrInvariantViolation)
		}
	case StateWontfix:
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

// issueFromFrontmatter builds a typed Issue from a validated frontmatter map.
func issueFromFrontmatter(fm map[string]any, status State, path, body string) Issue {
	iss := Issue{
		SchemaVersion: fm["schema_version"].(int),
		ID:            asString(fm["id"]),
		Slug:          asString(fm["slug"]),
		Severity:      Severity(asString(fm["severity"])),
		Category:      Category(asString(fm["category"])),
		Source:        Source(asString(fm["source"])),
		FoundDuring:   asString(fm["found_during"]),
		FoundAt:       asString(fm["found_at"]),
		LapsedAt:      asString(fm["lapsed_at"]),
		Grounds:       groundsEntries(body),
		Resolution:    asString(fm["resolution"]),
		WontfixReason: asString(fm["wontfix_reason"]),
		Status:        status,
		Path:          path,
		Body:          body,
	}
	iss.RelatedIntents = asStrList(fm["related_intents"])
	iss.RelatedSpecs = asStrList(fm["related_specs"])
	iss.RelatedIssues = asStrList(fm["related_issues"])
	iss.BlockedBy = asStrList(fm["blocked_by"])
	if rb, ok := fm["resolved_by"].(map[string]any); ok {
		iss.ResolvedBy = &ResolvedBy{
			Intent: asString(rb["intent"]),
			Spec:   asString(rb["spec"]),
			Commit: asString(rb["commit"]),
		}
	}
	return iss
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asStrList(v any) []string {
	l, _ := v.([]string)
	if len(l) == 0 {
		return nil
	}
	return l
}

// groundsEntries reads the record's `## Grounds` section through core/grounds's
// own reader, so what an ENTRY is has one definition and a bullet one writer
// appends is a bullet the other finds.
//
// It asks ParseSection where the intent half asks ParseSectionAboveFloor: the
// two families read the same section by the same rules and part company on the
// FLOOR alone. A wontfix stamps its grounds from a reason whose own contract is
// merely non-empty, so applying the floor here would drop entries the ledger has
// always accepted and leave a surface reporting no recorded grounds about a
// record that visibly carries one.
func groundsEntries(body string) []string {
	entries := grounds.ParseSection(body)
	if len(entries) == 0 {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, g := range entries {
		out = append(out, g.String())
	}
	return out
}

// acceptedValues renders a closed enum's legal set for a refusal message.
//
// One helper rather than three literal lists, so the message and the membership
// test read the same slice: a value added to core/issueschema appears in the
// refusal without anyone remembering to add it.
func acceptedValues(vals []string) string {
	return "accepted values: " + strings.Join(vals, " | ")
}
