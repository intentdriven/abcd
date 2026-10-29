---
schema_version: 1
id: "iss-2610262034034042"
slug: "itd-21-has-the-scaffold-s-starter-intent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 466 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-21-no-lifeboat-scaffolding.md:32"
remedy: "none (filed automatically)"
related_intents: [itd-21]
finding_ref: cfk-7ada5e8184cb0a19
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031362851. Same draft edit as iss-2610262031362851, set against a different statement of the same current state."
duplicates: [iss-2610262031362851]
---

itd-21 has the scaffold's starter intent captured as itd-1; the intent create mints a timestamp-numeric id

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-21-no-lifeboat-scaffolding.md:32` — "capturing the project's mission as `itd-1`"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:185` — "Mints a timestamp-numeric itd-N id through the shared record-id seam"

itd-21 promises a starter intent, kicked off through the intent interview, that captures the project's mission as itd-1. The intent surface's create mints a timestamp-numeric itd-N through the shared record-id seam, so an intent created through it is never itd-1. The scaffold could keep its promise only by bypassing the create path.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 466.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031362851. Same draft edit as iss-2610262031362851, set against a different statement of the same current state.
