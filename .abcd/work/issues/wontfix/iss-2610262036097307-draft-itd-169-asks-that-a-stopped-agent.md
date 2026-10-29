---
schema_version: 1
id: "iss-2610262036097307"
slug: "draft-itd-169-asks-that-a-stopped-agent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 627 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-169-an-agent-loop-that-stops-says-who-it-is.md:12"
remedy: "none (filed automatically)"
related_intents: [itd-169]
finding_ref: cfk-5614f667275e2f73
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034153896. Same overlap between itd-169 and the shipped waiting-on state, seen from the mode chapter instead of the board chapter."
duplicates: [iss-2610262034153896]
---

Draft itd-169 asks that a stopped agent loop say who it is asking; 26-mode already records whom the loop waits on in the badge's waiting-on state.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-169-an-agent-loop-that-stops-says-who-it-is.md:12` — "An agent loop that stops says who it is asking and what it needs, and never simply halts"
- End B: `.abcd/development/brief/04-surfaces/26-mode.md:4` — "waiting-on state behind the status line's badge: `managed`, `facilitator` or"

itd-169 promises that an agent loop that stops says who it is asking and what it needs, and never simply halts. 26-mode holds the waiting-on state behind the status line's badge (managed, facilitator or product-thinker), the fact the agent loop is judged by, delivered by itd-200 and spc-70. The core of the draft is ground a shipped surface already covers, and neither document names the other, so the ground is claimed twice.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 627.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034153896. Same overlap between itd-169 and the shipped waiting-on state, seen from the mode chapter instead of the board chapter.
