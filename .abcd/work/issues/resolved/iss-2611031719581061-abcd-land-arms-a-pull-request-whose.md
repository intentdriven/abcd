---
schema_version: 1
id: "iss-2611031719581061"
slug: "abcd-land-arms-a-pull-request-whose"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/land.go"
remedy: "Before arming, land checks whether the branch's base includes commits of another open pull request still queued or armed, and defers the arming (saying to arm after that one merges, or to rebase onto main) instead of arming into a sure removal; a test arms a stacked branch and asserts the deferral."
resolution: "abcd land now lists the pull requests queued or armed into the default branch before it arms, and stops unarmed while the branch shares a commit not on the default branch with one of them, naming it and saying to run abcd land again after it merges or to rebase onto the default branch; a head the checkout lacks is fetched by its pull-request ref, and an unreadable listing or head arms nothing."
impact: fix
resolved_by:
  commit: "ca34205da9b0bae954f6cb84178be14c4ec5d5a1"
---

abcd land arms a pull request whose branch carries another queued pull request's commits, and the merge queue then removes it as invalid_merge_commit: on 2026-11-03 #1376 (records-coord-1103c) carried #1375's two commits and was armed while #1375 was queued; it entered the queue at 14:46 and was removed 16 seconds later, missing the 15:00 freeze, and had to be re-armed by hand after #1375 merged.
