---
id: spc-2611201455051897
slug: the-drain-lands-one-pull-request-per
intent: itd-2611201450052326
origin: researcher-authored
production_mode: hand-written
---
# The drain fixes several small issues of one kind in one place with one pull request

## Summary

The drain opens one issue run per issue today. This spec lets one issue run carry several issues of one category in one directory: a lead issue and up to `drain.bundle_max - 1` members. The run keeps one lane, so the bundle gets one branch, one review set per round, one preflight and one pull request. Each member keeps what a single issue has today: its own eligibility, its own judgement, its own reservation, its own resolution commit and `Resolves:` line, and its own hand-back. A bundle that fails as a whole is split back into single lanes. A drain whose cap is 1, or that finds no second issue, behaves as it does today. The drain records the figures the intent is measured by. Six steps, each one pull request.

## Approach

### What is on the default branch today, and every place that assumes one issue per lane

Read at 81739e558.

- **The drain's lane** is `DrainLane{Issue, RunID, OpenedAt, Outcome, PR}` in `internal/core/implement/loop/drain.go`. The drain state's version is 3. `drainMove` walks the eligible set in drain order. For the first issue it can take, it reserves the issue (`reserveForJudgement`, footprint from `issueFootprint`, which is the one path its `found_at` names). It then asks one judgement (`st.Judging` holds one request). On a no it calls `startLane(repoRoot, v.ID, lane)` and returns. A move opens at most one lane.
- **Taken and passed over** are judged by issue id against `DrainLane.Issue` (`DrainState.taken` in `drain_readmit.go`) and against a run's `Key` (`r.Key == v.ID` in `drainMove`).
- **The run** is `State` in `state.go`, version 29. An issue run has `Key` set to the issue and `Issue()` returns it. It has one `ReservedAt`.
- **The reservation** lives in `reservation.go`. It is `refs/heads/abcd-reserved/<key>` for `run.Key` alone, and it is renewed, fenced before the push and the arming, and tombstoned or released per run. `widen.go` widens that one hold to what the branch changes beyond it (`receiptHoldGaps`). The drain's lease renewal (`drain_lease.go`) and `renewruns.go` renew per run.
- **The brief** is rendered by `issuebrief.go`, which reads one issue's record and remedy.
- **The receipt** is checked in `receipt.go`. It must resolve the run's issue (`resolvesIssue`), and `handBackGaps` refuses resolves beside a `handback`, so a lane either fixes everything or hands everything back. `Lane.Resolves` is already a list, capped at 50.
- **The hand-back** is `handBackLane` in `handback.go`, which hands back the whole lane. `laneOutcome` in `drain.go` reads one hand-back per lane, `routeHandBack(l.Issue, …)` routes it, and `releaseHandedBack` releases `run.Key`. `stuck.go`'s `handBackStuck` hands back the whole lane. The spot-check's `routedLane` (`handback.go`) maps a route to a lane by issue.
- **The landing** in `land.go` already writes one `Resolves:` trailer per entry of `lane.Resolves` and resolves each record (around lines 586 to 607). `prTitle` and `prBodyParked` name `st.Issue()` alone.
- **The finish** is `finishMerged` in `drain_finish.go`, which marks one lane done per merged pull request. `drainCounts` counts lanes and pull requests, not issues.

### The run carries members (step 1; decisions 1, 8 and 9)

`State` gains `bundle`, a list of members. Each member has the issue id, its footprint, its `reserved_at`, a `hand_back` once it is handed back alone, and `split` once the drain has split the run. `State.Issues()` returns the lead and every member not handed back, and every reservation act that iterates the run's reservations uses it. That covers the renewal, the push fence, the land fence, the release on landing, the release on hand-back and the widening. The widening judges the branch against the members' footprints together and widens the lead's hold. `covers(id)` replaces `r.Key == v.ID` wherever the drain asks whether a run holds an issue. A run with no members writes no `bundle` key, so its file differs from today's only in `schema_version`.

Migration: version 30. Version 29 is its strict subset, read as a run with no members and written back at 30. A file of version 29 or lower carrying `bundle` was not written by that version and is refused naming the key. Releases that write 29 refuse a version-30 file as a newer abcd's before any strict decode. This is the same pattern as versions 26 to 29 in `state.go`. The lane receipt stays at version 1 (decision 9).

### The brief, the receipt and the landing of a bundle (step 2; decision 6)

The issue brief renders every member's record and remedy from the lane's base, in drain order, with the lead first. It asks for one commit per member, with each commit named in that member's resolution. It asks the implementer to name every member exactly once, in `resolves` or in `handbacks`. `handbacks` is a list of `{issue, kind, reason, home}`, each checked as `handBackGaps` checks a lane's hand-back. The receipt check refuses:

- a member named in neither list, or in both;
- an issue that is not a member;
- a top-level `handback` in a run with members;
- `handbacks` in a run without members.

A member handed back is recorded on the run's member, and its reservation is released with its tombstone through the same release a lane's hand-back uses. If every member, the lead included, is handed back, the lane is handed back as today. The landing already writes one `Resolves:` per resolution. `prTitle` names the area and the count, for example `fix(.abcd/development/brief/04-surfaces): 4 issues, iss-A and 3 more`. The body lists each member with its resolution, and gives a `Refs:` line for each member handed back, so every record id the text names carries a trailer.

### The cap and who joins (step 3; decisions 2, 3, 4 and 9)

`drain.bundle_max` is a new key under a `drain` namespace. It is resolved through `layered` as the pace keys are (`pace.go`): the repository's `.abcd/config.json` over the machine's `config.json` over the bundled 5, from 1 to 10. The namespace is claimed, so a misspelt key is refused naming the file. The cap is read once when a drain starts and kept on its state as `bundle_max`, with its layer and origin, as `Pace` is kept. A running drain keeps its cap.

`bundleCandidates` is a pure function over the plan's dispositions, the lead and the cap. It returns the issues after the lead in drain order that:

- are eligible;
- have the lead's category exactly;
- have the lead's area: `path.Dir` of the `issueFootprint` path, or the path itself when it is a directory in the checkout;
- are not of the `security` category;
- have an area other than the repository root.

The lead itself must pass the same area and category tests to be bundled at all; otherwise its lane opens alone. `abcd drain --dry-run` shows, under each eligible issue that would lead, the members its next lane would take. It also says that judgements, reservations and `reservation.max_live` may make the bundle smaller.

Migration: drain state version 4 is defined whole in this step: the drain's `bundle_max`, a lane's `bundle` (member ids) and `split`, the drain's `forming`, and a lane's `ci_minutes`. Each field is written only by the step that uses it, so no later step raises the version again. Version 3 is its strict subset, read as a drain with the bundled cap's layer recorded as `bundled` and no bundles, and written back at 4. A version 3 file carrying any of them is refused naming the key, and releases that write 3 refuse 4 as a newer abcd's.

### The drain forms and opens bundles (step 4; decisions 3, 4, 5 and 8)

After the lead's judgement answers no, the drain does not start the lane at once when `bundleCandidates` is not empty and the cap is above 1. It records `forming`: the lead, its hold, and the members judged so far with their holds. The rest of the move, and each move after it, takes the next candidate the way the drain takes an issue today:

- it passes over a candidate that is held, skipped, lapsed or already run, as now;
- it reserves the candidate (`reserveForJudgement`) and asks its judgement;
- a no adds it to `forming`;
- a yes routes it as today, and it leaves the bundle.

The holds of the lead and the members are kept across moves. They are added to the move's `judgeHolds` as kept, so `settle` does not release them on the way out. `renewInProgress` renews them at half their lease. The lane opens when no candidate is left, the cap is reached, or the person's live reservations would reach `reservation.max_live` with the next one. At the cap of reservations the bundle opens as it stands rather than waiting. `startLane` takes the members through `Options`, and `Start` writes them on the new run with the hold each was reserved on, as `heldReservation` does for one issue today.

A stop, a pace window that elapses, or an origin that does not answer while a bundle is forming keeps `forming` and its holds, as `st.Judging` is kept today. A drain that ends releases every forming hold plainly, with no tombstone. `DrainState.taken` and the run lookups count a member as taken. The lane's routing reads each member handed back alone (step 2's record) and routes it with `routeHandBack(member, "lane", …)`. A whole-lane hand-back routes every member's hand-back in this step. Step 5 replaces that with the split. `finishMerged` marks the lane done for every member, and `routedLane` and `DrainSummaryLine` name every member.

A cap of 1, a lead that is not bundleable, or a lead with no candidates goes through the path the drain takes today, with no `forming` written. The test for criterion 8 compares that drain's state, run state, brief, receipt check, pull request text and summary with a fixture of today's, apart from `schema_version`.

### A failed bundle is split (step 5; decision 7)

When a lane with members is handed back as a whole, because `VerdictUnachievable` at the fix-round cap or `handBackStuck`, the drain does not route a hand-back. It marks the drain lane `split`. It releases every member's reservation plainly, with no tombstone, so no other copy reads the issues as a person's. It keeps each member's judgement, and the judgements are kept by remedy digest, so none is asked again for the same remedy. Each member is then taken up again in drain order as a bundle of one, which the drain never bundles again while it runs. A member that then fails its own fix rounds is handed back as today. A lane handed back because every member was handed back in its receipt (step 2) is not a split. Those members are a person's.

### The drain's figures (step 6; decision 10)

`DrainCounts` gains:

- `issues_resolved`, the resolutions on lanes that merged;
- `preflights`, the lanes whose landing reached its gate;
- `ci_minutes`, summed;
- the three figures per resolved issue.

`DrainPRHandBacks` (`hand_backs_by_pr`) gains the issues each pull request resolved. When `finishMerged` reads a merged pull request, it reads the pull request's check runs from the forge: the `statusCheckRollup` contexts, with `startedAt` added to the query in `internal/core/landing` and to `internal/ghcontract/graphql-fields.txt`. It stores the sum of `completedAt - startedAt` on the drain lane as `ci_minutes`. A check that never started or never completed adds nothing, and the summary says how many were left out. The merge queue's own runs are not counted, since they run on another ref. `abcd drain` prints the per-issue figures in its summary, and `--json` carries them. The intent's measurement over forty resolved issues is read from these counts, and no step builds a report for it.

### Coordination with other work in the loop package

`drain.go`, `state.go`, `reservation.go` and `land.go` are edited often by drain lanes and by other intents' steps. So each step names its files exactly, starts from the default branch as it then stands, and reserves through `abcd build`, which reserves the spec's footprint for the step. Steps 1, 2 and 3 have no file in common except `drain.go`, which steps 3 and 4 both edit, so 4 waits on 3.

## How the Acceptance Criteria are met

The criteria are numbered in the order the intent lists them.

1. Who joins: `bundleCandidates` (step 3), applied by the forming moves (step 4). Proved by a ledger with a lead and issues of the same category in the same directory, one of another category, one in another directory, a security issue and a root file, with only the first group joining.
2. The cap's layers: the `drain` namespace and its resolution (step 3). Proved by the repository's value, the machine's, both, neither, 0, 11 and a misspelt key.
3. `reservation.max_live` headroom: the forming loop opens the lane rather than reserving past the cap (step 4). Proved with `reservation.max_live` of 3 and a cap of 5.
4. One branch, review set, preflight and pull request with one `Resolves:` per resolved member: one run, one lane (step 1), and the landing and pull request text (step 2). Proved by landing a fixture bundle of three and reading the records commit's trailers and RS001 over its diff.
5. A reservation per member, renewed, fenced and released: `State.Issues()` across the reservation acts (step 1) and the forming holds (step 4). Proved by a fixture run of three members on a fixture origin.
6. A member handed back alone: the receipt's `handbacks` and the per-member release (step 2), routed by the drain (step 4). Proved by a receipt that resolves two members and hands one back.
7. A failed bundle is split: step 5. Proved by a bundle whose fix rounds are spent, then each member's single lane, one passing and one handed back.
8. A bundle of one is today's drain: the unchanged path (step 4) and the absent fields (steps 1 and 3). Proved by the fixture comparison in step 4.
9. The figures: step 6. Proved with `pr-state-1018-merged-timed.json` extended with `startedAt`, and a drain of two lanes, one of them a bundle of three.

## Open Questions

None for the build. The bar in the intent's Mechanism is for the person to confirm on 22 Nov, and it changes no step.

## Footprint

- packages: internal/core/implement/loop/state.go, internal/core/implement/loop/reservation.go, internal/core/implement/loop/widen.go, internal/core/implement/loop/renewruns.go, internal/core/implement/loop/issuebrief.go, internal/core/implement/loop/receipt.go, internal/core/implement/loop/handback.go, internal/core/implement/loop/handback_release.go, internal/core/implement/loop/land.go, internal/core/implement/loop/drain_bundle.go, internal/core/implement/loop/drain.go, internal/core/implement/loop/drain_hold.go, internal/core/implement/loop/drain_lease.go, internal/core/implement/loop/drain_readmit.go, internal/core/implement/loop/drain_finish.go, internal/core/implement/loop/drain_text.go, internal/core/implement/loop/loop.go, internal/core/implement/loop/drain_split.go, internal/core/landing/land.go, internal/ghcontract/graphql-fields.txt, internal/surface/cli/drain.go, commands/drain.md, .abcd/development/brief/04-surfaces/35-drain.md, .abcd/development/brief/05-internals/03-configuration.md
- tests: internal/core/implement/loop/state_bundle_test.go, internal/core/implement/loop/reservation_bundle_test.go, internal/core/implement/loop/issuebrief_bundle_test.go, internal/core/implement/loop/receipt_bundle_test.go, internal/core/implement/loop/land_bundle_test.go, internal/core/implement/loop/drain_bundle_test.go, internal/core/implement/loop/drain_schema_test.go, internal/core/implement/loop/drain_bundle_form_test.go, internal/core/implement/loop/drain_bundle_one_test.go, internal/core/implement/loop/drain_bundle_split_test.go, internal/core/implement/loop/drain_bundle_counts_test.go, internal/surface/cli/drain_surface_test.go, internal/surface/cli/drain_counts_surface_test.go

## Steps

1. A run carries members
   - criteria: 4, 5, 8
   - needs: none
   - packages: internal/core/implement/loop/state.go, internal/core/implement/loop/reservation.go, internal/core/implement/loop/widen.go, internal/core/implement/loop/renewruns.go
   - tests: internal/core/implement/loop/state_bundle_test.go: a version-29 file reads as a run with no members and is written back at 30; a version-29 file carrying `bundle` is refused naming it; a version-30 file is refused by a reader at 29 as a newer abcd's; a run with no members writes no `bundle` key; `Issues()` and `covers` over a lead, members and a member handed back. internal/core/implement/loop/reservation_bundle_test.go: on a fixture origin, a run with three members renews each at half its lease, the push fence and the land fence refuse when any member's hold is another's and name it, the release on landing tombstones every member, and the widening judges the branch against the members' footprints together. Every existing reservation test passes unchanged.
2. The brief, the receipt and the landing of a bundle
   - criteria: 4, 6
   - needs: 1
   - packages: internal/core/implement/loop/issuebrief.go, internal/core/implement/loop/receipt.go, internal/core/implement/loop/handback.go, internal/core/implement/loop/handback_release.go, internal/core/implement/loop/land.go
   - tests: internal/core/implement/loop/issuebrief_bundle_test.go: the brief renders each member's record and remedy from the lane's base in drain order, asks for one commit per member, and names `handbacks`; a run without members renders today's brief byte for byte. internal/core/implement/loop/receipt_bundle_test.go: a receipt that resolves two members and hands one back is taken and the member's reservation is released with its tombstone; a member in neither list, one in both, a non-member, a top-level `handback` with members, and `handbacks` without members are each refused naming the gap; every member handed back hands the lane back. internal/core/implement/loop/land_bundle_test.go: the records commit carries one `Resolves:` per resolved member and a `Refs:` per member handed back; the pull request's title names the area and the count, and its body lists every member.
3. The cap and who joins
   - criteria: 1, 2
   - needs: none
   - packages: internal/core/implement/loop/drain_bundle.go, internal/core/implement/loop/drain.go, internal/surface/cli/drain.go, commands/drain.md, .abcd/development/brief/04-surfaces/35-drain.md, .abcd/development/brief/05-internals/03-configuration.md
   - tests: internal/core/implement/loop/drain_bundle_test.go: `drain.bundle_max` from the repository, the machine, both and neither; 0, 11 and a misspelt key refused naming the file; the cap kept on the drain state with its layer and not changed by a running drain; `bundleCandidates` over a ledger with one category in one directory, another category, another directory, a security issue, a root file, a lead that is not bundleable, and a cap of 1. internal/core/implement/loop/drain_schema_test.go: version 4's new fields; a version-3 file read and written back at 4; a version-3 file carrying any new field refused naming it; version 4 refused by a reader at 3. internal/surface/cli/drain_surface_test.go: `--dry-run` shows the members each leading issue's next lane would take, and the `--json` envelope carries them. The command page and the two brief pages state the key, its range, its layers and the rule for who joins.
4. The drain forms and opens bundles
   - criteria: 1, 3, 4, 5, 6, 8
   - needs: 1, 2, 3
   - packages: internal/core/implement/loop/drain.go, internal/core/implement/loop/drain_hold.go, internal/core/implement/loop/drain_lease.go, internal/core/implement/loop/drain_readmit.go, internal/core/implement/loop/drain_finish.go, internal/core/implement/loop/drain_text.go, internal/core/implement/loop/handback.go, internal/core/implement/loop/loop.go, internal/surface/cli/drain.go, commands/drain.md, .abcd/development/brief/04-surfaces/35-drain.md
   - tests: internal/core/implement/loop/drain_bundle_form_test.go: a lead and three candidates form over moves, with each reserved before its judgement, a yes routed and left out, and the lane opening with the rest; the forming holds kept by `settle` and renewed at half their lease; a stop, an elapsed window and an origin that does not answer keep `forming`; a drain that ends releases forming holds with no tombstone; `reservation.max_live` of 3 with a cap of 5 opens a bundle of three without waiting; a member counted as taken; a member handed back in the receipt routed alone, and the lane's other members finished when its pull request merges; a whole-lane hand-back routes every member. internal/core/implement/loop/drain_bundle_one_test.go: with a cap of 1, and with a lead that has no candidates, the drain's state, the run's state, the brief, the pull request's text and the summary equal a fixture of today's, apart from `schema_version`.
5. A failed bundle is split
   - criteria: 7
   - needs: 4
   - packages: internal/core/implement/loop/drain_split.go, internal/core/implement/loop/drain.go, internal/core/implement/loop/drain_readmit.go
   - tests: internal/core/implement/loop/drain_bundle_split_test.go: a bundle of three whose fix rounds are spent, and one handed back as stuck, are each split with no tombstone pushed, their reservations released and their judgements kept with none asked again; each member then opens as a bundle of one that is never bundled again by that drain; of two singles, one lands and one is handed back as today; a lane whose receipt handed every member back is routed, not split.
6. The drain's figures
   - criteria: 9
   - needs: 4
   - packages: internal/core/implement/loop/drain_finish.go, internal/core/implement/loop/drain.go, internal/core/landing/land.go, internal/ghcontract/graphql-fields.txt, internal/surface/cli/drain.go, commands/drain.md
   - tests: internal/core/implement/loop/drain_bundle_counts_test.go: on a drain of a single lane and a bundle of three that both merged, `issues_resolved` is 4, pull requests per resolved issue 0.5, preflights per resolved issue 0.5, and the CI minutes are the check runs' durations from a fixture of the forge's reply extended with `startedAt`; a check that never started is left out and counted as left out. internal/surface/cli/drain_counts_surface_test.go: the summary prints the three figures per resolved issue, and `--json` carries them.

## Size

Six pull requests. Steps 1, 2 and 6 are contained: each widens code that already handles a list (reservations per run, resolutions per lane, a pull request's checks). Step 3 is a setting and a pure function. Step 4 is the largest, because forming spans moves and changes what `settle` releases. It reuses the reservation and judgement path a single issue takes, so the new code is the loop over candidates and the kept holds. Step 5 is a new outcome and its release. No step runs a preflight in this planning lane. Each step's builder runs the repository's definition of done.
