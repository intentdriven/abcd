---
schema_version: 1
id: "iss-2610262030262031"
slug: "the-press-release-says-disembark-pack"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:23"
remedy: "none (filed automatically)"
finding_ref: cfk-460cc04582bff3e5
---

The press release says disembark pack runs three agent passes; the disembark chapter says pack is one deterministic run with no model passes

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:23` — "runs three passes (settled artefacts → targeted chat retrieval → distil/compose/audit)"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:75` — "Packing is one deterministic Go run. It dispatches no agents and runs no model passes: the synthesis artefacts are written later, by the synthesis sub-verbs, over an already-packed lifeboat."

The press release describes pack as three passes, including targeted chat retrieval and a distil/compose/audit pass. The disembark chapter says packing dispatches no agents, runs no model passes, and leaves synthesis to later sub-verbs over an already-packed lifeboat. A pack cannot both run a distil/compose/audit pass and run no model passes, so anyone building from the press release would build machinery the surface rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1.
