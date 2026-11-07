---
schema_version: 1
id: "iss-2611070737155587"
slug: "the-release-mark-is-not-lowered-when-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/owned_copy.go"
remedy: "Lower the mark on that restore path, or have the refusal text say the mark is stale and how to lower it."
---

The release mark is not lowered when a refused re-execution or a go-back restores the previous binary. It keeps naming a release the machine no longer runs, and the cache promotion then refuses an older cached release.
