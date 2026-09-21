---
id: itd-82
slug: drain-ledger-triage
kind: standalone
suggested_kind: standalone
bundle: null
spec_id: null
reclassification_history: []
builds_on: [itd-4, itd-46, itd-2609201916151817, itd-2609201925079472]
related_intents: [itd-2609211116005482, itd-84]
related_adrs: [adr-25, adr-27]
severity: major
impact: additive
---

# `abcd drain` fixes the issues that need no decision, and hands the rest back by kind

## Press Release

> **`abcd drain` works the open issue ledger unattended: it fixes what needs no decision, and routes what does to the place a person decides it.** It reads every open issue that nothing blocks, and lets through only the ones the rules say a machine may touch alone: a stated remedy, one area of the code, no change a user would see, no trust or safety boundary, not security, not `major` or `critical`. Each of those goes to the machinery `abcd build <itd-N>` uses, one lane, one pull request, the record resolved in the same change, the run never approving its own pull request. An issue that needs a decision is handed back by kind: one that would change what a user sees becomes an intent draft seeded from the issue, for a person to plan; one that turns on a rule about trust or safety is flagged as needing a decision record, with the question stated; anything else is flagged with the home the decision belongs in. A lane that discovers a decision inside a "mechanical" fix stops, discards its work, and hands the issue back the same way rather than guessing. Blocked issues are skipped with the blocker named. It runs until nothing eligible is left, paced by the pace rule, because "drain" means all; `--max <n>` caps a run. In the morning there are three lists: pull requests to review, drafts to plan, decisions to make.
>
> "I used to point the loop at the ledger and then hover, because the moment it hit something design-shaped it would either stall or, worse, decide it," said a product thinker who ran the first ledger drains by hand. "Now the mechanical ones arrive as pull requests, the ones that are mine arrive as drafts or as questions, and the one time a lane found a decision halfway through, it stopped and told me instead of finishing."

## Why This Matters

The pilot run of 2026-09-20 fixed a cluster of five captured issues through lanes, reviewers and a merge queue, and the part a person did by hand at every step was the sorting: which capture a lane may take alone, which is a design decision in disguise, which is blocked. That sorting is the load-bearing, human-shaped piece, and it is the one a person should not run every night. Its failure is one-directional: a machine that decides a thing needs no decision, and then makes one. The evidence gathered on 2026-09-21 is plain about it: models detect their own ambiguity badly (the best reaches 89 per cent with a 3 per cent false-positive rate only under strong prompting; weaker ones flag 93 per cent of clear tasks as ambiguous), two teams that let a triage pass auto-dispatch a coding agent reverted it after unwanted pull requests, and the largest dataset of agent pull requests (878 over ten months) shows clean-ups and tests merging at 85 and 76 per cent while features and performance work merge at 65 and 55. Industrial fix pipelines land 15 to 25 per cent of what they attempt; most lanes end without a merge, and that has to be a normal ending, not a failure.

The record already has the sorting rule. The four-piece routing every filing goes through (capability to an intent, trust rule to a decision record, stance to a principle, plumbing to the brief) is what a decision-shaped issue needs applied to it, and a capture is not an intent: an intent is a user moment, and an issue whose decision is a rule about trust is not one. So the hand-back routes by kind rather than promoting everything into drafts a person then has to sort again.

## Mechanism

We expect a rule-first eligibility filter, a lane that hands back on discovering a decision, and a hand-back routed by kind to drain the mechanical part of a ledger without a person and without a decision being made by a machine, because the rules that predict a safe unattended fix are fields the capture already carries (a stated remedy, category, severity, the blocked-by edges) and the hand-back is a write the verbs already make (`capture promote` for a user moment, a flag for the rest); shown wrong if a merged drain pull request is later found to have decided something a person should have, or if the hand-back queue is mostly issues the filter should have let through.

## Scope Conditions

- Holds for a repository abcd manages with a merge queue or branch protection on the default branch, so a pull request the run opens has a path to the default branch that the run does not control.
- Holds for issues captured through `abcd capture` with a category, a severity and a stated remedy; a record without a remedy is not eligible and is listed as such.
- Holds at the ledger sizes seen so far (hundreds of open records), where every eligible issue is classified on every run; a ledger of thousands would want the classification recorded once and reused, which is an open question below.
- The classifier is rules first; where a host-delegated judgement is used for the residual call, its errors run in one direction by construction (an uncertain issue is handed back, never fixed), and that judgement is recorded with the disposition.

## What's In Scope

- **Eligibility, rules first**: open, nothing unshipped in `blocked_by`, a stated remedy, category not `security` and not one of the decision categories (`architectural-insight`, `future-work-seed`), severity `minor` or `nitpick`, the remedy naming one package or one surface, no user-visible change and no trust-boundary change on the run's reading of the remedy; `major` and `critical` go to the hand-back by default. A host-delegated judgement (adr-25) is allowed only for the residual "is this genuinely self-contained" call and only to hand back, never to let through.
- **The lane**: the implement machinery (`itd-2609201916151817`), one lane per issue, the record resolved with its commit in the same change, one pull request per issue with the repository's merge rule applied, the run's identity never an approver, nothing pushed to a pull request after its merge is armed.
- **Reproduce, then fix**: the lane arms a detector that fails before the fix and passes after; the adversarial reviewer stays the oracle and the detector is evidence for it, not the gate.
- **The hand-back, routed by kind**: a user-moment issue is promoted to an intent draft (`capture promote`, seeded from the issue, the back-link written); a trust or safety rule is flagged as needing a decision record with the question stated, and nothing is minted; anything else is flagged with the proposed home named. The issue is never edited; the lane's partial work is discarded.
- **The mid-lane stop**: a lane that meets a decision (a reviewer raising a design question, a remedy that turns out to change what a user sees, a second package the remedy did not name) stops, discards, and hands back with the reason, as a first-class outcome.
- **Order**: category and footprint first, smallest first within a category, oldest among equals; the run's summary states the rule.
- **Ceilings**: the pace rule bounds the run; `--max <n>` caps issues attempted; the pull requests opened, the promotions made and the spend are counted in the summary, and a cap hit is named.
- **The summary**: every issue's disposition (pull request number, draft id, flag with home, skipped with blocker, stopped with reason), machine-readable and re-runnable; "no pull request, reason logged" is a normal terminal state.

## What's Out of Scope

- The lane, the validators and the landing: `itd-2609201916151817`; the pace: `itd-2609201925079472`.
- Picking among intents: `abcd build next` (`itd-2609211116005482`), the sibling run.
- Merging pull requests, planning promoted drafts, writing decision records: human gates by design.
- Any change to the ledger schema beyond consuming `list`, `resolve` and `promote`.
- Any estimate the agent makes of its own confidence as a reason to let an issue through.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that revived this draft:

1. **This record is the drain run**, updated for the implement machinery; no new record, and the verb keeps its name, `abcd drain`, which is what a person types. It hands over to `abcd implement`, the machinery an agent calls.
2. **The hand-back routes by kind**: user moments become intent drafts; trust rules are flagged for a decision record; the rest are flagged with their home. Not everything promoted, not everything merely listed.
3. **All by default, capped by flag**: "drain" reads all and "next" reads one, so the two runs' defaults differ on purpose and their flags are the same.
4. **The rule for "needs no decision" is a recorded rule.** It decides what a machine may touch alone, so it is written as a decision record and a brief invariant before this path ships, the way `--auto-plan` owes its record; until then the rules above are the draft's statement of it.

## Prior Art

- **`itd-2609201916151817`** (the implement machinery): the lane, the validators and the landing this run dispatches onto; it supersedes `itd-29`, which this draft first named.
- **`itd-46`** / **spc-30**: owns `capture promote <iss-N>`, the issue-to-intent elevator the user-moment hand-back uses; shipped, so the earlier cut-A stub is gone.
- **`itd-4`**: the ledger substrate (`list`, `resolve`) the run reads and mutates.
- **`itd-84`**: the four-piece routing the hand-back applies to a decision-shaped issue.
- **adr-25** (host-delegated judgement): the residual classifier call rides the host and only hands back.
- **adr-27** (receipt-gated autonomous runs): the lane's review discipline.

## SOTA

Surveyed 2026-09-21 by an independent research pass, primary sources where they exist:

- **Triage that dispatches is the failure mode.** GitHub's own reference triage workflows label and comment only, never assign or close; two independent repositories that let a triage label auto-dispatch a coding agent reverted it after unwanted pull requests. Adopted: the run's eligibility rules are the only thing that lets an issue through, and the hand-back's only write is a promotion for a user moment.
- **The classifier is rules first.** Copilot's published guidance keeps ambiguous, open-ended, security, PII, authentication and production-critical work for people and assigns bugs, tests, docs and debt to the agent; repositories converged independently on `agent-ready` / `needs-design` labels. Ambiguity self-detection by models is unreliable (Ambig-SWE, ICLR 2026), so the residual judgement may only hand back.
- **Order by category and footprint, not severity.** The dotnet/runtime dataset (878 agent pull requests, ten months): clean-up 84.7, testing 75.6, bug fix 69.4, feature 64.5, performance 54.5 per cent merged; one platform's own fleet report had refactoring and performance queues at zero closure. Adopted for this run because its scope is already bounded to no-decision issues.
- **Detector-first is contested.** Impact-aware test selection cut regressions from 6.08 to 1.82 per cent while a bare "write a failing test first" instruction worsened them to 9.94 (TDAD, 2026); 29.6 per cent of plausible patches on SWE-bench behave differently from the reference. Adopted as reproduce-then-fix with the reviewer as the oracle.
- **The safety envelope is GitHub's two designs combined**: one branch, one pull request per task, the requester cannot approve, a hard session cap (Copilot's cloud agent); read-only agent job with writes through a scoped step, `max: 1` pull request, daily credit and per-user rate limits, `stop-after` (gh-aw). Translated: per-run caps, the run's identity is not an approver, the merge queue is the path to `main`.
- **Expect most lanes not to merge.** Google's sanitizer-fix pipeline landed 15 per cent of candidates; Meta's test-repair agent 25.5 per cent over three months with a judge before human review. "No pull request, reason logged" is a normal ending.
- **Not adopted**: WSJF and cost-of-delay (no outcome evidence, needs human estimates); the agent's self-reported confidence as a gate (vendor feature, no published threshold); fully automatic readiness promotion by a triage model.

## Open Questions

- **Whether the classification is recorded on the issue.** Re-derived every run (simple, and the rules may change) or written once as a disposition field the next run reuses and a person can override (durable, and `capture` could set it at file time). Leaning durable at ledger sizes past a few hundred.
- **The decision-record flag's home.** A line in the summary only, or a marker on the issue the person finds without the summary; the marker is a write to a record the run otherwise never edits.
- **Whether `major` may ever be let through.** The default sends it to the hand-back; a `--allow-major` flag is the obvious opt-in, and whether a person wants it is not yet known.

## Acceptance Criteria

- **Given** an open ledger holding a `minor` issue with a stated one-package remedy, a `major` issue, an issue whose remedy would change a command's output, an issue in category `security`, and an issue blocked by an open record, **when** `abcd drain` runs, **then** only the first goes to a lane; the `major`, the user-visible and the `security` issues are handed back with the kind named; the blocked one is skipped naming its blocker; and every open issue receives exactly one recorded disposition.
- **Given** an eligible issue, **when** its lane runs, **then** a detector is armed and watched to fail before the fix and pass after, the record is resolved with the fix commit in the same change, one pull request is opened with the repository's merge rule applied, and nothing is pushed to it afterwards.
- **Given** a decision-shaped issue that describes a user moment, **when** it is handed back, **then** an intent draft exists seeded from the issue with the back-link written, the issue is unchanged, and no design or adoption verdict is recorded.
- **Given** a decision-shaped issue that turns on a trust or safety rule, **when** it is handed back, **then** it is flagged as needing a decision record with the question stated, and nothing is minted.
- **Given** a lane whose reviewer raises a design question, or whose fix reaches a second package or a user-visible change the remedy did not name, **when** the lane sees it, **then** the lane stops, its work is discarded, the issue is handed back with the reason, and the summary records the stop as its own outcome.
- **Given** the default run, **when** eligible issues remain at a window's end, **then** the run pauses under the pace rule and resumes; **given** `--max <n>`, **then** it stops at the cap and names it.
- **Given** a completed or paused run, **when** its summary is read, **then** every issue's disposition is there (pull request number, draft id, flag with home, skip with blocker, stop with reason) with the ordering rule and every cap, in text and in `--json`, and a run whose lanes all ended without a merge exits 0 with that stated.
- **Given** a crafted issue id, **when** the run resolves it, **then** it is validated by shape and refused otherwise, and no file outside the ledger directories is read, written or moved.

## Typed Links

- **builds_on `itd-2609201916151817`** (the implement machinery) and **`itd-2609201925079472`** (the pace): the lane and its clock.
- **builds_on `itd-46`** (`capture promote`) and **`itd-4`** (the ledger): the writes the hand-back and the resolve make.
- **refines `itd-84`** (decompose before filing): the routing applied to a decision-shaped issue at hand-back.
- **refines `itd-2609211116005482`** (`abcd build next`): the sibling run over intents; the two share the flags and the hand-over and differ in their default count by the meaning of their names.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
