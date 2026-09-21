---
id: itd-82
slug: drain-ledger-triage
kind: standalone
suggested_kind: standalone
bundle: null
spec_id: spc-2609212015054359
reclassification_history: []
builds_on: [itd-4, itd-119, itd-2609201916151817, itd-2609201925079472, itd-50]
related_intents: [itd-2609211116005482, itd-84]
related_adrs: [adr-25, adr-27]
severity: major
impact: additive
---

# `abcd drain` fixes the issues that need no decision, and hands the rest back by kind

## Press Release

> **`abcd drain` works the open issue ledger unattended: it fixes what needs no decision, and routes what does to the place a person decides it.** It reads every open issue that nothing blocks, and lets through only the ones the rules say a machine may touch alone: a `remedy:` field on the record (captures carry one from now on; a record without it is ineligible until someone adds it), a category the run may fix, not security, not `major` or `critical`, nothing unshipped blocking it; a host judgement over the remedy may hand an issue back for a user-visible or trust-boundary change, and can only ever make the set smaller. Each of those goes to the same loop `abcd build` uses, keyed by the issue (the machinery grows an issue key for it: the brief is the record and its remedy, the landing resolves the record), one lane, one pull request, the run never approving its own pull request. An issue that needs a decision is handed back by kind: one that would change what a user sees becomes an intent draft seeded from the issue, for a person to plan; one that turns on a rule about trust or safety is flagged as needing a decision record, with the question stated; anything else is flagged with the home the decision belongs in. A lane that discovers a decision inside a "mechanical" fix stops, discards its work, and hands the issue back the same way rather than guessing. Blocked issues are skipped with the blocker named. It runs until nothing eligible is left, paced by the pace rule, because "drain" means all; `--max <n>` caps a run, and at a window's end it writes when it may continue and exits, the next invocation carrying on. In the morning there are three lists: pull requests to review, drafts to plan, decisions to make.
>
> "I used to point the loop at the ledger and then hover, because the moment it hit something design-shaped it would either stall or, worse, decide it," said a product thinker who ran the first ledger drains by hand. "Now the mechanical ones arrive as pull requests, the ones that are mine arrive as drafts or as questions, and the one time a lane found a decision halfway through, it stopped and told me instead of finishing."

## Why This Matters

The pilot run of 2026-09-20 fixed a cluster of five captured issues through lanes, reviewers and a merge queue, and the part a person did by hand at every step was the sorting: which capture a lane may take alone, which is a design decision in disguise, which is blocked. That sorting is the load-bearing, human-shaped piece, and it is the one a person should not run every night. Its failure is one-directional: a machine that decides a thing needs no decision, and then makes one. The evidence gathered on 2026-09-21 is plain about it: models detect their own ambiguity badly (the best reaches 89 per cent with a 3 per cent false-positive rate only under strong prompting; weaker ones flag 93 per cent of clear tasks as ambiguous), two teams that let a triage pass auto-dispatch a coding agent reverted it after unwanted pull requests, and the largest dataset of agent pull requests (878 over ten months) shows clean-ups and tests merging at 85 and 76 per cent while features and performance work merge at 65 and 55. Industrial fix pipelines land 15 to 25 per cent of what they attempt; most lanes end without a merge, and that has to be a normal ending, not a failure.

The record already has the sorting rule. The four-piece routing every filing goes through (capability to an intent, trust rule to a decision record, stance to a principle, plumbing to the brief) is what a decision-shaped issue needs applied to it, and a capture is not an intent: an intent is a user moment, and an issue whose decision is a rule about trust is not one. So the hand-back routes by kind rather than promoting everything into drafts a person then has to sort again.

## Mechanism

We expect a rule-first eligibility filter, a lane that hands back on discovering a decision, and a hand-back routed by kind to drain the mechanical part of a ledger without a person and without a decision being made by a machine, because the rules that predict a safe unattended fix are fields on the capture (a `remedy:` field, category, severity, the blocked-by edges) and the hand-back is a write the verbs already make (`capture promote` for a user moment, a flag for the rest); shown wrong if a hand-back is then let through by a person unchanged more often than not, counted per run.

## Scope Conditions

- Holds for a repository abcd manages with a merge queue or branch protection on the default branch, so a pull request the run opens has a path to the default branch that the run does not control. <!-- cond: cond-2609212015056112 -->
- Holds for issues captured through `abcd capture` with a category, a severity and a `remedy:` field; a record without the field is not eligible and is listed as such, so the first drains over today's ledger are small. <!-- cond: cond-2609212015051754 -->
- Holds once the decision record for the eligibility rule exists (decision 4): an autonomous lane refuses to start without it, and the record is a prerequisite this intent names, not a step it performs. <!-- cond: cond-2609212015056014 -->
- The rules are the fields (`remedy:`, category, severity, `blocked_by`); the host-delegated judgement reads the remedy for a user-visible or trust-boundary change and can only hand back, never let through, and that judgement is recorded with the disposition. <!-- cond: cond-2609212015052582 -->

## What's In Scope

- **Eligibility, by field**: open; nothing unshipped in `blocked_by`; a `remedy:` field present; category one of `bug`, `documentation`, `drift`, `inconsistency`, `tech-debt`, `ux` (the fixable set); severity `minor` or `nitpick`. Excluded by field: `security` (always a person's), `major` and `critical` (the hand-back by default), and the decision categories `process`, `observation`, `architectural-insight`, `future-work-seed`, `lapse` (the run file's merits set). A host-delegated judgement (adr-25) reads the remedy for a user-visible or trust-boundary change and may only hand back.
- **The `remedy:` field**: `capture` gains it (the optional `suggested_fix` made first-class under the name the eligibility reads); a record without it is ineligible and listed as such.
- **The lane**: the implement loop keyed by the issue (decision 10 on `itd-2609201916151817`: the brief is the record and its remedy, the checks are this record's eligibility, the landing is `capture resolve --commit`), one lane per issue, one pull request per issue with the repository's merge rule applied, the run's identity never an approver, nothing pushed to a pull request after its merge is armed.
- **Reproduce, then fix**: the issue-keyed lane's definition of done, owned by the loop (decision 10 on `itd-2609201916151817`): a detector fails before the fix and passes after, and the adversarial reviewer stays the oracle with the detector as evidence for it.
- **The hand-back, routed by kind**: a user-moment issue is promoted to an intent draft (`capture promote`, `itd-119`'s verb, on `main` with its spec open; seeded from the issue; the issue gains `promoted_to` and nothing else); a trust or safety rule is flagged as needing a decision record with the question stated, and nothing is minted; an issue above the run's severity, or in a category the run does not fix, is flagged as a person's with that reason; anything else is flagged with the proposed home named. The lane's partial work is discarded.
- **The mid-lane stop**: a lane that meets a decision stops, discards, and hands back with the reason, as a first-class outcome; the `handback:` field on the lane report is the loop's (decision 10 on the parent), and this record names the values drain reads from it (a reviewer's design finding, a second package, a user-visible change).
- **Order**: by category in the declared order `tech-debt`, `documentation`, `inconsistency`, `drift`, `bug`, `ux` (clean-ups and text first, on the evidence that they merge most often; behaviour changes last), then `nitpick` before `minor`, then oldest first; the run's summary states the rule.
- **Ceilings**: the pace rule bounds the run (at a window's end it writes `next_eligible_at` and exits; the next invocation continues); `--max <n>` caps issues attempted and `--until-empty` is the default's own name; the pull requests opened, the promotions made and the spend are counted in the summary, and a cap hit is named.
- **The command page**: `commands/capture.md` (or the drain page) says what `drain` does, its flags, the order, the hand-back kinds and every refusal.
- **The summary**: every issue's disposition (pull request number, draft id, flag with home or reason, ineligible with the missing field named, skipped with blocker, stopped with reason), machine-readable and re-runnable; "no pull request, reason logged" is a normal terminal state.

## What's Out of Scope

- The lane, the validators and the landing: `itd-2609201916151817`; the pace: `itd-2609201925079472`.
- Picking among intents: `abcd build next` (`itd-2609211116005482`), the sibling run.
- Merging pull requests, planning promoted drafts, writing decision records: human gates by design.
- Any change to the ledger schema beyond the `remedy:` field; the run consumes `list`, `resolve` and `promote`.
- Any estimate the agent makes of its own confidence as a reason to let an issue through.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that revived this draft:

1. **This record is the drain run**, updated for the implement machinery; no new record, and the verb keeps its name, `abcd drain`, which is what a person types. It hands over to `abcd implement`, the machinery an agent calls.
2. **The hand-back routes by kind**: user moments become intent drafts; trust rules are flagged for a decision record; the rest are flagged with their home. Not everything promoted, not everything merely listed.
3. **All by default, capped by flag**: "drain" reads all and "next" reads one, so the two runs' defaults differ on purpose and their flags are the same.
4. **The rule for "needs no decision" is a recorded rule.** It decides what a machine may touch alone, so it is written as a decision record and a brief invariant before this path ships, the way `--auto-plan` owes its record; the spec's first delivery mints it with `abcd decide` from this decision's text and the eligibility scope above, the reviewers judge it with the diff, and until it exists an autonomous drain refuses to start.
5. **The build machinery grows an issue key** (ruled 2026-09-21 on the design review's first finding): `abcd implement` takes an intent or an issue; drain hands it `iss-N`. Recorded as decision 10 on `itd-2609201916151817`.
6. **A `remedy:` field, required from now on** (ruled 2026-09-21): eligibility reads it; the backlog without it waits for someone to add it.
7. **Order and categories declared** over the ledger's own enum (scope above), so no implementer decides them alone.
8. **The classification is re-derived every run** and recorded with each disposition in the summary; nothing is written onto the issue for it (ruled 2026-09-21; the durable-field alternative is left for a ledger that outgrows re-derivation).

## Prior Art

- **`itd-2609201916151817`** (the implement machinery): the lane, the validators and the landing this run dispatches onto; it supersedes `itd-29`, which this draft first named.
- **`itd-119`** / **spc-24**: owns `capture promote <iss-N>`, the issue-to-intent elevator the user-moment hand-back uses; the verb is on `main` with its spec open.
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

- **The decision-record flag's home.** A line in the summary only, or a marker on the issue the person finds without the summary; the marker is a write to a record the run otherwise never edits.
- **Whether `major` may ever be let through.** The default sends it to the hand-back; a `--allow-major` flag is the obvious opt-in, and whether a person wants it is not yet known.

## Acceptance Criteria

- **Given** an open ledger holding a `minor` `bug` with a `remedy:` field, a `minor` `bug` without one, a `major` issue, an issue in category `security`, a `process` issue, and an issue blocked by an open record, **when** `abcd drain` runs, **then** only the first goes to a lane; the one without a remedy is listed as ineligible; the `major`, the `security` and the `process` issues are handed back with the kind named; the blocked one is skipped naming its blocker; and every open issue receives exactly one recorded disposition.
- **Given** an eligible issue whose remedy the host judgement reads as a user-visible or trust-boundary change, **when** the pass classifies it, **then** it is handed back with that reason, and the judgement never makes an ineligible issue eligible.
- **Given** an eligible issue, **when** its lane runs, **then** it is the implement loop's issue-keyed lane: the brief is the record and its remedy, the loop's definition of done holds (a detector fails before the fix and passes after), the record is resolved with the fix commit in the same change, one pull request is opened with the repository's merge rule applied, and nothing is pushed to it afterwards.
- **Given** a decision-shaped issue that describes a user moment, **when** it is handed back, **then** an intent draft exists seeded from the issue, the issue gains `promoted_to` and nothing else, and no design or adoption verdict is recorded.
- **Given** a decision-shaped issue that turns on a trust or safety rule, **when** it is handed back, **then** it is flagged as needing a decision record with the question stated, and nothing is minted.
- **Given** a lane whose report carries `handback:` (a reviewer's design finding, a second package or a user-visible change the remedy did not name), **when** the loop reads it, **then** the lane stops, its work is discarded, the issue is handed back with the reason, and the summary records the stop as its own outcome.
- **Given** the default run, **when** eligible issues remain at a window's end, **then** the run writes `next_eligible_at`, exits, and the next invocation after it continues; **given** `--max <n>`, **then** it stops at the cap and names it.
- **Given** the command page, **when** it is read, **then** it states what `drain` does, its flags, the category order, the hand-back kinds and every refusal.
- **Given** the decision record for the eligibility rule does not exist, **when** an unattended drain starts, **then** it refuses naming the record it needs.
- **Given** a completed or paused run, **when** its summary is read, **then** every issue's disposition is there (pull request number, draft id, flag with home, skip with blocker, stop with reason) with the ordering rule and every cap, in text and in `--json`, and a run whose lanes all ended without a merge exits 0 with that stated.
- **Given** a crafted issue id, **when** the run resolves it, **then** it is validated by shape and refused otherwise, and no file outside the ledger directories is read, written or moved.

## Typed Links

- **builds_on `itd-2609201916151817`** (the implement machinery) and **`itd-2609201925079472`** (the pace): the lane and its clock.
- **builds_on `itd-119`** (`capture promote`) and **`itd-4`** (the ledger): the writes the hand-back and the resolve make.
- **refines `itd-84`** (decompose before filing): the routing applied to a decision-shaped issue at hand-back.
- **related `itd-2609211116005482`** (`abcd build next`): the sibling run over intents; the two share `--max` and `--until-empty` and the hand-over to `abcd implement`, and differ in their default count by the meaning of their names.
- **builds_on `itd-50`**: the audit-driven fix round runs on an issue lane as on an intent lane.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the autonomous run's capture batches are a person's hand-written routing today; we expect the drain to be what replaces them after the run; shown wrong if the next run still starts from a hand-written table
