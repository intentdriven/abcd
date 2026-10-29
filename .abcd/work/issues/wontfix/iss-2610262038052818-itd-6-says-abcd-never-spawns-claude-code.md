---
schema_version: 1
id: "iss-2610262038052818"
slug: "itd-6-says-abcd-never-spawns-claude-code"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 770 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:53"
remedy: "none (filed automatically)"
related_intents: [itd-6]
finding_ref: cfk-d3a85b684caa0211
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The never-spawn line was written about RP bypass and is overtaken by the shipped runner (internal/core/runner/claude.go runs print mode); itd-6's builds_on already lists that runner intent."
---

itd-6 says abcd never spawns Claude Code subprocesses; 35-drain has the drain move start the claude runner for the drain judge itself

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:53` — "Spawning Claude Code subprocesses with `claude -p` — explicitly NOT how abcd works; abcd is the consumer, not the spawner"
- End B: `.abcd/development/brief/04-surfaces/35-drain.md:522` — "When the person's own layer routes `roles.drain-judge.runner` to the claude runner, the move that writes the host's request also writes the runner a brief of its own, the same question over the same quoted record with an answer shape that carries no digest, and starts the runner on it through the dispatch every routed role takes"

itd-6 records that spawning Claude Code subprocesses with `claude -p` is not how abcd works, abcd being the consumer and never the spawner. 35-drain states that when `roles.drain-judge.runner` is routed to the claude runner, the drain move writes the runner a brief and starts the runner through the dispatch every routed role takes. Either abcd starts a claude runner process or it never spawns one; both cannot stand as written.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 770.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The never-spawn line was written about RP bypass and is overtaken by the shipped runner (internal/core/runner/claude.go runs print mode); itd-6's builds_on already lists that runner intent.
