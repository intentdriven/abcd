---
schema_version: 1
id: "iss-2609231012180437"
slug: "itd-158-ac-2-narrowed-spc-51-says-a-provenance-record"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat/embark_render.go"
resolution: "itd-158's second criterion holds (the record is read as exempt); spc-51's per-section reclassification overreached and coverage has no per-pass attribution, so this is an erratum for spc-51"
impact: internal
---

itd-158 ac-2 narrowed: spc-51 says a provenance record carrying pass_b_exemption makes the consumer classify the section Pass B would have grounded as declared exempt — 'reported as an exemption with its reason, not enumerated among the coverage blanks — answer these first'. The delivery (PR #555) recognises the exemption at record level only: renderCoverageBlanks (internal/core/lifeboat/embark_render.go:85) prints one 'pass B (transcripts): declared exempt — <reason>' line above the blanks, and TestEmbarkUnmarkedProvenanceReadsAsBefore (embark_test.go:1167) pins that the marker changes nothing else, so every blank Pass B would have filled is still listed as work the human is left. The intent's criterion (the record is recognised as exempt rather than as an unmarked gap) holds; the spec's per-section reclassification does not, and coverage carries no per-pass attribution to make it possible. Either the spec's approach should be corrected to what shipped or the handoff should stop asking the human for Pass-B-shaped blanks

## Grounds

- pursued: the intent's criterion is met at record level and the overreach is the spec's; shown wrong if coverage gains per-pass attribution and the record-level exemption is then still the only one
