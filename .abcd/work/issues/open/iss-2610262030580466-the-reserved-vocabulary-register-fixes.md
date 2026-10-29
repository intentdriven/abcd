---
schema_version: 1
id: "iss-2610262030580466"
slug: "the-reserved-vocabulary-register-fixes"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 68 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:187"
remedy: "In 04-naming.md (line 187) scope the row to lifeboat review verdicts (family 1) and register the other closed verdict sets in use: PROMOTE, HOLD, INCONCLUSIVE for the docs review receipt and the implement loop's reviewer words."
finding_ref: cfk-3abc0592b519dd23
---

The reserved-vocabulary register fixes review verdicts as SHIP / NEEDS_WORK / MAJOR_RETHINK; the docs chapter's saved docs review returns PROMOTE, HOLD or INCONCLUSIVE, which the register does not hold.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:187` — "Review verdicts ∈ `{SHIP, NEEDS_WORK, MAJOR_RETHINK}`"
- End B: `.abcd/development/brief/04-surfaces/10-docs.md:78` — "A receipt for HEAD that is PROMOTE lets the change proceed."

04-naming registers review verdicts as the closed enum {SHIP, NEEDS_WORK, MAJOR_RETHINK}, and requires every new term to be registered. 10-docs describes layer 2 of the doc-fidelity gate as a host-delegated docs review whose receipt reads PROMOTE, HOLD or INCONCLUSIVE. A second, unregistered set of review verdicts is in use beside the registered one, so 'review verdict' now names two different closed sets.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 68.
