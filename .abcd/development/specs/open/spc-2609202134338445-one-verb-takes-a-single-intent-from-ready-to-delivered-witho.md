---
id: spc-2609202134338445
slug: one-verb-takes-a-single-intent-from-ready-to-delivered-witho
intent: itd-2609201916151817
origin: researcher-authored
production_mode: hand-written
---
# one-verb-takes-a-single-intent-from-ready-to-delivered-witho

## Summary

The design record for itd-2609201916151817, from the seven decisions on the
intent (2026-09-20). Decision 5 fixes the shape: A loop over a state file,
driven by a host session step by step by default, or by a process through
the CLI adapter when opted in. The ADR decision 6 owes (the `--auto-plan`
substitute for the sign-off act, and the opt-in reversal of the host-delegated
boundary) is minted before either path ships and is a delivery of this spec.

## Scope

1. **The state file** under `.abcd/.work.local/run/<run-id>/state.json`:
   run id, intent, lanes (each with branch, base sha, head sha, worktree
   path, step, receipt path, PR number), window clock and
   `next_eligible_at` (the pacing intent writes and reads those two), and
   the run record accumulated as steps complete. One reader and one atomic
   writer (`fsutil.WriteFileAtomic`); every verb below reads it first and
   writes it last.
2. **The step interface** (decision 5, the default): `abcd build <itd-N>`
   creates the run after the checks (criteria 1 and 2); `abcd implement step`
   performs the next binary-owned step and, when a step needs an agent,
   returns the brief path, the agent role and the receipt path it expects
   (criterion 8); `abcd implement receipt <path>` verifies and advances
   (criterion 4); `abcd implement status` renders the state. Every
   invocation exits after one step (criterion 7).
3. **The process driver** (decision 5, opt-in by configuration): the same
   loop calling itself through `step` and `receipt`, starting each agent
   through the CLI adapter (`itd-2609201916056194`) and waiting for it;
   named in the run record (criterion 9).
4. **The checks** (criteria 1, 2): the readiness gate, the claim sections,
   the open-question count, the hold (prose until `iss-2609200830076665`
   ships), and the peer reader (`itd-2609091416295622`).
5. **The brief renderer**: intent, spec, the conventions section of
   `AGENTS.md` and the decision lines the intent cites, into one Markdown
   file under the run directory (criterion 3).
6. **The lane**: worktree at `~/.abcd/worktrees/<root-sha>/<run>-<lane>`
   in abcd's form (the store's verb once `itd-2609091014076309` ships; a
   plain `git worktree add` until then), branch off the default branch.
7. **The receipt** (criterion 4): a file the agent writes naming its commits,
   the definition of done's output and its report; the verifier checks the
   commits exist on the branch and the report exists, and refuses otherwise.
8. **Validators** (criterion 5): the ruthless and security reviewer briefs
   on the lane's diff, each a fresh agent; findings are applied by a fresh
   implementer or rejected in the report; the fidelity request is completed
   with the delivered range from base to head before the auditor runs.
9. **The landing** (criterion 6): `spec close` and `capture resolve` invoked
   by the loop with the lane's commit, the pull request through the forge
   client the repository already uses (`gh`), the merge rule from the
   repository's ruleset, no push after arming, cleanup after the ancestor
   check.
10. **The run record and transcripts** (criterion 10): the state file's
    record rendered at the end, and `history capture` per transcript path
    (one call once `iss-2609202046145653` ships).
11. **`--auto-plan`** (decision 6): the planning path run by the loop on a
    draft whose decisions are all recorded, with the two adversarial reviews
    as validator steps and the ADR's substitute checked before `plan`;
    refused on anything less.

## Out of scope

The pace (its spec); the runner's implementation (the adapter's); the
worktree store's verbs; the peer reader; the claim and the register.

## Approach

Test-first, in the numbered order, three lanes: 1 to 4 (state, steps,
checks), 5 to 8 (brief, lane, receipt, validators), 9 to 11 (landing,
record, auto-plan with its ADR). The step interface is exercised end to
end by a test that plays the host: it reads the brief, writes a receipt,
and calls `receipt`, so the loop is proven without any model.

## How the criteria are satisfied

1 and 2 by piece 4; 3 by pieces 1, 5, 6; 4 by piece 7; 5 by piece 8; 6 by
piece 9; 7 by pieces 1 and 2; 8 by piece 2; 9 by piece 3; 10 by piece 10;
11 by the refusal shape every piece shares.

## Invariant folded in on 2026-09-21 (itd-58)

Only the loop writes a verdict. The validator stage records each validator's
verdict into the state file from the validator's own return, before the
advance is decided; the lane's receipt and report carry no verdict field the
loop reads, and a report that carries one is refused at the advance naming
it. The end-to-end test enters through the loop's step interface, has a
fake validator return SHIP, and asserts the advance; a second run has the
lane write a SHIP into its report and asserts the refusal.
