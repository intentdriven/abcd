---
schema_version: 1
id: "iss-2610262032580449"
slug: "itd-37-extends-principle-distiller-into"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 342 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-37-modification-grammar.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-37]
finding_ref: cfk-8b3b271fa8056cb2
---

itd-37 extends principle-distiller into a curator that writes memory pages from specs when they complete; disembark says its lifeboat agents, principle-distiller among them, read the packed lifeboat's files

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-37-modification-grammar.md:25` — "At spec completion, `principle-distiller` (the role-extended curator from [itd-36] (../shipped/itd-36-memory-unification.md)) extracts the `## Modification Grammar` section into two memory pages"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:307` — "The four lifeboat agents read the packed lifeboat's files"

itd-37 has `principle-distiller` read a spec's Modification Grammar section when the spec completes and write two memory pages from it. That role needs no lifeboat. The disembark chapter counts principle-distiller among the four lifeboat agents and says they read the packed lifeboat's files. One agent cannot both work only from a packed lifeboat and curate specs in the live repository unless one of the two descriptions changes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 342.
