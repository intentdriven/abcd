---
id: itd-2609201916151817
slug: one-verb-takes-a-single-intent-from-ready-to-delivered-witho
spec_id: spc-2609202134338445
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916056194, itd-2609091014076309, itd-2609091416295622]
supersedes: [itd-29, itd-58]
related_intents: [itd-29, itd-50, itd-2, itd-2609170822093401]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# One verb takes a READY intent to delivered, and refuses while a decision is open

## Press Release

> **`abcd build <itd-N>` runs one intent from READY to delivered with nobody in the loop, and refuses to start while a design decision is still open.** `build` is the verb a person types; `implement` is the machinery it hands over to, the loop an agent or a driving host calls. The verb is a loop over a state file, not a model holding the run in its head: every invocation reads the state, does one step, writes the state and exits, so a killed process loses nothing and a pause is a timestamp. A host session drives it step by step by default, and a process can drive the same loop through a configured runner. Two modes are explicit: `abcd intent plan` stays the human's act, and `implement --auto-plan` is the agent's, adversarial reviews included, ruled on in its own decision record before it ships.
>
> "The pilot did all of this by hand: a brief composed from three files, a receipt checked by eye, a range typed into the auditor's prompt, eight transcript captures one by one," said a technical facilitator who ran one intent through a coordinator, an implementer, three reviewers and an adjudicator. "Every one of those is a thing the binary already knows."

## Why This Matters

Four sessions in two managed repositories ran the whole lifecycle through sub-agents between 2026-09-17 and 2026-09-20, and one of them asked outright whether the pattern becomes an abcd recipe. The Dessau pilot then measured it: an orchestrator reading reports only reached 187k of context, a coordinator 188k, and a single implementer 349k after one fix round on a nine-commit intent, which is past the point where compaction loses the evidence the reports need. The product thinker there rebuilt the orchestrator as a loop script over a state file, launching one fresh process per lane, and listed the pieces abcd would have to own: the brief renderer, the receipt writer and verifier, the state file, the worktree creation, the spec close and the capture resolve after the receipt, the audit request's delivered range, and the capture of the run's own transcripts. adr-27 already rules that the autonomous run's loop is a thin native seam that gates on receipts; this intent is that seam with its pieces named.

## Mechanism

We expect a re-entrant loop over a state file, launching one fresh process per lane, to take an intent to delivered without a human because every step the pilot did by hand is a read of the record or a write the verbs already make, and the state the run needs between steps fits in a file rather than a context window; shown wrong if a lane needs a judgement the record does not hold and the loop cannot refuse on it, or if the receipts a lane writes cannot be verified without reading the lane's transcript.

## Scope Conditions

- Holds for a repository abcd manages, with a merge queue or branch protection on the default branch, and an intent in `planned/` whose gate reads READY. <!-- cond: cond-2609202134335223 -->
- Holds where the harness or a configured runner can start a fresh agent process for a lane and return; a harness that offers no way to launch one is a stop condition the run names. <!-- cond: cond-2609202134336678 -->
- Holds at the pilot's measured scale: one intent of nine commits and four packages per lane, with the lane's own context under 350k; a lane that would exceed it is split by the spec, not by the loop. <!-- cond: cond-2609202134334806 -->
- The model a lane ran is recorded as the runner reported it; the binary cannot verify it. <!-- cond: cond-2609202134336746 -->

## What's In Scope

- **The loop and its state file** under `.abcd/.work.local/run/`: lanes, status, branch heads, receipts, pull-request numbers, the window clock; every invocation advances it and exits.
- **The readiness and decision check** before anything starts: READY, no open question, no unanswered claim section, not held, and, from the peer listing, no peer holding the record. The held state (`iss-2609200830076665`) is a prerequisite this intent depends on; until it ships the check reads the hold as prose and says so.
- **The brief renderer**: Intent, spec, the conventions section of `AGENTS.md`, the decision lines the intent cites, into one file the implementer is handed.
- **The lane**: A worktree in abcd's form, one fresh implementer, the receipt (commits on the branch, the definition of done's output, the report file), verified by the loop before any validator runs.
- **Validators that never implemented**: The ruthless and security reviewers on the lane's diff, the fidelity auditor on the request the loop completes (it supplies the delivered range from the branch's base and head); findings applied by a fresh implementer or rejected in writing.
- **The landing**: `spec close` and `capture resolve` invoked by the loop after the receipt, the pull request opened with the repository's own merge rule, the branch cleaned after the merge is an ancestor of the default branch.
- **The run record and the transcripts**: One record per run under the local tier, and `history capture` of the run's own transcripts; the current-session capture (`iss-2609202046145653`) is a prerequisite, and until it ships the loop captures each transcript by path.
- **Refusals**: An unplanned intent, an open decision, a peer holding the record, a runner that cannot be started, a receipt that does not verify.

## What's Out of Scope

- The pace and the ceiling (`itd-2609201925079472`); the runner a lane goes through (`itd-2609201916056194`); the worktree store's own verbs (`itd-2609091014076309`); the peer listing (`itd-2609091416295622`); the claim and the register.
- The interview itself: A draft is planned by a human through `abcd intent plan`, or by `--auto-plan` once its decision record exists.

## Decisions

Settled on 2026-09-20 with one defensible answer each, on the product thinker's ruling that such a decision is recorded rather than asked (`iss-2609202055103741`):

1. **The orchestrator is the verb's own loop over a state file, never a model.** The pilot's context numbers and adr-27 leave one shape. Roles are implementer and validators only.
2. **Re-entrant, never blocking.** A multi-hour process launched from a harness tool dies at the tool's timeout; every invocation does one step and exits, and a pause is `next_eligible_at` in the state file.
3. **The merge target is the repository's own rule**: A pull request with auto-merge armed where the ruleset gates it through a queue, a pull request left open where it does not. The loop never pushes to a pull request after arming it.
4. **The binary-owned pieces the pilot listed are this intent's scope**, not seven records: They share the state file and are worthless apart.

Asked and answered on 2026-09-20:

5. **Both drivers, host-driven by default.** A host session drives the loop through a step interface (`abcd implement step` returns the brief; `abcd implement receipt <path>` advances the state), so its context stays bounded because the binary holds the state, and no boundary changes. A process driver, the same loop calling the CLI adapter (`itd-2609201916056194`) for each lane, is opt-in. That opt-in is a **reversal flag** on the host-delegated boundary in `AGENTS.md`, confirmed by the product thinker for the opt-in path only, and the ADR for `--auto-plan` records it beside the planning reversal.
6. **`--auto-plan`, explicit and autonomous.** The flag is named for what it does: it lets the planning path run without a human, the two adversarial reviews included, on a draft whose decisions are all recorded. `abcd intent plan` stays the human's act. The decision record states the substitute for the sign-off (recorded decisions, grounds present, the two review receipts, criteria provenance) and is minted before the path ships.
7. **`itd-29` is superseded by this intent.** Its budget check before a run and its checkpoint on a rate-limit response move into the pacing intent as criteria; its telemetry and hand verbs are dropped until someone wants them.

Ruled by the product thinker on 2026-09-21, in the interview that filed `itd-2609211116005482` (`abcd build next`) and revived `itd-82` (`abcd drain`):

8. **`build` for people, `implement` for the machinery.** `abcd build <itd-N>` is what a person types and is the verb this record's press release names; `abcd build next` and `abcd drain` are the two customised runs that hand over to the same loop. `abcd implement` is that loop, with the step-level words a driving host calls renamed from `next` to `step` (`implement step` returns the brief, `implement receipt <path>` advances) so that `next` is free to mean the pick. Which verbs the command list shows a person and which it shows an agent is its own record (`iss-2609211119023345`).
9. **Only the loop writes a verdict** (itd-58 folded in). A validator's verdict is recorded by the loop from the validator's own return, into the state file, before the advance is decided; a lane has no write to it, and a receipt carrying a verdict the loop did not record is refused at the advance, naming the receipt.

## Open Questions

_None open; decisions 5 to 7 settle the interview's questions._

## Acceptance Criteria

- **Given** an intent not in `planned/`, or READY with an open question, an unanswered claim section or a hold, **when** the verb runs without `--auto-plan`, **then** it refuses naming the check that failed and writes no state.
- **Given** a READY intent a peer holds, **when** the verb runs, **then** it refuses naming the peer, and writes no state.
- **Given** a READY intent nobody holds, **when** `abcd build <itd-N>` runs, **then** the state file exists with one lane, the worktree exists in abcd's form on a branch off the default branch, and the brief file names the intent, the spec and the conventions it was rendered from.
- **Given** a lane whose implementer has returned, **when** the loop verifies the receipt, **then** a receipt without commits on the branch, without the definition of done's output or without the report refuses and names what is missing, and the lane is not advanced.
- **Given** a verified receipt, **when** the validators run, **then** each runs as a fresh agent that did not implement, every finding is either applied by a fresh implementer or rejected in writing in the lane's report, and the fidelity request carries the delivered range from the branch's base to its head.
- **Given** validators passed, **when** the loop lands the lane, **then** the spec is closed in the landing change, every capture the lane fixed is resolved with its commit, the pull request exists with the repository's merge rule applied, and nothing is pushed to it afterwards.
- **Given** the loop's process is killed at any step, **when** the verb is invoked again, **then** it resumes from the state file at the step that did not complete, and repeats no completed step.
- **Given** a host with no configured runner, **when** `abcd implement step` returns a brief, **then** the host is told which agent to start and where the receipt goes, and the loop advances only on `abcd implement receipt`.
- **Given** a configured runner and the process driver opted in, **when** the loop reaches a lane, **then** it starts the lane through the runner itself, and the run record names the runner.
- **Given** a completed run, **when** the run record is read, **then** it names every lane, receipt, reviewer verdict, the model each runner reported, and the transcripts captured into the history store.
- **Given** a validator has returned, **when** the loop records its verdict, **then** the verdict in the state file is the one the loop parsed from the validator's return, and a lane report carrying a verdict the loop did not record is refused at the advance, naming the report; a real SHIP recorded by the loop lets the advance proceed.
- **Given** any refusal, **when** it is rendered, **then** it names the step, the reason and the remedy, in text and in `--json`.

## Typed Links

- **builds_on `itd-2609201916056194`** (the CLI adapter), **`itd-2609091014076309`** (the worktree store) and **`itd-2609091416295622`** (the peer listing): The runner, the worktree and the peer read this loop calls.
- **refines adr-27** (the autonomous run's pluggable seam): This is the thin native loop it names, with its pieces.
- **refines `itd-50`** (the audit loop to acceptance): The verb's validator stage is where that loop would run.
- **refines `itd-2609170822093401`** (the model tier): Decides which runner a role gets.
- **supersedes `itd-29`** (the autonomous spec run): Its run verbs re-land here on the intent key; its budget check and rate-limit checkpoint live in the pacing intent.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect a re-entrant loop over a state file, launching one fresh process per lane, to take an intent to delivered without a human, because every step the pilot did by hand is a read of the record or a write the verbs already make, and the state the run needs between steps fits in a file rather than a context window; shown wrong if a lane needs a judgement the record does not hold and the loop cannot refuse on it
