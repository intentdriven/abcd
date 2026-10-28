---
schema_version: 1
id: "iss-2610262040575327"
slug: "the-glossary-says-signing-off-a"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 967 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/technical-facilitator.md:38"
remedy: "none (filed automatically)"
related_intents: [itd-2610030810370060]
finding_ref: cfk-684832595c724cee
---

The glossary says signing off a dependency is never the technical facilitator's; the Terminal-interview intent records the technical facilitator signing one off.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/technical-facilitator.md:38` — "Not for what is built, a ruling, an adoption, a dependency sign-off or an irreversible act"
- End B: `.abcd/development/intents/planned/itd-2610030810370060-a-person-can-run-abcd-s-interviews-in-a.md:79` — "the technical facilitator, signing off a new dependency"

technical-facilitator.md lists dependency sign-off among the acts that are the product thinker's, never the facilitator's, and product-thinker.md says the same. itd-2610030810370060 decision 5 records golang.org/x/term being adopted as a new direct dependency on the technical facilitator's sign-off. A dependency cannot be signed off by a role the glossary says has no authority to do it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 967.
