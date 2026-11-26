---
id: adr-2611251806496719
slug: the-spot-check-s-store-branch-is-pushed
status: accepted
date: 2026-11-25
supersedes: null
superseded_by: null
related_intents: [itd-2610301750162657]
related_rfcs: []
related_adrs: [adr-2610031352212697, adr-2610150729333337]
---

# ADR-2611251806496719: The spot-check's store branch is pushed without a preflight receipt

## Context

[itd-2610301750162657](../../intents/planned/itd-2610301750162657-a-person-spot-checks-what-abcd-s-agents.md) keeps the spot-check's record on one branch of the shared remote, `abcd-spotcheck`: an orphan whose tree holds small JSON files under `handbacks/`, `decisions/`, `cycles/`, `items/` and `judgements/`, beside one `window-open.json`, each written once and never edited. The draw runs inside a landing (`abcd land`, a lane's landing in `abcd implement step`, a drain move), and the review verb pushes a judgement per item.

Three constraints were already locked:

- **The push hook needs a receipt.** `.githooks/pre-push` refuses any push whose new commit carries no preflight receipt, and the store's push runs with the repository's hooks, never with `--no-verify`.
- **A store commit has nothing a preflight vouches for.** Its tree is the store's records alone, never the product's code, and it never merges into the default branch. A landing cannot mint a receipt for it: the commit is made after the landing, by abcd, in no working tree.
- **dcn-2611130729517836 and dcn-2611131053182459** require the hook to have a row of its own for the branch, with its own ruling and shape check, before any verb pushes to it.

The hook already admits two families of receipt-less pushes, reservations ([adr-2610031352212697](2610031352212697-claims-are-branches-on-the-shared-remote.md)) and status branches ([adr-2610150729333337](2610150729333337-a-status-branch-moves-without-a.md)), each by one row of its receipt-less table and a shape check that admits only commits that change no file. Store commits do change files, so neither check fits.

## Decision

The push hook's receipt-less table gains a third row: the branch `abcd-spotcheck` itself, checked by its own `spotcheck_shape` check. A row whose prefix does not end in a slash names one branch exactly, never a prefix of others. The check admits a push only when:

- it is a fast-forward of the branch, or its creation;
- every commit it moves (`<remote>..<local>` on an update, everything reachable from `<local>` on a creation) has a tree holding nothing but regular, non-executable files, each at `window-open.json` or at a path of `.json` names under `handbacks/`, `decisions/`, `cycles/`, `items/` or `judgements/`, made of letters, digits, `.`, `_` and `-`;
- every file is within the store reader's own bound, 16 KiB, or 256 KiB for a cycle under `cycles/`, and the push adds or changes no more than 1,000 files, the most the store's writer puts in one push;
- the walk read every commit in full, as the reservation check requires of a shallow clone.

A deletion of the branch is refused outright: the store is kept for ever. Any other push to the branch is gated as any branch is. The forge ruleset that blocks deletion and non-fast-forward pushes of `refs/heads/abcd-spotcheck` is the person's act, applied from its mirror, `.abcd/work/rulesets/spotcheck-protection.json`, as every ruleset is, before the first draw on any machine.

Because the row admits content, abcd's reader never trusts what it reads there ([dcn-2611260013430233](../../../work/decisions/dcn-2611260013430233-the-technical-facilitator-under.md)). Every cycle is verified wherever it is read: its id is its close's, it starts where its window began, its close is no later than the reader's clock allows, its round is the one its close fixes on the chain's pinned clock, its signature verifies under the pinned key, and its drawn lists are the ones that round gives over its populations. A cycle that fails is set aside, named, and its window drawn afresh, so a hand-made or future-dated cycle can neither end a window nor stop a later draw. A window's close, and so its round and its draw, is read from the default branch's history alone, the same for every drawer at any moment. Each machine keeps a high-water mark of the last copy it verified, in the person's home store, and refuses a branch that no longer holds it, deleted or rewritten, opening no first window over it. Records join a population by their arrival on the branch, and a judgement counts only when it reads as the judgement of the item and role its path names.

## Alternatives Considered

- **A receipt for every store push.** Rejected: no preflight can run over a commit made after a landing in no working tree, so the store could never reach `origin`.
- **The reservation shape check.** Rejected: it admits only commits that change no file, and the store's commits add files.
- **A hidden ref namespace (`refs/abcd/spotcheck`) on the remote.** Rejected for the reasons [adr-2610150729333337](2610150729333337-a-status-branch-moves-without-a.md) gives for status branches: the forge's rulesets cannot protect it, and a hosted session's proxy refuses it.
- **The chosen option: a row of its own with a path-and-mode shape check.** It admits only files the store writes, under the store's folders, on a branch no gate reads code from.

## Consequences

- **Easier:** a landing's draw and its records reach `origin` from any machine whose hooks run, so the queue is one queue for every machine.
- **Harder:** the shape check judges paths, modes, sizes and the count, never content: a push under the row can carry any JSON at a store path, within the bounds. abcd's reader is the guard on content: it reads every file strictly, verifies every cycle, and names one it cannot read or verify; nothing on the branch is ever built, run or merged.
- **The residual:** whoever holds the person's push rights can still write a cycle that verifies over a population it chose, hold a record back until a window's draw is known, or write judgements that halve the next sample. Verification removes the free choice of round and of drawn list and makes every cycle recomputable; it cannot tell an agent from the person when both push with one credential. Who may judge is the review lane's to settle.
- **The hook is a local-only check.** A push made where no hook runs is held only by the forge's ruleset, as for the other two rows.
- **A new obligation:** a change to the store's folders changes this row's check in the same change.
