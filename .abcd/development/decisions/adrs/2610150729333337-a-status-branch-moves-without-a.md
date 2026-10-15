---
id: adr-2610150729333337
slug: a-status-branch-moves-without-a
status: proposed
date: 2026-10-15
supersedes: null
superseded_by: null
related_intents: [itd-2609151838312703, itd-2609091034175565]
related_rfcs: []
related_adrs: [adr-2610031352212697, adr-38]
---

# ADR-2610150729333337: A status branch moves without a preflight receipt

## Context

[itd-2609151838312703](../../intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md) has sessions post announcements to the shared remote. They post from any account, any machine and hosted cloud sessions. Each announcement is an empty-tree commit on the person's own branch, `abcd-status/<forge username in lower case>`.

Three constraints were already locked:

- **The push hook needs a receipt.** `.githooks/pre-push` refuses any push whose new commit carries no preflight receipt.
- **Cloud sessions have no receipt.** A hosted cloud session cannot mint one, because it cannot pass preflight there (itd-2610120738075547). <!-- record-lint: forward-looking -->
- **The cloud proxy accepts branches only.** It refuses every ref that is not a branch, so a hidden ref namespace is out (itd-2609091034175565 decision 10).

The hook already admits one family of commits that change no file: reservations, under [adr-2610031352212697](2610031352212697-claims-are-branches-on-the-shared-remote.md). It does so with one row of its receipt-less table and the `reservation_shape` check.

## Decision

The push hook's receipt-less table gains a second row: the prefix `abcd-status/`, checked by the same `reservation_shape` check that reservations pass. That check admits only a fast-forward of commits that change no file. Every other push under the prefix is gated as any branch is: one that changes a file, rewrites the branch or deletes it.

Rewrites and deletions are blocked on the forge too. A ruleset on `refs/heads/abcd-status/**` blocks deletion and non-fast-forward pushes, with no bypass. Its mirror is kept in `.abcd/work/rulesets/`, and it is applied through the rulesets' own apply verb (brief invariant 10).

Announcements are fetched only by verbs whose documented meaning is that fetch: `abcd announce`, `abcd status` and the land step (brief invariant 7, adr-38). The prompt hook renders the last fetch from disk.

Invariants 7 and 10 gain clauses naming these verbs and the branch, written in the change that builds them.

## Alternatives Considered

- **A receipt for every announcement.** Rejected: a cloud session cannot mint one, so it could never announce.
- **A hidden ref namespace (`refs/abcd/status/*`) outside the branch list.** Rejected:
  - the cloud proxy refuses it;
  - the forge's rulesets cannot protect it;
  - the forge's activity record, which verifies senders, covers branches.
- **A second, announcement-specific shape check that parses trailers in shell.** Rejected: it would be a second parser beside the Go reader, which breaks one-canonical-primitive. The reader already flags an announcement it cannot parse (itd-2609151838312703 C11).
- **The chosen option: a second row reusing `reservation_shape`.** It admits nothing the reservation row does not already admit: commits that change no file.

## Consequences

- **Easier:** a cloud session, or any session without a receipt, can announce. The trust boundary is unchanged, because a push the row admits changes no file a gate would read.
- **Harder:** status branches can never be deleted. Their history is permanent and public, which the product thinker accepted (itd-2609151838312703 decision 19). They grow by one branch per person, never per session.
- **New obligations:**
  - keep the ruleset mirror and the live ruleset in step;
  - add the invariant clauses in the building change.
