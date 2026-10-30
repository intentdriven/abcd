---
schema_version: 1
id: "iss-2610262043581439"
slug: "dashboard-is-the-new-name-for-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1174 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:120"
remedy: "none (filed automatically)"
related_intents: [itd-2609151838312703, itd-2610032150577708]
finding_ref: cfk-14f7c7fad3bb67a8
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037281449. Same pending rename seen from the browser dashboard intent."
duplicates: [iss-2610262037281449]
---

'dashboard' is the new name for the terminal status board in itd-2609151838312703 and the browser page reached through Tailscale in itd-2610032150577708

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:120` — "that change and renaming the status board to the dashboard are captured as their own records"
- End B: `.abcd/development/intents/planned/itd-2610032150577708-a-product-thinker-opens-abcd-s-dashboard.md:21` — "A product thinker opens abcd's dashboard on their phone, iPad or computer, wherever they are, through their own private Tailscale network."

itd-2609151838312703 decides to rename the status board (bare `abcd` in a terminal) to 'the dashboard', so that the word 'status' is free. itd-2610032150577708 already uses 'abcd's dashboard' for a separate browser surface, opened on a phone, iPad or computer through Tailscale. One name would cover two different surfaces.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1174.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037281449. Same pending rename seen from the browser dashboard intent.
