---
schema_version: 1
id: "iss-2611251741261124"
slug: "drive-start-fetches-before-it-refuses-an"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/remoteid.go:1096, internal/core/implement/loop/lane.go:309, internal/core/implement/loop/terminal.go:686"
remedy: "When the tip origin reports is not held, the drive's start and the lane's worktree stage fetch origin's default branch once, as a drain move does, under the hold the start has just read, and compare again; only a second miss refuses. Tests start a drive on a repository whose origin moved since its last fetch, re-execute a drive after origin moved, and open a lane on a tip fetched by the stage, and assert each carries on."
resolution: "A drive's start and a lane's worktree stage now fetch origin's default branch once when the tip origin reports is not held, under the hold the start has just read, and refuse only a second miss."
impact: fix
---

A drive's start refuses when origin reports a tip this repository does not hold yet, and its remedy asks the person to fetch by hand, although every drain move fetches origin's default branch itself. A lane that opens between a move's fetch and its own ask of origin meets the same words as a plain refusal of the worktree stage. A drive that re-executes itself for a new release reads its hold afresh, so it can meet that refusal with nobody at the terminal to fetch.
