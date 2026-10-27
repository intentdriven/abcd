---
schema_version: 1
id: "iss-2610262032072943"
slug: "invariant-8-requires-an-acceptance-block"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 223 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:153"
remedy: "none (filed automatically)"
related_intents: [itd-2609211913453478]
finding_ref: cfk-011e8e588d41dad0
---

Invariant 8 requires an Acceptance block in 'every brief phase'; phase is a superseded term.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:153` — "Every brief phase has an `## Acceptance` block in the same format."
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:40` — "**Superseded terms**: `phase`, `milestone` and `roadmap` marked `status: superseded`"

itd-2609211913453478 marks `phase` as superseded. Invariant 8 still states its rule over 'every brief phase', a unit the record no longer has, so the invariant's own scope is written in retired vocabulary.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 223.
