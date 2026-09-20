---
id: itd-2609201925079472
slug: an-autonomous-implementation-run-paces-itself-by-default-and
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916151817]
related_intents: [itd-29]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# An autonomous implementation run paces itself by default, and the operator can set the pace and the agent ceiling in two flags. abcd implement <itd-N> ships with a pacing default the run follows without being told: a working window, a pause after it, and a ceiling on sub-agents alive at once, each a number the run reads from one place and reports at the start. Two flags override the defaults for one run: --pace <work-minutes>/<pause-minutes> (a working window and the pause that follows it, both in minutes) and --sub-agents <n> (the most sub-agents alive at once, reviewers counted). The default is the measured pattern the pilot runs established, one hundred and twenty minutes of work, three hundred minutes of pause, two sub-agents, and it lives in the repo's abcd configuration so a managed repository can declare its own; a flag wins over the config, the config over the bundled default, and the run record names which one applied. At the window's end the orchestrator spawns nothing new, lets a running lane finish its step and checkpoint to its branch, writes the handover, and sleeps through the pause in wake-ups the harness clamps at one hour; a run that would exceed the ceiling waits rather than spawning. Today every autonomous run carries the pace in its prompt as prose, and two runs on one machine can neither share it nor be sure the other honours it.

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Typed Links

Refines itd-2609201916151817 (the implement verb: this is the pace and the
ceiling that verb runs under) and itd-29 (the autonomous spec run's budget
check and resume; the pause here is the resume point that verb names).
Filed on the product thinker's word on 2026-09-20, after the two run files
carried the pace as prose.

## Why This Matters

An autonomous implementation run paces itself by default, and the operator can set the pace and the agent ceiling in two flags. abcd implement <itd-N> ships with a pacing default the run follows without being told: a working window, a pause after it, and a ceiling on sub-agents alive at once, each a number the run reads from one place and reports at the start. Two flags override the defaults for one run: --pace <work-minutes>/<pause-minutes> (a working window and the pause that follows it, both in minutes) and --sub-agents <n> (the most sub-agents alive at once, reviewers counted). The default is the measured pattern the pilot runs established, one hundred and twenty minutes of work, three hundred minutes of pause, two sub-agents, and it lives in the repo's abcd configuration so a managed repository can declare its own; a flag wins over the config, the config over the bundled default, and the run record names which one applied. At the window's end the orchestrator spawns nothing new, lets a running lane finish its step and checkpoint to its branch, writes the handover, and sleeps through the pause in wake-ups the harness clamps at one hour; a run that would exceed the ceiling waits rather than spawning. Today every autonomous run carries the pace in its prompt as prose, and two runs on one machine can neither share it nor be sure the other honours it.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
