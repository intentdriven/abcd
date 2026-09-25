---
schema_version: 1
id: "iss-2609250955207041"
slug: "the-cold-reading-exclusion-floor-can-be-walked-through-by-a"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
---

The cold-reading exclusion floor can be walked through by a fence shape both of its halves misread the same way, so an excluded heading's section travels in the bundle and nothing refuses. redactExcluded finds headings through site.Sections, whose private fence toggle (iss-2609250955051598) flips a boolean on every line that starts with three backticks. verifyRedaction re-checks with bodyFenceMask, a second private backtick-only toggle, so the two agree on the wrong answer. Probe at 70daf701: a document holding a four-backtick fence that quotes a bare three-backtick line, then '## Private Notes' with a body line, then a ~~~ block holding a three-backtick line. redactExcluded with the heading Private Notes excluded returns nil error and the body line still in the output. Under CommonMark (mdrecord.Mask) the heading is live. The four-backtick block is one fence, and the three-backtick line inside the tilde block is literal. So the manifest asserts a refusal that did not happen, the silent-bypass class iss-2608301350533102 closed for a delimiter inside the frontmatter. The fix is not a one-line swap. Moving the verifier onto mdrecord.Mask would make it refuse this shape, but mdrecord also masks HTML comments, and comment text travels to the reader, so the floor's view of comments needs a decision.
