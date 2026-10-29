---
schema_version: 1
id: "iss-2610262033055327"
slug: "the-decision-adapter-intent-treats"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 359 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609221009495079-abcd-s-typed-judgements-can-run-through.md:28"
remedy: "In the decision-adapter intent drop embark's lesson ranking from the judgement-type lists (press-release rationale and scope bullet), or add a host judgement for it first: the shipped ranking is a deterministic term-overlap computation with no host answer to pair."
related_intents: [itd-2609221009495079]
finding_ref: cfk-0fa68fc1ec4b7f88
---

The decision-adapter intent treats embark's lesson ranking as a judgement the host answers by default; the embark page ranks lessons by term overlap, with no judgement in that step.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609221009495079-abcd-s-typed-judgements-can-run-through.md:28` — "the docs lint's present-tense advisory, embark's lesson ranking"
- End B: `.abcd/development/brief/04-surfaces/03-embark.md:59` — "ranked by term overlap against the new voyage's brief"

itd-2609221009495079's interface is (state, typed question) to (value, probability), and by default the host's own judgement answers it. Embark's lesson ranking is among the judgement types it can be configured for. The embark page says the lessons sub-verb ranks every retrospective lesson by term overlap against the new voyage's brief and writes nothing. That ranking is a deterministic computation, not a host judgement, so the shadow lab has no host answer to pair with the adapter's, and turning the type on would replace a computation rather than a judgement.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 359.
