---
schema_version: 1
id: "iss-2610262031074393"
slug: "the-mental-model-gives-intent-auditor-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 89 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:85"
remedy: "Edit 03-mental-model.md: remove the discipline role from the intent-auditor sentence and mark the third (reclassification) role as a design target, as 05-internals/01-agents.md does."
finding_ref: cfk-c6b5f1fa00b592de
---

The mental model gives intent-auditor a discipline role that checks every spec, but the agent catalog lists three roles and none of them is a discipline role

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:85` — "the same agent's discipline role checks every spec against the active disciplines"
- End B: `.abcd/development/brief/05-internals/01-agents.md:135` — "The auditor is one agent with three roles, sharing its prompt scaffolding, oracle resolution and receipts."

The mental model says the intent-auditor's discipline role checks every spec against the active disciplines. It also says, in the present tense, that the auditor's third role suggests reclassifications as the corpus grows. The agent catalog says the auditor is one agent with three roles: single-document fidelity, cross-document consistency and kind classification (`intent shape`). That third role is marked a design target with no registered sub-verb. No discipline role is among the three, and the reclassification role does not run today.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 89.
