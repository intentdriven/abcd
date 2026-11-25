---
schema_version: 1
id: "iss-2611251739559427"
slug: "guard-counts-abcds-own-build-as-work"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/abcd.sh"
remedy: "Count the binary `scripts/abcd.sh` builds and the allocator lock as disposable in the proof for a worktree removal, by the exact paths abcd writes. Grounds: abcd makes both, and either is rebuilt by the next command."
---

The shell guard counts the binary `scripts/abcd.sh` builds for itself in a worktree as uncommitted work, so a worktree in which any abcd command was run can never be proven safe to remove, even with HEAD on the default branch and nothing else in it. Met on 2026-11-25: the two ignored files were the built binary and the issue allocator lock, both made by abcd. Refines iss-2610311249121083, whose remedy names a lock file and an empty directory as disposable and not the build.
