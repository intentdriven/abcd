---
schema_version: 1
id: "iss-2610262033202441"
slug: "itd-6-says-abcd-makes-no-direct-llm-api"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 392 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:38"
remedy: "none (filed automatically)"
related_intents: [itd-6]
finding_ref: cfk-5cb379aa151ac1e2
---

itd-6 says abcd makes no direct LLM API calls, only RP over MCP; 01-ahoy has delegating verbs send steps straight to a configured provider

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:38` — "abcd calls RP via `mcp__RepoPrompt__*` tools. Period. No subprocess spawning of `claude -p`, no direct LLM API calls"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:204` — "A delegating verb whose agent's `oracle.roles` entry points at a configured provider sends the step there itself"

itd-6 makes RepoPrompt over MCP abcd's single integration point, with no direct LLM API calls, and puts direct OpenAI, Google and Anthropic integrations out of scope. 01-ahoy describes an OpenAI-compatible provider adapter: a delegating verb whose `oracle.roles` entry points at a configured provider sends the step there itself, after the provider setup has checked the provider with a real call. abcd cannot both never call an LLM API directly and send steps to a provider it calls.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 392.
