---
schema_version: 1
id: "iss-2611070737239713"
slug: "abcd-land-said-commits-name-no-issue"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/announce.go"
remedy: "Reproduce with a branch of commits carrying all three trailers, fix the reading, and add a test."
refines: [iss-2611011155419019]
---

abcd land said commits name no issue, intent or pull request in a Refs:, Resolves: or Delivers: trailer on a branch whose commits carry all of those, and the announce step was skipped on that false reading.
