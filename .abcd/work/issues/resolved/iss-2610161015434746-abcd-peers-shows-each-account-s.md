---
schema_version: 1
id: "iss-2610161015434746"
slug: "abcd-peers-shows-each-account-s"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/peers.go"
remedy: "Show each drain lane's stage and recent hand-backs in abcd peers"
related_intents: [itd-2610161016401115]
resolution: "Each renewal a drain lane makes carries its stage as Abcd-Stage from the loop's own stage names, a drain's re-reserve of an issue a concurrent build's lane holds included, and abcd peers lists each live reservation's stage and each holder's hand-backs of the last day with their kinds"
impact: additive
resolved_by:
  intent: "itd-2610161016401115"
  spec: "spc-2610161044461448"
  commit: "bab97a0ae0d683655793766bbe2a5d78de63d250"
---

abcd peers shows each account's reservations but nothing of a drain lane's stage, its judgements or its hand-backs, so one person cannot see what another account's drain is doing.
