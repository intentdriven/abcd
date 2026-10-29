---
schema_version: 1
id: "iss-2610262045177576"
slug: "itd-6-still-calls-the-lifeboat-verdict"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1264 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:45"
remedy: "none (filed automatically)"
related_intents: [itd-6, itd-125]
finding_ref: cfk-a94ef1548d20f592
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The retired names appear in a historical same-chat bullet that predates the rename; the rename record already says which spelling holds."
---

itd-6 still calls the lifeboat verdict pass lifeboat-oracle and re-audit; itd-125 renamed it a review so that one word names it

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:45` — "lifeboat-oracle → fix → re-audit"
- End B: `.abcd/development/intents/shipped/itd-125-the-lifeboat-verdict-says-what-it-is.md:22` — "produced by an oracle, holding a review verdict — three words for one thing."

itd-125 says the lifeboat verdict file was produced by an oracle and held a review verdict, three words for one thing, and that the verb, flag, artefact and prose now all say review. itd-6's same-chat rule still describes the pass as lifeboat-oracle, fix, re-audit. That keeps the oracle and audit names itd-125 retired.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1264.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The retired names appear in a historical same-chat bullet that predates the rename; the rename record already says which spelling holds.
