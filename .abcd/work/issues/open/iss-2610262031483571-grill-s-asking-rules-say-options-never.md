---
schema_version: 1
id: "iss-2610262031483571"
slug: "grill-s-asking-rules-say-options-never"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 180 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:221"
remedy: "none (filed automatically)"
related_intents: [itd-99]
finding_ref: cfk-48fd93d851096803
---

GRILL's asking rules say options never recommend, but itd-99 frames every technical choice put to the thinkers as options plus a recommendation.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:221` — "options that widen and never recommend, deferral recorded as an answer"
- End B: `.abcd/development/intents/drafts/itd-99-a-team-of-product-thinkers-decides-as.md:67` — "proposal is framed as options + recommendation + stated costs"

04-naming records GRILL as the asking rules every abcd interview follows, including 'options that widen and never recommend'. itd-99's proposal contract frames any technical choice in the joint proposal as 'options + recommendation + stated costs', and its press release promises every proposal names 'its recommendation'. A choice put to the product thinkers cannot both carry a recommendation and follow a rule that never recommends.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 180.
