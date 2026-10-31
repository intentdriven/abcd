---
schema_version: 1
id: "iss-2610312044522893"
slug: "the-checkoutdigest-and-indexdigest"
severity: "nitpick"
category: "documentation"
source: "impl-review"
found_during: "landing PR #1301 (run-2610310228179894, lane-1)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/autoplan.go"
remedy: "Name both in each comment: the tree objects write-tree stores and the loose blobs add -A stores, all of them unreferenced."
---

The checkoutDigest and indexDigest comments in internal/core/implement/loop/autoplan.go (about lines 181-183 and 239) say nothing outside the scratch directory is written but the tree objects write-tree stores, yet the working-tree half's add -A on the scratch index also stores a loose blob for each admitted modified or untracked file.
