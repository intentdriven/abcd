---
schema_version: 1
id: "iss-2610262045365573"
slug: "itd-6-says-abcd-never-spawns-claude-p"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1285 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:38"
remedy: "none (filed automatically)"
related_intents: [itd-6, itd-2609201916056194]
finding_ref: cfk-4de04260ae4a9ed0
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/runner/claude.go runs the claude CLI in print mode; itd-6's scope text is the pre-re-file prohibition."
---

itd-6 says abcd never spawns claude -p; itd-2609201916056194 ships a claude CLI print-mode runner

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:38` — "No subprocess spawning of `claude -p`, no direct LLM API calls, no model detection."
- End B: `.abcd/development/intents/shipped/itd-2609201916056194-abcd-runs-a-delegated-agent-through-a.md:44` — "the claude CLI in print mode with the bare flag"

itd-6's scope makes RepoPrompt over MCP the single integration point and forbids spawning claude -p, repeating it out of scope. itd-2609201916056194 ships the claude CLI in print mode as one of its runners. itd-6's own decision 1 narrows it to one opt-in route, but its scope text still states the prohibition, so the two shipped records disagree on whether abcd starts a claude CLI subprocess.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1285.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/runner/claude.go runs the claude CLI in print mode; itd-6's scope text is the pre-re-file prohibition.
