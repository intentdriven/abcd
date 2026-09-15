---
schema_version: 1
id: "iss-2608311301521073"
slug: "the-read-block-eval-s-projected-carriers-draw-both-of-their"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "third delta review of itd-186"
origin: researcher-authored
production_mode: hand-written
found_at: "evals/coldreading_fixture_test.go"
resolution: "The shipped-intent carrier now draws four markers from four DISTINCT sections, and spec_id is pinned off the manifest's field column instead, because its projected text is the bare string spc-1 which also travels inside the whole spec file — no bytes.Contains marker could ever have reached it. Carriers gained a Fields list checked against the manifest, so the two checks reach different halves of the contract rather than being belt and braces. All 31 proper subsets of the five contracted fields are red and the full five is green."
impact: internal
resolved_by:
  intent: "itd-186"
  spec: "spc-64"
---

The read-block eval's projected carriers draw both of their markers from the same two fields, so the intent projection can be cut from five contracted fields to two and the lane stays green. The previous round added a second marker to each of the three projected carriers and drew every one of them from the Mechanism section, so the carrier set pins Press Release and Mechanism and nothing else. Narrowing the projection to those two drops Acceptance Criteria, Scope Conditions and spec_id, four manifest items vanish, and nothing notices; narrowing to one field alone is red. This is the third instance of one defect: the floor first measured nothing, then measured paths rather than bytes, and now measures two of five fields. The coverage matrix compounds it by carrying a row whose stated rule is the five contracted fields with a caught-by-carrier mechanism, so the newest row in the matrix is false for three of the five. Note spec_id cannot be pinned by a bundle-bytes marker at all: its projected text is the bare string spc-1, which also travels inside the whole-file spec, so the marker is satisfied without the projection; it has to be pinned off the manifest field column instead.

## Grounds

- pursued: the floor landed short three times because one of the five fields was not pinnable by the mechanism in use, so more markers of that kind could never have found it; the fix is a second mechanism, not a higher threshold. Proven by sweeping every proper subset rather than the one case that failed. This is wrong if a later fixture edit gives two carriers markers from the same field again, which the distinct-field rule is stated to prevent but nothing mechanically enforces.
