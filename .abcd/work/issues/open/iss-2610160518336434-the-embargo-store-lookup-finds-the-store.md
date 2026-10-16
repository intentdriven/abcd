---
schema_version: 1
id: "iss-2610160518336434"
slug: "the-embargo-store-lookup-finds-the-store"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "messaging step 2 review, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/repo.go"
remedy: "Key the store lookup on the root commit of origin's default branch, read with --no-replace-objects, falling back to HEAD only when there is no origin or no default branch."
---

The embargo store lookup finds the store by the root commit of HEAD, read through replace refs and grafts, so with HEAD on an orphan branch or a replaced root commit it looks in the wrong place, and a store it does not find reads as nothing under embargo: an embargoed record can then be announced or published.
