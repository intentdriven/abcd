---
id: itd-2610222215485265
slug: abcd-drain-run-from-a-plain-terminal
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610222000271964]
related_intents: [itd-2610161621364075, itd-2610170634394993, itd-2610100704270254]
refines: [itd-2610161016401115]
origin: extracted-from-record
production_mode: hand-written
---

# Typing abcd drain in a plain terminal works through the ledger on its own

## Press Release

> **Type `abcd drain` in a plain terminal and it works through the ledger on its own.** Frank opens a terminal on a spare machine and types `abcd drain`. abcd asks him once whether it should start Claude sessions itself to judge each fix and work each lane, using his model budget. He says yes. The terminal then shows one line per step (an issue judged, a lane opened, its review passed, its pull request queued) and moves on to the next. It stops at its cap, when nothing eligible is left, or when Frank presses Ctrl-C, and it hands back cleanly whatever it was holding. Frank never answers anything by hand, never keeps a second agent session open, and never runs the command again and again.
>
> _Facilitator-proposed on 2026-10-22 from iss-2610222000271964, for the product thinker's sign-off._

## Why This Matters

On 2026-10-22, after the crash, the product thinker typed `abcd drain` in a plain terminal on a second machine. It reserved the first of 213 eligible issues, wrote a judgement request and waited for an agent session that never came. It opened no lanes and left the issue reserved for two hours (iss-2610222000271964).

The shipped drain bundle (itd-2610161016401115) promised a drain that "runs unattended on a hosted session or a plain terminal for a day". It delivered the pieces: a judge routable to a runner, and lane roles that `implement step` starts through the runner. But nothing loops. Each `abcd drain` is one move, by design, for a host that calls it again, and a plain terminal has no such host. This intent closes that gap, so it `refines` the shipped one rather than reopening it.

## Mechanism

_Facilitator-proposed._

We expect a terminal drain to work through issues with no person and no host session. Every step that today waits for a host is either a call abcd already knows how to make itself (the routed judge, the runner-started roles) or a wait that resumes by itself, so the driver only has to repeat what a host repeats. This is shown wrong if a terminal drain, started on a machine with the runner set up and left for eight hours, opens fewer lanes than the same drain driven by a host session in the same time, for a reason other than its cap.

## Scope Conditions

_Facilitator-proposed._

- Holds on a machine where the claude runner is set up and admitted (`runner.claude`), within the model budget abcd's sub-agent setting allows.
- Holds for one drain per working copy, which the drain's lock already enforces.
- Holds while the terminal stays open, or under a tool the person chooses to keep it open (`tmux`, `nohup`); abcd starts no background service.
- Once the sandbox (itd-2610161621364075) ships, it holds only where its fence check passes; once the coordination intent (itd-2610170634394993) ships, only where its login check allows the run (that intent's open choice 1).

## Acceptance Criteria

_Facilitator-proposed on 2026-10-22, for the product thinker's walk; none confirmed._

- **Given** Frank types `abcd drain` in a plain terminal with no host session, **when** the drain would need a judgement or a lane step, **then** abcd asks him once whether to start Claude sessions itself, naming that they use his model budget, and starts none without his yes; `--drive` gives the yes in advance for a scripted start, and a non-interactive start without it does what it does today, one move and exit.
- **Given** Frank said yes, **when** the drain runs, **then** it judges each eligible issue through the runner, opens its lane, steps the lane until it lands or is handed back, and moves to the next, with no further input, printing one line per step and logging the detail under the local tier.
- **Given** a terminal drain, **when** it reaches its cap, finds nothing eligible, or Frank presses Ctrl-C, **then** it stops, releases every reservation it holds that no open lane needs, and says what it did; a second Ctrl-C stops it at once, as today.
- **Given** a judgement or a lane step whose runner fails, times out or answers out of shape, **when** the driver meets it, **then** that issue is handed back with the reason and the drain moves on; it never waits for a person to answer by hand.
- **Given** the machine's load gate or the sub-agent budget is full, **when** the driver would start a role, **then** it waits under the existing deadline and pacing and says so, never starting beyond them.
- **Given** a host session is driving the drain, **when** `abcd drain` runs, **then** it behaves exactly as today, one move per call.
- **Given** the routed roles, **when** the terminal drain starts them, **then** each launch goes through the runner's one launch point, so the guard approver, and once shipped the sandbox's fence, apply as they do for `abcd build`.

## Open Questions

_None for the product thinker; each technical point is recorded as a decision below. The criteria are walked at the sign-off._

## Decisions

Each is the facilitator's, on the product thinker's delegation of 2026-10-14 and 2026-10-16 for technical choices, recorded with `abcd decide` on 2026-10-22:

1. One intent, refining the shipped drain bundle (itd-2610161016401115) rather than reopening it, and FILE-AS-IS through the decomposition table: the fence, the routed judge's isolation and consent are existing rules, cited.
2. The driver is a loop inside `abcd drain`, not a new verb and not a background service.
3. A host is detected from the environment Claude Code sets for its shell (`CLAUDECODE=1`), and `--drive` and `--one-move` override the detection.
4. Consent to start agents is asked once per drain at its first launch and never remembered across drains; `--drive` is the scripted yes.
5. While driving, the judge and every lane role default to the claude runner wherever the person's config routes no role; a role routed elsewhere keeps its route.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
