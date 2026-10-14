---
schema_version: 1
id: "iss-2610112042253783"
slug: "lab-and-parity-checkouts-get-lfs"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "LFS fix lane, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lab/mint.go"
remedy: "Decide per caller whether its tree must hold real file content (a lab snapshot that runs the repository's code does; a parity check that compares blobs may not), route the ones that do through gitutil.WorkingTreeCommand, and prove each with the stand-in global filter test used for lanes"
---

abcd's lab snapshot (internal/core/lab/mint.go) and launch parity check (internal/core/launch/parity.go) run `git checkout` through gitutil's isolated environment, which drops the person's global git config, so in a repository whose Git LFS filter is set up by a normal global `git lfs install` their snapshot trees hold LFS pointer text instead of the real files. Same cause as iss-2610111729375108 (fixed for lane worktrees by gitutil.WorkingTreeCommand); noticed by that fix's sweep on 2026-10-11, not reproduced for these two callers.
