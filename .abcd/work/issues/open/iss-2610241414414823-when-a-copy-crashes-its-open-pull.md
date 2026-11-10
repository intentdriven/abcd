---
schema_version: 1
id: "iss-2610241414414823"
slug: "when-a-copy-crashes-its-open-pull"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "drain session abcd-0d, 2026-10-24 autonomous test run"
origin: researcher-authored
production_mode: hand-written
found_at: "open pull requests left by a crashed copy"
remedy: "A sweep that lists open pull requests whose author copy holds no live reservation and whose branch conflicts with the default branch, merges the default branch into each in its own store worktree, lands the clean ones and hands back the rest, with the conflicts named."
---

When a copy crashes, its open pull requests stay open and drift: by 2026-10-24 two of them (#1110 and #1116) conflicted with main after 89 and 128 commits, and nothing noticed or repaired them. A session had to find them, merge main into each by hand, resolve the conflicts and queue them, while the work they carried (a major issue's fix among it) stayed unmerged.
