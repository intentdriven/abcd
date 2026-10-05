---
schema_version: 1
id: "iss-2610040847166532"
slug: "an-ai-written-interview-s-role-can-slow-the-change-guard-by"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "abcd-e8: focused re-check of plain-Terminal step 4's worktree fix, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/interview/treewatch.go"
remedy: "Bound a followed link's target before hashing it: stop and refuse, naming the link, once a followed target passes a small file count or byte budget (receipts and hook directories hold tens of small files), rather than walking to the general 500,000-path bound. Grounds: the re-check's probe of followLink at treewatch.go:567."
resolution: "What the links one reading of the change guard follows lead to is held to 1,024 entries and 64 MiB in all, charged before each entry is read, and a reading past it is refused naming the link (TestAFollowedLinkIsBounded: a turn-directory receipts link to a 128 GiB sparse tree refuses in under half a second, where it read for minutes)."
impact: fix
resolved_by:
  spec: "spc-2610040847280931"
  commit: "9e1c3e24e20b90dc928a66f65e1adefc754be445"
---

An AI-written interview's role can slow the change guard by planting a link to a huge directory tree where the guard follows links (for example its turn directory's preflight-receipts made a symlink and registered as a worktree): followLink content-hashes up to maxWatched (500,000) files before refusing. The guard still fails closed (the interview stops and nothing is accepted), but a turn can take minutes first.

## Grounds

- pursued: no legitimate followed target (git's hooks and info, a dotfiles hooks directory, at most 50 push receipts) comes near the budget, so no real interview is refused by it; a person's linked hooks directory past 1,024 entries or 64 MiB being refused would show the budget too small
