---
id: itd-2609211116005482
slug: abcd-build-next-picks-the-readiest-planned-intent-itself-wri
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916151817, itd-2609201925079472]
related_intents: [itd-78, itd-82, itd-2609201916151817, itd-50]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# `abcd build next` picks the readiest planned intent itself, and writes down why

## Press Release

> **`abcd build next` chooses the next intent to build, says why on the record, and builds it.** It reads every planned intent that passes every refusal check `abcd build <itd-N>` would run (READY, no open question, no unanswered claim, not held, not blocked, no peer holding it), scores each on facts its own record holds — how clear its acceptance criteria are, whether there is an obvious way to test it, how small the change it asks for looks — takes the readiest, oldest among equals, and writes onto that intent a grounds entry marked as the run's: who else was considered, why each lost, and what would show the pick wrong. The entry is the lane's first commit, so the reason reaches `main` with the work and stays on the branch as history if the lane is discarded. Then it hands the intent to the same machinery `abcd build <itd-N>` uses and takes it to delivered. One intent per run, because "next" means one; `--until-empty` or `--max <n>` keeps it going under the pace rule, which is how `abcd drain` reads by default, because "drain" means all.
>
> "I used to open the run file every morning and pick by feel, then forget by Friday why Tuesday's pick was that one," said a product thinker who had planned forty-eight intents and could build one a day. "Now the run picks, the intent says why in its own record, and when a pick turns out wrong the reason is there to argue with."

## Why This Matters

Forty-eight planned intents were READY in this repository on 2026-09-20, and the file that routes them into batches was written by hand, ordered by a person's sense of what comes first, with no line saying why. Every autonomous coding platform surveyed on 2026-09-21 (Copilot's coding agent, OpenHands, Jules, Devin, Cursor's cloud agents, Claude Code and Codex routines) is triggered by a human assigning a task or a label; none publishes a policy for choosing among a backlog, so a run that is meant to be unattended still begins with a human's choice. Two things the same survey found do have evidence: readiness is predictable from the record alone (static features of the task predict an agent's success at 0.76 to 0.84 AUC before any code is written), and small, tidy work merges far more often than large features (84.7 per cent for clean-ups against 64.5 per cent for features across 878 agent pull requests in one large repository). A pick made from those facts, written where a person can later check it, is what turns the run file's hand-ordered batches into something the binary does.

The reason has to be checkable. Agent-written rationales, when audited, were factually wrong more than half the time in one repository's decision log; the survey found no evidence that recording a reason improves later review, only that an unrecorded one cannot be reviewed at all. So the record the run writes is computed, not composed: the candidate set, each candidate's score and the rule that placed it, and a falsifier the lane's own outcome can meet.

## Mechanism

We expect a pick made from the record's own facts, with its reason written as computed comparison and a falsifier, to choose intents that deliver at a higher rate than hand-ordering and to be correctable when it does not, because the facts that predict an agent's success are in the acceptance criteria, the test path and the footprint before a lane starts, and a written falsifier is what a later reader checks the outcome against; shown wrong if the falsifier on a failed pick did not name what actually failed, or if picks scored readiest end unachievable (itd-50's verdict) more often than not over ten or more picks.

## Scope Conditions

- Holds for a repository abcd manages whose planned intents carry acceptance criteria in Given-When-Then form and a linked spec, so the readiness score has fields to read; an intent with a placeholder criteria section is not a candidate.
- Holds where the intent store's grounds section accepts an entry the run writes and marks as its own beside a human's; the human's entry is never edited or displaced, and the readiness gate reports the human's entry as the most recent conjecture, skipping the run's marker when it summarises.
- The ordering is a heuristic with the evidence stated above; where `blocked_by` and `builds_on` edges exist, an intent with an unshipped blocker is not a candidate, and the edges are not otherwise ranked on (the dependency-graph draft, `itd-78`, is where a derived priority would come from, and this intent does not build it).

## What's In Scope

- **The candidate set**: every intent in `planned/` that passes every refusal check `abcd build <itd-N>` runs before it starts (READY, no open question, no unanswered claim section, not held, no peer holding it, nothing unshipped in `blocked_by`); the pick never writes onto an intent the build would then refuse.
- **The score**, from the record alone, each component named in the reason: criteria clarity (count and shape of the Given-When-Then bullets), a test path (the spec's `## Footprint` section naming the tests, absent today on every spec), and the expected footprint (that section's package list); three parts at equal weight, the bundled default, revisable after ten picks; age is not a part of the score and breaks ties only. Until a spec carries the section, its test-path and footprint parts read zero and the reason says so.
- **The reason**, written onto the chosen intent's `## Grounds` section as one entry marked as the run's (`pursued:` prefixed with the run's identity and date), carrying the candidates considered with their scores, the rule that placed the winner, the runner-up and why it lost, and the falsifier: a lane outcome the state file can meet (fix rounds past the pace rule's count, or the lane handing back as unachievable under itd-50's stage). The marker ("picked by run <id> on <date>") goes inside the entry's text after the `pursued:` token, since the token vocabulary is closed. The entry is committed as the lane's first commit on the lane branch.
- **The hand-over**: the chosen intent goes to the implement machinery exactly as `abcd build <itd-N>` would send it; nothing in the lane differs.
- **The run count**: one by default; `--max <n>` and `--until-empty` continue under the pace rule, each further pick written the same way.
- **Refusals**: no candidate (says so, names each excluded intent and the check that excluded it, writes nothing); a tie the score cannot break beyond age (takes the oldest and says the tie was broken by age).
- **The spec template**: gains a `## Footprint` section (packages, tests) the score reads; `intent plan` seeds it empty.
- **The command page**: `commands/intent.md` (or the build page once it exists) says what `next` does, its two flags and its refusals.

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
4. **One by default, many by flag**, and the defaults differ from `drain` on purpose: "next" reads one and "drain" reads all; `--max <n>` and `--until-empty` are the two flags both share.
5. **The reason is computed, never composed.** Candidates, scores, placing rule, falsifier; no prose the agent writes about its own choice.
6. **The reason is the lane's first commit** on the lane branch (ruled 2026-09-21 on the design review's finding): it reaches `main` with the work and stays on the branch if the lane is discarded.
7. **Equal weights at first**, declared as the bundled default and revisable after ten picks; age breaks ties and is not scored.

## Open Questions

- **Whether a run-made grounds entry needs its own token.** `pursued:` with the run's identity in the text is the shape assumed here; if the vocabulary gains a token for a machine's conjecture, this intent takes it.

## Acceptance Criteria

- **Given** planned intents of which some pass every refusal check of `abcd build <itd-N>` and some fail one (not READY, an open question, an unanswered claim, held, blocked, a peer holding it), **when** `abcd build next` runs, **then** only those that pass are candidates, and the refusal for an empty candidate set names each excluded intent and the check that excluded it, writing nothing.
- **Given** two or more candidates, **when** the pick is made, **then** the chosen intent's `## Grounds` section gains exactly one `pursued:` entry whose text opens with the run's marker, naming every candidate with its score, the rule that placed the winner, the runner-up and why it lost, and a falsifier the state file can meet; no existing entry is changed; the entry is the lane branch's first commit; and `abcd intent ready` still reports the human's entry as the most recent conjecture.
- **Given** candidates whose scores tie, **when** the pick is made, **then** the oldest is taken and the entry says the tie was broken by age.
- **Given** a pick, **when** the lane starts, **then** it is the lane `abcd build <itd-N>` would start for that intent, with the same brief, worktree, validators and landing.
- **Given** the default run count, **when** the picked intent is delivered or its lane stops, **then** the run reports and exits without picking again; **given** `--max <n>` or `--until-empty`, **then** it picks again under the pace rule and writes each further reason the same way.
- **Given** a pick, **when** the lane's state file shows fix rounds past the pace rule's count or an unachievable hand-back, **then** the run record names the pick as falsified and the intent's entry is not edited.
- **Given** a spec without a `## Footprint` section, **when** the score is computed, **then** its test-path and footprint parts read zero and the reason says the spec carries no footprint.
- **Given** the command page, **when** it is read, **then** it states what `next` does, `--max` and `--until-empty`, and every refusal.
- **Given** `--json`, **when** any of the above renders, **then** the candidate set, scores, the reason and every refusal are in the payload.

## Typed Links

- **builds_on `itd-2609201916151817`** (the implement machinery, `abcd build <itd-N>`): the pick hands over to it and adds nothing to the lane.
- **builds_on `itd-2609201925079472`** (the pace): `--until-empty` and `--max` run under it.
- **refines `itd-78`** (the dependency graph): this intent filters on `blocked_by` and does not derive a priority; that draft is where a derived one would come from.
- **related `itd-82`** (`abcd drain`): the sibling run over the issue ledger; the two share `--max` and `--until-empty` and the hand-over to `abcd implement`, and differ in their default count by the meaning of their names.
- **builds_on `itd-50`** (the audit-driven fix round): the unachievable verdict is one of the pick's falsifiers.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
