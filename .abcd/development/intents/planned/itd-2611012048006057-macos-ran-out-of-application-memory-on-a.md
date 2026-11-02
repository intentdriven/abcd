---
id: itd-2611012048006057
slug: macos-ran-out-of-application-memory-on-a
spec_id: spc-2611020732419632
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2611011737317454]
origin: extracted-from-record
production_mode: hand-written
---

# abcd's load gate waits for memory as it waits for load

## Press Release

> A preflight never starts on a machine that is out of memory. When several agent sessions build, review and test on one machine at once, abcd's load gate already waits for the processor load to fall and refuses when the disk is nearly full; now it also reads how much memory pressure the machine is under. When the system reports memory pressure at or above the floor the person set (macOS's own pressure level, or on Linux the share of recent time tasks stalled waiting for memory), a preflight or a gated test run waits at the gate, bounded as the load wait is, instead of starting a test stage that would push the machine past its memory and stall every session on it. The wait says what it is waiting for, and a reading abcd cannot take lets the work through and says so, so the floor never wedges a run.

## Why This Matters

Graduated from `iss-2611011737317454`. On 2026-11-01, at about 17:32, a 128 GB, 16-core development machine ran out of application memory while several agent sessions drained, built and reviewed in parallel, with only this person's sessions active. The machine's counters showed about 155 GiB written to swap over its five days of uptime, so it had run past its memory before. abcd's load gate reads the processor load, the process table and free disk, but never memory, so nothing in abcd held a test stage back while memory was short. The person ruled on 2026-11-02 that the gate waits rather than refuses below the floor, that the floor is set by macOS's own memory-pressure level, that Linux is guarded from the start through the kernel's pressure stall information, that waiters admitted after the pressure falls start one at a time with a settle period between them, and that attributing a spike to a process belongs with the intent that records how each preflight behaved (itd-2610262140595217).

## Mechanism

We expect a gate that waits while the system reports memory pressure at or above its floor (macOS's warning level, or Linux's stalled share) to stop a new test stage from starting into a memory shortage, because the pressure level is the signal the operating system itself acts on before it runs out, and the stages that use the most memory (race-enabled test binaries, several at once) are the ones the gate already admits one by one; waiting costs a stage minutes where starting it costs every session on the machine.

## Scope Conditions

- The person's own macOS or Linux machine, where the system reports memory pressure (macOS's pressure level; on Linux, kernel 4.20 or later, `/proc/pressure/memory`); on any other platform, or where the reading fails, the floor is reported as not checked and the gate admits as it does today. <!-- cond: cond-2611020732414692 -->
- Runs that pass abcd's load gate: `make preflight` and the command form `abcd implement load --wait -- <command>`; a test an agent runs outside the gate is not held, as iss-2610241130157455 settled. <!-- cond: cond-2611020732413735 -->
- One person's machine shared by that person's sessions and accounts; the floor is a machine setting in `~/.abcd.noindex/config.json`, never read from a repository file. <!-- cond: cond-2611020732419293 -->

## Acceptance Criteria

- Given the system reports memory pressure at or above the floor when a preflight reaches the load gate, when the gate decides, then it waits rather than admitting the test stage, re-reads the pressure before it opens, and names the pressure level and the floor in its wait line.
- Given the pressure falls below the floor while the gate waits, when the gate next checks, then it admits the stage as it would with no floor.
- Given the pressure stays at or above the floor until the gate's own time limit, when the limit passes, then the gate ends as its load wait ends today, naming memory pressure as what it waited for.
- Given the pressure level cannot be read (another platform, or the read fails), when the gate decides, then it admits the stage and says once that the memory floor was not checked, and why.
- Given no floor is set in the person's machine configuration, when the gate decides, then it uses abcd's default floor (macOS's warning level; on Linux, 10% of time stalled over the last 10 seconds), and given a floor set in a repository's `.abcd/config.json`, then that setting is ignored and named as ignored.
- Given `abcd implement load` without `--wait` (the load check), when memory pressure is at or above the floor, then the check reports it as a warning and never fails.
- Given several runs wait at the gate for memory pressure to fall, when it falls, then they are admitted one at a time in queue order, and after each admission the gate waits a settle period before reading memory again for the next waiter, so a stage's own memory use shows before another starts.
- Given a Linux machine whose `/proc/pressure/memory` reports a stalled share at or above the floor, when a preflight reaches the gate, then it waits as on macOS, and given a kernel without that file, then the floor is reported as not checked and the stage is admitted.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: no test stage starts while the system reports memory pressure at or above its floor, so a machine running several agent sessions no longer runs out of memory; shown wrong if the machine runs out of application memory again with the floor in force and every stage admitted through the gate.
