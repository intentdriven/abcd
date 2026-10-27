---
schema_version: 1
id: "iss-2610262044520698"
slug: "itd-55-s-foundations-auditor-claims-to"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1236 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-55-first-principles-foundations-auditor.md:30"
remedy: "none (filed automatically)"
related_intents: [itd-55, itd-42]
finding_ref: cfk-6954ce1f19dcefc8
---

itd-55's foundations auditor claims to dig out an intent's unstated premises; the grill shipped by itd-42 already questions hidden assumptions

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-55-first-principles-foundations-auditor.md:30` — "excavate the implicit premises as a load-ranked inventory"
- End B: `.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:34` — "It still grills vague terms and hidden assumptions"

itd-55 adds an audit that digs out the unstated premises of a document, including an intent's Why This Matters. It sets itself apart from the grill, which it says checks criteria only. itd-42 says the grill still questions vague terms and hidden assumptions, on top of its new coherence tier. Both surfaces claim an intent's unstated premises, and itd-55's line between itself and the grill ignores the questioning of hidden assumptions that itd-42 delivered.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1236.
