---
schema_version: 1
id: "iss-2609260552246303"
slug: "abcd-history-list-show-staged-help-say-writes-nothing-and"
severity: "minor"
category: "inconsistency"
source: "drift-detection"
found_during: "v0.11.0 release gate: brief-surface cross-check (autonomous run A, abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/history.md"
---

`abcd history list|show|staged --help` say "Writes nothing" and commands/history.md:15 says they perform zero writes, while the store resolve seam (internal/core/history/location.go) creates the user-level store chain and moves a legacy corpus on first resolve, which commands/history.md:29 itself admits: the help and the page contradict each other and the behaviour. Found by the v0.11.0 brief-surface cross-check (x-036, x-037).
