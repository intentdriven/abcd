---
id: itd-2609211116005482
slug: abcd-build-next-picks-the-readiest-planned-intent-itself-wri
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916151817, itd-2609201925079472]
related_intents: [itd-78, itd-82, itd-2609201916151817]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# `abcd build next` picks the readiest planned intent itself, and writes down why

## Press Release

> **`abcd build next` chooses the next intent to build, says why on the record, and builds it.** It reads every planned intent whose gate says READY, scores each on facts its own record holds — how clear its acceptance criteria are, whether there is an obvious way to test it, how small the change it asks for looks — takes the readiest, oldest among equals, and writes onto that intent a grounds entry marked as the run's: who else was considered, why each lost, and what would show the pick wrong. Then it hands the intent to the same machinery `abcd build <itd-N>` uses and takes it to delivered. One intent per run, because "next" means one; `--until-empty` or `--max <n>` keeps it going under the pace rule, which is how `abcd drain` reads by default, because "drain" means all.
>
> "I used to open the run file every morning and pick by feel, then forget by Friday why Tuesday's pick was that one," said a product thinker who had planned forty-eight intents and could build one a day. "Now the run picks, the intent says why in its own record, and when a pick turns out wrong the reason is there to argue with."

## Why This Matters

Forty-eight planned intents were READY in this repository on 2026-09-20, and the file that routes them into batches was written by hand, ordered by a person's sense of what comes first, with no line saying why. Every autonomous coding platform surveyed on 2026-09-21 (Copilot's coding agent, OpenHands, Jules, Devin, Cursor's cloud agents, Claude Code and Codex routines) is triggered by a human assigning a task or a label; none publishes a policy for choosing among a backlog, so a run that is meant to be unattended still begins with a human's choice. Two things the same survey found do have evidence: readiness is predictable from the record alone (static features of the task predict an agent's success at 0.76 to 0.84 AUC before any code is written), and small, tidy work merges far more often than large features (84.7 per cent for clean-ups against 64.5 per cent for features across 878 agent pull requests in one large repository). A pick made from those facts, written where a person can later check it, is what turns the run file's hand-ordered batches into something the binary does.

The reason has to be checkable. Agent-written rationales, when audited, were factually wrong more than half the time in one repository's decision log; the survey found no evidence that recording a reason improves later review, only that an unrecorded one cannot be reviewed at all. So the record the run writes is computed, not composed: the candidate set, each candidate's score and the rule that placed it, and a falsifier the lane's own outcome can meet.

## Mechanism

We expect a pick made from the record's own facts, with its reason written as computed comparison and a falsifier, to choose intents that deliver at a higher rate than hand-ordering and to be correctable when it does not, because the facts that predict an agent's success are in the acceptance criteria, the test path and the footprint before a lane starts, and a written falsifier is what a later reader checks the outcome against; shown wrong if the readiest-scored intents deliver no more often than oldest-first over a run of ten or more picks, or if the falsifier on a failed pick did not name what actually failed.

## Scope Conditions

- Holds for a repository abcd manages whose planned intents carry acceptance criteria in Given-When-Then form and a linked spec, so the readiness score has fields to read; an intent with a placeholder criteria section is not a candidate.
- Holds at the scale the run file recorded: tens of READY intents, not thousands, so the whole candidate set is scored on every run and written into the reason.
- Holds where the intent store's grounds section accepts an entry the run writes and marks as its own beside a human's; the human's entry is never edited or displaced.
- The ordering is a heuristic with the evidence stated above; where `blocked_by` and `builds_on` edges exist, an intent with an unshipped blocker is not a candidate, and the edges are not otherwise ranked on (the dependency-graph draft, `itd-78`, is where a derived priority would come from, and this intent does not build it).

## What's In Scope

- **The candidate set**: every intent in `planned/` whose `abcd intent ready` verdict is READY, that is not held, that no peer holds, and whose `blocked_by` names nothing unshipped.
- **The score**, from the record alone, each component named in the reason: criteria clarity (count and shape of the Given-When-Then bullets), a test path (a spec that names the packages and the tests, or criteria a test can hold), the expected footprint (packages and surfaces the spec names), and the record's age; the weights are a declared configuration, not a prompt.
- **The reason**, written onto the chosen intent's `## Grounds` section as one entry marked as the run's (`pursued:` prefixed with the run's identity and date), carrying the candidates considered with their scores, the rule that placed the winner, the runner-up and why it lost, and the falsifier: the lane outcome that would show this pick wrong (more than the pace rule's fix rounds, a reviewer raising a design question, a footprint outside what the spec named).
- **The hand-over**: the chosen intent goes to the implement machinery exactly as `abcd build <itd-N>` would send it; nothing in the lane differs.
- **The run count**: one by default; `--max <n>` and `--until-empty` continue under the pace rule, each further pick written the same way.
- **Refusals**: no READY intent (says so, writes nothing); every READY intent held or blocked (lists them); a tie the score cannot break beyond age (takes the oldest and says the tie was broken by age).

## What's Out of Scope

- The build itself, the lane, the validators, the landing: `itd-2609201916151817`.
- The pace, the ceiling and the pause: `itd-2609201925079472`.
- A derived priority over the dependency graph: `itd-78`; this intent filters on edges and does not rank on them.
- The issue ledger: `abcd drain` (`itd-82`) is the run over issues, and it hands over to the same machinery.
- Any estimate the agent makes of its own confidence as an input to the pick.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed this draft:

1. **The verb is `build`, for people; `implement` is the machinery, for agents.** `abcd build <itd-N>` and `abcd build next` are what a person types; both hand over to `abcd implement`, whose step-level words (`step`, `receipt`) a driving host calls. The planned implement intent is amended to say so; which verbs the command list shows a person and which it shows an agent is a separate record (`iss-2609211119023345`).
2. **Readiest first, oldest among equals.** Not oldest-first, not most-important-first; the evidence for readiness as the predictor is the reason, and the order is declared a heuristic.
3. **The reason lives on the intent itself**, as a grounds entry marked as the run's, so opening the intent shows why it was picked and what would prove the pick wrong.
4. **One by default, many by flag**, and the defaults differ from `drain` on purpose: "next" reads one and "drain" reads all; the flags are the same on both.
5. **The reason is computed, never composed.** Candidates, scores, placing rule, falsifier; no prose the agent writes about its own choice.

## Open Questions

- **Whether a run-made grounds entry needs its own token.** `pursued:` with the run's identity in the text is the shape assumed here; if the vocabulary gains a token for a machine's conjecture, this intent takes it.
- **The score's weights.** Declared in configuration with a bundled default; the default is set from the first ten picks' outcomes, not chosen up front.

## Acceptance Criteria

- **Given** planned intents of which some are READY, some held, some blocked and some not READY, **when** `abcd build next` runs, **then** only the READY, unheld, unblocked ones are candidates, and the refusal for an empty candidate set names each excluded intent and the check that excluded it, writing nothing.
- **Given** two or more candidates, **when** the pick is made, **then** the chosen intent's `## Grounds` section gains exactly one entry marked as the run's, naming every candidate with its score, the rule that placed the winner, the runner-up and why it lost, and a falsifier, and no existing entry is changed.
- **Given** candidates whose scores tie, **when** the pick is made, **then** the oldest is taken and the entry says the tie was broken by age.
- **Given** a pick, **when** the lane starts, **then** it is the lane `abcd build <itd-N>` would start for that intent, with the same brief, worktree, validators and landing.
- **Given** the default run count, **when** the picked intent is delivered or its lane stops, **then** the run reports and exits without picking again; **given** `--max <n>` or `--until-empty`, **then** it picks again under the pace rule and writes each further reason the same way.
- **Given** a delivered pick, **when** the falsifier's condition is met by the lane's outcome (a design question raised by a reviewer, a footprint outside the spec, fix rounds past the pace rule's count), **then** the run record names the pick as falsified and the intent's entry is not edited.
- **Given** `--json`, **when** any of the above renders, **then** the candidate set, scores, the reason and every refusal are in the payload.

## Typed Links

- **builds_on `itd-2609201916151817`** (the implement machinery, `abcd build <itd-N>`): the pick hands over to it and adds nothing to the lane.
- **builds_on `itd-2609201925079472`** (the pace): `--until-empty` and `--max` run under it.
- **refines `itd-78`** (the dependency graph): this intent filters on `blocked_by` and does not derive a priority; that draft is where a derived one would come from.
- **refines `itd-82`** (`abcd drain`): the sibling run over the issue ledger; the two share the flags and the hand-over and differ in their default count by the meaning of their names.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
