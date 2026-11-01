package lifeboat

import "reflect"

// inputs.go — the four disembark synthesis inputs as the input registry in
// the front door names them (spc-2610301431075964 step 1): each input's Go
// type, its schema version (LessonsSchemaVersion, PrinciplesSchemaVersion,
// PressReleaseSchemaVersion, ReviewSchemaVersion), a worked example held
// beside the type, and the structural half of the check its writing command
// runs, here run without writing. The other half, each entry's citations
// against the packed lifeboat, needs the lifeboat, and drops an entry rather
// than refusing the payload. Each Check takes the root a refusal is redacted
// against, as the writing command passes the lifeboat's.

// The Go types the four inputs decode into.
var (
	LessonsFileType      = reflect.TypeOf(LessonsFile{})
	PrinciplesFileType   = reflect.TypeOf(PrinciplesFile{})
	PressReleaseFileType = reflect.TypeOf(PressReleaseFile{})
	ReviewArtefactType   = reflect.TypeOf(ReviewArtefact{})
)

// LessonsExample is one `disembark graveyard --lessons-json` payload at
// LessonsSchemaVersion that passes CheckLessons.
const LessonsExample = `{
  "schema_version": 1,
  "lessons": [
    {"id": "les-auth-spike-abandoned",
     "lesson": "The bespoke auth spike was reverted once the managed provider landed; owning session crypto was not worth the maintenance.",
     "confidence": "high",
     "evidence": ["rev-9f2a1b3c4d5e", "branch-spike-auth"]}
  ]
}`

// PrinciplesExample is one `disembark principles --principles-json` payload
// at PrinciplesSchemaVersion that passes CheckPrinciples.
const PrinciplesExample = `{
  "schema_version": 2,
  "mode": "delegated",
  "prompt_version": "0.2.1",
  "principles": [
    {"id": "prn-oracle-cascade-fixed",
     "principle": "The oracle cascade is fixed; capability routing is a pre-cascade selector.",
     "confidence": "high",
     "claim_type": "causal",
     "reference": "adr-24",
     "comparison": null,
     "evidence": ["adr-24", "docs/adrs/0024-oracle-cascade.md"]}
  ]
}`

// PressReleaseExample is one `disembark press-release --press-release-json`
// payload at PressReleaseSchemaVersion that passes CheckPressRelease.
const PressReleaseExample = `{
  "schema_version": 1,
  "mode": "delegated",
  "prompt_version": "0.1.2",
  "headline": "The project's theory crosses a session boundary.",
  "subhead": "A configuration layer for development.",
  "body": "The press-release prose, composed from the packed brief and spine.",
  "quotes": [
    {"attribution": "a product thinker", "text": "The record survives the session; the lifeboat is how."}
  ],
  "evidence": ["brief/01-product/01-press-release.md", "rescue/spine.md", "principles.json"]
}`

// ReviewExample is one `disembark review --review-json` payload at
// ReviewSchemaVersion that passes CheckReview.
const ReviewExample = `{
  "schema_version": 1,
  "mode": "delegated",
  "prompt_version": "0.1.2",
  "verdict": "NEEDS_WORK",
  "findings": [
    {"id": "fnd-coverage-thin",
     "severity": "warning",
     "finding": "12 sections are blank against 7 grounded; the record is too thin to ship as-is.",
     "evidence": ["coverage.json"]}
  ]
}`

// CheckLessons runs the lessons ingest's structural checks on raw: the cap,
// the strict decode, the schema gate and the lesson cap.
func CheckLessons(repoRoot string, raw []byte) error {
	_, err := decodeLessons(repoRoot, raw)
	return err
}

// CheckPrinciples runs the principles synthesis's structural checks on raw:
// the cap, the strict decode, the schema, mode and prompt-version gates, the
// principle cap, and every entry readable as an object.
func CheckPrinciples(repoRoot string, raw []byte) error {
	_, _, err := decodePrinciples(repoRoot, raw)
	return err
}

// CheckPressRelease runs the press release's structural checks on raw: the
// cap, the strict decode, and the schema, mode and prompt-version gates.
func CheckPressRelease(repoRoot string, raw []byte) error {
	_, err := decodePressRelease(repoRoot, raw)
	return err
}

// CheckReview runs the review's structural checks on raw: the cap, the strict
// decode, the schema, mode and prompt-version gates, the verdict and the
// finding cap.
func CheckReview(repoRoot string, raw []byte) error {
	_, err := decodeReview(repoRoot, raw)
	return err
}
