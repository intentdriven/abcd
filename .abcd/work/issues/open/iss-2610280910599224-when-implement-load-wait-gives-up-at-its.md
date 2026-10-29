---
schema_version: 1
id: "iss-2610280910599224"
slug: "when-implement-load-wait-gives-up-at-its"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous-run-2026-10-28"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_wait.go"
remedy: "On timeout, print one final line naming the wait limit reached, how long it waited and what held it at that moment (load, the live race run or preflight, the waiter ahead), and exit with a distinct code."
---

When implement load --wait gives up at its timeout inside a make recipe, the log ends with only exit status 1 after the waiter's last status line, which can be hours old, so a session sees a failed gated run with no cause named. Gated runs timed out after 2-3h on 27-28 Oct showing nothing more.
