---
schema_version: 1
id: "iss-2610160518345930"
slug: "c4-s-public-record-read-peers-scantree"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "messaging step 2 review, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/peers/read.go"
remedy: "Run scanTree's ls-tree with --no-replace-objects, so the default branch is read as stored."
---

C4's public-record read (peers scanTree, the ls-tree of origin's default branch) follows replace refs, so a local replace ref can make a record read as present on the default branch when the stored tree does not hold it.
