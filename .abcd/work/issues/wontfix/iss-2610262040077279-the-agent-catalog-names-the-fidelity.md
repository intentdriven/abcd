---
schema_version: 1
id: "iss-2610262040077279"
slug: "the-agent-catalog-names-the-fidelity"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 910 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/01-agents.md:133"
remedy: "none (filed automatically)"
related_intents: [itd-80]
finding_ref: cfk-3eda56feb434be79
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. One agent, one recorded rename; the only agents/ file carries the current name."
---

The agent catalog names the fidelity agent `intent-auditor`; itd-80, which created it, names it `intent-fidelity-reviewer`

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/01-agents.md:133` — "`intent-auditor`: one agent, three roles"
- End B: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:35` — "A host-delegated `intent-fidelity-reviewer` markdown agent"

01-agents names the one agent with three roles `intent-auditor`. itd-80 delivers the same host-delegated single-document fidelity reviewer as the `intent-fidelity-reviewer` markdown agent. That is two names for one agent, and only one of them is a file under agents/.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 910.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. One agent, one recorded rename; the only agents/ file carries the current name.
