---
schema_version: 1
id: "iss-2611052230565049"
slug: "a-terminal-drive-renews-what-it-holds"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "release cut preparation"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/terminal.go"
remedy: "After an hour's wait on a limit, or at once when the named reset is over an hour off, with no agent in a call and no open pull request: save the lane and release its claims; on waking, drop the lane and take the next issue when the issue has left open on the default branch or a peer holds it, and otherwise reserve again and catch the branch up."
---

A terminal drive given an end date with `--until` renews what it holds through every wait, so a subscription limit lasting days keeps its issue reserved for the whole wait with nothing happening on it, and refuses any peer's work whose paths overlap. With no end date the same wait is bounded at eight hours, after which the run ends and lets the lane go.
