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
resolution: "A drive's start lets go a run no drain lists that it took up, or took over from a settled stop's mark, and that comes back from its lane still in progress (letGoTakenUp in terminal.go, after both of takeEarlier's steps), releasing its reservation and recording the let-go for a later start, so the drain's first move never opens a lane beside a second reserved run it does not step; a run a live driver elsewhere holds, or one set aside, keeps its hold."
impact: fix
resolved_by:
  intent: "itd-2611141610197753"
  spec: "spc-2611141617199772"
---

A drive's start can take up an earlier run that no drain lists, re-reserve it in driveLane, and give it back still in progress (stuck, unable to move, refused for good, or taken by a peer). No lane lists that run, so the drain's first move counts no lane in work and opens one. The drive then holds two reserved runs while stepping only one, and keeps the idle run's reservation until the drive ends or the lease lapses. Found while reviewing the drive's one-lane-in-work hold on surge-b-s2; the path is main's internal/core/implement/loop/terminal.go, not that change.
