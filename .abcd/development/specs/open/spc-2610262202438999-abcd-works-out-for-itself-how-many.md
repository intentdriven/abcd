---
id: spc-2610262202438999
slug: abcd-works-out-for-itself-how-many
intent: itd-2610262127138979
origin: researcher-authored
production_mode: hand-written
---
# A preflight waits for its turn as one whole run

## Summary

Delivers itd-2610262127138979: `make preflight` waits at the load gate once, for the whole run, before any of its stages, so a queued preflight does no lint, eval, build, vet or test work; the number allowed at once is read again at every look of a wait; and two runs reaching the gate together are ordered without waiting on each other until timeout.

Grounding (the two pre-interview reviews of 2026-10-26): today the lint gates, both eval lanes, build and vet are the recipe's prerequisites and run before the first gate (`Makefile` preflight target), and the first gate and the race gate are the plain form, which exits as soon as it opens, so nothing before the tests waits. `WaitLoad` reads the slot count once, before its polling loop (`internal/core/implement/load_wait.go`). The command form of the gate (`implement load --wait -- <cmd>`) already holds the turn for the whole command, and sessions that wrapped a whole preflight in it ran one at a time and passed at the first attempt.

### Approach

1. **Whole-run admission.** The `preflight` target becomes a thin entry that runs the existing recipe through the gate's command form, so the waiter is the parent of the whole run and its turn holds until the run ends. The recipe's own first gate goes, because a second waiter inside a held turn can deadlock against a run queued meanwhile (traced in the review: each shut by the other until the two-hour timeout). The race gate stays inside the run for slot counts above one, judged as today, and must not count its own parent run as another preflight. The command-form `turnWon` path in `machineload/queue.go` decides between two runs arriving together.
2. **The count read at every look.** `PreflightSlots` is read inside `WaitLoad`'s loop, not once before it, and the race-gate shortcut that depends on it reads the same value at the same look. `cmd/test-parallelism` and `GoTestParallelism` divide the cores by that same value, through one reader, so the gate and `-p` never disagree. A value that becomes unusable mid-wait refuses at the next look, naming the file. The report names the value and its source (config file or environment).
3. **What is taught.** The LOAD rules (`internal/core/rules/defaults/rules.json`), the conventions block abcd installs (`internal/core/ahoy/defaults/`), the Makefile's comments and AGENTS.md say that a preflight waits as one whole run, and that the gate's command form is no longer needed around `make preflight`.

Out of scope, recorded elsewhere: releasing a slot whose holder is idle (iss-2610262155232329); admitting on projected headroom and changing the count automatically (itd-2610262141001890, after the run record of itd-2610262140595217); the short-run lane (iss-2610260845422800).

### How each criterion is met

- **Three runs, one admitted, the others idle.** Approach 1. Test: a Makefile-structure test asserts no stage runs before the gate's command form in the `preflight` target, and a gate test with a fake process table of three `make preflight` waiters asserts exactly one is admitted.
- **A change to the number reaches waiting runs.** Approach 2. Test: a `WaitLoad` test changes the configured count between looks and asserts the next look judges by it and its report names value and source; a second changes it to an unusable value and asserts the refusal names the file.
- **Two runs at one moment.** Approach 1's `turnWon` path. Test: two waiters of different accounts with equal start times; one opens within the jitter-plus-stand-back bound and neither reaches its timeout.
- **A repository's number is ignored.** Existing refusal kept and covered by a test that sets a load key in a repository's `.abcd/config.json` and asserts the count is unchanged.

## Footprint

- packages: internal/core/implement/load.go, internal/core/implement/load_wait.go, internal/core/implement/load_slots.go, Makefile, internal/core/machineload, cmd/test-parallelism, internal/core/rules, internal/core/ahoy
- tests: internal/core/implement/load_wait_test.go, internal/core/implement/load_slots_test.go, internal/core/machineload, cmd/test-parallelism/main_test.go

## Steps

_No steps listed: the spec is built as one step. To split it, list the steps in order as `1. <title>`, each with `- packages:` and `- tests:` indented beneath it; `- landed: <pull request or commit>` marks a step that has landed._
