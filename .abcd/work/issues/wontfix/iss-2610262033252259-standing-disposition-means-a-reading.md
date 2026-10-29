---
schema_version: 1
id: "iss-2610262033252259"
slug: "standing-disposition-means-a-reading"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 403 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:331"
remedy: "none (filed automatically)"
finding_ref: cfk-a6d851b0cc3922c0
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The shared word is a recorded glossary choice, and the two chapters name different records with their own state sets, so no statement conflicts."
---

'Standing disposition' means a reading item's answer in 06-capture and a scope condition's current standing in 05-intent.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:331` — "every scope condition the intent carries, with its standing disposition and the block that disposition came from"
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:217` — "makes the standing disposition the one no sibling supersedes"

In 06-capture a disposition is a record answering one reading item (accepted, declined, held, rejected), and the standing disposition is the one no sibling supersedes. In 05-intent a condition disposition is a dated Audit Notes block on a shipped intent's scope condition (survived, narrowed, falsified, untested), read back as its standing disposition. Such a block can be occasioned by a reading item that has a disposition of its own. So one term names two records with different state sets, and they meet at the same reading item.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 403.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The shared word is a recorded glossary choice, and the two chapters name different records with their own state sets, so no statement conflicts.
