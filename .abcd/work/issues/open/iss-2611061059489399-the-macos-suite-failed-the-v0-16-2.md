---
schema_version: 1
id: "iss-2611061059489399"
slug: "the-macos-suite-failed-the-v0-16-2"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 release gate"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/macos-suite.yml"
remedy: "Give the package a macOS leg of its own so a slow runner still clears the limit, make its slowest whole-drive tests faster, and warn at a package's share of its leg as well as at the job's clock."
---
The macOS suite failed the v0.16.2 release gate and the nightly before it the same way: internal/core/implement/loop hit the 40-minute test limit in all three test processes with no test failing. A rerun of the same commit passed: `loop` took 1504 s and `internal/core/capture` 268 s, against more than 2400 s and 554 s on the failed attempt, so the runner's speed varied by about two. `loop` took 1173 s the day before, so the package's growth narrowed the margin and a slow runner used it up. Its leg shares a runner of about three cores with 47 other packages. The Linux queue passes the same tree, so the queue gave no warning. The only earlier signal was the status block's line on the scheduled `ci.yml` verdict, and iss-2611061059497955 holds why the cut did not act on it.
