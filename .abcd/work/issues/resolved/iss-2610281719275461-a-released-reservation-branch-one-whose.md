---
schema_version: 1
id: "iss-2610281719275461"
slug: "a-released-reservation-branch-one-whose"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "lane E, abcd-e8, on the coordinator's relay of a product ruling, 2026-10-28"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reservesweep/judge.go"
remedy: "Judge a branch whose tip is a plain release spent at the next sweep with no age wait, keep the age for a lapsed reservation, and amend adr-2610161940004937 and the reserve page and brief to match."
resolution: "a plain release is spent at the next sweep once the take-over margin has passed; any other tip still waits the age"
impact: fix
resolved_by:
  commit: "ef8d6edae"
---

A released reservation branch, one whose tip is a plain release commit, waits seven days before the daily reservation sweep deletes it, so spent branches pile up on the forge for a week after the work was handed in. The product thinker ruled on 2026-10-28 that such a branch is deleted at the next daily sweep run instead, keeping the sweep's other safety rules.
