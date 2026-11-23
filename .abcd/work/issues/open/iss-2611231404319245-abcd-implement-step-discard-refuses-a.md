---
schema_version: 1
id: "iss-2611231404319245"
slug: "abcd-implement-step-discard-refuses-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/stop.go"
remedy: "let --discard end a lane whose record is resolved or whose intent shipped, one with no commits, and one handed back for any reason, saving its unique commits to a bundle first; test each case"
---

abcd implement step --discard refuses a lane that is stale (its record resolved, its intent shipped, or no commits) or handed back as unachievable: it takes only a held lane or one handed back as stuck or closed. Nine such lanes in this checkout sit at implement, validate or land with no route to an end, and two handed-back lanes (run-2611020739372478, run-2611211059241594) have none either; implement --help names no verb that ends them.
