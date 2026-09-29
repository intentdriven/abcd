package changelog

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/launch"
)

// Derivation is the deterministic outcome of a release cut: what the next
// version is, which records decide it, and — when it cannot be decided — why.
//
// A refusal is modelled as a VALUE, not an error, in the same shape as
// launch.RetentionPlan: "this cut cannot be derived" is a legitimate result a
// read-only preview must render, not an exceptional failure. Errors are reserved
// for "the repository could not be read at all".
//
// Three states are deliberately distinguishable, because a caller that conflated
// them would write a wrong CHANGELOG heading:
//
//	Refused          — do not derive; RefusalReason says what to fix.
//	!Refused, !Bumped — nothing to release; write no heading.
//	!Refused, Bumped  — Next/NextTag are the release.
type Derivation struct {
	// Base is the version of the anchor tag the cut is measured from.
	Base launch.Semver
	// BaseTag is Base as a git tag ("v0.3.0"), empty when no anchor resolved.
	BaseTag string
	// Records is the cut: what entered and left the terminal record folders.
	Records RecordSet
	// Bump is the strongest impact in the cut — the judgement that decides the
	// version. ImpactInternal means nothing user-facing shipped.
	Bump Impact
	// Next is the derived version; only meaningful when Bumped.
	Next launch.Semver
	// NextTag is Next as a git tag, empty when nothing is released.
	NextTag string
	// Bumped reports whether the cut moves the version at all.
	Bumped bool
	// Refused reports that the cut must not be derived.
	Refused bool
	// RefusalKind classifies the refusal for a caller that must act on it. The
	// prose in RefusalReason is for a human; this is for the ship verb, which
	// decides whether to name a record, a version, or a missing tag — and must
	// not pattern-match English to tell them apart.
	RefusalKind RefusalKind
	// RefusalReason names what to fix; empty unless Refused.
	RefusalReason string
}

// RefusalKind is the machine-readable classification of a derivation refusal.
// The string values are what a machine-readable front door emits, so renaming a
// constant is safe and changing a value is a contract change.
type RefusalKind string

// The refusal kinds, one per fail-closed case in Derive.
const (
	// RefusalNone is the zero value: this derivation did not refuse.
	RefusalNone RefusalKind = ""
	// RefusalNoReleaseTag: no immutable base to measure the cut from.
	RefusalNoReleaseTag RefusalKind = "no-release-tag"
	// RefusalReleaseInFlight: the newest CHANGELOG heading is ahead of the
	// newest tag, so a release is between its merge and its tag.
	RefusalReleaseInFlight RefusalKind = "release-in-flight"
	// RefusalUnlabelledRecord: a record ADDED by the cut carries no valid impact.
	RefusalUnlabelledRecord RefusalKind = "unlabelled-record"
)

// Derive runs the whole deterministic release derivation over the repository at
// root and writes nothing. It is the one composition of this package's parts:
// resolve the anchor tag, refuse a release already in flight, diff the record
// end-states, then apply the version policy to the strongest impact in the cut.
//
// It refuses (rather than deriving a number that would be wrong) in three cases,
// each fail-closed:
//
//   - No release tag. There is no immutable base; inventing one would report
//     every record ever written as this release's contents.
//   - The newest CHANGELOG heading is ahead of the newest tag. auto-release.yml
//     tags AFTER the ship PR merges, so this is the post-merge/pre-tag window:
//     the heading and the tag describe different releases and the base is
//     mismatched. The next cut derives correctly once the tag lands.
//   - A record ADDED by the cut carries no valid impact. An unlabelled record
//     ranks below every real impact, so deriving over it would silently
//     under-bump a release that may contain a break. The lints gate this at the
//     record lifecycle; this is the backstop at the cut, and it names both the
//     record and the file to edit. The removed side cannot refuse — its blob is
//     read from the anchor tag's immutable tree, so an unlabelled one is
//     unfixable by definition (see RecordSet.UnlabelledAdded).
func Derive(root string) (Derivation, error) {
	var d Derivation

	base, hasTag, err := LatestReleaseTag(root)
	if err != nil {
		return Derivation{}, err
	}
	if !hasTag {
		return refuse(d, RefusalNoReleaseTag, "no release tag found — a cut needs an immutable base (tag the current release first)"), nil
	}
	d.Base, d.BaseTag = base, base.Tag()

	inFlight, reason, err := releaseInFlight(root, base)
	if err != nil {
		return Derivation{}, err
	}
	if inFlight {
		return refuse(d, RefusalReleaseInFlight, reason), nil
	}

	records, err := ShippedSince(root, d.BaseTag)
	if err != nil {
		return Derivation{}, err
	}
	d.Records = records

	if unlabelled := records.UnlabelledAdded(); len(unlabelled) > 0 {
		names := make([]string, 0, len(unlabelled))
		for _, rec := range unlabelled {
			names = append(names, rec.ID+" ("+rec.Path+": "+rec.ImpactErr+")")
		}
		return refuse(d, RefusalUnlabelledRecord, "records added by the cut carry no valid impact: "+strings.Join(names, "; ")), nil
	}

	d.Bump = records.Impact()
	next, bumped := DeriveNext(base, d.Bump)
	if bumped {
		d.Next, d.NextTag, d.Bumped = next, next.Tag(), true
	}
	return d, nil
}

// refuse stamps a refusal on a partially-filled derivation, clearing anything
// that could read as a derived release. Whatever was resolved before the refusal
// (the anchor, the records) is kept, because it is what the operator needs to
// see to fix the cut.
func refuse(d Derivation, kind RefusalKind, reason string) Derivation {
	d.Refused = true
	d.RefusalKind = kind
	d.RefusalReason = reason
	d.Bumped = false
	d.Next = launch.Semver{}
	d.NextTag = ""
	return d
}

// ReleaseInFlight reports whether a release sits between its cut and its tag —
// the newest CHANGELOG heading ahead of the newest release tag — and the reason
// Derive refuses it with. It reads only the tags and the CHANGELOG, so a caller
// can ask it before work that would otherwise refuse first on a symptom of the
// same window: the ship's payload parity diff, whose baseline advice names a flag
// the ship does not take (iss-2609252117203691). No tag is not in flight; Derive
// refuses that case on its own.
func ReleaseInFlight(root string) (bool, string, error) {
	base, hasTag, err := LatestReleaseTag(root)
	if err != nil || !hasTag {
		return false, "", err
	}
	return releaseInFlight(root, base)
}

// releaseInFlight is the one statement of the in-flight rule, against base, the
// newest release tag.
func releaseInFlight(root string, base launch.Semver) (bool, string, error) {
	heading, hasHeading, err := LatestChangelogVersion(root)
	if err != nil {
		return false, "", err
	}
	if hasHeading && launch.CoreGreater(heading, base) {
		return true, "release " + heading.Tag() + " in flight — tag pending (the newest CHANGELOG heading is ahead of " + base.Tag() + ")", nil
	}
	return false, "", nil
}
