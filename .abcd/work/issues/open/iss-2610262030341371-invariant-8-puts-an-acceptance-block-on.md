---
schema_version: 1
id: "iss-2610262030341371"
slug: "invariant-8-puts-an-acceptance-block-on"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 16 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:153"
remedy: "In 03-invariants.md invariant 8 delete 'Every brief phase has an `## Acceptance` block in the same format' and say the brief's press release and surface chapters carry the `## Acceptance` block, as 01-product/03-mental-model.md states; phases were retired by adr-2609212115255771. Folds in iss-2610262031228119 and iss-2610262032072943."
finding_ref: cfk-451dfed62851cdac
---

Invariant 8 puts an Acceptance block on every 'brief phase'; the mental model puts it on the press release and surface chapters, with phases retired

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:153` — "Every brief phase has an `## Acceptance` block in the same format."
- End B: `.abcd/development/brief/01-product/03-mental-model.md:85` — "The brief's press release and surface chapters carry an `## Acceptance` block in the same format."

Invariant 8 says every brief phase has an `## Acceptance` block. The mental model says the brief's press release and surface chapters carry that block, and states that the phase is retired. The invariant uses a retired unit for the home of brief-level acceptance, so the invariant and the model name different homes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 16.
