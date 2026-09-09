---
schema_version: 1
id: "iss-2609090722466403"
slug: "a-staged-raw-transcript-can-live-indefinitely-and-the-store"
severity: "critical"
category: "security"
source: "agent-finding"
found_during: "sub-agent transcript capture conceptual review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/staging.go"
---

A staged raw transcript can live indefinitely, and the store's own comment says it cannot. Staging is the one place abcd holds unredacted transcript text on purpose, and its contract is that a staged file survives only until the next session starts. That holds only for a repository someone opens again. The drain runs from the session-start hook of the repository the staged file belongs to, so a repository that is finished with, or merely quiet, keeps its raw transcripts forever. On this machine right now there are four staged files totalling about thirteen megabytes of unredacted transcript, the oldest fourteen days old, and the per-repo status verb reports nothing from any other repository, so standing in one checkout cannot reveal a pile in another. A drain failure compounds it: the staged file is deliberately left in place, correctly, because deleting the only copy would be worse, but nothing ever retires it, so the population of permanently raw files only grows. Three things would close it: a drain that can run while a session is live rather than only at its start, a maximum staged age after which a file is redacted or deleted with a notice, and a cross-repository notice at session start so the pile in the repository nobody is standing in is still visible.
