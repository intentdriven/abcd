---
schema_version: 1
id: "iss-2611270555396370"
slug: "drive-start-keeps-an-unlisted-run"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "review of the drive's one-lane hold (B step 2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/terminal.go"
remedy: "In terminal.go, release the reservation of a run driveLane gives back still in progress when no drain lists it, or have the drain list a run the drive's start took up, so the in-work count sees it; add a test where the start takes up an unlisted run that then cannot move and the first move opens no second reserved lane."
---

A drive's start can take up an earlier run that no drain lists, re-reserve it in driveLane, and give it back still in progress (stuck, unable to move, refused for good, or taken by a peer). No lane lists that run, so the drain's first move counts no lane in work and opens one. The drive then holds two reserved runs while stepping only one, and keeps the idle run's reservation until the drive ends or the lease lapses. Found while reviewing the drive's one-lane-in-work hold on surge-b-s2; the path is main's internal/core/implement/loop/terminal.go, not that change.
