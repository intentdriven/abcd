---
schema_version: 1
id: "iss-2610262044285635"
slug: "tier-routing-sends-a-row-to-a-provider"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1209 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:105"
remedy: "none (filed automatically)"
related_intents: [itd-2609170822093401, itd-2609081951381895]
finding_ref: cfk-0c6d8a7698a0cb19
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262039253719. Same contradiction checked against the shipped adapter intent instead of the adapters page."
duplicates: [iss-2610262039253719]
---

Tier routing sends a row to a provider that can serve its tier, but the shipped adapter rules that no provider serves any tier

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:105` — "provider where reachable and able to serve the tier, harness otherwise"
- End B: `.abcd/development/intents/shipped/itd-2609081951381895-abcd-ships-an-openai-compatible-api.md:69` — "never by a tier alone, so `Connections.Serves` answers false for every tier"

itd-2609170822093401 resolves each agent's tier row at run time to a configured provider that is reachable and able to serve the tier, and falls back to the harness only otherwise. Its persona describes economy rows running on a local server untouched. itd-2609081951381895, which the routing intent names as owner of the adapters, rules that a provider claims no tier and is never reached by a tier alone, so `Connections.Serves` is false for every tier. Under the shipped rule every tier row falls back to the harness, and the planned resolution cannot happen.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1209.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262039253719. Same contradiction checked against the shipped adapter intent instead of the adapters page.
