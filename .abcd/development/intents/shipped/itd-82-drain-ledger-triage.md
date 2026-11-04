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

- **Eligibility, by field**: open; nothing unshipped in `blocked_by`; a `remedy:` field present; category one of `bug`, `documentation`, `drift`, `inconsistency`, `tech-debt`, `ux` (the fixable set); severity `minor` or `nitpick`. Excluded by field: `security` (always a person's), `major` and `critical` (the hand-back unless the repository's drain record loosens it; no run flag does, decision 11), and the decision categories `process`, `observation`, `architectural-insight`, `future-work-seed`, `lapse` (the run file's merits set). A host-delegated judgement (adr-25) reads the remedy for a user-visible or trust-boundary change and may only hand back.
- **The `remedy:` field**: `capture` gains it (the older `suggested_fix` made first-class under the name the eligibility reads) and requires it of every new issue (ruling BX3 of 2026-09-29); an automatic filer with no fix writes `none (filed automatically)` (ruling H12 of the same day), which is ineligible until a person writes a real remedy with `capture remedy`; a record filed before the rule carries none, and is ineligible and listed as such.
- **The lane**: the implement loop keyed by the issue (decision 10 on `itd-2609201916151817`: the brief is the record and its remedy, the checks are this record's eligibility, the landing is `capture resolve --commit`), one lane per issue, one pull request per issue with the repository's merge rule applied, the run's identity never an approver, nothing pushed to a pull request after its merge is armed.
- **Reproduce, then fix**: the issue-keyed lane's definition of done, owned by the loop (decision 10 on `itd-2609201916151817`): a detector fails before the fix and passes after, and the adversarial reviewer stays the oracle with the detector as evidence for it.
- **The hand-back, routed by kind**: a user-moment issue is promoted to an intent draft (`capture promote`, `itd-119`'s verb, on `main` with its spec open; seeded from the issue; the issue gains the intent in `related_intents` and nothing else); a trust or safety rule is flagged as needing a decision record with the question stated, in the summary and as a marker on the issue (decision 10), and nothing is minted; an issue above the run's severity, or in a category the run does not fix, is flagged as a person's with that reason; anything else is flagged with the proposed home named. The lane's partial work is discarded.
- **The mid-lane stop**: a lane that meets a decision stops, discards, and hands back with the reason, as a first-class outcome; the `handback:` field on the lane report is the loop's (decision 10 on the parent), and this record names the values drain reads from it (a reviewer's design finding, a second package, a user-visible change).
- **Order**: by category in the declared order `tech-debt`, `documentation`, `inconsistency`, `drift`, `bug`, `ux` (clean-ups and text first, on the evidence that they merge most often; behaviour changes last), then `nitpick` before `minor`, then oldest first; the run's summary states the rule.
- **Ceilings**: the pace rule bounds the run (at a window's end it writes `next_eligible_at` and exits; the next invocation continues); `--max <n>` caps issues attempted and `--until-empty` is the default's own name; the pull requests opened, the promotions made and the spend are counted in the summary, and a cap hit is named.
- **The command page**: `commands/capture.md` (or the drain page) says what `drain` does, its flags, the order, the hand-back kinds and every refusal.
- **The summary**: every issue's disposition (pull request number, draft id, flag with home or reason, ineligible with the missing field named, skipped with blocker, stopped with reason), machine-readable and re-runnable; "no pull request, reason logged" is a normal terminal state.

## What's Out of Scope

- The lane, the validators and the landing: `itd-2609201916151817`; the pace: `itd-2609201925079472`.
- Picking among intents: `abcd build next` (`itd-2609211116005482`), the sibling run.
- Merging pull requests, planning promoted drafts, writing decision records: human gates by design.
- Any change to the ledger schema beyond the `remedy:` field and the decision-record marker (decision 10); the run consumes `list`, `resolve` and `promote`.
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
9. **The drain's remaining work is one plan** (ruled 2026-10-09): the product thinker grouped what the drain still needs to run end to end into this intent's plan: the judgement before a lane opens and the summary's remaining counts, plus three steps from issues (merged lanes finished, iss-2610090936460789; one issue or package held back for now, iss-2610090642385898; a pull request that leaves the merge queue put back, iss-2610090642376032). Pacing stays its own intent (itd-2609201925079472), built alongside; this intent builds on it. The drain joins the person's command list when this intent ships (ruling BX1 kept). Answer verbatim: "Group as proposed".
10. **The decision-record flag is also a marker on the issue** (ruled 2026-10-10, the first open question): a trust-rule hand-back writes a marker onto the open issue, naming the decision record it needs and stating the question, as well as the line in the summary, so a person finds it without the summary. It is a write to a record the run otherwise never edits, chosen knowing that; it is not the classification, which decision 8 keeps re-derived every run. Answer verbatim: "Marker on the issue too".
11. **`major` is let through by the repository's drain record, never by a flag** (ruled 2026-10-10, the second open question): invariant 19 already lets a repository's record loosen its rule to `major`, and a drain never runs looser than its record, so there is no `--allow-major`; abcd's own record keeps `nitpick` and `minor`. Answer verbatim: "Record only, no flag", given when the first answer, "Opt in with --allow-major", was put beside invariant 19.

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

_All resolved: the decision-record flag's home by decision 10, and whether `major` may ever be let through by decision 11 (both ruled 2026-10-10)._

## Acceptance Criteria

- **Given** an open ledger holding a `minor` `bug` with a `remedy:` field, a `minor` `bug` without one, a `major` issue, an issue in category `security`, a `process` issue, and an issue blocked by an open record, **when** `abcd drain` runs, **then** only the first goes to a lane; the one without a remedy is listed as ineligible; the `major`, the `security` and the `process` issues are handed back with the kind named; the blocked one is skipped naming its blocker; and every open issue receives exactly one recorded disposition.
- **Given** an eligible issue whose remedy the host judgement reads as a user-visible or trust-boundary change, **when** the pass classifies it, **then** it is handed back with that reason, and the judgement never makes an ineligible issue eligible.
- **Given** an eligible issue, **when** its lane runs, **then** it is the implement loop's issue-keyed lane: the brief is the record and its remedy, the loop's definition of done holds (a detector fails before the fix and passes after), the record is resolved with the fix commit in the same change, one pull request is opened with the repository's merge rule applied, and nothing is pushed to it afterwards.
- **Given** a decision-shaped issue that describes a user moment, **when** it is handed back, **then** an intent draft exists seeded from the issue, the issue gains the intent in `related_intents` and nothing else, and no design or adoption verdict is recorded.
- **Given** a decision-shaped issue that turns on a trust or safety rule, **when** it is handed back, **then** it is flagged as needing a decision record with the question stated, in the summary and as a marker on the open issue, and nothing is minted.
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
- **built on by `itd-2610301750162657`** (abcd's unattended work is spot-checked against specs and intents): Draws a few of the drain's fixes and hand-backs and has them checked against their specs, filing each finding against this intent's rule (adr-2609291342092738) for a person to amend.
- **built on by `itd-2610301850454860`** (abcd records how each change landed): Records which changes a drain lane landed and under which rule.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-e731cc8c71a1 -->
Fidelity review — receipt rcp-e731cc8c71a1 (verifier intent-auditor claude (the harness exposed no model id)).

Provenance: intent-auditor@claude (the harness exposed no model id) · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:b02e64727895ee1512b9bba5901a773d4dadf01b26bec31aad53e790df9ab744
Input attestations: diff:e1f96e610526a5a14bf93e95f6632daa8bafa51e..2e297483e6873e9bcc65ce90dc43e775456cd465 (lane-2, spec step 6)@sha256:f3f411552cfb165d4a8847d8c5bd0daa7a3abc1682dc01586a21ab75782d19fb; diff:87a5dd1b41e3786b81ec24822dd25edee3a5e022..ade000d23d6078368df4d8babe9e1fc67e4fc889 (lane-1, spec step 5, PR #1370)@sha256:993dc3ba64deb9dd4a44ff6198f8301e2162dba01b9fcd80863cd1bd0496e97a; commit:80fa3da48 (spec step 1), 378348a14 (step 2), 91765d9c5 (step 3), 6594725b5953 (step 4); repository read at 2e297483e6873e9bcc65ce90dc43e775456cd465@-; intent:.abcd/development/intents/planned/itd-82-drain-ledger-triage.md@sha256:31a47f4b6c2da6e383395969302c8c4d03322a801c978755e9227eb2cb611064; spec:.abcd/development/specs/open/spc-2609212015054359-drain-ledger-triage.md@sha256:d13def14dda90a34d81a45e7ba6e0030004c318eed75ad91beb9707da9c8bd39;

Acceptance rollup: MET 9 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: The field rule asks blocked, security, category, severity and remedy in a fixed order, so each issue gets exactly one disposition. A fixture with the criterion's six issues gives one eligible, one ineligible (remedy), three hand-backs (severity, security, category naming process) and one skip naming its blocker. The test passed when I re-ran it at 2e297483e.
  evidence: internal/core/capture/eligible_test.go:142 — "func TestDrainRoutesAMixedLedgerByField(t *testing.T) {"
  evidence: internal/core/capture/eligible_test.go:153 — "want 6 dispositions (one per open issue)"
  evidence: internal/core/capture/eligible.go:155 — "case len(iss.BlockedByOpen) > 0:"
  evidence: internal/core/capture/eligible.go:200 — "case strings.TrimSpace(iss.Remedy) == "":"
- ac-2 — MET: A host-judgement yes hands the issue back, before any lane opens, under its kind and with its reason. judgeAnswer records an answer over an issue that is no longer eligible and gives it no effect, so the judgement can never admit an issue. Tests cover the user-visible yes, the trust-boundary yes and the never-eligible case, and they passed when I re-ran them.
  evidence: internal/core/implement/loop/judgement.go:377 — "if verdict == nil || verdict.Outcome != capture.DrainEligible {"
  evidence: internal/core/implement/loop/judgement.go:384 — "out.Note = j.Issue + " is no longer eligible, so the judgement decides nothing: " + why"
  evidence: internal/core/implement/loop/judgement_test.go:198 — "func TestTheJudgementNeverMakesAnIneligibleIssueEligible(t *testing.T) {"
  evidence: internal/core/implement/loop/judgement_test.go:142 — "func TestAUserVisibleRemedyIsHandedBackBeforeAnyLaneOpens(t *testing.T) {"
- ac-3 — MET_WITH_CONCERNS: The issue key opens one lane whose brief is the record and its remedy. The landing resolves the issue in the lane's own change, with a Resolves: trailer and the commit, and opens one pull request armed by the ruleset's method; nothing is pushed after arming. Concern: the reproduce-then-fix definition of done is an instruction in the brief, judged by the validators. Nothing in the loop checks that a detector failed before the fix and passed after, so 'holds' rests on the implementer and the reviewer. The intent accepts that (the reviewer stays the oracle), but the criterion's wording is stronger.
  evidence: internal/core/implement/loop/issuebrief.go:154 — "Reproduce, then fix: write a detector (a test) that fails before the fix and passes after"
  evidence: internal/core/implement/loop/land.go:389 — "if _, err := capture.Resolve(capture.ResolveRequest{RepoRoot: wt, ID: r.Issue, Resolution: r.Note, Impact: r.Impact,"
  evidence: internal/core/implement/loop/issue_test.go:349 — "func TestAnIssueLaneLandsOnePullRequestThatResolvesItsIssue(t *testing.T) {"
  evidence: internal/core/implement/loop/land.go:42 — "Nothing is pushed to the lane"
  evidence: internal/core/implement/loop/land_test.go:460 — "func TestNothingIsPushedAfterArming(t *testing.T) {"
- ac-4 — MET: A user-visible hand-back goes through capture promote and mints an intent draft. The drain test diffs the record line by line and finds exactly one added line, related_intents naming the draft, with nothing removed. Promote has no verdict path: I found no verdict or adoption write in promote.go.
  evidence: internal/core/implement/loop/drain_test.go:137 — "if r.Route != RoutePromoted || !strings.HasPrefix(r.Draft, "itd-") {"
  evidence: internal/core/implement/loop/drain_test.go:165 — "if len(removed) != 0 || len(added) != 1 || !strings.HasPrefix(added[0], "related_intents:") {"
  evidence: commands/drain.md:562 — "| `user-visible` | `promoted`: an intent draft seeded from the issue (`capture promote`) | the draft, and the issue's `related_intents`; nothing else |"
- ac-5 — MET_WITH_CONCERNS: Lane-2 delivers the marker. A trust-rule hand-back, from a lane or from the judgement, is flagged in the summary with its question, and markDecisionNeeded writes `decision_needed: <question>` onto the open record as the only changed line; no draft or decision record is minted. Tests check the one-line diff, the dry run and a later drain reading the marker, and no double write; they passed when I re-ran them. Concerns: first, a move planned from origin writes the marker only where this checkout holds the record as origin's default branch does; otherwise there is only the summary flag. Second, the marker is written uncommitted in the checkout, so no other clone sees it until a person commits it. Third, decision 10 says the marker names the decision record it needs; the marker holds only the question, because nothing is minted to name. Fourth, no verb clears the marker; a person edits the line out by hand.
  evidence: internal/core/implement/loop/drain.go:1441 — "wrote, err := markDecisionNeeded(repoRoot, issue, from, hb.Reason, origin)"
  evidence: internal/core/capture/decisionneeded.go:116 — "newContent, err := setScalarField(content, issueschema.DecisionNeededKey, question)"
  evidence: internal/core/implement/loop/drain_marker_test.go:92 — "if len(removed) != 0 || len(added) != 1 || added[0] != "decision_needed: \""+question+"\"" {"
  evidence: internal/core/implement/loop/drain.go:1481 — "return "nothing here: " + why + ", so the decision_needed: marker is not written here; a flag in this summary", nil"
  evidence: commands/drain.md:580 — "checkout, uncommitted, as a promotion is; a move planned from `origin` writes"
- ac-6 — MET: A receipt carrying handback ends the lane before its validators. The worktree and branch are removed, the discarded head is recorded, the lane is handed back with its kind and reason, and every later step refuses. The drain routes it by kind and counts it as its own outcome (handed-back, lanes_handed_back). The tests passed when I re-ran them.
  evidence: internal/core/implement/loop/issue_test.go:289 — "func TestALaneReportHandBackStopsTheLaneAndDiscardsItsWork(t *testing.T) {"
  evidence: internal/core/implement/loop/drain.go:114 — "DrainLaneHandedBack = "handed-back""
  evidence: internal/core/implement/loop/drain.go:318 — "LanesHandedBack int `json:"lanes_handed_back"`"
- ac-7 — MET: At the window's end the drain writes next_eligible_at to its state file and opens nothing. Before that time a drain opens nothing; after it, the next invocation opens the next lane. --max stops the drain at the cap, names it in next and refuses a negative value. The tests passed when I re-ran them.
  evidence: internal/core/implement/loop/drain_test.go:225 — "func TestADrainPausesAtItsWindowsEnd(t *testing.T) {"
  evidence: internal/core/implement/loop/drain_test.go:208 — "if capped.Stopped != DrainStoppedCap || !capped.Complete || capped.Start != nil || !strings.Contains(capped.Next, "--max 1") {"
- ac-8 — MET: commands/drain.md states what the drain does, lists every flag in its argument hint and prose, and gives the category order, a rule table of hand-backs, a hand-back-by-kind table (now including the marker) and a Refusals section. A test holds the page to the marker rows.
  evidence: commands/drain.md:4 — "argument-hint: "[--dry-run] [--max < n>] [--pace < work-minutes>/< pause-minutes>] [--sub-agents < n>] [--fix-rounds < n>] [--judgement < file>] [--take-over] [--skip < iss-N|path> --reason < why>] [--unskip < iss-N|path>]""
  evidence: commands/drain.md:96 — "`tech-debt`, `documentation`, `inconsistency`, `drift`, `bug`, `ux`, then"
  evidence: commands/drain.md:641 — "## Refusals"
  evidence: internal/core/capture/eligible_test.go:714 — "func TestTheDrainPageNamesTheDecisionRecordMarker(t *testing.T) {"
- ac-9 — MET: Without an accepted record of the eligibility rule, both the dry run and the bare drain refuse and write nothing. The refusal names 'drain eligibility record', its four fields and how to add it. The record exists for abcd itself (adr-2609291342092738). The rule moved to each repository's own record (rulings BX2/H11), so the refusal names the kind of record needed rather than one fixed ADR id.
  evidence: internal/core/drainrule/drainrule.go:134 — "ErrUnrecorded = errors.New("this repository holds no drain eligibility record")"
  evidence: internal/core/implement/loop/drain_test.go:263 — "if _, err := Drain(repo.Root(), Options{}, DrainOptions{}); err == nil || !strings.Contains(err.Error(), "drain eligibility record") {"
  evidence: internal/core/capture/eligible_test.go:442 — "func TestADrainRefusesARepositoryWithoutItsOwnRule(t *testing.T) {"
- ac-10 — MET: In text and --json, the summary carries every disposition, the routes (pull request number, draft, decision-record question, home), the passed issues with their blockers, the order rule and max, plus the counts and spend. A drain whose lanes all ended without a pull request says this is a normal ending, and each move exits 0. The tests passed when I re-ran them.
  evidence: internal/core/implement/loop/drain.go:568 — "res.Next = joinNext(res.Next, "no lane ended in a pull request, which is a normal ending: every lane's outcome and hand-back is above")"
  evidence: internal/core/implement/loop/drain.go:255 — "Order string `json:"order"`"
  evidence: internal/core/implement/loop/drain_counts_test.go:87 — "func TestADrainWhoseLanesAllEndedWithoutAPullRequestSaysSo(t *testing.T) {"
  evidence: commands/drain.md:164 — "Each `abcd drain` without `--dry-run` performs one move and exits 0, saying"
- ac-11 — MET: Issue ids are checked against IssIDRe before anything is read. A traversal-shaped key such as iss-../../x is refused at the key check with no run tier written. The new marker writer checks the id shape and requireHeld before the preamble, and ledger writes are guarded against redirection outside the checkout.
  evidence: internal/core/implement/loop/issue_test.go:158 — "for _, key := range []string{"iss-../../x", "iss-", "iss-12a", "iss-1/2", "iss-0", "iss-02609292352131344"} {"
  evidence: internal/core/capture/decisionneeded.go:76 — "if !reIssID.MatchString(req.ID) {"
  evidence: internal/core/capture/ledgerroot_test.go:80 — "func TestLedgerRecordWriteCannotBeRedirectedOutsideTheCheckout(t *testing.T) {"

Gap audit:
- honoured:
  - Eligibility is rule-first, by field, and the host judgement can only hand back.
    evidence: internal/core/capture/eligible.go:142 — "func eligibility(iss Issue, r drainrule.Rule, anchor deferralAnchor, opts ...eligibilityOption) DrainVerdict {"
    evidence: internal/core/implement/loop/judgement.go:378 — "// The judgement never makes an issue eligible: it is recorded, and"
  - Decision 10: a trust-rule hand-back marks the open issue with its question, changes nothing else and mints nothing. Invariant 19 names the marker as the hand-back's flag, not the classification, so decision 8 keeps re-deriving the classification every run.
    evidence: internal/core/capture/decisionneeded.go:71 — "func MarkDecisionNeeded(req DecisionNeededRequest) (DecisionNeededResult, error) {"
    evidence: .abcd/development/brief/02-constraints/03-invariants.md:218 — "The one write a hand-back makes to an issue it does not promote is the decision-record marker"
  - The hand-back is routed by kind: a user moment is promoted, a trust rule gets a decision-record flag and marker, a design finding or second package is flagged with its home.
    evidence: internal/core/implement/loop/drain.go:1439 — "case HandBackTrustRule:"
    evidence: commands/drain.md:563 — "| `trust-rule` | `decision-record`: flagged, with the lane's reason as `question` | `decision_needed:` on the open issue, holding the question; nothing else; no record is minted |"
  - The merge queue is the run's path to main: an armed pull request that leaves the queue is put back or reported with the failed check (lane-1, step 5).
    evidence: internal/core/implement/loop/queue_test.go:77 — "func TestALandedPullRequestLeftUnqueuedAtCleanIsEnqueued(t *testing.T) {"
    evidence: internal/core/implement/loop/queue_test.go:171 — "func TestAPullRequestDroppedAfterAFailedMergeGroupCheckIsReportedWithTheCheck(t *testing.T) {"
  - 'No pull request, reason logged' is a normal terminal state.
    evidence: internal/core/implement/loop/drain.go:568 — "no lane ended in a pull request, which is a normal ending"
- diverged:
  - The intent says the rules are the fields remedy, category, severity and blocked_by. The delivered rule also hands back on related_intents (promoted), the new decision_needed marker, a remedy opening 'Waits on', deferred_after, and the automatic filers' machine remedy. Each of these only narrows the set, so the safety direction holds, but the rule reads more than the four fields the intent names.
    evidence: internal/core/capture/eligible.go:168 — "case strings.TrimSpace(iss.DecisionNeeded) != "":"
    evidence: commands/drain.md:83 — "| `handback` | `waits-on-ruling` | its remedy opens "Waits on": the fix waits on a ruling a person has not given |"
  - Decision 10 says the marker names the decision record it needs and states the question. The marker holds the question only: nothing is minted, so there is no record id to name. It is written uncommitted in the checkout, and on a move planned from origin only where the checkout holds the record as origin does.
    evidence: internal/core/issueschema/issueschema.go:187 — "const DecisionNeededKey = "decision_needed""
    evidence: internal/core/implement/loop/drain.go:1481 — "so the decision_needed: marker is not written here; a flag in this summary"
  - The eligibility record is no longer one ADR this intent names as a prerequisite. It is each repository's own accepted record carrying four drain_ fields, which abcd ahoy install offers to write, and a repository's record may loosen the floors.
    evidence: internal/core/drainrule/drainrule.go:146 — "const HowToAdd = "add it: run `abcd ahoy install` at a terminal and accept the drain rule it offers, " +"
  - The detector-first definition of done is an instruction to the implementer that the validators judge. It is not a check the loop enforces.
    evidence: internal/core/implement/loop/issuebrief.go:156 — "conventions below state it, in your worktree. The validators judge the fix with the detector as"
- missing:
  - No verb clears the decision-record marker once the decision record is written. A person removes the decision_needed: line by hand. The implementer judged a clearing verb to be out-of-scope schema surface.
    evidence: commands/drain.md:586 — "question as its reason. Once the decision record is written, take the"

Scope-condition dispositions:
- cond-2609212015056112 — survived: The landing arms the merge through landing.DecideMerge and the ruleset mirror's queue method; where the gate does not allow it, the pull request is left open. Lane-1 adds a watch that re-queues or reports an armed pull request that leaves the queue. The default branch is still reached through a path the run does not control.
  evidence: internal/core/implement/loop/land.go:968 — "d, err := landing.DecideMerge(landing.MergeQuery{RepoRoot: c.RepoRoot, Roots: roots, Tip: tip, TipName: Remote + "/" + def, Default: def,"
  evidence: internal/core/implement/loop/queue.go:3 — "// queue.go is the landing's watch over the merge queue (spc-2609212015054359"
- cond-2609212015051754 — survived: capture refuses a new issue without a remedy. A record without one, or carrying the automatic filers' machine value, gets the outcome ineligible under the remedy rule and is listed with how to add one.
  evidence: internal/core/capture/capture.go:466 — "ErrRemedyRequired = errors.New("remedy required")"
  evidence: internal/core/capture/eligible.go:200 — "case strings.TrimSpace(iss.Remedy) == "":"
- cond-2609212015056014 — survived: No drain starts without an accepted record of the eligibility rule: both the dry run and the bare verb refuse and write nothing. abcd's own record, adr-2609291342092738, exists, so the condition holds in this repository. The record is now each repository's own rather than a single named ADR; that move is recorded under diverged.
  evidence: internal/core/drainrule/drainrule.go:134 — "ErrUnrecorded = errors.New("this repository holds no drain eligibility record")"
  evidence: internal/core/implement/loop/drain_test.go:261 — "func TestADrainRefusesWithoutTheRepositorysRule(t *testing.T) {"
- cond-2609212015052582 — narrowed: Only the field rule can make an issue eligible. The judgement can only hand back, and it is recorded with the disposition in the drain's state and summary. But the delivered rule reads more than remedy, category, severity and blocked_by: lane-2 adds decision_needed, and the rule also reads related_intents, a 'Waits on' remedy, deferred_after and the machine remedy. So 'the rules are the fields' holds only in the admitting direction.
  narrowing: The four named fields (remedy, category, severity, blocked_by) are the only fields that can make an issue eligible, and the host judgement only hands back. Further fields can also hand an issue back: related_intents, decision_needed (written by the drain's own trust-rule hand-back), a remedy opening 'Waits on', deferred_after and the machine remedy. So the four fields are necessary for eligibility, but they are not the whole rule.
  evidence: internal/core/capture/eligible.go:168 — "case strings.TrimSpace(iss.DecisionNeeded) != "":"
  evidence: internal/core/implement/loop/judgement.go:384 — "out.Note = j.Issue + " is no longer eligible, so the judgement decides nothing: " + why"
<!-- abcd-review-end receipt=rcp-e731cc8c71a1 -->

## Grounds

- pursued: the autonomous run's capture batches are a person's hand-written routing today; we expect the drain to be what replaces them after the run; shown wrong if the next run still starts from a hand-written table
