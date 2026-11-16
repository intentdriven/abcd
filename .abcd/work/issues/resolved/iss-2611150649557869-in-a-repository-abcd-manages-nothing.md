---
schema_version: 1
id: "iss-2611150649557869"
slug: "in-a-repository-abcd-manages-nothing"
severity: "major"
category: "process"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610222214476887-abcd-has-no-single-verb-that-cleans-up-a.md"
remedy: "Take the later decision itd-2610222214476887 leaves open: once a landing sees its pull request merged, and at the end of a drain, a build run and an autonomous run, abcd runs the provably-spent part of abcd sweep --apply on its own (the same proofs: merged into origin's default branch, clean, not live, no reservation, local tier archived first) and reports what it removed and what it left; the guard admits that sweep's removals as proven, so no person has to allow them one by one."
resolution: "abcd now runs the provably spent part of an applied sweep on its own in a managed repository: after a landing sees its pull request merged, when a build start or resume closes a merged lane, when a drain move finishes a merged lane or ends the drain, and at a terminal drive's end unless it was killed, halted or stopped; it is bounded at two minutes, stops between removals, and deletes only the branches abcd made, a branch named exactly as the build loop names a lane, keeping every other branch, a store worktree's included, for the applying sweep; sweep.automatic false turns it off"
impact: additive
resolved_by:
  commit: "dddab8beff7cc8d450188af1695ecbe3c5abcbf7"
---

In a repository abcd manages, nothing clears provably finished worktrees and branches on its own, so they pile up until a person removes them. On 2026-11-15 a person had to run a cleanup of 91 worktrees and their branches, every one already merged, because an agent cannot: the shell guard refuses an agent's worktree removal whenever the worktree holds ignored files such as its local tier (iss-2610311249121083), and abcd sweep (itd-2610222214476887), still being built, runs only when a person runs it and puts a scheduled or post-merge sweep out of scope as a later decision. The product thinker ruled on 2026-11-15 that this cleanup happens autonomously in a repository abcd manages. Third occurrence after iss-2610050532499080 and iss-2610130501296659.
