---
id: itd-2609212103565953
slug: a-spec-lists-its-steps-and-the-build
spec_id: spc-2609212138246060
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916151817, itd-2609211116005482]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609211913453478]
related_adrs: [adr-2609212115255771]
---

# A spec lists its steps, and the build lands them one at a time

## Press Release

> **A spec lists its steps, and `abcd build` lands them one at a time, each its own lane and pull request.**
>
> "The implement verb was three lanes in the spec's order, and nothing had a word for the three," said a technical facilitator reading the run file. "Now the spec says its steps, the loop builds them in that order, and a step that does not fit the cut leaves the rest as the remainder. No new record family, no task tracker: a list in the design record."

## Why This Matters

The record's only unit below a spec was another spec, minted as a remainder, and the build machinery's scope condition says a lane too large for one context is split by the spec, not by the loop. The run file already split the implement verb into three lanes with no name for the pieces. The field's answers are Shape Up's scopes (pieces inside the pitch) and Kubernetes' in-record graduation; a child record family (Linear's sub-issues) adds folders and verbs the vocabulary is shedding. Ruled on 2026-09-21 (adr-2609212115255771, decision 4): a section in the spec, called steps, sharing the loop's own word.

## Mechanism

We expect specs split into named steps to land with fewer fix rounds and smaller lane contexts than the pilot's 349k-token lane, because a step is what one implementer can hold and land; shown wrong if stepped specs show no smaller lane contexts than unstepped ones of the same footprint.

## Scope Conditions

None stated.

## What's In Scope

- **The section**: `## Steps` in the spec, an ordered list, each step with a title and its footprint (packages, tests); the spec template seeds it empty at `intent plan`; a spec without steps is one step.
- **The loop**: `abcd build` reads the list and runs one lane per step in order, each landing as its own pull request; the next step's lane starts after the previous has merged.
- **The remainder**: a step that does not fit the cut leaves the unfinished steps in the remainder spec `spec close --remainder` mints.
- **The brief**: a lane's brief names the step it builds and the steps before it; the run record lists steps as it lists lanes.
- **One word**: `implement step` (the loop's step interface) and this section share the word, and the command page says so.

## What's Out of Scope

- The build proposing a split (the author writes the steps).
- A task record family.
- Reordering steps mid-run.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. The unit below a spec is a section called steps, not a record family (adr-2609212115255771, decision 4).
2. The spec's author writes the steps; a spec without any is one step.
3. A step that does not fit the cut leaves the rest as the remainder.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** `abcd intent plan` mints a spec, **when** the stub is read, **then** it carries an empty `## Steps` section beside `## Footprint`, and a spec with no steps listed is built as one step.
- **Given** a spec with three steps, **when** `abcd build` runs it, **then** three lanes run in order, each landing as its own pull request, and the second starts only after the first has merged.
- **Given** a step that does not fit the cut, **when** the spec is closed with `--remainder`, **then** the remainder spec carries the unfinished steps.
- **Given** a lane for a step, **when** its brief is rendered, **then** it names the step and the steps before it, and the run record lists the steps.
- **Given** the command page, **when** it is read, **then** `implement step` and the spec's steps are described as one word for one thing.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-2257cc3a38d9 -->
Fidelity review — receipt rcp-2257cc3a38d9 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:9917c69a7540149729e8eb4af6ad1265e655f9d85331a4d795333c961f001612
Input attestations: diff:e5dfe0fc4 3f20b5167 3ebb7b0e3 8ede4810f 3c19408cb 3130ebca8 4651b460b (ancestors of 4651b460b, docs/close-spec-steps)@sha256:1041e178e12eb68b3ea1faca87bc624fd726a8b8643aff919dd04eb09321131a;

Acceptance rollup: MET 2 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: stubSections renders ## Footprint then ## Steps with the italic placeholder; spec.Steps turns a section listing none into one implicit step, startStepsRow hands the loop one step for such a spec, and TestPlanMintsASpecWithAnEmptyStepsSection proves both at the plan verb (passes on an archive copy of HEAD)
  evidence: internal/core/spec/spec.go:308 — "{heading: "## Summary", body: summary}, {heading: FootprintHeading, body: footprintStubBody}, {heading: StepsHeading, body: stepsBody}"
  evidence: internal/core/spec/spec.go:289 — "const stepsPlaceholder = "_No steps listed: the spec is built as one step."
  evidence: internal/core/spec/steps.go:294 — "if len(steps) == 0 { return []Step{{Number: 1, Title: ImplicitStepTitle, Implicit: true}}, nil }"
  evidence: internal/core/intent/startcheck.go:320 — "row.Detail = sp.ID + " lists no steps: it is built as one step""
  evidence: internal/core/intent/steps_test.go:24 — "func TestPlanMintsASpecWithAnEmptyStepsSection"
  evidence: internal/core/spec/steps_test.go:67 — "func TestStepsOfAnUnsteppedSpecIsOneImplicitStep"
- ac-2 — MET_WITH_CONCERNS: readyPending opens a step's lane only when landedInRun holds for every step requires() names (all earlier steps without a needs line, DR6b), landedInRun demands Stage == StageDone, and a lane reaches done only after landMerged fetches origin's default branch and proves the pushed head an ancestor of it (an armed-but-unmerged pull request is a contention, not done); each lane opens its own pull request (lane.PR). Concerns: the merge-gate test runs two steps, not three, and the three-step in-order test (TestTheRecordNamesEachLaneAsItOpens) uses fake stages with no landing or pull request, so the criterion's three-lanes-three-pull-requests scenario is composed from parts, never exercised end to end; and because the gate is an ancestor check, a squash- or rebase-merged pull request is refused rather than counted merged, so under those merge methods the second lane never opens
  evidence: internal/core/implement/loop/schedule.go:182 — "func (s State) readyPending() []int"
  evidence: internal/core/implement/loop/schedule.go:172 — "if l.SpecStep == n && l.Stage != StageDone { return false }"
  evidence: internal/core/implement/loop/schedule.go:151 — "func (p PendingStep) requires() []int"
  evidence: internal/core/implement/loop/land.go:989 — "netGit(c.RepoRoot, "fetch", "--quiet", "--no-tags", Remote, "+refs/heads/"+def+":"+tracking)"
  evidence: internal/core/implement/loop/land.go:996 — "on, err := gitutil.IsAncestor(c.RepoRoot, ld.Pushed, tracking)"
  evidence: internal/core/implement/loop/land.go:1015 — "return Outcome{}, contend(string(StageLand), "", lane.ID, "pull request #"+n+" is not merged yet"
  evidence: internal/core/implement/loop/land.go:1008 — "pull request #"+n+" merged, but the lane's head "+shortSHA(ld.Pushed)+" is not on "+Remote+"/"+def+" (a squash or rebase rewrote it)"
  evidence: internal/core/implement/loop/parallel_test.go:498 — "f := newParFixture(t, "1. One\n2. Two\n", Options{SubAgents: strp("3")})"
  evidence: internal/core/implement/loop/parallel_test.go:518 — "no lane opens for step 2 while lane 1's pull request is not merged, with every slot free"
  evidence: internal/core/implement/loop/parallel_test.go:273 — "f.repo.Git("push", "-q", "origin", l.Landing.Pushed+":refs/heads/main")"
  evidence: internal/core/implement/loop/loop_test.go:861 — "specWithSteps("1. The parser\n2. The loop\n3. The page\n")"
  evidence: internal/core/implement/loop/loop_test.go:866 — "f := &fakeSteps{calls: map[Stage]int{}}"
  evidence: internal/core/implement/loop/land_test.go:321 — "if r := mustRefusal(t, err); !r.Contention || !strings.Contains(r.Reason, "not on")"
  evidence: internal/core/implement/loop/parallel_test.go:613 — "strings.Count(f.ghLog(t), "pr merge 7 --auto") != 1"
- ac-3 — MET_WITH_CONCERNS: the --remainder close reads the closing spec's steps, carries the ones without a `- landed:` line into the minted remainder renumbered from one (CarryUnlanded, CreateWithSteps), and refuses writing nothing when the section cannot be read; TestReconcileRemainderCarriesTheUnlandedSteps passes on the archive copy. Concern: 'unfinished' is decided solely by a hand-written `- landed:` marker the delivery chose outside the record (e5dfe0fc4's message says so), and nothing in the loop's landing writes that line, so a step the loop landed as a pull request is carried into the remainder unless the author marks it by hand
  evidence: internal/core/intent/lifecycle.go:932 — "listed, err := spec.ReadSteps(repoRoot, sp)"
  evidence: internal/core/intent/lifecycle.go:939 — "carried, rewrites = spec.CarryUnlanded(listed)"
  evidence: internal/core/intent/lifecycle.go:987 — "minted, err = spec.CreateWithSteps(repoRoot, intentID, remainder.Slug, remainder.ProductionMode, carried)"
  evidence: internal/core/spec/steps.go:316 — "func Unlanded(steps []Step) []Step"
  evidence: internal/core/spec/steps.go:248 — "func CarryUnlanded(steps []Step) ([]Step, []NeedsRewrite)"
  evidence: internal/core/intent/steps_test.go:46 — "func TestReconcileRemainderCarriesTheUnlandedSteps"
  evidence: internal/core/implement/loop/land.go:1043 — "pull request #%s landed: %s is on %s/%s at %s; removed the lane's worktree and branch %s"
- ac-4 — MET: renderEarlierSteps writes the lane's step among the spec's and each earlier step with what landed it (the base's `landed:` line and the run's lane with branch, head and pull request), a first step or an unstepped spec saying no step comes before; the run record carries every lane's spec_step and step_title, the pending steps, and an `open` entry naming each lane's step as it opens, rendered as 'spec step N' in text; TestTheBriefNamesTheStepsBeforeItsOwn and TestTheRecordNamesEachLaneAsItOpens pass on the archive copy
  evidence: internal/core/implement/loop/brief.go:589 — "fmt.Fprintf(b, "This lane builds step %d of the %d %s lists", lane.SpecStep, len(steps), st.Spec)"
  evidence: internal/core/implement/loop/brief.go:595 — "for _, s := range steps[:lane.SpecStep-1]"
  evidence: internal/core/implement/loop/brief_steps_test.go:75 — "1. \"The parser\" — landed: #12 (the spec at the lane's base)"
  evidence: internal/core/implement/loop/record.go:114 — "rl := RecordLane{ID: l.ID, Key: l.Key, SpecStep: l.SpecStep, StepTitle: l.StepTitle, Stage: l.Stage"
  evidence: internal/core/implement/loop/record.go:44 — "Pending []PendingStep `json:"pending"`"
  evidence: internal/core/implement/loop/schedule.go:252 — "st.Record = append(st.Record, Entry{At: now, Lane: l.ID, Stage: "open", Note: fmt.Sprintf("%s opened for step %d of %s (%s)""
  evidence: internal/core/implement/loop/loop_test.go:905 — "lane-1: checks passed; lane-1 opened for step 1 of spc-1 (The parser)"
  evidence: internal/surface/cli/build.go:1003 — "fmt.Fprintf(w, " %s: spec step %d, %q — %s\n", l.ID, l.SpecStep, termsafe.Sanitize(l.StepTitle), l.Stage)"
- ac-5 — MET_WITH_CONCERNS: commands/intent.md and commands/build.md now say 'step' names the spec's piece and nothing else, a lane's are its stages, and the payloads keep them apart as spec_step and stage; the glossary says the same. Concern: this is the signed-off divergence BU1 (iss-2609291313276243), not the intent's 'share the word' plan — the verb `implement step` keeps a name that the same page says denotes one stage or one move of the run, not one step, so the verb's own name remains a second use of the word that the page explains rather than removes
  evidence: commands/intent.md:592 — "In the build loop, "step" names this and nothing else: the loop (`/abcd:build`) lands each step as one lane and takes the lane through its **stages**"
  evidence: commands/intent.md:594 — "`abcd implement step` performs one stage, and its payloads name the lane's stage under `stage` and the spec's step under `spec_step`"
  evidence: commands/build.md:16 — "Two words, two things. A **step** is a piece of the spec"
  evidence: commands/build.md:19 — "`implement step` performs one move of the run"
  evidence: .abcd/development/brief/glossary/core/step.md:18 — "`abcd implement step` performs one stage, not one step"
  evidence: .abcd/development/intents/shipped/itd-2609212103565953-a-spec-lists-its-steps-and-the-build.md:48 — "`implement step` (the loop's step interface) and this section share the word, and the command page says so"

Gap audit:
- honoured:
  - the spec template seeds an empty ## Steps section at intent plan and an unstepped spec is one step
    evidence: internal/core/spec/spec.go:308 — "{heading: StepsHeading, body: stepsBody}"
    evidence: internal/core/spec/steps.go:294 — "Implicit: true"
  - a step's lane opens only after every earlier step's pull request is an ancestor of origin's default branch, each step landing as its own lane and pull request
    evidence: internal/core/implement/loop/schedule.go:172 — "l.Stage != StageDone"
    evidence: internal/core/implement/loop/land.go:996 — "gitutil.IsAncestor(c.RepoRoot, ld.Pushed, tracking)"
  - spec close --remainder carries the steps not marked landed into the remainder
    evidence: internal/core/intent/lifecycle.go:939 — "spec.CarryUnlanded(listed)"
  - the brief names the lane's step and the steps before it, and the run record lists the steps
    evidence: internal/core/implement/loop/brief.go:595 — "steps[:lane.SpecStep-1]"
    evidence: internal/core/implement/loop/schedule.go:252 — "Stage: "open""
  - no task record family and no build-proposed split: steps are a section of the spec the author writes
    evidence: internal/core/spec/steps.go:20 — "const StepsHeading = "## Steps""
    evidence: .abcd/development/brief/glossary/core/step.md:22 — "a step is a section of the design record, not a record of its own"
- diverged:
  - `implement step` and the spec's steps share one word for one thing: delivered as a rename (BU1) under which 'step' is the spec's piece only and the verb `implement step` performs a stage
    evidence: commands/build.md:16 — "Two words, two things."
    evidence: commands/intent.md:594 — "`abcd implement step` performs one stage"
  - 'the next step's lane starts after the previous has merged' proven for three steps: the real-landing test runs two steps, and the three-step test runs fake stages with no pull request
    evidence: internal/core/implement/loop/parallel_test.go:498 — ""1. One\n2. Two\n""
    evidence: internal/core/implement/loop/loop_test.go:866 — "fakeSteps"
  - 'a step that does not fit the cut leaves the unfinished steps in the remainder' rests on a hand-written `- landed:` marker the loop never writes
    evidence: internal/core/spec/steps.go:319 — "if s.Landed == "" && !s.Implicit"
    evidence: internal/core/implement/loop/land.go:1042 — "ld.Merged = merged"
- missing: (none)
<!-- abcd-review-end receipt=rcp-2257cc3a38d9 -->

## Grounds

- pursued: the run file already builds the implement verb as three lanes in the spec's order with no word for the pieces; we expect named steps to land with smaller contexts and fewer fix rounds; shown wrong if stepped specs show no smaller lane contexts
