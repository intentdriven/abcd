---
schema_version: 1
id: "iss-2610100913096346"
slug: "the-run-start-budget-check-never"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "review of PR #886 (itd-2609201925079472, run-2610100550297632)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/budget.go"
remedy: "implement QuotaReporter for a runner that can report a quota, and count fix rounds in the estimate"
---

The run-start budget check never compares anything today: no shipped runner implements QuotaReporter, so every real run skips the check, and the estimate leaves out fix rounds, so it is a lower bound.
