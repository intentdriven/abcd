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
remedy: "Rewrite the checkoutDigest comment to name everything written outside the scratch directory: the tree objects write-tree stores, the loose blobs add -A stores (all unreferenced) and whatever a configured clean filter writes; leave the indexDigest comment as it is."
---

The checkoutDigest comment in internal/core/implement/loop/autoplan.go (about lines 181-183) says nothing outside the scratch directory is written but the tree objects write-tree stores, yet its add -A on the scratch index (about line 207) also stores a loose blob for each admitted modified or untracked file, and a configured clean filter may write objects of its own (Git LFS writes under .git/lfs/objects). The indexDigest comment is accurate: it runs only ls-files and write-tree.
