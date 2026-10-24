---
id: adr-2610241022541063
slug: each-person-s-commit-identity-lives-on
status: accepted
date: 2026-10-24
supersedes: null
superseded_by: null
related_intents: [itd-2610240918547728, itd-131, itd-178]
related_rfcs: []
related_adrs: [adr-38]
---

# ADR-2610241022541063: Each person's commit identity lives on their own machine, not in the repository

## Context

[itd-131](../../intents/shipped/itd-131-managed-repo-identity-gate.md) made
the commit identity gate check git's effective author and committer "against
the identity the repo pins" in a committed `.abcd/config/identity.json`, and
made adopting that pin a step of `abcd ahoy install`. The pin names one
person. In a repository that takes outside contributions that is wrong for
everyone else: setup reports any other contributor's identity as a required
`git_identity.mismatch` gap and offers to make them commit as the pinned
person, and abcd's own committed `.githooks/pre-commit` refuses their commits
with a fix line telling them to set the maintainer's name and email
([iss-2610240830397793](../../../work/issues/resolved/iss-2610240830397793-the-committed-identity-pin-names-one.md)).
That contradicts the attribution rule that every commit is honestly authored
by the human who made it.

The pin did two useful things, and both had to survive: it caught a commit
made under a bot's or an AI tool's identity, and it caught a mistyped email of
one's own. The same committed file also carried the repository's default
`production_mode` (itd-178), which is the repository's setting, not a
person's. And an unattended routine on a fresh machine (a cloud container)
has no person at a terminal to establish an identity for.

The product thinker accepted the reversal on 2026-10-24, in the Decisions of
[itd-2610240918547728](../../intents/planned/itd-2610240918547728-every-person-who-commits-to-a-repository.md),
after considering and declining the alternative of dropping the pin and
checking only that the author is a human.

## Decision

We will keep each person's commit identity on their own machine, and nothing
in a repository will name who may commit to it.

- The record is `{"name": "…", "email": "…"}` under the user-level home, read
  most specific first: `~/.abcd.noindex/identity/<root-sha>.json` for one
  repository, keyed on the full object name of its root commit (as the
  worktree and transcript stores are keyed), else
  `~/.abcd.noindex/identity.json`, the machine-wide default. Per repository
  with a machine-wide default, so a person with a work and a personal identity
  keeps each where it belongs while the common case is one file.
- It is read under the guards every home-scoped declaration takes
  (`fsutil.ReadHomeDeclaration`, the rules loader's): a regular file this
  account owns that no other account can write, capped in size, refused behind
  a symlinked `~/.abcd.noindex`. A record that is there and fails a guard, or
  does not parse, fails the check closed; it never reads as no record.
- The commit check (`identity.Check`, `abcd ahoy --identity`, and the
  self-contained shell gate in abcd's own `.githooks/pre-commit`, which reads
  the same two files in the same order from `$HOME`, with no binary) holds the
  author and committer to that record, and refuses a machine identity, from
  the one list `internal/core/identity/tool-identities.txt`, in either role
  whether or not a record exists. With no record, a human identity commits:
  an unattended run on a fresh machine commits as the human identity its
  launcher supplied, or is refused as a machine.
- Setup proposes a person's usual git identity and records it only on their
  yes, as its own question, only at a terminal; off a terminal, or under the
  approve-everything flag, it records nothing and says so.
- A repository that still stores the pin is migrated on its owner's machine
  (the person whose git identity is the stored one), on a yes at a terminal:
  the pin becomes that person's record for the repository, its
  `production_mode` moves to `.abcd/config/provenance.json`, and
  `identity.json` leaves the working tree. On anyone else's machine it is not
  copied, not removed and not named.
- The repository's default production mode stays committed, in
  `.abcd/config/provenance.json`; while a repository has not migrated, the
  legacy pin's `production_mode` member is still read when that file is
  absent.

This supersedes, from itd-131, the decision that the gate checks commits
against an identity the repository pins; the scope item that adopting the gate
writes `identity.json` behind the config-change approval; and the first link
of the disk-only proposal chain, "pinned → global", which becomes "the
person's own record → global". Its other decisions stand: detection here and
establishment by whatever launches a routine (iss-2608210932052003), a
disk-only proposal chain with no network lookup (adr-38), and
propose-and-confirm at a terminal before repo-local `user.*` is written.

## Alternatives Considered

- **Keep the committed pin** (itd-131 as shipped). Rejected: it names one
  person to every contributor and tells them to commit as that person.
- **Drop the pin and check only that the author is a human and matches the
  committer.** Considered and declined by the product thinker: it keeps the
  machine refusal but loses the check that catches a mistyped email of one's
  own.
- **A machine-wide record only.** Rejected: a person who commits to work
  repositories under a work identity and to their own under a personal one
  would be refused in one set or the other.
- **A per-repository record only.** Rejected as the only form: every
  repository would need its own question, while most people commit under one
  identity everywhere.
- **Chosen: per repository, with a machine-wide default**, kept on the
  person's own machine.

## Consequences

- No contributor is ever shown or offered another person's identity, and a
  repository that takes outside contributions needs no per-person setup.
- The commit check is now a person's own commitment: until they record an
  identity, only a machine identity is refused. A person who never runs setup
  at a terminal is checked for machines alone.
- An unattended run on a fresh machine needs no record: its launcher supplies
  the human identity, and a machine identity is still refused. A git identity
  git invents from the account and host name, when none is configured, is not
  distinguishable from a supplied one inside the hook; the Go check reports it
  as `git_identity.unset`.
- The shell gate in abcd's own pre-commit hook now reads the person's home
  directory, and is at least as strict as the Go reader (it refuses a
  group-writable folder the Go reader admits when the group is the owner's
  own).
- `.abcd/config/provenance.json` is a new committed file, and
  `provenance.DeclaredMode` is the one reader of the repository's default
  production mode; `identity.DeclaredProductionMode` is gone.
- abcd's own repository still stores the maintainer's pin until the
  maintainer's next setup moves it; nothing reads it to check commits in the
  meantime.
