---
schema_version: 1
id: "iss-2609290518278152"
slug: "fsutil-readdeclaration-refuses-a-home-scoped-declaration-as"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/fsutil.go"
---

fsutil.ReadDeclaration refuses a home-scoped declaration as replaced between its vetting and its read when another abcd process of the same user atomically renames a new version into place inside the lstat-to-open window, so two abcd processes on one machine (two sessions, or a hook and a verb) can make one refuse its own ~/.abcd/config.json, rules.json or credentials index; the merge-group macOS leg of PR 744 failed TestConcurrentConnectsKeepEveryKeyAndBlock on it.
