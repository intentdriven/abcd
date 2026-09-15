---
schema_version: 1
id: "iss-2609010818066790"
slug: "the-reading-assembler-test-s-tree-walk-lstats-every-path-it"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "treeSnapshot returns filepath.SkipDir at .git before the walk error is examined, so the walk never descends into a directory whose contents git mutates underneath it, and any other path that vanishes mid-walk is ignored rather than fatal."
impact: fix
---

The reading assembler test's tree walk lstats every path it enumerates without tolerating ENOENT, so a transient file under .git - git's background maintenance.lock - that exists at enumeration and is gone by the stat fails the test; it flaked TestAssembleDryRunWritesNothing on the macOS CI leg and is a time-of-check race any concurrent git activity can trigger

## Grounds

- pursued: a snapshot of the worktree has no business walking .git at all, and tolerating the error would have kept the race while hiding it. What would show this wrong is a dry run that writes into .git, which nothing in the assembler can do.
