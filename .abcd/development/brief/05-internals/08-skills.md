# Skills — Procedural Workflows That Aren't Commands

A new user-invokable surface has to be either a command or a skill, and the
choice is expensive to reverse: by the time you discover a skill is mutating state, downstream
contracts have hardened around the skill shape. This page holds the criterion that
makes the call at design time, and records where abcd came out.

**abcd ships zero skills.** The `/abcd:` namespace is commands only, and there is
no `skills/` directory in the tree. The three workflows once shipped as skills —
`consult`, `ingest`, and `prepare-this-repo` — are commands, because each mutates
state: the sources corpus, its ledger, the target repo. They live at
`commands/<name>.md` with chapters [`13-consult.md`](../04-surfaces/13-consult.md),
[`14-ingest.md`](../04-surfaces/14-ingest.md), and
[`15-prepare-this-repo.md`](../04-surfaces/15-prepare-this-repo.md).

The plugin carries two further surface kinds this choice does not cover, because
neither is something a person invokes by name: the agent prompts under `agents/`,
which verbs and reviewers dispatch ([`01-agents.md`](01-agents.md)), and the hook
entrypoints under `hooks/`. The release payload declares all four kinds. Only
commands and skills compete for the same slash-invoked namespace, and that
competition is what this page decides.

## Skill vs command — the criterion

A surface is a **skill** when all of these hold:

- The verb describes a *workflow that runs against existing content*: an
  interview, an audit, a review, a walkthrough, a stress test.
- Its output is *findings or suggestions only*, never artefact creation or
  modification.
- Re-running it on the same input has the same effect.
- It fits naturally as a workflow markdown file the agent reads and follows.

A surface is a **command** when **any** of these hold:

- The verb describes a *state change*: install, pack, unpack, capture, plan, ship.
- Its output includes new or modified artefacts, even alongside findings.
- Re-running it has different effects.
- It needs an acceptance block, side-effect documentation, or a checkpoint and
  resume protocol.
- It writes run reports under `.abcd/.work.local/logs/<verb>/`.

**When in doubt, ship as a command.** The earlier "ship as a skill first, promote
on mutation" guidance was overturned by the round-2 review, on the cost above.

The classification itself is enforced by reviewer judgement, not by lint: a lint
reads a directory, not a shape, so nothing mechanical tells a findings-only
workflow from a mutating one. The strict rule from that review: any logbook
output, any artefact mutation, any state change makes it a command.

## What a command carries

- A **brief surface chapter** under `04-surfaces/` describing acceptance criteria,
  interaction flow and side effects, or a sub-verb row in an existing parent's
  chapter. The CLI-only verbs that have a Go verb and no chapter of their own are
  listed under
  [§ Operator-internal verbs](../04-surfaces/README.md#operator-internal-verbs)
  with the record that delivered each.
- A **per-invocation report subdirectory** under the gitignored
  `.abcd/.work.local/logs/<verb>/` tier, for commands that emit run reports.
- A **status-and-help render** when called bare. That is a convention rather than
  a universal, and the [surfaces index](../04-surfaces/README.md#bare-invocation)
  carries the one enumeration of where it holds and where a parent prints usage
  instead.

Most of abcd's commands are **binary-backed**: a Go verb plus a `commands/` file.
Three are **host-delegated** — `/abcd:consult`, `/abcd:ingest` and
`/abcd:prepare-this-repo` — with a command page and no Go verb, so the workflow
runs in the host agent, and they have no bare-status render and no sub-verbs. That
is the shape a command takes when its work is host-delegated rather than owned by
the transport-agnostic core.

The mapping between command pages and binary verbs is one-to-one in neither
direction, and both exceptions are deliberate. Five verbs have a Go verb and no
command page: `changelog`, `completion`, `hook`, `rules`, and `spec`. Three
command pages carry no Go verb of their own name: the host-delegated three above.
Two of those three call no part of the binary at all; `/abcd:prepare-this-repo`
is the exception, running the binary's audit, install and identity verbs as steps
inside a workflow the host drives. See
[`04-surfaces/`](../04-surfaces) for per-command detail.

## Skills are not in `04-surfaces/`

`04-surfaces/` documents commands. A skill gets no surface chapter there: no
`NN-<verb>.md` file carrying acceptance criteria, interaction flow, and side
effects. The plugin's `skills/<skill-name>/SKILL.md` is the executable form; the
intent file, where one exists, is the canonical user-moment reference.

A skill does need a **row** in the surfaces registry,
[`04-surfaces/README.md`](../04-surfaces/README.md). The blocker-severity
`surface_coverage` rule reads the skills directory alongside the commands one and
counts every immediate subdirectory of `skills/` as a real surface, so the first
skill added there fails `make record-lint` until the table carries it. The registry
is the one place a skill and a command share: presence is machine-checked in both
directions, and only the classification is left to judgement.

## If a skill is ever added

The registration list is empty, and the auto-registration the plugin system
performs finds nothing. Moving the three workflows to `commands/` also closed
iss-61 — a shipped skill silently dropped from the cut artefact, because
`commands/` is in the release payload and `skills/` never was, and there are no
skills left to drop.

A later phase introducing a slash-invokable workflow that has no parent command and
is findings-only and idempotent per the criterion above gets: an intent file
capturing the user moment, a `skills/<name>/` directory holding the executable
form, an entry in this section, a row in the surfaces registry marked `shipped`,
the skills directory added to the release payload's include list, and **no**
surface chapter — because a skill needing one is command-shaped and ships as a
command instead.

The payload step is the one that is easy to forget and has not been taken for
anyone: the include list still names commands, agents and hooks and no skills
directory, which is exactly the condition that dropped a shipped skill from the
cut artefact before. A skill added without it passes the record gate and is
missing from the release.

itd-30 (design fictions, a later phase) is a **command extension** rather than a
new skill: it extends the canonical create `/abcd:intent "<text>"` with a format
flag.
