---
schema_version: 1
id: "iss-2609251639024615"
slug: "the-account-key-walk-gives-up-after-eight-blanks-which-fails"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
---

The account-key walk gives up after eight blanks, which fails away from the finding. maxKeyGap (internal/adapter/scanner/identity.go) bounds the blanks afterAccountKey reads between a key and its value at eight, so a column-aligned config dump (USER= or username: followed by ten blanks and the login) raises no finding for a login on the generic list. Every other bound in the position checks stops reading and keeps the match; this one stops reading and drops it.
