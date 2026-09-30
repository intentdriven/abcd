// Package changelog is abcd's transport-agnostic release-derivation domain: the
// version and the changelog are facts derived from the records that shipped, not
// numbers and lines a human types (itd-73, spc-10).
//
// This file holds the domain's smallest and most-shared piece — impact, the
// one-word product judgement a record carries. It lives here, not in
// internal/core/lint, because the lints that GATE the judgement and the
// derivation that CONSUMES it must agree on exactly one enum; two copies of it
// would drift the moment one side gained a member. Nothing here reads git, reads
// records, or does bump arithmetic: it is a pure value type plus its boundary
// validator, so both consumers can import it without inheriting a dependency.
package changelog

import "fmt"

// Impact is the product judgement a record (an intent or an issue) declares
// about what shipping it does to the public surface. It drives the SemVer bump
// and changelog inclusion, and nothing else — it deliberately does not decide a
// Keep-a-Changelog section, because a four-value enum cannot express
// Security/Deprecated/Removed granularity and conflating the two would make the
// version hostage to editorial judgement.
type Impact string

// The four members of the enum. The string values are the wire format: they are
// what a record's `impact:` frontmatter field carries, so renaming a constant
// without changing its value is safe and changing a value is a record migration.
const (
	// ImpactAdditive is a new capability that breaks nothing.
	ImpactAdditive Impact = "additive"
	// ImpactBreaking removes or narrows something callers depend on.
	ImpactBreaking Impact = "breaking"
	// ImpactFix corrects behaviour within the existing surface.
	ImpactFix Impact = "fix"
	// ImpactInternal is invisible to users: excluded from the changelog and
	// drives no bump. It exists so plumbing work (lint internals, atomic-write
	// hardening) is not forced into a user-facing changelog by a hard default.
	ImpactInternal Impact = "internal"
)

// impactValues lists the enum in the order the error messages name it, so the
// legal set is written once and every message stays in step with the constants.
var impactValues = []Impact{ImpactAdditive, ImpactBreaking, ImpactFix, ImpactInternal}

// ParseImpact is the boundary validator for the `impact:` frontmatter field.
//
// Matching is exact: no case folding and no trimming. The shared frontmatter
// scanner (internal/core/frontmatter) already trims the value it hands over, so
// any whitespace still present here came from inside the value and is an
// authoring defect worth naming rather than silently absorbing. Case is not
// folded because the records are the source of truth for a machine-read enum;
// accepting "Additive" would make the corpus inconsistent with no gate to pull
// it back.
//
// An empty value is an error, not a default. The rule it replaces — "issues
// default to fix" — under-bumped genuinely feature-adding issues, and the
// surface guardrail only backstops breaks, never additive under-bumps. There is
// therefore no defaulting anywhere in this package: the judgement is the
// author's, made explicitly, or the gate refuses.
func ParseImpact(v string) (Impact, error) {
	if v == "" {
		return "", fmt.Errorf("impact is required and has no default: set it explicitly to one of %s", impactList())
	}
	for _, known := range impactValues {
		if Impact(v) == known {
			return known, nil
		}
	}
	return "", fmt.Errorf("invalid impact %q: want exactly one of %s (lower-case, no surrounding whitespace)", v, impactList())
}

// impactList renders the legal set for an error message.
func impactList() string {
	out := ""
	for i, v := range impactValues {
		if i > 0 {
			out += "|"
		}
		out += string(v)
	}
	return out
}

// rank is the ONE ordering of the enum: breaking > additive > fix > internal.
// internal is the bottom because it drives no bump at all, which makes it the
// correct identity for a set with nothing user-facing in it. An unrecognised
// value ranks below internal so a zero-value or hand-built Impact that never
// passed ParseImpact can neither win a maximum nor satisfy a predicate.
func (i Impact) rank() int {
	switch i {
	case ImpactBreaking:
		return 3
	case ImpactAdditive:
		return 2
	case ImpactFix:
		return 1
	case ImpactInternal:
		return 0
	default:
		return -1
	}
}

// maxImpact returns the strongest impact in a set — the single comparison in
// this package, so callers never re-derive the ordering. A set with nothing
// user-facing in it (empty, all-internal, or all-unrecognised) yields
// ImpactInternal, which reads correctly at the call site as "nothing to
// release": the result drives no bump and belongs in no changelog. The input is
// not modified.
func maxImpact(impacts []Impact) Impact {
	best := ImpactInternal
	for _, i := range impacts {
		if i.rank() > best.rank() {
			best = i
		}
	}
	return best
}

// DrivesBump reports whether a record with this impact moves the version. Only
// internal (and anything that never passed ParseImpact) does not.
func (i Impact) DrivesBump() bool { return i.rank() > ImpactInternal.rank() }

// InChangelog reports whether a record with this impact earns a changelog line.
// It coincides with DrivesBump today because outcome 8 states one rule for both
// — internal is excluded from the changelog AND drives no bump — but the two are
// named apart because they gate different pipelines (version arithmetic versus
// the prose bijection) and a future class could plausibly split them.
func (i Impact) InChangelog() bool { return i.rank() > ImpactInternal.rank() }
