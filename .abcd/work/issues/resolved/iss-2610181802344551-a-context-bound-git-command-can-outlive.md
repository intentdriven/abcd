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
resolution: "A context-bound git command's partial-clone guard now ends at the caller's deadline: the promisor-config read takes the caller's context, and the git version probe is shared by every caller waiting on it and cancelled when the last caller with a deadline leaves, which waits for its git to end, so neither the caller nor the probe's git outlives the deadline. Callers with no deadline keep the probe to its own 30-second bound. Tests pin a hanging version probe (the caller returns at the deadline and the probe's git is gone) and a hanging config read on a git below the floor."
impact: fix
resolved_by:
  commit: "43fe855b8"
---

A context-bound git command can outlive its caller's deadline: isolatedGitContext runs the partial-clone guard (lazyFetchGuard) first, and the guard's own git version probe and its floor-config read run with no context, so a git that hangs holds a caller on a deadline (the status verb, the prompt hook's reads) for as long as git hangs, though the command itself is killed at the deadline. Found when TestRootCommitContextKillsGitAtTheDeadline, run alone, took 20 s against a 500 ms deadline because its hanging fake git answered the guard's probes.

## Grounds

- pursued: a status refresh or prompt hook on a deadline returns at it and leaves no git running whatever git does; a git process still alive after RootCommitContext returned at its deadline would show this wrong
