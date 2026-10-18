---
id: adr-2610181003541997
slug: the-away-mode-s-stop-check-reads-the
status: proposed
date: 2026-10-18
supersedes: null
superseded_by: null
related_intents: [itd-2610180912189125]
related_rfcs: []
related_adrs: [adr-38]
---

# ADR-2610181003541997: The away mode's stop check reads the forge once, as its documented meaning

## Context

Brief invariant 7 says abcd asks the network nothing on its own: implicit
checks read disk only, and a fetch happens only when the user invokes a verb
or flag whose documented meaning is that fetch
([adr-38](0038-implicit-checks-are-disk-only.md)). The stop check of
itd-2610180912189125 runs from the host's Stop hook at every turn's end, which
no one invokes by name.

Two of the four kinds of owed work live only on the forge: whether a
session's own pull request is behind its base or failing its checks, and
whether one carries a reservation's work. Plan review round 2 showed that the
disk cannot answer the first in the night: the last-fetched default branch is
stale, so a pull request that fell behind after the last fetch never reads as
behind (blocker 2, dcn-2610181003464379). The drain's own reads of the remote
were written into invariant 7 the same way, as the documented meaning of its
verb.

## Decision

We will treat setting the away mode (`abcd mode autonomous`, which the
person runs or asks a session to run, and which grants no push rights,
dcn-2610181016510153) as invoking the stop check's forge read while the mode holds:
at most one pull-request listing of the person's open pull requests per
check (head and base branch, author, body, merge state, review decision and
check rollup), under a ten-second deadline. There is no cache, so a run of
eight blocks costs up to eight listings, each bounded by the deadline; a
reuse cache is remaining work of itd-2610170638184486 (dcn-2610181017056307).

We will keep the check disk-only whenever the away mode does not hold: no
turn's end while the person is present touches the network.

We will name a failed or late read, never hide it: the check falls back to
the disk kinds alone, with the pull requests named as not read.

We will add this listing to invariant 7 as the away mode's documented
meaning, beside the drain's.

## Alternatives Considered

- **Disk only, always.** Keeps invariant 7 untouched. Rejected: "behind" is
  then measured against a default branch nothing refreshes overnight, and a
  reservation's carrying pull request cannot be seen, so the stall the slice
  fixes (iss-2610170619247185) recurs on exactly the pull requests that fell
  behind in the night.
- **A fetch of the default branch at each check.** Answers "behind" from
  disk after a fetch. Rejected: it is a network read all the same, it still
  cannot see checks, reviews or links, and it would run while the person is
  present unless also confined to the away mode.
- **One listing per check while away, chosen.** One call, bounded in time,
  and only inside a window the person opened.

## Consequences

- Invariant 7 gains a third verb-scoped meaning; the zero-network test
  harness must show the present-mode check makes no forge call
  (TestPresentReadsDiskOnly).
- The forge client must be signed in for the away check to read pull
  requests; when it is not, the check says so and judges from disk, which the
  intent's scope conditions already name.
- A later intent that wants the forge read while the person is present needs
  its own amendment; this record does not cover it.
