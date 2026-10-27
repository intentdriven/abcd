---
schema_version: 1
id: "iss-2610262044503777"
slug: "shipped-itd-6-says-abcd-only-uses-agent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1233 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:53"
remedy: "none (filed automatically)"
related_intents: [itd-6, itd-2610222215485265]
finding_ref: cfk-d03f26685d50f738
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-6's never-spawn line is overtaken by the shipped runner and by the draft's own stated reversal, so no edit to either document is owed now."
---

Shipped itd-6 says abcd only uses agent sessions and never starts Claude Code ones itself; itd-2610222215485265 has `abcd drain --drive` start Claude sessions

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:53` — "explicitly NOT how abcd works; abcd is the consumer, not the spawner"
- End B: `.abcd/development/intents/drafts/itd-2610222215485265-abcd-drain-run-from-a-plain-terminal.md:21` — "start Claude sessions on his API key, about forty agent runs at most"

itd-6 says that spawning Claude Code subprocesses is explicitly not how abcd works: abcd is the consumer, not the spawner. In itd-2610222215485265, the driver tells the person it will start Claude sessions on their API key, up to about forty agent runs, from a plain terminal. The drive record notes that it reverses the host-delegated boundary in a proposed ADR. itd-6 still states the opposite rule.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1233.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-6's never-spawn line is overtaken by the shipped runner and by the draft's own stated reversal, so no edit to either document is owed now.
