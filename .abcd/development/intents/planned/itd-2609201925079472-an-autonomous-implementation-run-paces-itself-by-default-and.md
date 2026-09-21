---
id: itd-2609201925079472
slug: an-autonomous-implementation-run-paces-itself-by-default-and
spec_id: spc-2609202134341288
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916151817]
related_intents: [itd-29]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# An autonomous run paces itself by default, and one flag sets the pace

## Press Release

> **A run started with `abcd build` (or `abcd drain`, or the `abcd implement` loop they hand over to) follows a pace it was never told: a working window, a pause after it, and a ceiling on lanes alive at once.** The three numbers come from one place, layered: a flag on this run wins over the repository's abcd configuration, which wins over the machine's, which wins over the bundled default; the run record names which layer applied. `--pace <work-minutes>/<pause-minutes>` and `--sub-agents <n>` set them for one run. At the window's end the loop starts no new lane, lets a running lane finish its step and checkpoint to its branch, writes `next_eligible_at` into the state file and exits; an invocation before that time refuses and names it, so the pause survives a killed process and needs no sleeping model. Today every autonomous run carries its pace in its prompt as prose, and two runs on one machine cannot share it.
>
> "Two pilots, two prompts, two paces typed by hand, and no way to know the other one was keeping to its ceiling," said a technical facilitator who ran both. "Now the pace is a line in the config and a timestamp in the state file, and a run that starts early is refused."

## Why This Matters

Two pilots on one machine in the week of 2026-09-15 ran on two paces written into their prompts by hand, two hours of work then five off with two lanes here, two hours then four off with three lanes there, and neither could be sure the other honoured its ceiling. The pace exists because the model budget resets on a window; a run that ignores it runs into the reset mid-lane and loses the lane. 

## Mechanism

We expect a pace read from layered configuration and enforced by a timestamp in the state file to be honoured by every run without being told, because the loop that reads the state is the only thing that starts a lane; shown wrong if a run starts a lane inside a pause or above the ceiling, which the run record would show, or if the budget window the pause exists for moves in a way minutes cannot express.

## Scope Conditions

- Holds for a run driven by the implement verb's loop; a session pacing itself by prompt is outside it. <!-- cond: cond-2609202134346930 -->
- Holds per run: The ceiling counts this run's lanes and validators. <!-- cond: cond-2609202134341920 -->
- The bundled default is a choice, not a measurement: The two pilots ran 120/300 with two lanes and 120/240 with three, and a repository overrides it in its configuration. <!-- cond: cond-2609202134341625 -->

## What's In Scope

- The three numbers, their four layers, the flags, the window clock and `next_eligible_at` in the state file, the ceiling on this run's lanes and validators, the budget check before a run starts, and the checkpoint on a rate-limit response.

## What's Out of Scope

- A ceiling across runs on one machine: The register's (`itd-2609150819440345`), where a lane registry can live.
- Mid-run telemetry and an operator's hand verbs over the state file: Dropped with `itd-29` until wanted.

## Decisions

Settled on 2026-09-20 with one defensible answer each, on the product thinker's ruling that such a decision is recorded rather than asked (`iss-2609202055103741`):

1. **Re-entrant pause.** The pause is `next_eligible_at` in the state file; an invocation before it refuses and names the time. No process sleeps.
2. **Units are minutes**, as the product thinker specified the flag; a quota-window signal, where a harness exposes one, is a later refinement and is recorded as such.
3. **Cross-run enforcement is the register's.** This intent bounds one run; the design review found a repository file cannot bound two runs across repositories.
4. **Layering is flag, then repository, then machine, then bundled**, the order the model-tier intent already uses.
5. **The bundled default is 120 minutes of work, 300 of pause, two lanes**, the product thinker's numbers for this repository's runs on 2026-09-20; a repository that measured otherwise writes its own.
6. **The budget check and the rate-limit checkpoint come here from `itd-29`**, superseded on 2026-09-20 by the implement verb: A run refuses to start when the estimated cost exceeds the remaining quota where the runner reports one, and a rate-limit response checkpoints the lane and ends the window early.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** no flag and no configuration, **when** a run starts, **then** it runs on 120/300 with two lanes and the run record names the bundled layer.
- **Given** a repository configuration and a machine configuration that disagree, **when** a run starts without a flag, **then** the repository's values apply and the record says so.
- **Given** `--pace 90/240 --sub-agents 3`, **when** a run starts, **then** those values apply over every configured layer and the record names the flag.
- **Given** a window that has elapsed, **when** the loop is invoked, **then** it starts no lane, lets a running lane finish its current step and checkpoint to its branch, writes `next_eligible_at`, and exits 0 naming the time.
- **Given** `next_eligible_at` in the future, **when** the loop is invoked, **then** it refuses naming the time and changes no state.
- **Given** the ceiling reached, **when** the loop would start a lane or a validator, **then** it starts nothing, names the lanes alive, and exits; the next invocation fills the slot, and the record counts the minutes the slot was waited for.
- **Given** a runner that reports remaining quota and an estimate that exceeds it, **when** a run starts, **then** it refuses naming both numbers and writes no state; a runner that reports no quota is named and the check is skipped out loud.
- **Given** a rate-limit response from a runner mid-lane, **when** the loop reads it, **then** the lane is checkpointed to its branch, the window ends early with `next_eligible_at` set, and the record names the response.
- **Given** a malformed pace or ceiling, **when** a run starts, **then** it refuses naming the value and the accepted form, and writes no state.

## Typed Links

- **builds_on `itd-2609201916151817`** (the implement verb): The loop this pace bounds.
- **refines `itd-2609170822093401`** (the model tier): The same configuration layering.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect a pace read from layered configuration and enforced by a timestamp in the state file to be honoured by every run without being told, because the loop that reads the state is the only thing that starts a lane; shown wrong if a run starts a lane inside a pause or above the ceiling, which the run record would show
