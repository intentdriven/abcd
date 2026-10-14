---
schema_version: 1
id: "iss-2610100913096412"
slug: "the-run-start-quota-query-has-no"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "review of PR #886 (itd-2609201925079472, run-2610100550297632)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/budget.go"
remedy: "give the quota query a deadline and pass its error through fsutil.RedactHome like the loop's other record notes"
resolution: "The run-start quota query now runs under a deadline (the loop's ghTimeout, two minutes), so a runner that reports its quota over I/O cannot hang build or drain at start, and its error is passed through fsutil.RedactHome before it reaches the start's budget row and the run record."
impact: fix
---

The run-start quota query has no deadline and its error is not home-redacted: Config.Quota runs on context.Background, so the first runner that reports a quota over I/O can hang build and drain at start, and its error reaches the run record and the checks row with the home path in it.
