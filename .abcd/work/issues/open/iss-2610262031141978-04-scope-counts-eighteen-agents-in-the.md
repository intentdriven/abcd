---
schema_version: 1
id: "iss-2610262031141978"
slug: "04-scope-counts-eighteen-agents-in-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 104 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/04-scope.md:32"
remedy: "Edit 07-memory.md section 6: replace 'The agent count stays at 15' with a count-free statement that principle-distiller gains role responsibilities, not agent peers."
finding_ref: cfk-073bbd86f43efab8
---

04-scope counts eighteen agents in the canonical roster; 07-memory says the agent count stays at 15.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/04-scope.md:32` — "eighteen agents — the canonical roster is the catalog in"
- End B: `.abcd/development/brief/05-internals/07-memory.md:158` — "The agent count stays at 15"

04-scope states the plumbing as 'eighteen agents — the canonical roster is the catalog in' 01-agents.md, while 07-memory asserts 'The agent count stays at 15' as the premise the curator role on principle-distiller is reasoned under. Both state the current size of the one agent roster and give different numbers, so one of them is stale.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 104.
