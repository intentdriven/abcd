---
schema_version: 1
id: "iss-2611060835502818"
slug: "the-prompt-s-status-block-read-status"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/status_notes.go"
remedy: "Measure which step eats the hook's deadline and name it in the block."
---

The prompt's status block read `status unavailable: no time was left to read it at this prompt` on every prompt for an hour on 6 November with load low (`internal/core/claim/status_notes.go:177`). A hook that never reaches its read tells the person nothing about status for that whole stretch, and nothing names the earlier step that used the deadline.

2026-11-25: Seen again at four or more prompts in one evening, on hooks at 0.16.0 with the load average between 9 and 19. Three runs each of 0.17.1 and of the tree at that load answered in under half a second with no such line.
