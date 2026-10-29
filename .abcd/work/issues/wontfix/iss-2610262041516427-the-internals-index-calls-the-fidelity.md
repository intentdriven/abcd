---
schema_version: 1
id: "iss-2610262041516427"
slug: "the-internals-index-calls-the-fidelity"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1029 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/README.md:20"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-062c31c5460b8b6e
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The name in itd-34 was the agent's name when the intent shipped; the rename is recorded in .abcd/development/agents/CHANGELOG.md, so a reader can resolve it."
---

The internals index calls the fidelity agent intent-auditor; itd-34 calls the same agent intent-fidelity-reviewer.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/README.md:20` — "`intent-auditor` or `documentation-auditor`"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:88` — "`intent-fidelity-reviewer` shape-classification role"

itd-34 identifies 'the same agent that performs single-document fidelity audits ... and cross-document fidelity audits' and names it intent-fidelity-reviewer. That agent is the one the 05-internals index and the rest of the brief call intent-auditor. One agent has two names, so a reader cannot tell whether itd-34's third role belongs to the shipped auditor or to another agent.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1029.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The name in itd-34 was the agent's name when the intent shipped; the rename is recorded in .abcd/development/agents/CHANGELOG.md, so a reader can resolve it.
