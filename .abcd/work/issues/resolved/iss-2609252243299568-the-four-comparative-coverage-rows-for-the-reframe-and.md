---
schema_version: 1
id: "iss-2609252243299568"
slug: "the-four-comparative-coverage-rows-for-the-reframe-and"
severity: "minor"
category: "drift"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "evals/coldreading_coverage_test.go"
resolution: "each comparative derived-family row is split into its leak half, declared a Gap with the reason no plant can die there, and its manifest half, caught by path; the row and gap pins move 80 to 84 and 7 to 11"
impact: internal
resolved_by:
  commit: "3d06143e"
---

The four comparative coverage rows for the reframe and derived families state a two-clause Rule (never reach the bundle AND the manifest says so) while only the manifest clause is falsified; the unfalsified leak half is disclosed in a code comment, not in the row's Gap field the matrix header names as the place for an unfalsifiable claim

## Grounds

- pursued: every coverage row states one rule and is either caught or a declared gap; a row whose Rule claims a leak its Caught mechanism does not produce would show it wrong
