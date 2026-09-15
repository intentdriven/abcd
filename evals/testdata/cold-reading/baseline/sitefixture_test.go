package site

import "testing"

// The shape the itd-183 fidelity audit found leaking live on this repository's
// own corpus: a Go test file carrying record-shaped markdown in a raw string
// literal, headings and all. The floor's key and heading signals are record
// shapes only a markdown file carries, so this file is admitted by a row the
// floor does not parse — and before itd-194 it travelled intact while the same
// run's manifest asserted that `Audit Notes` had been refused
// (iss-2608301450065320).
//
// itd-194 does not stop it travelling. Both design documents name the shipped
// tree's code and tests as a reading's object, so the answer is disclosure
// rather than refusal: the item arrives whole and its manifest entry is marked
// `unscanned`, which is the difference between a scan that ran and a scan that
// never ran (adr-56 as refined 2026-09-02).
//
// It is corpus rather than a test: nothing under testdata/ is built by the Go
// toolchain, and this file exists to be ASSEMBLED.
const recordShapedPage = `---
id: itd-0
origin: hand-written
---

# A record-shaped page inside a Go fixture

## Press Release

ABCD-EVAL-UNSCANNED-CARRIER travels whole and marked, never examined.

## Audit Notes

The verdict the exclusion floor's heading signal names, sitting in a container
the floor does not parse.
`

func TestSiteFixtureIsCorpusAndNeverRuns(t *testing.T) {
	t.Fatal(recordShapedPage)
}
