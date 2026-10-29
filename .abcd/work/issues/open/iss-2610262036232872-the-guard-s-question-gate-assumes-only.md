---
schema_version: 1
id: "iss-2610262036232872"
slug: "the-guard-s-question-gate-assumes-only"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 644 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md:127"
remedy: "Add 17-guard.md:127 to the intent's owed brief edits (its decision 4 names only 08-abcd.md and commands/abcd.md): the menu is a third setter of the waiting-on state, so either it sets it through the abcd mode primitive and always clears it, or the gate's ownership test is amended when the menu ships."
related_intents: [itd-2610031215002409]
finding_ref: cfk-0c85a14992d529ee
---

The guard's question gate assumes only abcd mode and the gate itself set a mode that names somebody; itd-2610031215002409 has bare abcd's Terminal menu set that waiting-on state too.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/17-guard.md:127` — "naming somebody, which only `abcd mode` and this gate set and the answer resets"
- End B: `.abcd/development/intents/planned/itd-2610031215002409-under-the-board-a-short-what-next-menu.md:22` — "While the menu waits, the status line says an answer is waiting on whoever the view is for, and it clears when you answer or leave; in a Terminal this is the one thing opening the board ever changes."

17-guard treats a question without an abcd chip as abcd's when the mode names somebody, and states the premise behind that: only abcd mode and the gate set the mode, and the answer resets it. itd-2610031215002409 has the 'what next?' menu under bare abcd set the status line's waiting-on state in a Terminal while it waits, and clear it on the answer. That is a third setter outside the host's question tool, so the premise behind the gate's ownership test no longer holds.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 644.
