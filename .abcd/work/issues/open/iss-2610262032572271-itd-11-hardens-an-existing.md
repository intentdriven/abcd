---
schema_version: 1
id: "iss-2610262032572271"
slug: "itd-11-hardens-an-existing"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 339 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-11-pass-b-pitfall-mitigation.md:36"
remedy: "none (filed automatically)"
related_intents: [itd-11]
finding_ref: cfk-493be1d7112c6274
---

itd-11 hardens an existing chat-distiller; disembark says the transcript-reading distiller is not built and waits on itd-11

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-11-pass-b-pitfall-mitigation.md:36` — "Replacing chat-distiller with an entirely different approach (this is hardening, not redesign)"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:114` — "Not built yet: routing such disagreements to a transcript-reading distiller that emits a finding for the unrecorded-decisions report. Viability is gated on itd-11 (draft)"

itd-11 is scoped as hardening a chat-distiller that already reads transcripts, and it rules out a redesign. The disembark chapter says that distiller is not built yet and that whether it is viable waits on itd-11. Each document waits on the other: the draft assumes the distiller exists, and the surface holds off building it until the draft lands.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 339.
