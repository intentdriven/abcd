---
id: spc-2611050023402644
slug: the-ledger-reaches-the-status-line-and-a
intent: itd-2611031342011261
origin: researcher-authored
production_mode: hand-written
---
# The launch ledger reaches the status line and a run's own summary

## Summary

_Filed by the remainder close of spc-2611041701500441, which delivered the record and `abcd agents`. Not READY: the first step below needs a ruling before it is built._

Delivers what is left of itd-2611031342011261: the figures the ledger holds reach the two places a person meets them without asking. Criterion 7 and criterion 9 (the status line), criterion 8's alternative basis and criterion 12's warning are step 1; criterion 11 (a run's summary) is step 2. The intent ships when this spec closes.

**Step 1: what the row has to live within.**
- The status verb has 500 ms for all of its work, and `runner.ReadLedger` is no bounded pass: it opens every file in the store, up to the retention's 1,000,000, and reading the process table to judge a running entry starts `/bin/ps` on macOS, where the composition starts no subprocess at all. A reader for the row therefore takes one session's figures in a bounded pass, and says it stopped short rather than showing a partial sum as a whole one.
- The status verb is read-only by rule (`internal/core/statusline/readonly_guard_test.go`): one write is admitted, the session's fill file, on the product thinker's ruling. Decision 15's cached read, 30 seconds across refreshes, is a second write on that path, so it needs a ruling of its own. **Open:** admit one cache file beside the fill file, or have each launch's writer keep a per-session rollup the verb only reads, which adds no write to the status path but makes the rollup a second record of what the entries say. This is the question that keeps the spec from READY.
- The session comes from the harness's payload, `session_id`, not the environment, and a row with no session id shows no agents' element.
- The elements, their words and their dropping order are the sibling draft's (itd-2611031342000215), which this step supplies figures to and does not pre-empt. If that draft is not planned when this is built, the step adds one element of its own, which that draft then takes over.
- The explanation does not exist yet: `abcd statusline --json` gives the row's ordered elements and no source, basis or threshold. Decision 5 asks for a machine-readable form of the status line's state, in which each ledger figure names the ledger as its source, the token basis it counts and the threshold it warns at, and which counts two launches for one await as a warning.
- The settings are the token basis (every count by default, prompt and completion as the alternative, decision 16), showing cache reads apart, off by default, the cache interval (decision 15) and the day, local by default and UTC as the alternative (decision 20). They join `ledger.*` in `internal/core/runner/config.go`, whose claim refuses an unknown key, and the verb reads them without loading the runner's whole configuration on every refresh.
- Spend against a key's budget and its burn rate have no key to show until itd-2611011230226453 hands one: nothing sets the key route today. The row shows them only once that intent ships, and the step tests them over entries laid by hand.

**Step 2: what a run's summary has to read.**
- A drain is no one run in the ledger. Each lane has its own run id, in the drain's state, and the judge's launches name `drain-<started>`. `LedgerQuery` takes one run, so a drain's totals need a query over several, which the drain's state lists. `--run` on the verb becomes repeatable for the same reason, so the line the summary prints lists exactly what it summed; the store gains no field, and the run state still names nothing of the ledger (criterion 14).
- `renderDrainCounts` already reports a spend in agent runs, from the run states. It stays, as what the lanes' runs say, and the ledger's totals sit beside it: launches, failed or stopped, key spend, tokens and API time, with the cost note and the token basis any render of them owes.
- Each `abcd drain` makes one move and exits, so the totals belong on the move that reports the drain ended or stopped. The terminal drive (itd-2610222215485265, on its own branch) and the build drive (iss-2611031146255825, not built) have no summary on the default branch yet. Whichever lands first carries its half here, and a drive this spec cannot reach when it closes stays a stated remainder rather than an untested claim.
- A read here is not on the status path, so it reads the whole store; a hosted runner's home is as short-lived as the runner, so this summary is the account that survives there (the third scope condition).

**Out of scope.** The budget's own check against a cap, a session's by-hand launches (decision 19) and the row's elements beyond the ledger's figures.

## Footprint

- packages: internal/core/statusline, internal/core/runner, internal/core/implement/loop, internal/surface/cli, commands, docs/reference/cli, .abcd/development/release, .abcd/development/brief/04-surfaces, .abcd/development/brief/05-internals
- tests: internal/core/statusline (the row and its read-only guard), internal/core/runner (a bounded read for one session, a query over several runs), internal/core/implement/loop (a drain's totals across its lanes and its judge), internal/surface/cli (agents_surface_test.go, the statusline verb and the drain's render)

## Steps

1. AFTER: the status line's row, its cached read and its explanation, with the token basis as a setting
   - packages: internal/core/statusline, internal/surface/cli
   - tests: the row from the ledger alone, within the status command's deadline, cached for the configured interval
   - needs: none
2. AFTER: an unattended run's summary carries the ledger's totals
   - packages: internal/core/implement/loop, internal/surface/cli
   - tests: a drain's and a driven build's summary naming their totals and the verb
   - needs: none
