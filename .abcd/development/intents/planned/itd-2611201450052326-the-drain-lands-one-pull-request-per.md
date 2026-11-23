---
id: itd-2611201450052326
slug: the-drain-lands-one-pull-request-per
spec_id: spc-2611201455051897
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2611201421262152, iss-2610161652341096]
origin: extracted-from-record
production_mode: hand-written
impact: additive
---

# The drain fixes several small issues of one kind in one place with one pull request

## Press Release

> **The drain fixes several small issues of one kind in one place with one pull request.** A person runs `abcd drain`. When the ledger holds several eligible issues of the same category in the same place, such as five corrections to one chapter of the brief or three bugs in one package, the drain takes them together. They share one branch, one review, one preflight and one pull request, which carries a `Resolves:` line for each issue. Each issue is still judged and reserved on its own. If one issue in the group needs a person, only that issue is handed back and the rest still land. If the group as a whole fails its review, the drain tries each issue again on its own, so one bad fix does not cost the others. The size of a group is one setting, `drain.bundle_max`, five unless a person changes it. Setting it to 1 makes the drain work exactly as it does today.

> "On Thursday the drain opened forty-five pull requests, and most of them changed one sentence of the brief and one test. Now five sentences in one chapter come as one pull request, and I review them together." — Iris, a product thinker (they/them)

> "One preflight and one CI run now cover five fixes. The drain's summary tells me how many pull requests, preflights and CI minutes each resolved issue cost, so I can see whether the groups pay for themselves." — Nia, a technical facilitator (they/them)

## Why This Matters

Between 19 Nov 01:06Z and 20 Nov 14:04Z the drain (itd-82) opened 45 pull requests from build/run branches, one per issue, and 42 had merged by 14:21Z. Each took 0.9 to 1.9 hours from opening to merge, and 19 of the 42 took over 1.5 hours. Most were documentation fixes. 33 of the 45 changed exactly one brief or docs file, the issue's record and one Go test that pins the corrected sentence. 3 more paired a brief file and a test with other Go changes, and 7 changed docs only. The pinning test makes CI's fail-closed classifier treat the change as code and run the full check job (42 minutes on #1694), where a docs-only change runs it in 19 (#1698). Each issue also pays for its own local `make preflight` (about 40 minutes), its own review pair and its own merge-queue entry (iss-2611201421262152).

iss-2610161652341096 asked the drain to bundle issues whose `found_at` paths overlap. Most of the issues above touch different files in one place, the brief's chapters, so that rule alone would bundle few of them. The person ruled on 20 Nov 2026 that the drain may bundle eligible issues of one category in one area, which is one brief chapter folder or one package, up to a setting whose default is five (dcn-2611201429107457).

There is room for it. On 20 Nov the open ledger held 1,858 issues, and 1,647 of them share both their category and the directory of their `found_at` with at least one other. For example, 204 inconsistencies name a page in `.abcd/development/brief/04-surfaces/` and 48 bugs name a file in `internal/core/implement/loop/`. Not all of them are eligible for the drain, but the issues the drain did fix come from these groups.

### Who it is for

- The product thinker (Iris), who reviews and merges what the drain lands, and reads fewer, larger pull requests that each stay in one place.
- The technical facilitator (Nia), who runs the drain and pays for its preflights and CI time, and who reads the drain's figures to judge the setting.

### What a person does

- Runs `abcd drain` or `abcd drain --drive` as today.
- Optionally sets `drain.bundle_max` in the machine's `config.json`, from 1 to 10, and may lower it for one repository in its `.abcd/config.json`, never raise it (dcn-2611222342270127).

### What a person sees

- A lane that names its lead issue and the issues bundled with it.
- One pull request with a `Resolves:` line for each issue it fixed.
- A hand-back for an issue that needs a person, routed as today, while the other issues in the lane still land.
- In the drain's summary: the issues it resolved and handed back, and its pull requests, preflights and CI minutes per resolved issue.

### What a person never has to know

- How the drain forms a group, and that each member was judged and reserved separately.
- That a failed group was split and its issues tried one at a time.

## Mechanism

We expect fewer pull requests, preflights and CI minutes per resolved issue, because a preflight, a review pair, a CI check job and a merge-queue entry cost about the same for one small fix as for five in the same place, and the fixes the drain makes most often are small and cluster by place: 33 of the last 45 changed one docs file and one test. We expect no more hand-backs per issue than today, because each member is still judged, reserved and resolved on its own, a member that needs a person is handed back alone, and a group that fails as a whole is split back into single lanes.

Shown wrong if, over the first forty issues the drain resolves with bundling on at the default, the pull requests per resolved issue do not fall below 0.5, or the share of issues handed back is higher than over the forty resolved before it. Pull requests per resolved issue counts merged pull requests only: the drain's pull requests that merged over the issues they resolved. The share handed back is counted per issue: the issues handed back over the issues resolved plus the issues handed back, where a member handed back alone counts once and a split lane's issues count once, in the single lanes they are taken up in.

The comparison is made once, by hand, from the forge, the way iss-2611201421262152 measured its baseline, and no code reads it (dcn-2611201605030535). The drain's pull requests are those whose records commit carries the `Admitted-by: drain` trailer, which tells them from `abcd build`'s on the same `build/run` branches; `gh pr list` gives each one's files, its `Resolves:` and `Refs:` lines and its opening and merge times. The forty before are the last forty issues resolved by drain pull requests that merged before step 4, the first step that can open a bundle. The forty with bundling on are the first forty resolved by drain pull requests opened by the drains this intent's Audit Notes names as running a build carrying step 6, which raises the default to 5, taken in the order their pull requests merged, with the pull request that reaches forty counted whole. Each forty's span runs from the first of its pull requests opening to the last merging, and a hand-back counts in a forty when it falls in that span; an issue counts once in a forty however often it was handed back. A member handed back alone is counted from its bundle pull request's `Refs:` lines, which step 2 writes and the forge keeps, merged or closed, leaving out a `Refs:` naming the record the pull request's `Admitted-by:` names, which is the admission's; its tombstone is not counted again. Any other lane's hand-back is a tombstone carrying `Abcd-Handed-Back` on origin's reservation branches, released in the span, either of a lane's own kind (`unachievable`, `stuck` or `closed`) or of a judgement's kind over a reservation whose history beneath it holds a renewal carrying `Abcd-Stage`, since a release never carries the stage (`37-reserve.md`). One blind spot remains, in both forties alike: a single lane's receipt hand-back made before its first renewal, and a member's in a bundle that opened no pull request, leave a reservation and a tombstone with no stage beneath it, exactly as a judgement's hand-back does, and are not counted. The reservation sweep deletes a hand-back's branch once its record is finished, so the technical facilitator lists origin's hand-back tombstones for the older forty when step 4 merges, and for the newer forty at least once a day while it runs, and records both lists in Audit Notes. Only a hand-back swept before it was listed is lost, so both shares are floors, and the count hides a rise in the share only when more of the newer forty's hand-backs were finished and swept before they were listed than of the older forty's. The technical facilitator records the count once forty issues have resolved after the default is raised, which falls after this intent is in `shipped/`, so Audit Notes gains it then, apart from the intent-auditor's fill on the move. Audit Notes also names the drains counted, because a drain still on an older build opens single pull requests that the forge cannot tell apart from a drain that found no members.

## Scope Conditions

- Issues the drain's own rule already makes eligible. Bundling widens no eligibility: an issue the drain would not fix alone is never fixed in a bundle. <!-- cond: cond-2611201455058439 -->
- Members of one category and one area, as the person ruled on 20 Nov 2026 (dcn-2611201429107457). The area is the directory holding the path an issue's `found_at` names (decision 2). An issue whose `found_at` names a root file or no path is never bundled, and a security issue is never bundled (decision 3). <!-- cond: cond-2611201455054086 -->
- A bundle no larger than `drain.bundle_max` and no larger than the person's free reservations under `reservation.max_live` (decision 4). <!-- cond: cond-2611201455050410 -->
- A repository with a remote named `origin`, where reservations are made. Without one the drain reserves nothing and a bundle has no reservations to take, as a single lane has none today. <!-- cond: cond-2611201455058214 -->
- The baseline figures were measured on one machine over 37 hours of one autonomous run, on pull requests that were mostly documentation fixes (iss-2611201421262152). Nothing here is evidence for other mixes of issues until the measurement covers them. <!-- cond: cond-2611201455053642 -->

## What's Out of Scope

- Bundling issues of different categories, or issues in different areas, even when their files overlap.
- Changing which issues the drain finds eligible, its judgement or its routes.
- Changing the drain's order beyond one thing. The order still chooses each lane's lead, and a lead's members are drawn after it from its category and area at either fixable severity, so a member can be taken ahead of an older or a `nitpick` issue in another area. adr-2609291342092738 carries a dated amendment of 20 Nov 2026 for it (dcn-2611201515310051).
- Making CI's classifier treat a docs fix with a pinning test as docs-only. That is a separate change to `.github/workflows/ci.yml`.
- Bundling intents, or building several spec steps of one intent in one pull request.
- A flag for the cap. The setting is the only way to change it.

## SOTA

Surveyed on 20 Nov 2026 from the tools' published documentation as the technical facilitator knows it, not re-fetched for this record. The independent fit-challenge is still owed before step 4 is built.

- **Grouped dependency updates.** Dependabot's `groups` setting and Renovate's `groupName` rules put the updates that match one pattern into one pull request, so one CI run and one review cover them. This is the shape the person ruled for the drain: one kind in one place, one pull request. Both tools group by a declared rule and not by inferred overlap, which supports grouping by category and directory over overlap of `found_at` (iss-2610161652341096).
- **Batching merge queues.** Prow's Tide tests a batch of pull requests together and merges the batch when it passes, and falls back to one at a time when it fails. bors-ng bisects a failed batch. GitHub's merge queue removes a failing group's changes and retries the rest. Zuul's dependent pipelines test changes speculatively and eject the one that fails. Each keeps one failing change from failing the others, which is decision 7's split.
- **What does not carry over.** Those queues batch finished changes, and the drain bundles issues before they are fixed, so one implementer writes every fix in the bundle. That is why a member can also be handed back alone in the receipt (decision 6), which the queues have no counterpart for.

**Path 3, bespoke in the drain, with the grouping rule from grouped dependency updates and the failure split from batching merge queues.** There is nothing to adopt: the unit being grouped is an issue run inside abcd's own loop. No dependency.

## Acceptance Criteria

- **Given** eligible issues of one category whose `found_at` paths lie in one directory, and `drain.bundle_max` of 2 or more, **when** the drain opens its next lane, **then** that lane carries the lead issue and, in drain order, each further issue of the same category and directory that is eligible, unheld, judged no and reserved, up to the cap. An issue of another category, another directory, the security category, a root file or no path does not join it (decisions 2 and 3).
- **Given** `drain.bundle_max` set in the repository's `.abcd/config.json`, in the machine's `config.json`, in both, or in neither, **when** a drain starts, **then** the cap is the machine's value, or the bundled 5 where the machine sets none, lowered by the repository's value where that is lower, a repository value above it is ignored and named, the cap is kept on the drain's state with the layer that set it, and a value outside 1 to 10 is refused naming the file (decision 4, dcn-2611222342270127).
- **Given** a person whose free reservations under `reservation.max_live` are fewer than the cap, **when** the drain forms a bundle, **then** the bundle opens with no more members than those reservations allow, and the drain does not wait for more (decision 4).
- **Given** a bundle of several issues, **when** its lane runs, **then** it has one branch, one review set per round, one preflight and one pull request, and that pull request carries one `Resolves:` line for each issue the lane resolved, so RS001 finds each record in `resolved/` in the same diff (decision 1).
- **Given** a bundle of several issues, **when** the drain forms it, **then** every member holds its own reservation on `origin` with its own `found_at` path as the footprint, and each reservation is renewed, fenced before the push and the arming, and released as the lane ends, as a single issue's is today (decision 8).
- **Given** a bundle lane whose implementer finds that one member needs a person, in its first round or a later fix round, and whether that member is the lead or another, **when** its receipt hands that member back, **then** the drain routes that member's hand-back as it routes a lane's hand-back today and releases its reservation with its tombstone, the member's resolution and its commit are gone from the lane, the paths the lane widened stay held, and the lane lands the other members (decisions 6 and 13).
- **Given** a bundle lane that is handed back as a whole, because its fix rounds are spent or it is stuck, whether or not a member was handed back alone before, **when** the step hands it back and the drain reads it, **then** no member is handed back or tombstoned, the step itself pushes no tombstone, and each member still held has its reservation released plainly; its judgement is kept, and each is taken up again on its own, never bundled again by that drain, so only an issue that fails its own fix rounds is handed back. A bundle the drive drops because one issue's take-up backed off or its record moved is split the same way: the take-up releases what it re-held before it refuses, that issue alone is routed as a dropped lane is today, and the rest are taken up on their own (decisions 7, 13 and 15).
- **Given** `drain.bundle_max` of 1, or no other issue of the lead's category and directory, **when** the drain runs, **then** it writes the same drain state, run state, brief, receipt check, pull request and summary as the drain did before bundling, apart from the schema versions, the cap the drain's state records with its layer, and the figures the summary and its `--json` add by criterion 9 (decision 9).
- **Given** a drain whose lanes have merged, **when** a person reads its summary or its `--json`, **then** it gives that drain's pull requests opened, issues resolved, issues handed back, counted per issue, and pull requests closed unmerged, and, per resolved issue, the pull requests, the preflights and the CI minutes, counted over merged lanes only, the CI minutes read from the forge's checks as each pull request merged; a lane still in progress or parked is in no per-issue figure, and nothing is kept across drains (decisions 10 and 14).

## Open Questions

_None for the build. One for the person, recorded under Decisions: whether 0.5 pull requests per resolved issue is the bar the Mechanism should be judged by._

## Decisions

Recorded on 20 Nov 2026 by the technical facilitator, on the person's delegation while they are away until 22 Nov. Each answer is grounded in the code or in common practice, and is open to reversal when the person reviews it.

1. **A bundle is one run with members** (dcn-2611201453477828). One issue run keyed by its lead issue carries the others. A lane already keeps a list of resolutions and its landing writes one `Resolves:` line per entry (`internal/core/implement/loop/land.go`), so one run gives one branch, one review set, one preflight and one pull request without new coordination.
2. **A bundle's area is the directory of the `found_at` path** (dcn-2611201453486298). A Go package is its directory, and a brief chapter folder is the directory holding its pages. Same-file and file-and-directory overlap, which iss-2610161652341096 asked for, fall inside it. A root file or no path never bundles.
3. **Who joins** (dcn-2611201453492456). Members are drawn in drain order after the lead. Each must be eligible on its own, match the lead's category and area exactly, pass its own judgement and take its own reservation. The security category never bundles, because an embargoed or private fix must not share a public pull request with other work.
4. **The cap** (dcn-2611201453495520). `drain.bundle_max`, 1 to 10, set by the machine's `config.json` over the bundled 5 (1 until steps 5 and 6 land, decision 12), which a repository's `.abcd/config.json` may only lower: a higher repository value is ignored and named, because a merged commit must not raise how much an unattended drain bundles (dcn-2611222342270127, superseding this decision's first precedence), fixed when a drain starts, with no flag. 1 turns bundling off. A bundle opens smaller rather than waiting when `reservation.max_live` leaves fewer reservations. The ceiling of 10 is the default `reservation.max_live`.
5. **Forming over several moves** (dcn-2611201453501725). After the lead's no, the drain reserves and judges each next candidate in turn, keeps the holds of members already judged, and opens the lane when no candidate is left or the cap is reached. One judgement over the whole bundle was rejected, because it would change the judgement file and the routed judge.
6. **A member handed back alone** (dcn-2611201453512021). The receipt names every member once, in `resolves` or in a per-issue `handbacks` list, with one commit per resolved member.
7. **A failed bundle is split** (dcn-2611201453528321). Batching merge queues retry a failed batch in smaller parts instead of failing every change in it, for example bors-ng's batch bisection, GitHub's merge queue removing a failing group, and Zuul's dependent pipelines.
8. **One reservation per member** (dcn-2611201454578385). Every member is reserved under its own id with its own footprint, so another account's drain or `abcd build` sees each issue held, and a member handed back publishes its own tombstone. A single reservation over all the members' paths was rejected, because other copies look a held issue up by its id.
9. **Two schema raises** (dcn-2611201455394615). Run state 29 to 30 and drain state 3 to 4, each following the run state's pattern. The lane receipt keeps version 1: its per-issue `handbacks` list is accepted only in a run with members.
10. **The drain keeps its own figures** (dcn-2611201453538972). CI minutes are the sum of the pull request's check-run durations, as the forge reports them when it merges. The merge queue's own runs are not counted.

Added on 20 Nov 2026 after the plan's two reviews, on the same delegation:

11. **The order chooses each lane's lead** (dcn-2611201515310051). Members join after the lead from its category and area at either fixable severity, and adr-2609291342092738 carries a dated amendment for it. Letting members join only at the lead's severity was rejected, because a member would still pass older issues of its tier elsewhere, so it would not keep the order either.
12. **Bundling stays off by default until the split and the figures land** (dcn-2611201515430628). The bundled default of `drain.bundle_max` is 1 until steps 5 and 6 of the spec have both landed, and the later of the two raises it to 5, the person's default.
13. **The hand-back step releases a bundle's reservations itself** (dcn-2611201515431955). A whole-lane hand-back of a run with members releases every held issue plainly in the step that hands it back, because that step publishes the tombstone today before the drain reads the lane. Each reservation line names its issue, a member handed back in a fix round loses its resolution and its commit, and the widening moves to an issue still held.
14. **The drain keeps each drain's own figures, and the forty-issue comparison is counted once by hand from the forge** (dcn-2611201605030535, superseding dcn-2611201553101150 and, before it, dcn-2611201543068670, dcn-2611201533566242 and dcn-2611201515517724). Step 6 gives each drain's own figures in its summary and `--json` and keeps nothing across drains. The comparison the Mechanism is judged by is counted from the forge as iss-2611201421262152's baseline was, and recorded in Audit Notes. Four review rounds found each automated reading across drains unsound, a count from the forge sees the drains of every account and checkout, and a comparison made once does not pay for code.
15. **A bundle the drive drops is split, not routed whole** (dcn-2611201534051375). The issue the take-up was refused on is routed alone, and the others, released before the refusal, are taken up on their own.

For the person, on 22 Nov:

- **The bar.** The Mechanism says the idea is shown wrong if pull requests per resolved issue do not fall below 0.5 over the first forty resolved issues. Below 0.5 means groups that average more than two issues. The aim at the default cap of 5 is nearer 0.2, so 0.5 is a low bar that only catches bundling that barely happens. The technical facilitator chose the figure, and the count from the forge judges it once. Whether it is the right bar is the person's call.
- **The order.** From step 4 a lead's members join from its category and area at either fixable severity, so a member can be taken ahead of an older or a `nitpick` issue in another area. The technical facilitator amended the drain's declared order for it, with a dated amendment to adr-2609291342092738 (dcn-2611201515310051). The person ruled on bundling, not on the order, so it is listed here to be confirmed or reversed.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: one branch, review, preflight and pull request for several small issues of one category in one directory costs less per resolved issue than one each; shown wrong if pull requests per resolved issue stay at or above 0.5 over the first forty, or the hand-back share rises
