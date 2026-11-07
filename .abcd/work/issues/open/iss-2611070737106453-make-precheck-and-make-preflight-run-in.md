---
schema_version: 1
id: "iss-2611070737106453"
slug: "make-precheck-and-make-preflight-run-in"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Hold a lock around the site render, or give each run its own output directory."
---

make precheck and make preflight run in one worktree collide on the site output directory ('RemoveAll record: directory not empty'), so one of the two fails for a reason unrelated to the tree.
