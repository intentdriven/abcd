---
schema_version: 1
id: "iss-2610262036514870"
slug: "itd-6-says-abcd-never-spawns-claude-code"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 679 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md:385"
remedy: "none (filed automatically)"
related_intents: [itd-6]
finding_ref: cfk-b602dc555ff5fc97
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/runner starts harness processes today; the later shipped intent and ADR are the records that moved the boundary, and 34-build.md names the host-delegated reversal openly."
---

itd-6 says abcd never spawns Claude Code subprocesses; the guard chapter describes agents that abcd starts through its claude runner

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/17-guard.md:385` — "An agent abcd starts through its claude runner has no terminal"
- End B: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:53` — "Spawning Claude Code subprocesses with `claude -p` — explicitly NOT how abcd works; abcd is the consumer, not the spawner"

itd-6 rules out spawning Claude Code subprocesses with `claude -p` as 'explicitly NOT how abcd works', saying abcd is the consumer and not the spawner. The guard chapter describes agents abcd starts through its own claude runner: headless launches carrying their own command line, settings and MCP server, plus a second, opencode, runner. The two disagree on whether abcd launches harness processes at all.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 679.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/runner starts harness processes today; the later shipped intent and ADR are the records that moved the boundary, and 34-build.md names the host-delegated reversal openly.
