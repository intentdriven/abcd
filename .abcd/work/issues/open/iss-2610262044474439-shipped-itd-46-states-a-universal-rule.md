---
schema_version: 1
id: "iss-2610262044474439"
slug: "shipped-itd-46-states-a-universal-rule"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1230 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-46, itd-2610031215002409]
finding_ref: cfk-8fd46c40a9229c87
---

Shipped itd-46 states a universal rule that a bare command never changes state; in itd-2610031215002409, opening the bare `abcd` board writes the waiting-on status

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:18` — "The bare command (zero arguments) keeps doing what the universal abcd convention says it must: shows status and help, never mutates state."
- End B: `.abcd/development/intents/planned/itd-2610031215002409-under-the-board-a-short-what-next-menu.md:22` — "in a Terminal this is the one thing opening the board ever changes"

itd-46 states, as the universal abcd convention, that a bare command shows status and help and never changes state. In itd-2610031215002409, the menu under the bare `abcd` board sets the status line's waiting-on state in a Terminal. The record calls this the one thing opening the board ever changes. It reverses the read-only promise in the board's own chapter and page, but not the universal convention in itd-46. So the two records disagree on whether any bare command may write.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1230.
