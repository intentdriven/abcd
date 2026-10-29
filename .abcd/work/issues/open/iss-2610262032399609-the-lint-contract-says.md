---
schema_version: 1
id: "iss-2610262032399609"
slug: "the-lint-contract-says"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 298 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/06-lint.md:70"
remedy: "Edit 06-lint.md section 5 closing sentence and the first acceptance bullet in section 6: remove the claim that the documentation-auditor audits the contract on every disembark; state the contract is held by the armed gates (record-lint, docs-lint) and the surface_coverage checks."
finding_ref: cfk-c92970006c432a65
---

the lint contract says documentation-auditor audits it on every disembark; disembark says packing dispatches no agents

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/06-lint.md:70` — "The `documentation-auditor` agent (per `01-agents.md`) audits this contract on every disembark"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:75` — "Packing is one deterministic Go run. It dispatches no agents and runs no model passes"

The lint contract says, in the present tense, that the documentation-auditor agent audits the contract on every disembark, and its acceptance depends on that run. The disembark chapter says packing is one deterministic Go run that dispatches no agents. Its only agents are the four synthesis agents, and documentation-auditor is not one of them. No disembark step runs the audit the lint contract relies on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 298.
