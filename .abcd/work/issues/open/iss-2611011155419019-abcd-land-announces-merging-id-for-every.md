---
schema_version: 1
id: "iss-2611011155419019"
slug: "abcd-land-announces-merging-id-for-every"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous build and records landing, 2026-10-31"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/announce.go"
remedy: "Announce merging only for items named in Resolves: or Delivers: trailers (and the pull request), and either leave Refs: items out or post them under a distinct wording that says the change touches them, recording a decision that narrows dcn-2610170704191129."
---

abcd land announces 'merging <id>' for every issue or intent a branch's commits name in a Refs: trailer, as well as Resolves: and Delivers: (internal/core/landing/announce.go, per dcn-2610170704191129). Refs: means the change touches a record without fixing it, but the announcement reads as that record's work merging. On 2026-10-31 a peer session read a build session's records-PR announcements for Refs-only issues as competing work on those issues, and the two sessions spent a round of messages confirming nobody else was on them. A colleague reading abcd status cannot tell a fix that is merging from a record that is merely cited.
