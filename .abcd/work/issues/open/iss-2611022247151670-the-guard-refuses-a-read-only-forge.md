---
schema_version: 1
id: "iss-2611022247151670"
slug: "the-guard-refuses-a-read-only-forge"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-02"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/merge.go"
remedy: "Match the merge hazard on a graphql mutation that merges or arms (enablePullRequestAutoMerge, enqueuePullRequest, mergePullRequest), not on a query that reads those field names; a test approves the read and still refuses the mutation."
---

The guard refuses a read-only forge query as a merge: a script that ran gh api graphql reading a pull request's isInMergeQueue and autoMergeRequest fields after a cd was refused as merge-graphql-unread-human on 2026-11-02, though the query merges nothing.
