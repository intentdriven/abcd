---
schema_version: 1
id: "iss-2610262043071551"
slug: "itd-115-keeps-the-strict-merge-gate-to"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1116 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-115-abcd-managed-repos-merge-prs-without-the.md:24"
remedy: "none (filed automatically)"
related_intents: [itd-115, itd-114]
finding_ref: cfk-f85b029a07199952
---

itd-115 keeps the strict merge gate to catch duplicate ids from parallel sessions; itd-114 says ids are collision-proof with no coordination.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-115-abcd-managed-repos-merge-prs-without-the.md:24` — "strict merge gate stays intact: it is what catches a duplicate record id minted"
- End B: `.abcd/development/intents/shipped/itd-114-abcd-mints-collision-proof-record-ids.md:19` — "mints is collision-proof by construction, with no coordination between"

itd-115 states that the strict merge gate 'is what catches a duplicate record id minted by two parallel sessions' and must never be traded away. itd-114 states that every id abcd mints is collision-proof by construction, with no coordination between agents. If itd-114 holds, the gate catches no such duplicate; if itd-115 holds, itd-114's guarantee does not. The merge-queue design rests on a failure the shipped mint says cannot happen.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1116.
