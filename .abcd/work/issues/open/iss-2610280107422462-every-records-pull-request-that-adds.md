---
schema_version: 1
id: "iss-2610280107422462"
slug: "every-records-pull-request-that-adds"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "crash salvage lane D, landing records PRs in parallel, 2026-10-28"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/03-out-of-scope.md"
remedy: "Stop committing the drafts index: render it where it is read (the site build and abcd intent's listing) from drafts/ itself, and drop the region and its index_drift pair; or, while it stays committed, regenerate every generated region in the merge queue's own run and let the queue commit it, since a forge merge never runs a local merge driver"
---

Every records pull request that adds, plans or moves a draft intent rewrites the generated drafts index in .abcd/development/brief/06-delivery/03-out-of-scope.md, so two such pull requests open at once conflict as soon as the first merges: their insertions into the sorted one-per-line list land in neighbouring lines. On the night of 2026-10-27/28 it happened three times, #1131 against #1147, #1162, and lane E's rebase, and each was fixed by hand by merging the default branch, taking its side of the region and running go generate again. The forge applies no merge driver (iss-2609240646538011), and no verb regenerates the region when it files a record (iss-2610241256347199).
