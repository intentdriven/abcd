---
schema_version: 1
id: "iss-2610262032168367"
slug: "itd-80-moves-an-intent-to-shipped-when"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 245 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:34"
remedy: "none (filed automatically)"
related_intents: [itd-80]
finding_ref: cfk-e6c41d49576ea307
refines: [iss-2609100508566552]
---

itd-80 moves an intent to shipped when its linked spec closes; invariant 17 says a close never ships an intent that another open spec still names

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:34` — "on spec close, detect the linked intent via the immutable link, move it `planned → shipped` with an atomic same-filesystem rename"
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:171` — "closing a spec never ships an intent that still has an open spec, the intent moves `planned/` → `shipped/` on the close after which no open spec names it"

itd-80's reconcile finds the linked intent through the immutable link when a spec closes, and moves it planned → shipped. Invariant 17 makes the intent↔spec relation 1:n: closing a spec never ships an intent that still has an open spec, and the move happens only on the close after which no open spec names it. The two reconcile rules disagree about which close ships the intent.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 245.
