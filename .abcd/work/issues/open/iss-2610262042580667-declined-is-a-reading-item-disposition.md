---
schema_version: 1
id: "iss-2610262042580667"
slug: "declined-is-a-reading-item-disposition"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1106 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/disposition.md:4"
remedy: "none (filed automatically)"
related_intents: [itd-179]
finding_ref: cfk-3baa805a2785b4e7
---

'declined' is a reading-item disposition state in the glossary and a capture-triage grounds value in itd-179.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/disposition.md:4` — "in one of four states: accepted, rejected, declined or held."
- End B: `.abcd/development/intents/shipped/itd-179-the-reasoning-behind-what-was-pursued-no.md:22` — "`deferred`, `declined` — and the grounds name the conjecture being acted"

The glossary makes 'declined' one of the four disposition states (the widening proposal was admissible and the researcher chose otherwise) and asks that disposition vocabulary not be mixed with the issue ledger's. itd-179 gives the readiness gate and capture's triage routes a grounds vocabulary of 'pursued', 'deferred', 'declined', where 'declined' means a triage choice on an issue or intent. One token names two different acts on two different record kinds, so a reader meeting 'declined' on a ledger record must know which vocabulary wrote it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1106.
