---
schema_version: 1
id: "iss-2610150524235929"
slug: "push-after-a-passing-preflight"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "v0.15.0 run review 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Give abcd one landing path for any branch, not only a build lane: run the gate through the machine load waiter, and once it mints a receipt for an unchanged HEAD, plain-push the branch (never forced, never the default branch). Toggle it with a config key (repo config, overridable in ~/.abcd.noindex/config.json), default on; off leaves the push to the person and says so."
---

A branch whose preflight passed is pushed only if someone remembers to: abcd pushes after the receipt for a build lane alone. abcd build's land step already refuses a push without a receipt naming the head and then plain-pushes the lane. Every branch landed outside the loop gets no such step: hand-built branches, cloud-session pull requests, release branches, record-only chores. On 2026-10-14 one such branch's preflight passed at 13:17 and the branch sat unpushed for about two hours, because the step after the gate was lost across a context compaction. The coordinator now runs a hand-written scratchpad script that pushes after a minted receipt; that is one session's workaround, not abcd. The product thinker asked (2026-10-15) for this to be abcd-wide, toggled in config, default on.
