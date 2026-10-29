---
schema_version: 1
id: "iss-2610262034261541"
slug: "the-what-next-menu-makes-opening-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 496 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610031215002409-under-the-board-a-short-what-next-menu.md:22"
remedy: "none (filed automatically)"
related_intents: [itd-2610031215002409]
finding_ref: cfk-b63e7b36b2667a75
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The menu is unbuilt, and the intent records the reversal and the edits it makes to the chapter and page when it ships."
---

The what-next menu makes opening the board change the status in a Terminal; the board chapter promises bare `abcd` is strictly read-only.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610031215002409-under-the-board-a-short-what-next-menu.md:22` — "in a Terminal this is the one thing opening the board ever changes"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:6` — "read-only, so it is safe to type at any moment, including the moment you are"

itd-2610031215002409 says that in a Terminal, setting the waiting-on status is the one thing opening the board ever changes, and its decisions call this a reversal of the chapter's promise. 08-abcd.md still says bare `abcd` is strictly read-only and safe to type at any moment. The two documents disagree about whether opening the board writes anything.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 496.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The menu is unbuilt, and the intent records the reversal and the edits it makes to the chapter and page when it ships.
