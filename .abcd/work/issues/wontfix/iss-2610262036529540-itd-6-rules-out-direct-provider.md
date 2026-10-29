---
schema_version: 1
id: "iss-2610262036529540"
slug: "itd-6-rules-out-direct-provider"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 680 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/23-reading.md:365"
remedy: "none (filed automatically)"
related_intents: [itd-6]
finding_ref: cfk-05c13042a58d76bf
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-6's 'route everything through RP' was ruled when RepoPrompt was the only backend; later ADRs make the model host-delegated by default with opt-in adapters and the reading ingest follows them."
---

itd-6 rules out direct provider integrations and sends everything through RP; the reading chapter routes a cold-reading position straight to a configured provider through abcd's own tier routing

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/23-reading.md:365` — "when the person's `oracle.roles.cold-reading-<position>` points at a provider it sends the position's definition as the instructions and the parked bundle"
- End B: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:54` — "Direct OpenAI / Google / Anthropic API integrations — out of scope; route everything through RP"

itd-6 rules out direct OpenAI, Google and Anthropic API integrations and says to route everything through RP, with no abcd-side oracle selection. The reading chapter has abcd resolve a model-tier route per agent from layered oracle-routing tables. When `oracle.roles.cold-reading-<position>` names a provider, the ingest's dispatch form sends the definition and bundle directly to that provider and ingests the answer. abcd choosing a provider and calling it directly is what itd-6 excludes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 680.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-6's 'route everything through RP' was ruled when RepoPrompt was the only backend; later ADRs make the model host-delegated by default with opt-in adapters and the reading ingest follows them.
