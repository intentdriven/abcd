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
resolution: "The redactor reads by mdrecord through the site walk (iss-2609250955051598), so the probe's excluded section is dropped. The verifier no longer shares that reader: it scans every body line for an ATX heading at up to three spaces of indent, treats a line as code only where every mdrecord reading agrees (FencedUnderEveryRule), lets no HTML comment hide a heading because comment text travels, and refuses a fence either exported rule leaves unclosed. The floor's view of comments: a comment is live text; an excluded heading inside one is refused, not redacted by guess. The verifier is independent of the redactor in its HEADING reader only (floorATXRe and its siblings, comments ignored, an unclosed fence refused). Its FENCE reader is mdrecord.Read under three rules, the same primitive the redactor reaches through the site walk, so a closer-rule defect in mdrecord.Read would mask the same lines under every rule and both halves would agree on it. That residue is deliberate: the one-fence ruling forbids a second fence parser, so the fence reading is shared and is guarded by mdrecord's own CommonMark tests rather than by a second reader."
impact: fix
resolved_by:
  commit: "6fb055ea"
---

The cold-reading exclusion floor can be walked through by a fence shape both of its halves misread the same way, so an excluded heading's section travels in the bundle and nothing refuses. redactExcluded finds headings through site.Sections, whose private fence toggle (iss-2609250955051598) flips a boolean on every line that starts with three backticks. verifyRedaction re-checks with bodyFenceMask, a second private backtick-only toggle, so the two agree on the wrong answer. Probe at 70daf701: a document holding a four-backtick fence that quotes a bare three-backtick line, then '## Private Notes' with a body line, then a ~~~ block holding a three-backtick line. redactExcluded with the heading Private Notes excluded returns nil error and the body line still in the output. Under CommonMark (mdrecord.Mask) the heading is live. The four-backtick block is one fence, and the three-backtick line inside the tilde block is literal. So the manifest asserts a refusal that did not happen, the silent-bypass class iss-2608301350533102 closed for a delimiter inside the frontmatter. The fix is not a one-line swap. Moving the verifier onto mdrecord.Mask would make it refuse this shape, but mdrecord also masks HTML comments, and comment text travels to the reader, so the floor's view of comments needs a decision.

## Grounds

- pursued: the capture's probe no longer travels with a nil error (TestRedactExcludedDropsTheSectionTheFenceProbeHides, RED at 6034ae70 on a scratch copy) and the verifier alone refuses the probe, a list-exited fence, a commented heading and both unclosed fences while still admitting a fenced example (TestVerifyRedactionRefusesWhatAnyReadingShowsLive, TestVerifyRedactionStillAdmitsAFencedExample); an excluded body line in redactExcluded's output with a nil error, or a refusal of a committed-corpus document, would show it wrong
