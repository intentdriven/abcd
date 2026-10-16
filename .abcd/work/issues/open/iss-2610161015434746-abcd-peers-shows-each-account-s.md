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
---

abcd peers shows each account's reservations but nothing of a drain lane's stage, its judgements or its hand-backs, so one person cannot see what another account's drain is doing.
