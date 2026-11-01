---
id: adr-2611011813190527
slug: a-repository-declares-a-sequence-s-shape
status: proposed
date: 2026-11-01
supersedes: null
superseded_by: null
related_intents: [itd-2611011813162141]
related_rfcs: []
related_adrs: [adr-42, adr-2610091150447054, adr-2610161621415708]
---

# ADR-2611011813190527: A repository declares a sequence's shape but never a widening flag

## Context

itd-2611011813162141 lets a repository name a sequence of abcd's own verbs, which a person runs with `abcd run <name>`.

The chains people want most often carry flags that widen what abcd does: `--approve`, `--apply`, `--release`, a push or enqueue, an update. Three facts make a repository file the wrong place for those flags:

- **The guard is a mistake filter, not a boundary** (adr-42). A renamed command hides an agent's flag from the guard's registry. A sequence named `ship` that expands to `land --approve` would present the agent's command as `abcd ship`, and the registry entry that refuses an agent's approval would never fire.
- **Make targets and package scripts are launchers the guard never sees** (adr-2610091150447054).
- **Running unconfined is the person's choice, and no repository file can make it for them** (adr-2610161621415708). Workspace-trust tools work the same way: VS Code never runs a task automatically in an untrusted workspace (https://code.visualstudio.com/docs/editing/workspaces/workspace-trust).

## Decision

We will let a repository declare a sequence's shape and nothing more:

- the sequence holds only abcd's own verbs and non-widening flags;
- it may contain no shell step;
- it may not add a top-level name or shadow a built-in verb.

abcd expands each step and judges it against the guard's registry before running it. A widening flag comes only from the person's own invocation or their home configuration, never from the committed sequence. A sequence that names one is refused when its file is read, and the refusal names the flag.

## Alternatives Considered

- **Sequences may carry any flag.** Rejected, because renaming would hide an agent's widening flag from the guard.
- **Sequences may run shell.** Rejected. That reopens the code-execution hole of package lifecycle scripts in an untrusted clone, and a Makefile or script behind a host slash command already does this.
- **Widening flags allowed once the person trusts the repository from their home.** Not adopted now. It is a possible later step, if a repository's owners show they need it.
- **Shape from the repository, teeth from the person. Chosen.** A committed sequence then does no more than typing its verbs one by one.

## Consequences

- A sequence is safe to run in someone else's clone: at worst it does what the clone's verbs already allow.
- The verbs that sequences most often chain need stable exit codes and safe re-runs, because a sequence stops on a refusal or a back-off and is run again after the cause is fixed.
- Work that genuinely needs approval, release or push keeps those flags in the person's own command, or in a native composite verb with its own refusals (`land`, `build`, `sweep`).
