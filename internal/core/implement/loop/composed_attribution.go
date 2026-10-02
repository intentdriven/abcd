package loop

// composed_attribution.go is the trailer of the commits the loop composes from
// record facts (ruling PC1, the technical facilitator, 2026-10-02): the pick's
// record-only commit (pickcommit.go) and the sync's merge commit (sync.go).
// Their text is computed by abcd from the run's state and records, so no model
// wrote it and `Assisted-by: None` (no tool touched it) would be a false
// disclosure. They carry the third form the attribution convention accepts,
// `Assisted-by: abcd:<version>`, beside a model's `<Vendor>:<model-version>`
// and None.
//
// The landing's records commit (land.go) is not one of them: it carries prose
// a model composed, so it names the models the lane's receipts reported, and a
// receipt that names abcd as its model is refused there rather than copied.

import (
	"regexp"

	"github.com/intentdriven/abcd/internal/core"
)

// releaseVersionPattern is a release version as the release workflow admits a
// tag (.github/workflows/release.yml, "Refuse a tag that is not a vX.Y.Z
// release tag"). TestComposedLabelGrammarMatchesTheGate ties it to the gate's
// ABCD_RE (scripts/check-attribution.sh).
const releaseVersionPattern = `v[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.-]+)?`

var releaseVersionRe = regexp.MustCompile(`^` + releaseVersionPattern + `$`)

// devLabel is the label of a binary built without a release version: a
// development build, which says so rather than borrowing a release's name.
const devLabel = "dev"

// composedLabel is the version half of the label for a binary whose version
// is v: the release version when v is one, and devLabel otherwise (the
// unstamped "dev" default, or any stamp that is not a release version, which is
// never copied into a trailer).
func composedLabel(v string) string {
	if releaseVersionRe.MatchString(v) {
		return v
	}
	return devLabel
}

// composedAssistedBy is the trailer line of a commit the loop composes from
// record facts, naming the running binary's version (what `abcd version`
// prints).
func composedAssistedBy() string {
	return "Assisted-by: abcd:" + composedLabel(core.Version)
}
