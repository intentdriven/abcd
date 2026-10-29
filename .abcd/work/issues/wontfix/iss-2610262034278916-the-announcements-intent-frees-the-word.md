---
schema_version: 1
id: "iss-2610262034278916"
slug: "the-announcements-intent-frees-the-word"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 497 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:120"
remedy: "none (filed automatically)"
related_intents: [itd-2609151838312703]
finding_ref: cfk-b91467eba5339a1e
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037281449. The board still carries its old name only because the ruled rename has not been applied; the finding describes that interim state."
duplicates: [iss-2610262037281449]
---

The announcements intent frees the word 'status' for `abcd status` by renaming the board; the board chapter still calls the bare board the Status Board.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:120` — "The status board's rename also frees the word "status" for this command."
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:1` — "Top-Level Where-Am-I Status Board"

itd-2609151838312703 rules that `abcd status` is everyone's view of announcements and reservations, and that renaming the status board frees the word 'status' for that command. 08-abcd.md is titled the Status Board and describes a status block (Now, Next, Later), yet also points to `abcd status` as a separate verb. So 'status' names two different read-outs.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 497.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037281449. The board still carries its old name only because the ruled rename has not been applied; the finding describes that interim state.
