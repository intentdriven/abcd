---
schema_version: 1
id: "iss-2610262033270378"
slug: "drain-names-both-the-intent-audit-s"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 408 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:553"
remedy: "none (filed automatically)"
finding_ref: cfk-01b2499c7920cf18
---

'drain' names both the intent audit's owed-review queue and the top-level verb that works the issue ledger.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:553` — "the owed set is paid on demand by the bounded drain"
- End B: `.abcd/development/brief/04-surfaces/35-drain.md:3` — "`/abcd:drain` is the verb a person types to have the open issue ledger worked unattended"

05-intent calls the capped owed form of the intent audit 'the bounded drain': the command that pays the fidelity-review debt, one audit at a time. 35-drain gives the same word to a different verb, `/abcd:drain`, which works the open issue ledger unattended through implement-loop lanes and never audits. 05-intent has to say 'the top-level drain verb' to tell the two apart, and 35-drain never mentions the other use. Across the two pages, 'drain' names two different things.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 408.
