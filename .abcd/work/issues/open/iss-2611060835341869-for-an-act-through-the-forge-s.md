---
schema_version: 1
id: "iss-2611060835341869"
slug: "for-an-act-through-the-forge-s"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/terminal_wait.go"
remedy: "Prove the forge's API for such an act before the two-hour clock runs."
---

For an act through the forge's command-line client, origin answering shows only that git works, so a forge API incident longer than two hours with git healthy reads as the act's own failure (`terminal_wait.go:64`, `terminal.go:1114`). A finished lane then goes back as stuck, with a tombstone, over an incident that was not its own.
