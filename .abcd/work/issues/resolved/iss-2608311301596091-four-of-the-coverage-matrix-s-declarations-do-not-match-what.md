---
schema_version: 1
id: "iss-2608311301596091"
slug: "four-of-the-coverage-matrix-s-declarations-do-not-match-what"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "third delta review of itd-186"
origin: researcher-authored
production_mode: hand-written
found_at: "evals/coldreading_coverage_test.go"
resolution: "Both per-position asymmetry rows now state the two-part falsifier they actually need, matching the family rows above them. The body redaction's markdown-only scope has a row and a corpus file that makes it falsifiable: a source file carrying an unterminated fence at the left margin, so deleting the early return refuses the assembly. The agents and evals structural-deny segments have rows of their own, and the first-row-wins tie-break has a gap row with its reason."
impact: internal
resolved_by:
  intent: "itd-186"
  spec: "spc-64"
---

Four of the coverage matrix's declarations do not match what the eval does. The two per-position asymmetry rows state a one-part falsifier and claim a leak, but run verbatim both produce a refusal at exit 2 because the path-shaped exclusion entry fires first, and the plant plays no part; the leak appears only under the two-part mutation the rows do not state, which is the pattern the matrix documents twenty lines above for the family rows and then does not apply here. The markdown-only scope of the body redaction is named by no row at all, neither as a catch nor as a gap, and it is falsifiable in two corpus lines: a config file admitted by the root row carries its keys unredacted and the oracle reports three violations, so a runnable falsifier is simply unexercised, which by the matrix's own rule is the worse case than a declared gap. The structural-deny mutation for two of the four denied segments is named by no row, though both are caught. And the first-row-wins tie-break is stated as contract in the assembler and rowed nowhere, which is exactly what a gap row is for. A mislabelled catch is the same class as a mislabelled gap, one notch down, and this branch's own standard says a mislabelled gap would be worse than no matrix.

## Grounds

- pursued: a matrix row that misstates its own falsifier is worse than no row, because the next reader runs it, sees the wrong failure and concludes the rule is covered. Every one of the four was run verbatim after the fix. This is wrong if a row's falsifier drifts from the assembler again, which nothing mechanical catches — the matrix cannot check that a stated mutation still applies.
