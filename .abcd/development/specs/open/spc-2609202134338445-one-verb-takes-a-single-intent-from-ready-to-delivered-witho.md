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

## Steps

1. **The state file**
   Under `.abcd/.work.local/run/<run-id>/state.json`: run id, intent, lanes
   (each with branch, base sha, head sha, worktree path, step, receipt path,
   PR number), window clock and `next_eligible_at` (the pacing intent writes
   and reads those two), and the run record accumulated as steps complete.
   One reader and one atomic writer (`fsutil.WriteFileAtomic`); every verb
   below reads it first and writes it last.
   - packages: internal/core/implement/loop
   - landed: 7d3f3276
2. **The step interface**
   Decision 5, the default: `abcd build <itd-N>` creates the run after the
   checks (criteria 1 and 2); `abcd implement step` performs the next
   binary-owned step and, when a step needs an agent, returns the brief path,
   the agent role and the receipt path it expects (criterion 8); `abcd
   implement receipt <path>` verifies and advances (criterion 4); `abcd
   implement status` renders the state. Every invocation exits after one step
   (criterion 7).
   - packages: internal/core/implement/loop, internal/surface/cli
   - landed: 7d3f3276
3. **The process driver**
   Decision 5, opt-in by configuration: the same loop calling itself through
   `step` and `receipt`, starting each agent through the CLI adapter
   (`itd-2609201916056194`) and waiting for it; named in the run record
   (criterion 9).
4. **The checks**
   Criteria 1 and 2: the readiness gate, the claim sections, the
   open-question count, the hold (prose until `iss-2609200830076665` ships),
   and the peer reader (`itd-2609091416295622`).
   - packages: internal/core/implement/loop, internal/core/intent, internal/core/peers
   - landed: 7d3f3276
5. **The brief renderer**
   Intent, spec, the conventions section of `AGENTS.md` and the decision lines
   the intent cites, into one Markdown file under the run directory
   (criterion 3).
   - packages: internal/core/implement/loop
   - landed: 72ae4a2b
6. **The lane**
   Worktree at `~/.abcd.noindex/worktrees/<root-sha>/<run>-<lane>` in abcd's form (the
   store's verb once `itd-2609091014076309` ships; a plain `git worktree add`
   until then), branch off the default branch.
   - packages: internal/core/implement/loop, internal/core/peers
   - landed: 72ae4a2b
7. **The receipt**
   Criterion 4: a file the agent writes naming its commits, the definition of
   done's output and its report; the verifier checks the commits exist on the
   branch and the report exists, and refuses otherwise.
   - packages: internal/core/implement/loop, internal/surface/cli
   - landed: 72ae4a2b
8. **Validators**
   Criterion 5: the ruthless and security reviewer briefs on the lane's diff,
   each a fresh agent; findings are applied by a fresh implementer or rejected
   in the report; the fidelity request is completed with the delivered range
   from base to head before the auditor runs.
   - packages: internal/core/implement/loop, internal/core/intent, internal/surface/cli
   - landed: 1ed950b3a
9. **The landing**
   Criterion 6: `spec close` and `capture resolve` invoked by the loop with
   the lane's commit, the pull request through the forge client the
   repository already uses (`gh`), the merge rule from the repository's
   ruleset, no push after arming, cleanup after the ancestor check.
   - packages: internal/core/implement/loop, internal/surface/cli
   - landed: 8ede4810f
10. **The run record and transcripts**
   Criterion 10: the state file's record rendered at the end, and `history
   capture` per transcript path (one call once `iss-2609202046145653` ships).
   - packages: internal/core/implement/loop, internal/surface/cli
   - landed: 8ede4810f
11. **`--auto-plan`**
   Decision 6: the planning path run by the loop on a draft whose decisions
   are all recorded, with the two adversarial reviews as validator steps and
   the ADR's substitute checked before `plan`; refused on anything less.

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
11 (the issue key) by the fold-in below; 12 (only the loop writes a verdict)
by the invariant below; 13 by the refusal shape every piece shares. The
criteria are numbered in the order the intent lists them, which is the
order the fidelity auditor numbers them.

## Invariant folded in on 2026-09-21 (itd-58)

Only the loop writes a verdict. The validator stage records each validator's
verdict into the state file from the validator's own return, before the
advance is decided; the lane's receipt and report carry no verdict field the
loop reads, and a report that carries one is refused at the advance naming
it. The end-to-end test enters through the loop's step interface, has a
fake validator return SHIP, and asserts the advance; a second run has the
lane write a SHIP into its report and asserts the refusal.

## The issue key, folded in on 2026-09-21 (decision 10, for itd-82)

The state file's lane carries `key: itd-N | iss-N`. For an issue: the
pre-start checks are itd-82's eligibility rule (fields only: `remedy:`
present, fixable category, severity below major, nothing unshipped in
`blocked_by`); the brief renderer takes the record and its remedy in place
of the intent and spec; the validators run unchanged; the landing runs
`capture resolve <iss-N> --commit <sha>` in the lane's change instead of
`spec close`; the fidelity audit does not run (an issue has no criteria);
itd-50's fix round runs from the reviewers' findings instead; the lane's definition of done is the repository's (a detector watched to fail before the fix and pass after), rendered into the issue brief. The lane report
schema gains `handback: {kind, reason}`; the loop reads it before the
validators and ends the lane with that outcome when present. `abcd build
<iss-N>` is the person's form; `drain` calls the same loop with the key.

## Progress

The run builds the steps above; this section says which have landed and which
remain, and the `landed:` lines under `## Steps` say the same to the loop. The
spec stays open until the last lane closes it.

- **Landed (lane 1): pieces 1, 2 and 4.** The state file and its one reader and
  one atomic writer (`internal/core/implement/loop`), under
  `.abcd/.work.local/run/<run-id>/state.json` with the tier's advisory lock; the
  step interface — `abcd build <itd-N>` creates the run after the checks,
  `implement step` performs one step and exits, `implement receipt <path>`
  completes an agent step on a verified receipt, `implement status` renders —
  with the lane sequence (worktree, brief, implement, validate, land) named and
  every step body left to the piece that delivers it; and the checks: the
  readiness gate, the open questions, the claim sections, the hold (read from
  `held:`, since iss-2609200830076665 shipped), the spec's steps through the
  spec store's reader, and the peers (the peer listing and the shared run's
  live claims). The end-to-end test plays the host with fake step bodies.
- **Landed (lane 2): pieces 5, 6 and 7.** The lane's worktree in the
  machine-scoped store, `~/.abcd.noindex/worktrees/<root-sha>/<run-id>-<lane-id>`, on a
  branch `build/<run-id>-<lane-id>` cut from the default branch, its path derived
  from the run and lane ids and refused on any component that could leave the
  store; the brief, rendered from that base (the intent, the spec, the
  conventions of `AGENTS.md`, the cited ADRs and the decision-log entries naming
  the intent or spec) into the lane's directory of the run; and the implement
  step, which awaits a fresh implementer's receipt and verifies it strictly:
  every commit on the lane's branch past its base, a passing definition of
  done's output and the report, each inside the lane's directory, or a refusal
  naming every gap.
- **Landed in part (lane runner2): piece 3**, the process driver. `implement
  step` drives through `loop.Drive`: when a stage hands the lane to a role
  that `roles.<role>.runner` routes to a command-line runner
  (itd-2609201916056194), the loop starts the agent itself through the runner
  and hands its receipt back through `Receipt`, and the run record names the
  runner that ran it (criterion 9); a role left on the host is handed to the
  host exactly as before. Not built: the loop driving itself with no host
  session, which is decision 6's reversal of the host-delegated boundary and
  waits on the ADR that decision owes.
- **Landed (lane fidelityOnce): piece 8**, the validators with the itd-58
  verdict invariant. The validate stage hands the lane's head to a fresh
  ruthless reviewer and a fresh security reviewer, one at a time, and on the
  lane whose landing closes the spec (and ships the intent) to the
  intent-auditor, once, over the whole delivery: from the base of the run's
  first lane to the closing lane's head, with each lane's range and the steps
  landed before the run (ruling AI, 2026-09-29, which settles the question this
  section carried: audit once, on the lane that closes the spec, over the whole
  delivery). The fidelity request is composed before the close as the close's
  own emit composes it (`intent.ComposeDeliveryAudit`), keyed on the receipt the
  close parks, so its verdict is the one the close consumes. The loop parses
  each verdict from the validator's own return and records it in the state
  (schema version 5); a lane report stating a verdict is refused at the advance,
  naming it. A round that does not pass goes to a fresh implementer, who applies
  each finding or rejects it in writing in its report, and the next round
  judges the new head afresh; the rounds are counted, and their bound is
  itd-50's.
- **Landed (lane loopLanding): pieces 9 and 10**, the landing and the run
  record. The land stage takes a validated lane to the default branch one step
  per invocation, each recorded in the lane's `landing` (state schema 7) so a
  killed step resumes where it stopped: it checks the worktree is clean at the
  judged head; on the closing lane it runs `spec close` in the lane's worktree
  and ingests the verdict the closing lane's audit returned into the receipt the
  close parks, and for each capture the lane's receipts declared fixed
  (`resolves`, with the fixing commit) it runs `capture resolve`, committing
  both on the lane with `Delivers:` and `Resolves:` trailers; it pushes only
  once a preflight receipt names the head (the pre-push hook runs; nothing is
  forced or skipped); it opens the pull request through `gh` with a body from
  the records through the outbound scrub, re-reading and stripping it after
  creation; it arms auto-merge with the merge-queue method the ruleset mirror
  at the lane's base names, or leaves the pull request open; it pushes nothing
  after arming; and it removes the lane's worktree and branch only once the
  pushed head is an ancestor of the default branch on `origin`. `implement
  record` renders the run record in text and JSON (lanes, verified receipts
  with each runner's reported model, every verdict, fixes, landings,
  transcripts) and, on a complete run, captures each transcript by path through
  the history capture's own code, one capture per path. Marking a step's
  `landed:` line in the spec on a non-closing lane is not made by the landing.
- **Landed (lane drainLoop): the issue key (decision 10).** The key check
  admits an issue id by shape; its pre-start checks are itd-82's eligibility
  rule, read as `abcd drain` reads it, and the peers; its run has one lane, whose
  brief is the record and its remedy with the reproduce-then-fix definition of
  done; its validators take no fidelity audit; its receipt must declare the
  issue fixed in `resolves`, and the landing resolves it with that commit. The
  receipt carries `handback: {kind, reason, home}`, which the loop reads at the
  receipt, before the validators, discarding the lane's worktree and branch
  and ending the lane handed back.
- **Remaining: 11** (`--auto-plan` with its ADR), and piece 3's no-host
  driving, behind the same ADR. `--auto-plan` is not a flag yet.
