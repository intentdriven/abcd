---
schema_version: 1
id: "iss-2610302057380175"
slug: "the-status-line-s-merging-element-drops"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "the merge queue's status line test failure, 2026-10-30"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/config.go"
remedy: "Cache the default branch's facilitator.mode once per fetch, keyed on the fetched tip, so a status line render reads the cached answer and asks git nothing extra; the page says the row shows the mode as of the last fetch, the same reading the board already makes"
---

The status line's merging element drops from a person's row on a loaded machine. It reads the default branch's facilitator.mode with git questions of its own, inside the row's 300ms git share, alongside the badge, branch and root-commit reads. When the machine is busy (load above 100 is common here in the evening) those questions outlast the share, the element drops, and every render writes a note to stderr. The race-enabled test run on a loaded CI runner failed the same way, and the tests now widen their budgets; the person's row has no such room.
