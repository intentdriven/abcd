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
remedy: "Profile the two packages' slowest tests (the git spawns) and trim them, or raise their shard count, so each shard stays well clear of the 40-minute package budget and the 45-minute job ceiling"
---

The implement loop package's plain test time on the ubuntu CI runner, run as one process, grew from 466s to 593s across recent runs and then crossed 600s; internal/surface/cli ran at 400 to 550s. The goroutine dumps show the time going to git process spawns in the state and tracked-file reads. CI's plain step now shards both packages as make preflight does and declares a 40-minute package budget (iss-2610302058222905), but the packages keep growing, and the margin narrows again at this rate.

Two plain runs of the `loop` package on 6 November hit the 40 minute limit with no test failing, under ordinary shared load, each costing its builder the full 40 minutes; the same package finished in about 15 minutes under three shards. A plain run of it is no longer usable, so the page that tells someone to run one package should name the sharded form.
