---
id: adr-2610150603484802
slug: a-fresh-install-s-facilitator-is-a
status: accepted
date: 2026-10-15
supersedes: null
superseded_by: null
related_intents: [itd-97, itd-2610150602547271]
related_rfcs: []
related_adrs: [adr-2609151528057260]
---

# ADR-2610150603484802: A fresh install's facilitator is a person until the repository says auto

## Context

[adr-2609151528057260](2609151528057260-three-roles-who-each-artefact-addresses.md)
settled that the technical facilitator is a mode: "AI by default and human on
activation". Since then the security standard rule
(`.abcd/development/principles/abcd-fixes-what-it-adds.md`) has said that abcd
adds no vulnerability and never changes how a repository works. These two
records collide the first time the mode governs an outward act. Under an AI
default, installing abcd in someone's repository and changing nothing would
let abcd merge their pull requests with no person involved.

The collision became concrete on 2026-10-15. iss-2610150524246898 asks abcd to
queue every ready pull request at once, and the product thinker ruled that the
queueing follows the facilitator's mode: with abcd as facilitator it happens at
once, and with a person the merge waits for them. itd-2610150602547271 builds
that declaration as the first slice of itd-97. It needs a default.

## Decision

We will start every repository with a person as its facilitator. The
declaration `facilitator.mode` (`human` or `auto`, in `.abcd/config.json`) is
`human` when no layer sets it, and `human` from any layer wins over `auto`. A
repository becomes `auto` only by a deliberate setting. abcd's own repository
sets it in its committed config.

This amends the default clause of adr-2609151528057260, and only that clause.
The three roles, the facilitator being a mode, and every gate declaring its
behaviour in both modes all stand.

Ruled by the product thinker, 2026-10-15.

## Alternatives Considered

- **A person by default** (chosen): installing abcd changes nothing about who
  merges, so the security rule holds with no install-time question. Cost: a
  repository that wants abcd to facilitate must say so.
- **abcd by default**, as the roles decision said: the solo-by-default promise
  with no setup. Rejected: a silent default that hands merging to abcd is the
  change to how a repository works that the security rule forbids.
- **Ask at install**: an explicit choice with no silent default. Not chosen
  now: it still needs a default for repositories installed before it, and that
  default is this decision. It may be added later as an `ahoy` question that
  writes the key.

## Consequences

- itd-2610150602547271's merge gate reads `human` in every repository that has
  not opted in. There `abcd land` and the build loop arm no merge, and the
  guard refuses the merge commands.
- abcd's own repository must carry `facilitator.mode: auto` in the change that
  ships the gate, or its own sessions stop queueing.
- The README's "automated facilitator" ambition becomes opt-in. Docs that
  describe the facilitator as AI by default must say a fresh install starts
  with a person.
