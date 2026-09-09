---
schema_version: 1
id: "iss-2609090951277880"
slug: "auditismet-sums-every-section-titled-audit-notes"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/compose.go"
related_issues: ["iss-2609081941073604"]
resolution: "auditIsMet now reads the FIRST Audit Notes section only and refuses a document carrying two, the way it already refuses a negative count; the 26 shipped intents the real tree features are unchanged."
impact: fix
---

auditIsMet was narrowed to read the Audit Notes section fence-aware and to refuse a negative count, but it still walks EVERY section whose title is Audit Notes and accumulates the met and not-met counts across all of them, with no first-wins rule and no refusal of a duplicate title. An intent record carrying an honest concerns-only rollup plus a second section with the same title, at any heading level, holding a rollup that reads MET 1 therefore ends with met above zero and not-met still zero, and is lifted onto the homepage feature block: exactly the outcome the fence-awareness was added to prevent, reached by duplication instead of by fencing. Verified by reading the accumulation loop, where the title comparison is an equality test inside a loop over all sections and nothing breaks after the first match. It matters because the record body is the input the site trusts for the one testimonial the homepage carries, and a forged section is no harder to write than a forged fenced line was. Fix direction: read the FIRST Audit Notes section only and treat a second one as malformed, the same way a negative count is already treated as malformed. Detector: an intent whose honest Audit Notes rollup is concerns-only, plus a second section titled Audit Notes carrying a passing rollup, must not be featured, while the same intent with one honest passing rollup still is. Residual of iss-2609081941073604.

## Grounds

- pursued: an intent has one audit, so first-wins plus a duplicate refusal removes the forged-section route onto the homepage without narrowing the honest reading; it would be shown wrong by a real record that legitimately carries two sections of that title, which the committed tree does not and which would surface as a featured intent disappearing rather than as a false feature
