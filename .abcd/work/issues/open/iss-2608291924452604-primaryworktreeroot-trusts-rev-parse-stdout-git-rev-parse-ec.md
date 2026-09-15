---
schema_version: 1
id: "iss-2608291924452604"
slug: "primaryworktreeroot-trusts-rev-parse-stdout-git-rev-parse-ec"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "v0.6.9-security-review"
found_at: "internal/core/banlist/worktree.go"
---

PrimaryWorktreeRoot trusts rev-parse stdout: git rev-parse echoes an unrecognised option to stdout and exits 0, and --path-format only exists from git 2.31, so on an older git the --git-dir, --git-common-dir and --show-toplevel answers read as the flag text plus the path and every comparison silently fails closed to no primary store; validate each answer as a single absolute path or drop the flag
