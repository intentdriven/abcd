---
schema_version: 1
id: "iss-2610110939547495"
slug: "the-account-runs-at-most-5-macos-jobs-at"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "CI-cycles research, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/rulesets/main-protection.json"
remedy: "ask GitHub Support to raise the account's concurrent macOS job limit above 5, and record the answer here; larger runners are always billed, so they are not the fallback"
---

The account runs at most 5 macOS jobs at once, which is the binding limit on the merge queue: on 2026-10-11 queue entries waited for macOS slots past the check timeout and were removed. GitHub documents that support can raise concurrency limits; nobody has asked.
