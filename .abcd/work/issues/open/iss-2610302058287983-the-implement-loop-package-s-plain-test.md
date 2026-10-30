---
schema_version: 1
id: "iss-2610302058287983"
slug: "the-implement-loop-package-s-plain-test"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "merge-queue failure of a records-only pull request, 2026-10-30"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Profile the two packages' slowest tests (the git spawns) and shard or trim them, so the 40-minute ceiling stays well clear"
---

The implement loop package's plain test time on the ubuntu CI runner grew from 466s to 600s across recent runs, and internal/surface/cli runs at 400 to 550s. The goroutine dumps show the time going to git process spawns in the state and tracked-file reads. A declared 40-minute package budget now holds the step, but at this rate the margin narrows again.
