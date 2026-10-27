---
schema_version: 1
id: "iss-2610262039567036"
slug: "status-names-both-the-board-s-now-next"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 897 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/39-status.md:3"
remedy: "none (filed automatically)"
related_intents: [itd-2609212103568351]
finding_ref: cfk-2364a6c066cbd0ef
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038029972. The only difference is which shipped intent states the view's name; the fix is the same disambiguation note."
duplicates: [iss-2610262038029972]
---

'Status' names both the board's Now / Next / Later block and `/abcd:status` with its `## STATUS` block of announcements.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/39-status.md:3` — "`/abcd:status` is the people's view of what everyone else is doing on the"
- End B: `.abcd/development/intents/shipped/itd-2609212103568351-the-bare-abcd-status-board-and-the-site.md:58` — "The word roadmap is retired; this block is called status."

itd-2609212103568351 rules that the Now / Next / Later block that replaced the roadmap 'is called status'. 39-status defines `/abcd:status` as the people's view of announcements, reservations and releases being cut on the shared remote, and gives the prompt hook a `## STATUS` block that carries those announcements. One word names two unrelated renders, so 'the status block' could mean either.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 897.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038029972. The only difference is which shipped intent states the view's name; the fix is the same disambiguation note.
