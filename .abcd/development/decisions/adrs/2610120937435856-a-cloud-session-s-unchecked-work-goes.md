---
id: adr-2610120937435856
slug: a-cloud-session-s-unchecked-work-goes
status: accepted
date: 2026-10-12
supersedes: null
superseded_by: null
related_intents: [itd-2610120738075547, itd-2609091034175565]
related_rfcs: []
related_adrs: [adr-2610031352212697, adr-39, adr-44]
---

# ADR-2610120937435856: A cloud session's unchecked work goes only to a declared work-in-progress prefix that never reaches the default branch

## Context

The pre-push hook refuses any push whose new commit carries no preflight
receipt, and skipping the hook is forbidden. On 2026-10-12 a cloud session
finished a fix (two commits and a resolved record) on a machine that could not
pass preflight, so it could mint no receipt, could not push, and the work was
redone by hand on the person's computer
([iss-2610120729087253](../../../work/issues/open/iss-2610120729087253-cloud-session-work-trapped-without.md)).
The intent that answers the six findings of that day,
[itd-2610120738075547](../../intents/planned/itd-2610120738075547-a-person-hands-an-item-from-their.md),
deferred the question of whether such work may reach the shared remote (its
Q3) and the branch name it would use (Q3b).

Already locked when this was decided:

- **No cloud receipt is trusted** (the intent's decision 9, ruled by the
  product thinker): every cloud result is re-checked by preflight on the
  person's computer before it lands. The receipt was therefore never what made
  cloud work trusted.
- **Reservations are receipt-less branches too.**
  [adr-2610031352212697](2610031352212697-claims-are-branches-on-the-shared-remote.md)
  and the reservation build (itd-2609091034175565) admit
  `abcd-reserved/<record-id>` without a receipt when every commit has the
  empty tree. That build is turning the hook's exception into a declared list
  of receipt-less prefixes, each entry naming the prefix, the shape a commit
  under it must have, and the record that rules it.
- **The core is host-agnostic** ([adr-39](0039-host-tier-policy.md)): one
  host's names live in its connector, never in the core.
- **The hosted proxy accepts only branch pushes**, refuses deletions and
  blocks the forge's GraphQL interface (the intent's evidence of 2026-10-12).

The product thinker delegated the question on 2026-10-12: "decide the deferred
item so that you can plan the intent. I want it ready to be build in a few
days (while I'm away)". The technical facilitator decided it the same day.

## Decision

1. **A cloud session that cannot pass the checks still pushes its work, to a
   work-in-progress branch.** Every cloud session abcd starts pushes only to
   `<wip-prefix><item-id>`, whether or not its own checks passed, because no
   cloud receipt is trusted either way.
2. **The prefix belongs to the host connector.** The core and the hook never
   name it; each connector declares its prefix in its own declaration file,
   and the hook reads every declaration. A declared prefix must end in the
   path segment `abcd-wip/` and may hold at most one segment before it, so no
   declaration can name the default branch, a bare host namespace or every
   branch. For Anthropic's hosted sessions the prefix is `claude/abcd-wip/`:
   inside the host's own branch namespace in case its proxy admits only that,
   and narrow enough that a person's own hand-pushed `claude/` branches still
   need a receipt. That the proxy accepts a push to this name is a hypothesis
   the intent's first rung proves on a real hosted session.
3. **The hook admits a receipt-less push only as one entry of its declared
   list.** The entry names the connector's prefix, the commit shape (the
   branch is the prefix followed by one valid record id, and every commit in
   the pushed range passes the attribution gate: a human author, an
   `Assisted-by:` trailer, no AI co-author trailer, no session link), and this
   record as the rule. Every other branch, including every other branch under
   the host's namespace, needs a receipt as before. The non-fast-forward
   refusal and the default-branch refusal still apply first.
4. **No work-in-progress branch reaches the default branch.** A required check
   fails on any pull request or merge-queue entry whose source branch holds
   the path segment `abcd-wip`, or whose diff adds a cloud report file; the
   default branch's ruleset requires that check. The check matches the
   host-agnostic segment, never a host's prefix.
5. **The work lands only through the person's own collect.** Collecting a
   work-in-progress branch re-runs preflight on a copy on the person's
   computer and offers a new branch, under the person's name with a local
   receipt, for landing in the usual way (the intent's C9).

The invariant, carried in the brief as invariant 23: **no work-in-progress
branch reaches the default branch, and a receipt-less push is admitted only
for a host connector's declared work-in-progress prefix.**

## Alternatives Considered

- **No exception; push only work that passed preflight, and describe the rest
  in a report.** Rejected: it keeps the push gate as it is, but the work of
  2026-10-12 would still be redone by hand, which is the defect being fixed,
  and the receipt it protects is one decision 9 no longer trusts.
- **Admit the host's whole `claude/` namespace.** Rejected: a person who uses
  the host by hand also pushes `claude/` branches, and they would lose the
  receipt check without asking for it.
- **An abcd-only prefix outside every host namespace (`abcd-wip/`).** Rejected
  for the first host: its proxy may admit only its own namespace, and a
  session that cannot push is the trap again. The segment `abcd-wip` is kept
  inside the host's prefix so the required check stays host-agnostic.
- **A custom ref namespace (`refs/abcd/wip/*`).** Rejected: the proxy refuses
  any ref that is not a branch.
- **A git bundle uploaded somewhere abcd collects.** Rejected: the hosted
  machine has no sanctioned place to put one, and it adds a store to keep.
- **A declared work-in-progress prefix, one entry in the hook's list, kept off
  the default branch by a required check and landed only by a local collect.**
  Chosen: the trap ends, the gate's trust rests where decision 9 already put
  it, and each half has a test that can hold it.

## Consequences

- On a public repository, unfinished cloud work is public. Accepted by the
  facilitator; the intent refuses to hand a cloud session any item under a
  security embargo or any record not yet public, because pushed work is
  public.
- The hook's receipt exception grows by one declared entry; it never becomes a
  second special case. The step that touches the hook builds on the
  reservation branch's declared list landing first.
- A new required status check joins the default branch's ruleset, and the
  ruleset mirror under `.abcd/work/rulesets/` changes with it. Applying the
  ruleset on the forge is the person's act.
- Spent work-in-progress branches accumulate, since the proxy refuses
  deletions; the tidy-up is the reservation record's sweep from an ordinary
  machine.
- The hook is not a security boundary (`--no-verify` skips every hook); the
  required check, the ruleset and the local collect are. A test proves the
  required check refuses a work-in-progress source branch.
- The brief carries the rule as invariant 23 in
  [`02-constraints/03-invariants.md`](../../brief/02-constraints/03-invariants.md).
  Invariant 22 is held by the private-advisory trust rule on the embargo
  branches, not yet on the default branch.
