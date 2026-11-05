---
id: spc-2611041701500441
slug: every-agent-abcd-launches-for-a-session
intent: itd-2611031342011261
origin: researcher-authored
production_mode: hand-written
---
# abcd keeps a per-session ledger of every agent it launches

## Summary

Delivers the record and the rollup of itd-2611031342011261: one entry per launch abcd starts, written under abcd's home before the harness starts and completed when it ends, and `abcd agents`, which lists and totals them. Planned by an agent under dcn-2611031615094092, since every open question of the draft has one answer on the record (decisions 14 to 21, dcn-2611041550129278). The first release ships the steps marked MUST; the steps marked AFTER stay on this spec, and the intent stays planned until they land (a close with `--remainder` carries them).

**Approach.** One function, `runner.RecordLaunch`, is the only way an entry is written. It writes the entry marked running, runs the launch it is handed, and completes the entry from what the launch returned; handed no launch, it writes the host's placeholder (decision 21). The dispatcher's `runOn` wraps every adapter run in it, and its two hand-offs to the host call it bare, so the build, the drain, the judge and the planning interviewer are all covered with no change to their launch code. A launcher outside the dispatcher (the unattended engine) calls the same function. The store is `~/.abcd.noindex/agents/`, one file per launch holding one line of JSON, so concatenating the folder gives one line per launch and two launches never write one file. A file is replaced by a rename when its launch ends, and no reader writes.

**Decisions this spec adds.**
- An entry that cannot be written refuses the launch, naming the store: the mechanism's claim is that no launch escapes the record, and a key's cap will be summed from it. A completion that cannot be written leaves the answer standing and the entry reading as lost, which is the record's own visible sign.
- The ledger hangs off the runner configuration, which every dispatcher already holds and which `runner.Load` builds with the home it read, so no launcher can forget to name it. A configuration built with no home refuses the launch.
- The billing route and the API time are read from the launch's transcript, not added to `Answer` or `Sides`, because `Sides` is written into the run state and criterion 14 keeps that unchanged. The harness's process id reaches the entry through the context, so no adapter signature changes.
- A launch that failed on the claude runner carries what its stream shows, as a killed one does: a failed launch spent too.
- The floor is 32 days, the longest calendar month plus one day (decision 14), so the shipped 30-day default reads as 32 until the person sets more; the setting is `ledger.retention_days` and `ledger.retention_count`, in the machine configuration, and a repository's figure stands only where it lengthens what the machine keeps (dcn-2611042248450089).

**How each criterion is met.**
- Criterion 1 (entry at start and end), criterion 3 (the store), criterion 6 (killed and lost) and criterion 14 (run state unchanged): steps 1 and 2.
- Criterion 2 (the orchestrator): step 3.
- Criterion 5 (the billing route): step 2. The key route's field exists, and nothing sets it until the budget intent hands a key.
- Criterion 13 (retention): step 4.
- Criterion 4 (the lenient reader), criterion 10 (the verb), criterion 12 (two launches for one await) and criterion 8's default basis: step 5.
- Criterion 7 and criterion 9 (the status line), and criterion 8's alternative basis: step 6.
- Criterion 11 (a run's summary): step 7.

**Out of scope.** A session's by-hand launches (decision 19), the budget's own check against a cap (itd-2611011230226453, which sums from this ledger), and any new fetch of a key or an account's figures: a cost is what the harness reports, and no credential is read.

## Footprint

- packages: internal/core/runner, internal/core/sessionname, internal/core/implement/loop, internal/core/interview, internal/surface/cli, internal/core/surface, commands, docs/reference/cli, .abcd/development/release, .abcd/development/release-gate, .abcd/development/brief/04-surfaces, .abcd/development/brief/05-internals, .abcd/rules.json
- tests: internal/core/runner (ledger_test.go, the dispatcher's launch tests), internal/core/sessionname (lookup_test.go), internal/surface/cli (agents_surface_test.go and the surface audits)

## Steps

1. MUST: the entry and the one function that writes it
   - packages: internal/core/runner
   - tests: an entry written before the launch runs and completed after, one file holding one line, the placeholder for a host-handed role, a launch refused when the entry cannot be written, a kept answer when the completion cannot
   - needs: none
   - landed: 7a02f0236
2. MUST: every dispatched launch is recorded, with its process, route, model, tokens, cost and times
   - packages: internal/core/runner, internal/core/implement/loop, internal/core/interview
   - tests: the dispatcher over a fake harness for a finished, failed, killed and host-handed launch, the subscription, key and unknown routes, the run-state schema unchanged
   - needs: 1
   - landed: 7c21e2642
3. MUST: an entry names the session that asked for it
   - packages: internal/core/sessionname, internal/core/runner
   - tests: the exported session id with its persona name, a host found above the process, and the run id when neither answers
   - needs: 1
   - landed: aa57aaffc
4. MUST: retention never drops an entry inside the longest cap period
   - packages: internal/core/runner
   - tests: the oldest whole files pruned past days and past count, nothing pruned inside 32 days whatever is set, a setting below the floor read as the floor and named
   - needs: 1
   - landed: fb6305133
5. MUST: `abcd agents` lists and totals the launches
   - packages: internal/core/runner, internal/surface/cli, internal/core/surface, commands, docs/reference/cli, .abcd/development/release, .abcd/development/release-gate, .abcd/development/brief/04-surfaces, .abcd/rules.json
   - tests: the verb bare, with --run, --session and --all, in text and --json; money summed for keys alone; an unreadable and a newer entry named; running against lost judged on the process; two launches for one await named
   - needs: 1, 2
   - landed: 981d898ae
6. AFTER: the status line's row, its cached read and its explanation, with the token basis as a setting
   - packages: internal/core/statusline, internal/surface/cli
   - tests: the row from the ledger alone, within the status command's deadline, cached for the configured interval
   - needs: 5
7. AFTER: an unattended run's summary carries the ledger's totals
   - packages: internal/core/implement/loop, internal/surface/cli
   - tests: a drain's and a driven build's summary naming their totals and the verb
   - needs: 5
