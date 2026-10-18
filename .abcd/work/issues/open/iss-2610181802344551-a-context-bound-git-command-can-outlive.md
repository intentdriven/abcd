---
schema_version: 1
id: "iss-2610181802344551"
slug: "a-context-bound-git-command-can-outlive"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "fixing iss-2610051434156079"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/lazyfetch.go"
remedy: "Thread the caller's context into guardObjectRead, lazyFetchGuard, readFloorConfig and the version probe when the command is context-bound, so every git the call starts ends at the caller's deadline; test with a hanging git that answers the version probe"
---

A context-bound git command can outlive its caller's deadline: isolatedGitContext runs the partial-clone guard (lazyFetchGuard) first, and the guard's own git version probe and its floor-config read run with no context, so a git that hangs holds a caller on a deadline (the status verb, the prompt hook's reads) for as long as git hangs, though the command itself is killed at the deadline. Found when TestRootCommitContextKillsGitAtTheDeadline, run alone, took 20 s against a 500 ms deadline because its hanging fake git answered the guard's probes.
