---
schema_version: 1
id: "iss-2610262044038958"
slug: "itd-44-writes-adrs-under-a-next-free"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1180 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:122"
remedy: "none (filed automatically)"
related_intents: [itd-44, itd-114]
finding_ref: cfk-e5ead98412c3933d
---

itd-44 writes ADRs under a next-free ordinal; itd-114 makes every minted id collision-proof across branches

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-44-fourth-intent-kind-decision.md:122` — "`adr-<next-free>-<slug>.md` into `.abcd/development/decisions/adrs/`, matching"
- End B: `.abcd/development/intents/shipped/itd-114-abcd-mints-collision-proof-record-ids.md:21` — "minting the same instant on different branches produce different ids"

itd-44 has capture write `adr-<next-free>-<slug>.md`, choosing the next unused number. itd-114 says every id abcd mints is collision-proof by construction, so two agents minting at the same moment on different branches get different ids. A next-free ordinal is the scheme that collides in exactly that case, and itd-114 names the adr family among those it covers.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1180.
