---
schema_version: 1
id: "iss-332"
slug: "abcd-development-readme-md-conventions"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: ".abcd/development/README.md"
resolution: "development/README.md conventions line now states the ledger lives at ../work/issues/ (adr-32) and graduates into intents/ or principles/"
impact: internal
---

.abcd/development/README.md conventions line says issues graduate into intents/ or principles/ rather than a ledger, but the repo runs on a committed issue ledger at .abcd/work/issues/ (adr-32); the map of the durable record denies the store it governs, in the file iss-42 ruled must be current

## Evidence

- `.abcd/development/README.md:22-24` -- says issues graduate into intents/ or principles/ rather than a ledger, inside a live Conventions paragraph.
- `.abcd/work/issues/README.md:1` titled Issue ledger; adr-32 sites it in the work tier; DECISIONS.md 2026-08-19 single canonical issue store; ~300 records exist.

## Refuter verdict -- CONFIRMED (substantive, low end)

Descends from adr-30 (issues are NOT a record folder), whose compression dropped scope and destination, turning a siting claim into a substitution claim. Flagged twice in review notes, routed to iss-36/iss-38 (both resolved, neither touched it). DECISIONS.md 2026-08-03 rules this file the living index. Fix: reword to state the ledger lives in the working tier at ../work/issues/ (adr-32) and a design-significant issue graduates from it into intents/ or principles/.
