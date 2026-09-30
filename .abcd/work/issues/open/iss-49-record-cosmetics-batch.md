---
schema_version: 1
id: "iss-49"
slug: "record-cosmetics-batch"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "2026-07-08 multi-agent review"
found_at: ".abcd/development"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed I): What is the scope of the cleanup pass: personas role-first, the arc42 crosswalk, duplicated doctrine?"
remedy: "Waits on the cleanup-scope ruling: if full: one records-only change removes .abcd/development/intents/superseded/.gitkeep, replaces each duplicated doctrine block with a link to its one home, notes the ccpm entry's late addition in ACKNOWLEDGEMENTS.md, and adds the arc42 crosswalk page and the open-questions precedence line to the brief, proven by docs-lint and record-lint; if narrowed: do the mechanical three (the .gitkeep, the duplicates, the ccpm note) and capture the crosswalk as its own record."
---

record cosmetics batch: an orphaned .gitkeep; two verbatim doctrine blocks duplicated in the record; the ACKNOWLEDGEMENTS ccpm entry was added retroactively against its own same-change rule; the brief lacks a one-page arc42 crosswalk and an open-questions precedence line; 05-personas.md should be rewritten role-first with personas.json as the roster source of truth. Low individual impact; batched so the cleanup pass has one handle. Detector: none warranted beyond the existing lints — this is the residue after detector-shaped clusters were extracted, fixed directly per the fix-the-detector one-off bound.

## Remedy grounds (2026-09-29)

- Read at the base: the superseded folder holds thirteen records beside its .gitkeep, and 05-personas.md already selects by role with personas.json as the canonical roster (brief/01-product/05-personas.md:3), so the persona rewrite looks done and drops from the scope.
- Rejected: a detector, which the record rules out as the residue after detector-shaped clusters were extracted.
