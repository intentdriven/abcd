---
schema_version: 1
id: "iss-2610230643556343"
slug: "two-releases-reserved-at-the-same-moment"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "release-hold round 4 (iss-2610222014550692)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/withdrawact.go"
remedy: "Re-read the release reservations after the push and back off (releasing the new one) when another release still being cut is found; refuse, not pass, when the view cannot be read"
---

Two releases reserved at the same moment both pass the one-release-at-a-time check, since each reads the view before the other's push lands; and a view that cannot be read lets a release reservation through unchecked.
