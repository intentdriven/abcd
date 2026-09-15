---
schema_version: 1
id: "iss-2608301251391469"
slug: "testrawheadingcaseisfolded-does-not-discriminate-the-fold-it"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-183-round-9-ruthless"
found_at: "internal/core/reading/assemble_test.go"
resolution: "The fold test gains the row that discriminates it: a mixed-case close with no blank line after it, where the close is the only bound there is. Byte equality in place of the fold now turns the test red."
impact: internal
resolved_by:
  intent: "itd-183"
---

TestRawHeadingCaseIsFolded does not discriminate the fold it is named for and the entire package stays green when the guard is removed

Found by the round-9 adversarial RUTHLESS review of build/itd-183, by mutation.
INTRODUCED BY 2225d6cb.

Replacing `strings.EqualFold(rest[m[2]:m[3]], name)` at project.go:595 with
byte equality leaves the ENTIRE `internal/core/reading` suite green. All three
rows survive for reasons unrelated to folding: the opener capture is now raw
(`<H2>` yields name `H2`, matching `</H2>` byte-for-byte), and the soft
blank-line bound rescues the mixed-case row -- every row has a blank line after
the close.

The guard is nonetheless load-bearing: under the mutant,
`<h2>Audit Notes</H2>` followed by trailing prose with NO blank line leaks end
to end, and refuses at HEAD.

This is the fourth mutation-vacuous guard the two reviewers have found across
this workstream, and the second found on a test written in the very round that
was closing two others. The lesson stands and is now in every builder brief:
a test that stays green when its guard is removed is decoration, and the only
way to know is to remove the guard.

Remedy: add a row with no blank line between the mixed-case close and the
following prose, e.g. `<h2>Audit Notes</H2>\nand trailing prose <sentinel>`.
