---
schema_version: 1
id: "iss-2610262044392775"
slug: "the-what-next-menu-makes-bare-abcd-write"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1221 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610031215002409-under-the-board-a-short-what-next-menu.md:88"
remedy: "none (filed automatically)"
related_intents: [itd-2610031215002409, itd-2609150819445595]
finding_ref: cfk-10d96b81929a60f6
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The convention sentence is made inside a shipped intent about its own verb; the planned exception does not touch that verb."
---

The what-next menu makes bare `abcd` write, against the stated convention that bare invocations are read-only

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610031215002409-under-the-board-a-short-what-next-menu.md:88` — "the menu is the one exception, in a Terminal only, and the brief chapter 04-surfaces/08-abcd.md and the plugin page commands/abcd.md say so"
- End B: `.abcd/development/intents/shipped/itd-2609150819445595-a-debt-nothing-lists-owed-fidelity.md:44` — "bare invocations are read-only by convention here"

itd-2610031215002409 rules that the menu under bare `abcd` writes the waiting-on status in a Terminal, and lists only the board chapter and the abcd command page as the promises it reverses. itd-2609150819445595 relies on the general convention that bare invocations are read-only. The convention as stated no longer holds, and the planned record does not amend it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1221.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The convention sentence is made inside a shipped intent about its own verb; the planned exception does not touch that verb.
