---
schema_version: 1
id: "iss-2610262031138048"
slug: "invariant-4-says-all-inter-agent-data-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 102 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:15"
remedy: "none (filed automatically)"
finding_ref: cfk-5cc8ac0b507d7ffe
---

Invariant 4 says all inter-agent data is JSON, but the agent catalog says the implement loop reads the reviewers' prose verdict headings

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:15` — "all inter-agent data is JSON; markdown is a render step."
- End B: `.abcd/development/brief/05-internals/01-agents.md:57` — "`ruthless-reviewer` and `security-reviewer` return prose ending in a verdict heading the implement loop reads"

Invariant 4, which the invariants page calls non-negotiable, says all inter-agent data is JSON and markdown is only a render step. The agent catalog says four shipped prompts declare no JSON contract. The two loop reviewers return prose ending in a verdict heading that the implement loop reads and records, and the docs reviewer returns a table. The loop takes in agent output as prose, which the invariant rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 102.
