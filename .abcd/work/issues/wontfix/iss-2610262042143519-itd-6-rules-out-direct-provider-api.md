---
schema_version: 1
id: "iss-2610262042143519"
slug: "itd-6-rules-out-direct-provider-api"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1056 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:54"
remedy: "none (filed automatically)"
related_intents: [itd-6]
finding_ref: cfk-5646a7eccee82f47
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The out-of-scope line is the original framing replaced by the re-file note and adr-25; internal/adapter/openaiapi exists and review is host-default."
---

itd-6 rules out direct provider API integrations and puts RP MCP first in the oracle cascade; the build sequence ships an opt-in OpenAI-compatible provider and makes the host the default.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:54` — "Direct OpenAI / Google / Anthropic API integrations — out of scope; route everything through RP"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:35` — "`openaiapi`, the opt-in OpenAI-compatible provider an agent's oracle role can be pointed at"

itd-6's scope routes everything through RP, puts direct OpenAI, Google and Anthropic API integrations out of scope, and keeps a three-transport cascade with RP MCP preferred. The build sequence lists `openaiapi`, an opt-in OpenAI-compatible provider an agent's oracle role can be pointed at. It also makes review host-delegated by default, with native, CLI, API and MCP adapters opt-in. So a direct API adapter exists that itd-6 says abcd never has, and the default is the host rather than RP.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1056.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The out-of-scope line is the original framing replaced by the re-file note and adr-25; internal/adapter/openaiapi exists and review is host-default.
