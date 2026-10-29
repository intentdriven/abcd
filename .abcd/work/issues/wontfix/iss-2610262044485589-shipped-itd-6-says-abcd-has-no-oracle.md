---
schema_version: 1
id: "iss-2610262044485589"
slug: "shipped-itd-6-says-abcd-has-no-oracle"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1231 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:22"
remedy: "none (filed automatically)"
related_intents: [itd-6, itd-2609170822093401]
finding_ref: cfk-30adb492eca833de
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. abcd now has a resolver and routing proposal (internal/core/oracle), consistent with adr-25's opt-in adapters; the itd-6 statement is the dropped single-integration premise."
---

Shipped itd-6 says abcd has no oracle logic and no routing of its own; itd-2609170822093401 has abcd propose and apply a model tier for each agent

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:22` — "Zero abcd-side `oracle` logic, zero "which preset?" prompts, zero hard-coded routing."
- End B: `.abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:22` — "abcd now ships its own proposal for which model tier each of its agents"

itd-6 says abcd never picks an oracle and has zero abcd-side oracle logic and zero hard-coded routing. itd-2609170822093401 ships a bundled table with a tier for each agent. At run time it picks the winning row against the provider connections configured, and sends a step straight to a provider that can be reached. abcd cannot have no routing logic and also ship its own routing proposal and resolver.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1231.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. abcd now has a resolver and routing proposal (internal/core/oracle), consistent with adr-25's opt-in adapters; the itd-6 statement is the dropped single-integration premise.
