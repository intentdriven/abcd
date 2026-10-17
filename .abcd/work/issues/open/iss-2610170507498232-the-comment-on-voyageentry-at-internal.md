---
schema_version: 1
id: "iss-2610170507498232"
slug: "the-comment-on-voyageentry-at-internal"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "PR8 review of the v0.15.0 deferred verb findings"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat/voyage.go"
remedy: "Say in the comment that the file list is the lifeboat's own tree, pinned through _provenance.json's manifest_sha256, so the ledger records the count."
---

The comment on voyageEntry at internal/core/lifeboat/voyage.go:27-31 says the full file list lives in the lifeboat's own _provenance.json, pinned by the manifest hash. The Provenance struct (internal/core/lifeboat/plan.go:66-86) holds no file list: the list is the lifeboat's own tree, which verifyManifest (internal/core/lifeboat/embark.go:181-235) walks and re-hashes against manifest_sha256. The wrong comment was copied into the brief's naming register during PR8 review.
