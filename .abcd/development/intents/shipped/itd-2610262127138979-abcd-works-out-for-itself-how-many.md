---
id: itd-2610262127138979
slug: abcd-works-out-for-itself-how-many
spec_id: spc-2610262202438999
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609231434459890, itd-2609201925079472]
severity: minor
impact: fix
origin: researcher-authored
production_mode: hand-written
---

# A preflight waits for its turn as one whole run

## Press Release

> When several sessions on one machine push work at once, each preflight waits for its turn as one whole run (a records-only change takes a shorter preflight that waits at no gate). A queued run does nothing until the gate admits it: no lint, no build, no test. So the runs waiting never add to the load the running one is judged by. The number allowed at once is read again at every look, so a change to it applies to the runs already queued at their next look, and the wait's report names the value it judged by and where that value came from (always in its JSON form, and on the line saying the gate opened when more than one run is allowed at once). Two runs that reach the gate together never wait on each other until they time out.

## Why This Matters

On a shared machine, how preflights queue decides how fast everything lands. In the autonomous run of 24 to 26 October 2026 on a 10-core laptop, only a preflight's test stages waited at the gate: its lint gates, its two evaluation lanes, its build and its vet ran at once, for every queued run, so thirty queued runs kept the machine overloaded while they waited, and the admitted run's tests timed out. Sessions that wrapped a whole preflight in the gate's command form ran one at a time and passed at the first attempt, so the work-around showed the fix. A person who changed the number allowed at once saw nothing change for the runs already queued, because each run read the setting once, when it began to wait.

The first of three pieces split from one draft at the product thinker's ruling of 26 October 2026: this piece makes the whole run wait; itd-2610262140595217 records how each run behaved; itd-2610262141001890 adjusts the number allowed at once from that record. The fast lane for short targeted test runs stays its own record, iss-2610260845422800, because it needs its own decision about what a run may claim about itself.

It resolves iss-2610241537197997 (concurrent preflights fail; the remedy is this piece) and iss-2610241214065079 (stages that never wait), and the parts of iss-2610241414124940 (every heavy stage behind one admission; the setting read again while waiting) and iss-2610241911052191 (the admitted run overloaded by others' unmetered stages) that admission decides. What those two records also ask moves elsewhere: releasing a slot whose holder is idle to iss-2610262155232329, and admitting on the headroom a run will need to itd-2610262141001890.

## Mechanism

We expect fewer test timeouts and faster landings, because runs that wait no longer load the machine the running one needs; the sessions that already waited as a whole run passed at the first attempt. The claim is wrong if timeouts stay as frequent once queued runs sit idle.

## Scope Conditions

- macOS and Linux machines. <!-- cond: cond-2610262202434440 -->
- Sessions running an abcd that carries this change; a session on an older abcd keeps today's queue and its unmetered stages. <!-- cond: cond-2610262202437482 -->
- One machine shared by several sessions, whichever account each runs in. <!-- cond: cond-2610262202435764 -->
- The number allowed at once is a whole number from 1 to 4, and 1 by default (product thinker, 26 October 2026; the bound was raised from 2 to 4 on 28 October 2026). <!-- cond: cond-2610262202435155 -->
- Not on CI runners, which never wait. <!-- cond: cond-2610262202436858 -->

## Acceptance Criteria

- Given one run allowed at once and three full preflights started within a minute, when they run, then only the admitted one does any work (lint, build, test or any other check), and the other two wait doing nothing.
- Given a run waiting, when the number allowed at once changes, then the waiting run judges by the new number at its next look, and its report names the value and where it came from (always in its JSON form, and on the opening line when more than one run is allowed at once).
- Given two runs from different accounts reaching the gate at the same moment, when both wait, then one is admitted within a minute and neither waits on the other until it times out.
- Given a repository whose own settings name a number of runs for the gate, when any run waits, then that number is ignored.

## Decisions

- Split from one draft into three pieces; this is the first (product thinker, 26 October 2026).
- The number allowed at once is adjusted automatically, never by hand. A person may name a starting default, and abcd uses even that only once the machine's capacity has been measured (product thinker, 26 October 2026). The adjustment is itd-2610262141001890's promise; this piece only makes a change to the number reach the runs already waiting. The ruling reverses the standing rule that the person writes the number, as the LOAD rules and dcn-2610161814279204 state it, and is carried to that piece's planning.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-a712216d677f -->
Fidelity review OWED (receipt rcp-a712216d677f).
<!-- abcd-review-end receipt=rcp-a712216d677f -->

## Grounds

- pursued: it clears two of the serious problems that block the next release, and it stops queued runs overloading the machine, which held up the whole autonomous run; it is shown wrong if the release is still blocked by them, or runs still time out while others wait
