---
schema_version: 1
id: "iss-2608311207469829"
slug: "the-read-block-eval-never-asserts-the-positive-half-of-the-a"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "delta review of the itd-186 fix commit"
origin: researcher-authored
production_mode: hand-written
found_at: "evals/coldreading_fixture_test.go"
resolution: "Planted an UNPROJECTED-SECTION sentinel in a section of the shipped, draft and planned intents that is neither in the projection's field list nor on the exclusion floor, so deleting Fields from any of the three intent rows leaks it. Gave every excluded heading a second home on a record type that travels whole, because on a projected type the projection keeps the heading out whatever the floor says."
impact: internal
resolved_by:
  intent: "itd-186"
  spec: "spc-64"
---

The read-block eval never asserts the positive half of the assembler's field projection. All three WARM-FIELD plants sit in headings that redactExcluded strips BEFORE projection runs, so each of them tests the redaction and none tests the projection. Deleting the projection from the shipped-intent include row leaves the whole lane green, and deleting it from all three intent rows is also green. The consequence is real and was demonstrated: with one extra unprojected section added to the fixture's shipped intent, the clean binary keeps that section out of the bundle and the mutant passes it in. The corpus cannot distinguish 'the projection is positive at field granularity' from 'the projection is gone and the redaction cleaned up after it', because it holds no section that is unprojected and unexcluded at the same time. The remedy is a plant, not another floor: a sentinel in a shipped-intent and draft section that is neither projected nor on the exclusion floor.

## Grounds

- pursued: a rule is covered only when a plant dies as it is removed; a floor over the assertion is not a substitute. Watched red: deleting Fields from the shipped row, and from all three rows, now fails naming UNPROJECTED-SECTION. This is wrong if a later corpus edit makes the Residue sections projected or excluded, which would silently restore the green.
