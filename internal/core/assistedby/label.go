// Package assistedby holds the third form of the `Assisted-by:` trailer, the
// label abcd declares on a commit it composes from record facts (ruling PC1,
// the technical facilitator, 2026-10-02): `Assisted-by: abcd:<version>`, beside
// a model's `<Vendor>:<model-version>` and `None`.
//
// It is also the one Go home of the model form's value grammar (model.go), the
// gate's TRAILER_RE, which the same two readers need.
//
// It is the label's one Go home. The implement loop writes it (the pick's
// record-only commit, the sync's merge commit, and the landing's pull-request
// body, where the models' lines follow it) and refuses a receipt that claims
// it; the site's authorship tally reads it and counts it apart from both
// assistance and None. scripts/check-attribution.sh decides the grammar
// (ABCD_RE and ABCD_ANY_RE) and runs in CI without Go, so the two halves share
// a test rather than code: TestComposedLabelGrammarMatchesTheGate.
package assistedby

import "regexp"

// Vendor is the vendor half of the label: abcd itself, never a model's vendor.
const Vendor = "abcd"

// ReleaseVersionPattern is a release version as the release workflow admits a
// tag (.github/workflows/release.yml, "Refuse a tag that is not a vX.Y.Z
// release tag").
const ReleaseVersionPattern = `v[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.-]+)?`

// DevLabel is the version half for a binary built without a release version: a
// development build, which says so rather than borrowing a release's name.
const DevLabel = "dev"

var (
	releaseVersionRe = regexp.MustCompile(`^` + ReleaseVersionPattern + `$`)
	// composedValueRe is the whole label, the value half of the gate's ABCD_RE.
	composedValueRe = regexp.MustCompile(`^` + Vendor + `:(` + DevLabel + `|` + ReleaseVersionPattern + `)$`)
	// namesAbcdRe is a value naming abcd as its vendor, in any case: the value
	// half of the gate's ABCD_ANY_RE.
	namesAbcdRe = regexp.MustCompile(`^(?i:` + Vendor + `):`)
)

// labelVersion is the version half of the label for a binary whose version is
// v: the release version when v is one, and DevLabel otherwise (the unstamped
// "dev" default, or any stamp that is not a release version, which is never
// copied into a trailer).
func labelVersion(v string) string {
	if releaseVersionRe.MatchString(v) {
		return v
	}
	return DevLabel
}

// ComposedValue is the trailer value a commit abcd composes carries, for a
// binary whose version is v.
func ComposedValue(v string) string { return Vendor + ":" + labelVersion(v) }

// isComposed reports whether a trailer value is the label exactly as the gate
// accepts it.
func isComposed(value string) bool { return composedValueRe.MatchString(value) }

// NamesAbcd reports whether a trailer value names abcd as its vendor, in any
// case and whatever follows. The gate refuses every such value that is not
// isComposed, but each one claims abcd's provenance, so none is ever read as a
// model's vendor.
func NamesAbcd(value string) bool { return namesAbcdRe.MatchString(value) }
