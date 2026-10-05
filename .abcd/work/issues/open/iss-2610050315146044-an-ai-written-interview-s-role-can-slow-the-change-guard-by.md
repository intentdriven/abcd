---
schema_version: 1
id: "iss-2610050315146044"
slug: "an-ai-written-interview-s-role-can-slow-the-change-guard-by"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25: lane e8term-r3, probing the followed-link budget"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/interview/treewatch.go"
remedy: "Hold every hooks directory core.hooksPath names outside the tree to the same budget as a followed link (maxFollowedEntries and maxFollowedBytes, shared with the reading's links), refusing past it and naming the configured value: a hooks directory holds tens of small scripts (this repository's .githooks is 5 files and 102 KiB, git's sample hooks 14 files and 26 KiB), so a hooks directory past 1,024 entries or 64 MiB is not one git should run from unread. Grounds: the probe on a scratch copy (TestProbeHooksPathToALargeTree, 16 GiB in about 7 s) and readGitDirs' hooksPath loop, which calls hashIn with no budget."
---

An AI-written interview's role can slow the change guard by writing core.hooksPath into .git/config, naming a large directory outside the tree: the reading after the turn hashes every file there (toward the 500,000-path bound, with no byte bound) before it reports the configuration change. A probe with a 16 GiB sparse file held the reading about 7 seconds, growing with the size. The guard still fails closed (the change to .git/config and the planted directory are named), but a turn can take minutes first. The followed-link budget (iss-2610040847166532) does not cover it, because the hooks directory is resolved and read whole, not reached through a followed link.
