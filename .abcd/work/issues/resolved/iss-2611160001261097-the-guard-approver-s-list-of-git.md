---
schema_version: 1
id: "iss-2611160001261097"
slug: "the-guard-approver-s-list-of-git"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/approvepaths.go"
remedy: "judge each such setting against git's documentation and list it, with one list serving the approver and the hook"
resolution: "The approver's list of git settings that start a program now holds each setting git 2.52's git-config(1) and Git LFS 3.7.1's git-lfs-config(5) document whose value is a command, a program path or a helper name, or that switches on a program git starts, refused as a local git config write and as a git command's own setting; the guard's hook never read this list and is unchanged; the drive's held set keeps its own list, and a test fails when a program class it holds is missing from the approver's."
impact: fix
---

The guard approver's list of git settings that name a program is not yet judged one by one against git's documentation
