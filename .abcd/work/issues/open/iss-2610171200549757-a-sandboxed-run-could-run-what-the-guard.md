---
schema_version: 1
id: "iss-2610171200549757"
slug: "a-sandboxed-run-could-run-what-the-guard"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "the approval-prompt ruling of 2026-10-17"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Once autonomous agents run in a sandbox, let a command the guard cannot fully read run inside it instead of being refused, since the sandbox bounds what it can touch; until then extend what the guard reads (other interpreters' files) so fewer commands need the refusal."
---

A command the guard cannot read is refused outright in an autonomous run, even where a sandbox would make running it safe. The guard already reads shell scripts a command runs, but not other interpreters' files or file text substituted into a command; the sandbox environment under way for autonomous agents would bound what such a command can touch.
