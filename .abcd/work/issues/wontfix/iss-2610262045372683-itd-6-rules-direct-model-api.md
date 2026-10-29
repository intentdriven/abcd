---
schema_version: 1
id: "iss-2610262045372683"
slug: "itd-6-rules-direct-model-api"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1286 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:54"
remedy: "none (filed automatically)"
related_intents: [itd-6, itd-2609081951381895]
finding_ref: cfk-ffd8797e43b3caaa
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The adapter exists in internal/adapter/openaiapi and is opt-in; itd-6's out-of-scope line is the superseded framing."
---

itd-6 rules direct model API integrations out and routes everything through RepoPrompt; itd-2609081951381895 ships an OpenAI-compatible API adapter

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:54` — "Direct OpenAI / Google / Anthropic API integrations — out of scope; route everything through RP"
- End B: `.abcd/development/intents/shipped/itd-2609081951381895-abcd-ships-an-openai-compatible-api.md:21` — "An OpenAI-compatible API adapter reaches a configured provider for the roles and judgements pointed at it"

itd-6 puts direct OpenAI, Google and Anthropic API integrations out of scope and says to route everything through RepoPrompt. itd-2609081951381895 ships an OpenAI-compatible API adapter that reaches a configured provider directly for roles and judgements. Both are shipped; one of them no longer describes how abcd reaches a model.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1286.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The adapter exists in internal/adapter/openaiapi and is opt-in; itd-6's out-of-scope line is the superseded framing.
