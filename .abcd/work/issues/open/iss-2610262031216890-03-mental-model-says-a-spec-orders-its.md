---
schema_version: 1
id: "iss-2610262031216890"
slug: "03-mental-model-says-a-spec-orders-its"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 118 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:83"
remedy: "Edit glossary/core/spec.md: replace 'ordered tasks', 'numbered tasks', 'task lists' and the 'Task spc-3.1' example with steps under `## Steps`, since the step entry forbids task as a synonym."
finding_ref: cfk-8d82b1b31246d91b
---

03-mental-model says a spec orders its work as steps under ## Steps; the spec glossary entry defines a spec as broken into ordered tasks.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:83` — "each able to order its work under `## Steps`"
- End B: `.abcd/development/brief/glossary/core/spec.md:4` — "broken into ordered tasks with acceptance criteria"

03-mental-model says specs are 'each able to order its work under `## Steps`', a step being one landable piece. The glossary's spec entry defines a spec as 'broken into ordered tasks with acceptance criteria' and decomposed into numbered tasks. The canonical definition and the mental model give two names, step and task, for the one ordered unit below a spec.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 118.
