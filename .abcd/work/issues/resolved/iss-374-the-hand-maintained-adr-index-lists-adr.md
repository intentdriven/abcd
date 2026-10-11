---
schema_version: 1
id: "iss-374"
slug: "the-hand-maintained-adr-index-lists-adr"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: ".abcd/development/decisions/adrs/README.md"
resolution: "Corrected the ADR index (adrs/README.md) adr-44 status column from proposed to accepted, matching the record frontmatter."
impact: internal
---

The hand-maintained ADR index lists adr-44 as proposed but the record frontmatter is accepted; the index drifted after status was flipped without updating the index
## Evidence

- `.abcd/development/decisions/adrs/README.md:147` — adr-44 row status column reads `proposed`.
- `.abcd/development/decisions/adrs/0044-remote-mutation-and-caller-identity.md:4` — frontmatter `status: accepted`.

Commit `9fa5f63` flipped the frontmatter status without touching the index; row was later added (`824ef83`) already stale. Control: adr-41 agrees (proposed/proposed) in both places. The ADR index is not one of `index_drift`'s four registered indexes, so structurally ungated.

## Adversarial verdict

CONFIRMED (substantive, minor). Frontmatter is authoritative; index is the stale side. Not prior art (iss-296 resolved covered missing *rows*, not the status *value*). Fix: change README:147 status column to `accepted`.
