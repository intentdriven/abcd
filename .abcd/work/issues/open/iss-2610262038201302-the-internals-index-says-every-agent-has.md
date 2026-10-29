---
schema_version: 1
id: "iss-2610262038201302"
slug: "the-internals-index-says-every-agent-has"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 788 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/README.md:7"
remedy: "In the index row for 01-agents.md say fourteen of the eighteen prompts declare JSON inputs and outputs and the four repo-workflow reviewer and researcher prompts do not, as 01-agents.md does."
finding_ref: cfk-beb5d2b696a02d00
duplicates: [iss-2610160608434377]
---

The internals index says every agent has declared JSON inputs and outputs; the agent catalog says four agents declare no JSON contract.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/README.md:7` — "The eighteen shipped agent prompts and the design roster still to be built, each with declared JSON inputs/outputs"
- End B: `.abcd/development/brief/05-internals/01-agents.md:56` — "The four repo-workflow reviewer and researcher prompts declare no JSON"

05-internals/README describes 01-agents as covering the eighteen agents, each with declared JSON inputs/outputs. 01-agents says only fourteen return JSON: `docs-currency-reviewer`, `ruthless-reviewer`, `security-reviewer` and `sota-researcher` return a table, prose or a ranked list.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 788.
