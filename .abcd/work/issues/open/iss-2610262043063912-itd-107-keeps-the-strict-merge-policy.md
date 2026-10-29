---
schema_version: 1
id: "iss-2610262043063912"
slug: "itd-107-keeps-the-strict-merge-policy"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1115 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-107-autonomous-routines-assemble-from-one.md:138"
remedy: "In itd-107 lines 136-142, drop 'a record id minted concurrently on two branches' as the reason strict up-to-date stays: itd-114 shipped collision-proof ids, so that gate no longer catches such a duplicate; keep the requirement only on a reason that survives, or relax it"
related_intents: [itd-107, itd-114]
finding_ref: cfk-a4e14cc3a9ed0fd8
---

itd-107 keeps the strict merge policy because ids minted on two branches can collide; itd-114 says such ids never collide.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-107-autonomous-routines-assemble-from-one.md:138` — "the only place a record id minted concurrently on two branches"
- End B: `.abcd/development/intents/shipped/itd-114-abcd-mints-collision-proof-record-ids.md:21` — "minting the same instant on different branches produce different ids"

itd-107 forbids relaxing the strict status-check policy on the ground that the merged-result run is 'the only place a record id minted concurrently on two branches' becomes visible, because the mint lock does not span checkouts. itd-114 makes every id collision-proof by construction with no coordination, so two agents 'minting the same instant on different branches produce different ids'. One premise is false; the routine template would carry a justification the shipped id scheme has removed.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1115.
