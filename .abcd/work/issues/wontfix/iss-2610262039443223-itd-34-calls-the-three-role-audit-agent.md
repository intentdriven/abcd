---
schema_version: 1
id: "iss-2610262039443223"
slug: "itd-34-calls-the-three-role-audit-agent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 883 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/01-agents.md:133"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-eee897549482b295
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037533417. Same End B (itd-34 shape-classification heading) against a different document that uses the current name."
duplicates: [iss-2610262037533417]
---

itd-34 calls the three-role audit agent `intent-fidelity-reviewer`; 01-agents calls the same agent `intent-auditor`.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/01-agents.md:133` — "## `intent-auditor`: one agent, three roles"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:88` — "### `intent-fidelity-reviewer` shape-classification role (third role)"

Both documents describe one agent with three roles: single-document fidelity, cross-document fidelity and shape (kind) classification. itd-34 uses `intent-fidelity-reviewer` throughout. 01-agents' catalogue uses `intent-auditor`, and 05-prompt-quality says the agent was renamed. That gives one agent two names, and itd-34 does not map one to the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 883.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037533417. Same End B (itd-34 shape-classification heading) against a different document that uses the current name.
