---
schema_version: 1
id: "iss-2610262031040280"
slug: "the-mental-model-defines-a-step-as-one"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 82 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:56"
remedy: "none (filed automatically)"
finding_ref: cfk-b341b06acbf8e1ff
---

The mental model defines a step as one of a spec's ordered pieces, while the implement chapter also uses 'step' for the landing's sub-stages

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:56` — "A **step** is one of the ordered pieces a spec lists under its `## Steps` section"
- End B: `.abcd/development/brief/04-surfaces/27-implement.md:383` — "the head judged, the landing step it stopped before"

The mental model defines a step as one of the ordered pieces a spec lists under ## Steps, the unit that lands as one lane. The implement chapter uses the same word for the landing's own sub-stages ('the landing step it stopped before'), next to spec steps and the step verb's run moves. So 'step' carries a sense the defining document does not give it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 82.
