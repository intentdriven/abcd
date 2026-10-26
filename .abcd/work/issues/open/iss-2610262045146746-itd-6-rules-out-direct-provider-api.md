---
schema_version: 1
id: "iss-2610262045146746"
slug: "itd-6-rules-out-direct-provider-api"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1261 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:54"
remedy: "none (filed automatically)"
related_intents: [itd-6, itd-166]
finding_ref: cfk-df3c221e1d1fd45d
---

itd-6 rules out direct provider API integrations and routes everything through RP; itd-166 records model calls made through command-line runners and provider connections

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:54` — "Direct OpenAI / Google / Anthropic API integrations — out of scope; route everything through RP"
- End B: `.abcd/development/intents/shipped/itd-166-each-run-should-record-the-model.md:18` — "A role run through the host, a command-line runner or a provider connection names the model it requested"

itd-6's scope puts direct OpenAI, Google and Anthropic API integrations out of scope, routes everything through RP, and also rules out spawning subprocesses. itd-166 ships receipts for roles run through the host, a command-line runner or a provider connection (opencode, OpenRouter, local servers). Those are the direct routes itd-6 excludes, so the two cannot both describe how abcd routes model calls.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1261.
