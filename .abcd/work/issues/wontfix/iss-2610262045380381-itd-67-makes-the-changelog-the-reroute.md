---
schema_version: 1
id: "iss-2610262045380381"
slug: "itd-67-makes-the-changelog-the-reroute"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1288 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:48"
remedy: "none (filed automatically)"
related_intents: [itd-67, itd-65]
finding_ref: cfk-a06be87f8f8301d7
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-65 itself says the reroute is superseded because a gate cannot append to a changelog nothing hand-writes."
---

itd-67 makes the changelog the reroute target for itd-65's doc-history gate; itd-65 supersedes the reroute whole

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:48` — "change-narration that gate strips from a doc body is appended here"
- End B: `.abcd/development/intents/shipped/itd-65-launch-preflight-gate-suite.md:106` — "*The reroute criterion (last).* Superseded whole: with no auto-append there"

itd-67 says change-narration the doc-history gate strips from a doc body is appended to the changelog. itd-65 records that the reroute criterion is superseded whole, because a gate cannot append prose to a changelog nothing hand-writes and the change reaches the changelog through its record. The appending path itd-67 relies on does not exist under itd-65.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1288.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-65 itself says the reroute is superseded because a gate cannot append to a changelog nothing hand-writes.
