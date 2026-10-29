---
schema_version: 1
id: "iss-2610262035238503"
slug: "the-identity-chapter-says-bare-abcd-lint"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 568 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/19-identity.md:89"
remedy: "Reword 19-identity.md:89: the identity-positioning rule runs under bare abcd lint, which no gate invokes today (16-lint.md:144), so a blocker severity gates nothing until a gate wires it; keep the identity report's exit 0."
finding_ref: cfk-12a2d39634113eea
---

The identity chapter says bare abcd lint is the gate for positioning drift, but the lint chapter says no gate invokes bare abcd lint and the identity target runs in no gate

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/19-identity.md:89` — "and exits `0` even when it reports drift: it is a status render, and the gate is"
- End B: `.abcd/development/brief/04-surfaces/16-lint.md:144` — "No gate in this repository invokes bare `abcd lint`."

19-identity justifies the identity report's exit 0 on drift by saying the gate is bare `abcd lint`, and says the `blocker` severity turns the family into a hard lint gate. 16-lint says no gate in this repository invokes bare `abcd lint` and that the identity target runs in no gate. The drift the identity chapter treats as gated is therefore gated nowhere, and a `blocker` severity has nothing to block.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 568.
