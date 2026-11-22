---
schema_version: 1
id: "iss-2611221134035558"
slug: "the-load-gate-s-trim-at-the-disk-floor"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "run-1122 fix-gate-trim-1122"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/cachetrim_floor.go"
remedy: "Add a machine-config key naming a toolchain's cache directory, its version files and an age keep rule, and a trim step for it under the guards cachetrim.go keeps for Go, or record why abcd trims only caches whose layout it knows."
---

The load gate's trim at the disk floor is a per-toolchain step, but Go's build cache is the only step: a repository in another language trims nothing and the waiter says so. Declaring another toolchain's cache directory and keep rule in the machine config (~/.abcd.noindex/config.json) was left out of iss-2611220942266387, because abcd cannot tell an entry from a tool's own index in a cache whose layout it does not know, so a declared trim needs its own guards: a directory the account owns and no one else can write, never a symlink, regular files only, each time re-read before the unlink.
