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
---

An AI-written interview's role can slow the change guard by planting a link to a huge directory tree where the guard follows links (for example its turn directory's preflight-receipts made a symlink and registered as a worktree): followLink content-hashes up to maxWatched (500,000) files before refusing. The guard still fails closed (the interview stops and nothing is accepted), but a turn can take minutes first.
