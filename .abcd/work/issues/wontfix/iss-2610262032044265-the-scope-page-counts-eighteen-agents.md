---
schema_version: 1
id: "iss-2610262032044265"
slug: "the-scope-page-counts-eighteen-agents"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 216 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/04-scope.md:32"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-b5568df3c9a4aeca
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. agents/ now holds 18 definitions, as the scope page says; the 15 was a point-in-time count in a shipped intent, whose roles-not-agents point still holds."
---

The scope page counts eighteen agents; itd-34 says the agent count in the catalog stays at 15.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/04-scope.md:32` — "(eighteen agents — the canonical roster is the catalog in"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:98` — "The agent count in the catalog stays at **15**"

The brief's plumbing paragraph states eighteen agents in the canonical catalog. itd-34 asserts that the catalog's count stays at fifteen. The two give different counts for the same roster.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 216.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. agents/ now holds 18 definitions, as the scope page says; the 15 was a point-in-time count in a shipped intent, whose roles-not-agents point still holds.
