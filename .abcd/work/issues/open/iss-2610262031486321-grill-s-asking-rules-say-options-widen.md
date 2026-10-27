---
schema_version: 1
id: "iss-2610262031486321"
slug: "grill-s-asking-rules-say-options-widen"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 179 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:221"
remedy: "none (filed automatically)"
related_intents: [itd-97]
finding_ref: cfk-245d87d510ef5a38
---

GRILL's asking rules say options widen and never recommend, but itd-97's escalation contract hands the product thinker a proposal with a recommendation.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:221` — "options that widen and never recommend, deferral recorded as an answer"
- End B: `.abcd/development/intents/drafts/itd-97-the-facilitator-is-a-mode-not-a-person.md:67` — "point with a proposal (options, recommendation, price), exactly where duo"

04-naming records GRILL as the asking rules every abcd interview follows, including 'options that widen and never recommend'. itd-97's escalation contract has solo mode stop at every product-thinker decision point with a proposal carrying options, a recommendation and a price. A decision put to the person cannot both carry a recommendation and follow a rule that never recommends.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 179.
