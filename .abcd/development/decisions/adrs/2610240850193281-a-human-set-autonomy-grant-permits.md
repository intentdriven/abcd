---
id: adr-2610240850193281
slug: a-human-set-autonomy-grant-permits
status: proposed
date: 2026-10-24
supersedes: null
superseded_by: null
related_intents: [itd-2610240850336695, itd-2610180912189125, itd-2610170638184486]
related_rfcs: []
related_adrs: [adr-2610150603484802]
---

# ADR-2610240850193281: A human-set autonomy grant permits, separately from the agent-written waiting-on badge

## Context

`abcd mode` carries two things. The badge (`managed`, `facilitator`,
`product-thinker`) reports whose answer the loop is waiting on; a question's
chip sets it and the answer resets it, so in practice the agent writes it. The
away mode (`abcd mode autonomous`, `queue`, `back`, itd-2610180912189125)
records that the person is away and grants no rights: no reader in `build`,
`implement`, `drain` or `landing` consults it, and `--hold-pushes` changes only
the stop hook's wording. The one setting meant to grant something,
`facilitator.mode: auto|human` (adr-2610150603484802), lives in the
repository's config and is not built.

The away mode first granted a narrow right, a session pushing its own branches
and opening their pull requests (`dcn-2610181003456444`).
`dcn-2610181016510153` reversed it: the OS sandbox fences only the agents abcd's
runner launches, never the person's own interactive sessions, so guard entries
"would never become a boundary" and would buy "a trust claim the plan cannot
make".

The consequence surfaced on 2026-10-24: a session refused to push a branch and
open a pull request without a standing permission, and the person found no
setting that gives one. The permission lived only in chat.

State of the art, read on 2026-10-24: Claude Code's permission modes, Codex's
approval and sandbox presets, Cline's auto-approve toggles and LangChain's
per-tool interrupts are all grants the harness enforces, none of them an OS
boundary for the person's own session. Each is kept apart from the session's
"needs input" status (Claude Code's agent view states the separation), each is
built from per-capability grants with named presets on top, and each keeps a
short list of irreversible acts gated at every level (force-push, merging an
unapproved pull request, production deploys; GitHub's coding agent cannot
merge at all). None of them expires a grant on a clock.

## Decision

We will treat a grant and a fence as different things. A grant says what a
session may do without asking; it needs no sandbox, because it widens nothing
the session could not already do and only removes the stop. A fence restricts,
and stays advisory where abcd cannot enforce it, as `--hold-pushes` already is.

We will therefore give the person one autonomy grant, kept apart from the
badge:

- The person alone sets it, in their home (`~/.abcd.noindex/`), keyed on the
  repository's root commit. A repository's `.abcd/config.json` may remove a
  capability from it and never add one.
- It is built from four capabilities, `push-branch`, `open-pr`, `queue-merge`
  and `decide-as <role>`, with presets on top; a grant that names none carries
  "ship work" (`push-branch`, `open-pr`, `decide-as facilitator`) once the
  pre-run sweep has confirmed the outstanding design work
  (itd-2610240850336695, Decisions 2 to 4).
- Merging to the default branch, cutting a release and force-pushing are in no
  preset, and abcd refuses them in code at every level.
- It ends at a time the person names; an ended or unreadable grant reads as no
  grant.
- It holds whether the person is present or away: being away stays its own
  record, and ending it does not end the grant.
- It is the one place that lets abcd merge on its own; the planned
  `facilitator.mode: auto` folds into its `queue-merge` capability.
- Only the person sets it: the guard refuses a session that sets or writes it.
- The badge stays the agent's report of whose answer is owed; the status line
  shows the grant beside it.

This supersedes `dcn-2610181016510153` on rights, not on the away mode's other
behaviour.

The grant and the fence are two axes, as Codex keeps its approval policy apart
from its sandbox mode. Where abcd puts up a fence, the grant is embedded in it:
the agents abcd's runner launches run inside the operating-system sandbox of
itd-2610161621364075 (its part A), and the grant says what they may do there
without asking. Where abcd cannot fence, the person's own interactive sessions
(that intent's part B), the grant stands alone, abcd claims no boundary, and
the person's host settings stay the outer fence (product thinker, planning
interview, 2026-10-24).

## Alternatives Considered

- **Keep the away mode as a record that grants nothing** (the standing
  decision). Rejected: the person's permission then lives in chat, one session
  hears it and its peers do not, and compaction can drop it. Every tool
  examined ties the human control to what the agent may do.
- **Grant rights only where abcd can fence them** (the runner's sandboxed
  agents). Rejected: it leaves the person's interactive sessions, the ones
  that paused on 2026-10-24, with no grant at all.
- **A single autonomy level** (off, low, full). Rejected: every tool examined
  builds its levels from per-capability grants, and a single level cannot say
  "push and open pull requests, but do not decide product questions".
- **Per-capability grants with named presets, irreversible acts outside every
  preset** (chosen). It matches the state of the art and keeps the acts the
  person cannot undo with the person.

## Consequences

- A session may push its own branch and open a pull request under a grant
  without asking, and `abcd mode back` must list what it did.
- abcd takes on a refusal it does not have today: merge to the default branch,
  release and force-push, refused at every grant level by the guard.
- The badge loses its human writer on the `/abcd:mode` page; the page leads
  with the grant.
- `facilitator.mode` must be reconciled with the grant: folded in, or kept as
  the repository's opt-in for merging.
- The grant is a permission, not a sandbox: a session that ignores it is not
  stopped by it, and the documentation must say so.
