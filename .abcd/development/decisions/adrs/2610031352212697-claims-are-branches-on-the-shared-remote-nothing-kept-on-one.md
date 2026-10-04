---
id: adr-2610031352212697
slug: claims-are-branches-on-the-shared-remote-nothing-kept-on-one
status: proposed
date: 2026-10-03
supersedes: null
superseded_by: null
related_intents: [itd-2609091034175565, itd-2609150819440345, itd-2610031259176838, itd-2609221656373558]
related_rfcs: []
related_adrs: [adr-2609091248200336, adr-38]
---

# ADR-2610031352212697: Claims are branches on the shared remote; nothing kept on one computer is the register; claim text is data

## Context

On 2026-10-03 a second person began working on abcd from their own computer,
the repository was public, and a hosted cloud session with its own clone was
admitted as one more participant. Everything abcd used to keep agents apart
lived on one computer: The shared run's exclusive-create lease under the
machine-scoped run store
([itd-2609221656373558](../../intents/shipped/itd-2609221656373558-two-orchestrator-sessions-in-one-user-account-share-an.md)),
the sibling-worktree listing, and notes the sessions left each other. None of
it reaches the second person or the cloud session, so their agents and ours
could each start the same item, and the first anyone would hear of it is the
collision at merge
([iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md)).

The product thinker ruled the same day that coordination must be visible to
every participant, that a claim is a marker on the shared remote which abcd
lists, and that reserving one item is built first
([itd-2609091034175565](../../intents/planned/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md),
decisions 1 to 4;
[itd-2609150819440345](../../intents/drafts/itd-2609150819440345-which-session-holds-which-worktree-branch-or-record-is-coord.md),
decision 4). The state of the art and the run lessons behind those rulings
are in
[`2026-10-03-multi-agent-coordination-sota-and-run-lessons.md`](../../research/notes/2026-10-03-multi-agent-coordination-sota-and-run-lessons.md).

Two adversarial reviews of the design followed, and their evidence fixed the
transport rather than leaving it to preference:

- A hosted cloud session's git proxy rejects branch deletions and pushes of
  anything other than a branch; a probe on 2026-10-03 got HTTP 403 on a ref
  under `refs/abcd/` while branch pushes succeeded. A ref outside
  `refs/heads/` cannot include the cloud session.
- abcd's hazard registry refuses every `--force-with-lease` push, and the
  proxy refuses deletions, so neither a lease flag nor a deletion is
  available. A plain push already compares and swaps: a parentless commit
  over an existing branch is rejected as non-fast-forward, and two children
  of one parent admit exactly one.
- The committed pre-push hook demands a preflight receipt for every new
  commit, so it refuses a claim push today; an exemption by branch name alone
  would let code past the receipt gate.
- A claim's text is written by whoever has write access and is read by
  agents, and the 2026 incidents in the state of the art are agents following
  instructions planted in exactly such channels.
- Anyone with write access can push to or delete any branch outside the
  default branch, which is all this repository's rulesets protect.

Brief invariant 7 already holds that implicit operations never touch the
network ([adr-38](0038-implicit-checks-are-disk-only.md)).

## Decision

We will hold a claim on a record as a branch on the shared remote,
`refs/heads/abcd-reserved/<record-id>` (a person reads it as a reservation; the record and the code call it a claim), and nowhere else:

1. **The branch is the transport, and the only one.** A claim branch is a
   chain of commits with the empty tree, which nobody merges and abcd lists.
   It is a branch because a hosted cloud session can push nothing else. No
   hidden-ref backend sits beside it. Every claim act is a plain fast-forward
   push: a parentless create, a renewal or takeover as a child, and a release
   as a tombstone child; abcd uses no lease flag and deletes no claim branch.
2. **Nothing kept on one computer is the source of truth for who holds
   what.** The local tier, the machine-scoped run store, a desktop note and
   same-account messaging may speed up notice; the claim branch on the remote
   decides. A claim that cannot reach the remote is refused loudly and is
   never recorded locally as if it were shared.
3. **Claim text is data.** A claim is parsed by its known trailer keys and
   rendered as quoted values; an unknown key is ignored, a malformed or
   oversized claim is reported as unreadable, and no part of a claim is ever
   handed to an agent as instructions. Footprint paths never reach a shell or
   a refspec, and the branch name is built only from a valid record id.
4. **Claims are advisory among trusted writers.** A claim refuses nothing at
   the forge: Any writer can push to it or delete it. abcd's verbs honour it,
   expiry is judged by the reader, and the land stage re-checks it before
   arming a merge.
5. **The pre-push exemption covers empty-tree claim commits only.** The
   repository's pre-push hook passes an update to `refs/heads/abcd-reserved/*`
   without a preflight receipt only when every commit in the pushed range has
   the empty tree; every other push, and every non-fast-forward push, is
   gated as before.
6. **Every network act is an explicit verb.** `abcd reserve` and its
   release, `abcd build`, `abcd drain` and `abcd peers` push or fetch
   reservations because that is their documented meaning, and `implement
   step` reads and writes the run's reservation because its documented
   meaning is extended to say so; a branch push is not remote configuration.
   A reservation stops counting once its record is terminal on the default
   branch, so no step after the merge is needed. Every render, gate, hook and session start
   reads the claims only as last fetched, from disk.

## Alternatives Considered

- **A ref outside `refs/heads/` (`refs/abcd/claims/<record-id>`), hidden from
  the forge's website.** The product thinker's first choice; rejected on the
  cloud probe, because the cloud session could neither claim nor release.
- **Both a hidden ref and a branch.** Rejected: Two backends are two bug
  surfaces, and the hidden ref buys nothing a cloud session can use.
- **A lease kept on one machine (the run store, the shared `.git`, or the
  host's session registry).** Rejected by the visibility ruling: The second
  person never sees it.
- **A `claimed_by` stamp on the record, pushed through a pull request.**
  Rejected: An older reader skips a record with an unknown key, so it needed a
  two-release migration, and each claim paid a merge-queue pass of fifteen to
  sixteen minutes.
- **A forge claim (an issue assignment or a draft pull request).** Kept for
  outside contributors, who cannot push a branch
  ([itd-2610031259176838](../../intents/drafts/itd-2610031259176838-someone-outside-the-project-can-reserve-a-piece-of-work-too.md));
  it needs a forge client and token that a cloud session lacks.

## Consequences

- A claim costs one push of seconds, with no pull request, queue pass or
  workflow run, since every push trigger filters on the default branch or
  release tags.
- Claim branches show in the forge's branch list and may raise a
  compare-and-pull-request banner; every claimed record leaves a tombstoned
  branch, so the clean-up sweep of itd-118 is owed before the forge's 5,000-branch limit
  matters.
- A malicious or careless writer can delete or overwrite a claim; the design
  accepts this among trusted writers and does not defend against it.
- Brief invariant 7's documented-meaning entries must name the reservation
  verbs and `implement step`'s reservation write, and invariant 10 gains one
  clause saying a git branch push is not remote configuration (the technical
  facilitator's ruling of 2026-10-03, itd-2609091034175565 decision 12); both
  land in the change that builds the reservation, not before.
- The shipped `implement claim` and the remote claim judge liveness through
  one function, so the lease arithmetic has one home.
