---
schema_version: 1
id: "iss-2609091829549814"
slug: "parallel-agents-share-one-scratch-path-and-mutate-each-other"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "release-gate"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/principles"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the technical facilitator (lapsed-deferral triage, run A 2026-09-29): May the scratch-copy convention require a private directory the agent creates for itself (mktemp -d, or a named directory made with a plain mkdir that fails if it exists), as run A's lane rules already do? Both homes of the convention still name a shared path: AGENTS.md (a lane may not edit it) and itd-193's protocol example, whose conventional <scratch>/mut is the collision this record describes."
remedy: "Waits on the technical facilitator's ruling on the private-directory requirement: if adopted, amend AGENTS.md's 'A verifier works on a copy' bullet and itd-193's protocol example to make the copy in a directory the agent creates for itself (mktemp -d, or a named plain mkdir that fails if the name exists), never a conventional shared path such as <scratch>/mut, removed by exact path afterwards; if declined, state in itd-193 that the protocol assumes one agent per scratch directory. Prove an adoption with a record-lint banned token refusing the shared <scratch>/mut example."
---

Two agents working in separate worktrees were given the same session scratch directory and both used the same conventional subdirectory name for a mutation-testing copy, so one agent's mutation script rewrote the other's copy of the tree and restored it afterwards, which the second agent had no way to notice while it ran. Nothing was lost, because the mutation was reverted within the same run and neither worktree was touched, but the window is the hazard rather than the outcome: while a mutation is applied a gate reports on code nobody wrote, and the repository's own conventions say exactly that about mutating a tree, requiring the work to happen on a scratch copy and the tree to be proven clean before anything is reported. The convention covers the copy and not the collision, because it was written for one agent at a time and says nothing about where a scratch copy goes when several are running. The correction is cheap and belongs with the convention rather than in each brief: a scratch copy is made in a private directory the agent creates for itself, never at a shared conventional path, so two agents cannot pick the same one. Worth stating alongside it that a shared scratch directory is also where two agents can read each other's intermediate output and draw conclusions from work that is not theirs. Detector: a convention names the private-directory requirement where it names the scratch-copy requirement, and an agent brief that hands out a shared scratch path is a review finding.

## Remedy grounds (2026-09-29)

- Why: the operating system already guarantees the property; run A's lane rules require it and the lanes kept to it.
- Sources (consulted 2026-09-29): POSIX mkdtemp creates a uniquely named directory as if by mkdir(path, S_IRWXU) (https://pubs.opengroup.org/onlinepubs/9799919799/functions/mkdtemp.html); CWE-377 names predictable shared temporary paths as the weakness and a unique owner-only creation as the mitigation (https://cwe.mitre.org/data/definitions/377.html).
- Rejected: a registry or lock over scratch names, a new mechanism where the operating system primitive suffices (script-first MVP).
